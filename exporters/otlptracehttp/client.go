// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlptracehttp

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
	"sync"
	"time"

	coltracepb "github.com/karelbilek/opentelemetry/proto/collector/trace/v1"
	tracepb "github.com/karelbilek/opentelemetry/proto/trace/v1"
	"google.golang.org/protobuf/proto"

	"github.com/karelbilek/opentelemetry/exporters/otlptracehttp/internal"
	"github.com/karelbilek/opentelemetry/exporters/otlptracehttp/internal/otlpconfig"
	"github.com/karelbilek/opentelemetry/retry"
)

const contentTypeProto = "application/x-protobuf"

// maxResponseBodySize is the maximum number of bytes to read from a response
// body. It is set to 4 MiB per the OTLP specification recommendation to
// mitigate excessive memory usage caused by a misconfigured or malicious
// server. If exceeded, the response is treated as a not-retryable error.
// This is a variable to allow tests to override it.
var maxResponseBodySize int64 = 4 * 1024 * 1024

// Keep it in sync with golang's DefaultTransport from net/http! We
// have our own copy to avoid handling a situation where the
// DefaultTransport is overwritten with some different implementation
// of http.RoundTripper or it's modified by other package.
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

type Client struct {
	cfg         otlpconfig.SignalConfig
	requestFunc retry.RequestFunc
	client      *http.Client
	stopCh      chan struct{}
	stopOnce    sync.Once
}

const (
	// DefaultTracesPath is a default URL path for endpoint that
	// receives spans.
	DefaultTracesPath string = "/v1/traces"
	// DefaultMaxRequestSize is the default maximum size of a serialized export
	// request, before compression.
	DefaultMaxRequestSize int = 64 * 1024 * 1024
	// DefaultTimeout is a default max waiting time for the backend to process
	// each span batch.
	DefaultTimeout time.Duration = 10 * time.Second
)

// NewClient creates a new HTTP trace client.
func NewClient(endpoint string, urlPath string, insecure bool, headers map[string]string, maxRequestSize int, timeout time.Duration, retryCfg retry.Config) *Client {
	cfg := otlpconfig.NewHTTPConfig(endpoint, urlPath, insecure, headers, maxRequestSize, timeout, retryCfg)

	httpClient := &http.Client{
		Transport: ourTransport,
		Timeout:   cfg.Traces.Timeout,
	}

	stopCh := make(chan struct{})
	return &Client{
		cfg:         cfg.Traces,
		requestFunc: cfg.RetryConfig.RequestFunc(retry.Evaluate),
		stopCh:      stopCh,
		client:      httpClient,
	}
}

// Stop shuts down the client and interrupt any in-flight request.
func (c *Client) Stop(ctx context.Context) error {
	c.stopOnce.Do(func() {
		close(c.stopCh)
	})
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	return nil
}

// UploadTraces sends a batch of spans to the collector.
func (c *Client) UploadTraces(ctx context.Context, protoSpans []*tracepb.ResourceSpans) (uploadErr error) {
	pbRequest := &coltracepb.ExportTraceServiceRequest{
		ResourceSpans: protoSpans,
	}

	rawRequest, err := proto.Marshal(pbRequest)
	if err != nil {
		return err
	}

	ctx, cancel := c.contextWithStop(ctx)
	defer cancel()

	if maxSize := c.cfg.MaxRequestSize; maxSize > 0 && len(rawRequest) > maxSize {
		return fmt.Errorf("request body too large: exceeded %d bytes", maxSize)
	}

	request, err := c.newRequest(rawRequest)
	if err != nil {
		return err
	}

	var statusCode int

	return errors.Join(uploadErr, c.requestFunc(ctx, func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		statusCode = 0
		request.reset(ctx)
		// nolint:gosec // URL is constructed from validated OTLP endpoint configuration
		resp, err := c.client.Do(request.Request)
		var urlErr *url.Error
		if errors.As(err, &urlErr) && urlErr.Temporary() {
			return retry.NewResponseError(http.Header{}, err)
		}
		if err != nil {
			return err
		}

		if resp != nil && resp.Body != nil {
			defer func() {
				if err := resp.Body.Close(); err != nil {
					uploadErr = errors.Join(uploadErr, err)
				}
			}()
		}

		statusCode = resp.StatusCode
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
				var respProto coltracepb.ExportTraceServiceResponse
				if err := proto.Unmarshal(respData.Bytes(), &respProto); err != nil {
					return err
				}

				if respProto.PartialSuccess != nil {
					msg := respProto.PartialSuccess.GetErrorMessage()
					n := respProto.PartialSuccess.GetRejectedSpans()
					if n != 0 || msg != "" {
						err := internal.TracePartialSuccessError(n, msg)
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

		switch statusCode {
		case http.StatusTooManyRequests,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout:
			// Retryable failure.
			return retry.NewResponseError(resp.Header, bodyErr)
		default:
			// Non-retryable failure.
			return fmt.Errorf("failed to send to %s: %s (%w)", request.URL, resp.Status, bodyErr)
		}
	}))
}

func (c *Client) newRequest(body []byte) (request, error) {
	u := url.URL{Scheme: c.getScheme(), Host: c.cfg.Endpoint, Path: c.cfg.URLPath}
	r, err := http.NewRequestWithContext(context.Background(), http.MethodPost, u.String(), http.NoBody)
	if err != nil {
		return request{Request: r}, err
	}

	userAgent := "karelbilek opentelemetry"
	r.Header.Set("User-Agent", userAgent)

	for k, v := range c.cfg.Headers {
		r.Header.Set(k, v)
	}
	r.Header.Set("Content-Type", contentTypeProto)

	req := request{Request: r}

	r.ContentLength = int64(len(body))
	req.bodyReader = bodyReader(body)
	req.GetBody = bodyReaderErr(body)

	return req, nil
}

// MarshalLog is the marshaling function used by the logging system to represent this Client.
func (*Client) MarshalLog() any {
	return struct {
		Type string
	}{
		Type: "otlptracehttp",
	}
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

func (c *Client) getScheme() string {
	if c.cfg.Insecure {
		return "http"
	}
	return "https"
}

func (c *Client) contextWithStop(ctx context.Context) (context.Context, context.CancelFunc) {
	// Unify the parent context Done signal with the client's stop
	// channel.
	ctx, cancel := context.WithCancel(ctx)
	go func(ctx context.Context, cancel context.CancelFunc) {
		select {
		case <-ctx.Done():
			// Nothing to do, either cancelled or deadline
			// happened.
		case <-c.stopCh:
			cancel()
		}
	}(ctx, cancel)
	return ctx, cancel
}
