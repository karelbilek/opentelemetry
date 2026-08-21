// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpmetrichttp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	colmetricpb "github.com/karelbilek/opentelemetry/proto/collector/metrics/v1"
	metricpb "github.com/karelbilek/opentelemetry/proto/metrics/v1"
	"google.golang.org/protobuf/proto"

	"github.com/karelbilek/opentelemetry/exporters/internal/partialsuccess"
	"github.com/karelbilek/opentelemetry/exporters/otlpmetrichttp/internal/oconf"
	"github.com/karelbilek/opentelemetry/retry"
)

type client struct {
	// req is cloned for every upload the client makes.
	req            *http.Request
	maxRequestSize int
	requestFunc    retry.RequestFunc
	httpClient     *http.Client
}

// Keep it in sync with golang's DefaultTransport from net/http! We
// have our own copy to avoid handling a situation where the
// DefaultTransport is overwritten with some different implementation
// of http.RoundTripper or it's modified by another package.
var ourTransport = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	ForceAttemptHTTP2:     true,
	MaxIdleConns:          100,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
}

// maxResponseBodySize is the maximum number of bytes to read from a response
// body. It is set to 4 MiB per the OTLP specification recommendation to
// mitigate excessive memory usage caused by a misconfigured or malicious
// server. If exceeded, the response is treated as a not-retryable error.
// This is a variable to allow tests to override it.
var maxResponseBodySize int64 = 4 * 1024 * 1024

// newClient creates a new HTTP metric client.
func newClient(cfg oconf.Config) (*client, error) {
	httpClient := &http.Client{
		Transport: ourTransport,
		Timeout:   cfg.Metrics.Timeout,
	}

	u := &url.URL{
		Scheme: "https",
		Host:   cfg.Metrics.Endpoint,
		Path:   cfg.Metrics.URLPath,
	}
	if cfg.Metrics.Insecure {
		u.Scheme = "http"
	}
	// Body is set when this is cloned during upload.
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, u.String(), http.NoBody)
	if err != nil {
		return nil, err
	}

	userAgent := "OTel Go OTLP over HTTP/protobuf metrics exporter/" + Version()
	req.Header.Set("User-Agent", userAgent)

	if n := len(cfg.Metrics.Headers); n > 0 {
		for k, v := range cfg.Metrics.Headers {
			req.Header.Set(k, v)
		}
	}
	req.Header.Set("Content-Type", "application/x-protobuf")

	// Initialize the instrumentation.

	return &client{
		maxRequestSize: cfg.Metrics.MaxRequestSize,
		req:            req,
		requestFunc:    cfg.RetryConfig.RequestFunc(retry.Evaluate),
		httpClient:     httpClient,
	}, err
}

// Shutdown shuts down the client, freeing all resources.
func (c *client) Shutdown(ctx context.Context) error {
	// The otlpmetric.Exporter synchronizes access to client methods and
	// ensures this is called only once. The only thing that needs to be done
	// here is to release any computational resources the client holds.

	c.requestFunc = nil
	c.httpClient = nil
	return ctx.Err()
}

// UploadMetrics sends protoMetrics to the connected endpoint.
//
// Retryable errors from the server will be handled according to any
// RetryConfig the client was created with.
func (c *client) UploadMetrics(ctx context.Context, protoMetrics *metricpb.ResourceMetrics) (uploadErr error) {
	// The otlpmetric.Exporter synchronizes access to client methods, and
	// ensures this is not called after the Exporter is shutdown. Only thing
	// to do here is send data.

	pbRequest := &colmetricpb.ExportMetricsServiceRequest{
		ResourceMetrics: []*metricpb.ResourceMetrics{protoMetrics},
	}
	body, err := proto.Marshal(pbRequest)
	if err != nil {
		return err
	}
	if maxSize := c.maxRequestSize; maxSize > 0 && len(body) > maxSize {
		return fmt.Errorf("request body too large: exceeded %d bytes", maxSize)
	}
	request, err := c.newRequest(ctx, body)
	if err != nil {
		return err
	}

	var statusCode int

	return errors.Join(uploadErr, c.requestFunc(ctx, func(iCtx context.Context) error {
		select {
		case <-iCtx.Done():
			return iCtx.Err()
		default:
		}

		statusCode = 0
		request.reset(iCtx)
		// nolint:gosec // URL is constructed from validated OTLP endpoint configuration
		resp, err := c.httpClient.Do(request.Request)
		var urlErr *url.Error
		if errors.As(err, &urlErr) && urlErr.Temporary() {
			return retry.NewResponseError(http.Header{}, err)
		}
		if err != nil {
			return err
		}
		if resp != nil {
			statusCode = resp.StatusCode
			if resp.Body != nil {
				defer func() {
					if err := resp.Body.Close(); err != nil {
						uploadErr = errors.Join(uploadErr, err)
					}
				}()
			}
		}

		if statusCode >= 200 && statusCode <= 299 {
			// Success, do not retry.

			// Read the partial success message, if any.
			var respData bytes.Buffer
			if _, err := io.Copy(&respData, http.MaxBytesReader(nil, resp.Body, maxResponseBodySize)); err != nil {
				var maxBytesErr *http.MaxBytesError
				if errors.As(err, &maxBytesErr) {
					return fmt.Errorf("response body too large: exceeded %d bytes", maxBytesErr.Limit)
				}
				return err
			}
			if respData.Len() == 0 {
				return nil
			}

			if resp.Header.Get("Content-Type") == "application/x-protobuf" {
				var respProto colmetricpb.ExportMetricsServiceResponse
				if err := proto.Unmarshal(respData.Bytes(), &respProto); err != nil {
					return err
				}

				if respProto.PartialSuccess != nil {
					msg := respProto.PartialSuccess.GetErrorMessage()
					n := respProto.PartialSuccess.GetRejectedDataPoints()
					if n != 0 || msg != "" {
						err := partialsuccess.Metrics(n, msg)
						uploadErr = errors.Join(uploadErr, err)
					}
				}
			}
			return nil
		}
		// Error cases.

		// server may return a message with the response
		// body, so we read it to include in the error
		// message to be returned. It will help in
		// debugging the actual issue.
		var respData bytes.Buffer
		if _, err := io.Copy(&respData, http.MaxBytesReader(nil, resp.Body, maxResponseBodySize)); err != nil {
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) {
				return fmt.Errorf("response body too large: exceeded %d bytes", maxBytesErr.Limit)
			}
			return err
		}
		respStr := strings.TrimSpace(respData.String())
		if respStr == "" {
			respStr = "(empty)"
		}
		bodyErr := fmt.Errorf("body: %s", respStr)

		switch resp.StatusCode {
		case http.StatusTooManyRequests,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout:
			// Retryable failure.
			return retry.NewResponseError(resp.Header, bodyErr)
		default:
			// Non-retryable failure.
			return fmt.Errorf("failed to send metrics to %s: %s (%w)", request.URL, resp.Status, bodyErr)
		}
	}))
}

func (c *client) newRequest(ctx context.Context, body []byte) (request, error) {
	r := c.req.Clone(ctx)
	req := request{Request: r}

	r.ContentLength = int64(len(body))
	req.bodyReader = bodyReader(body)
	req.GetBody = bodyReaderErr(body)

	return req, nil
}

// bodyReader returns a closure returning a new reader for buf.
func bodyReader(buf []byte) func() io.ReadCloser {
	return func() io.ReadCloser {
		return io.NopCloser(bytes.NewReader(buf))
	}
}

// bodyReaderErr returns a closure returning a new reader for buf.
func bodyReaderErr(buf []byte) func() (io.ReadCloser, error) {
	return func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(buf)), nil
	}
}

// request wraps an http.Request with a resettable body reader.
type request struct {
	*http.Request

	// bodyReader allows the same body to be used for multiple requests.
	bodyReader func() io.ReadCloser
}

// reset reinitializes the request Body and uses ctx for the request.
func (r *request) reset(ctx context.Context) {
	r.Body = r.bodyReader()
	r.Request = r.WithContext(ctx)
}
