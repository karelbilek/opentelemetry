// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelhttp

import (
	"fmt"
	"io"
	"net/http"
	"time"

	otel "github.com/karelbilek/opentelemetry"
	"github.com/karelbilek/opentelemetry/codes"
	"github.com/karelbilek/opentelemetry/propagation"
	sdkmetric "github.com/karelbilek/opentelemetry/sdk/metric"
	sdktrace "github.com/karelbilek/opentelemetry/sdk/trace"
	otelsemconv "github.com/karelbilek/opentelemetry/semconv"
	"github.com/karelbilek/opentelemetry/trace"

	"github.com/karelbilek/opentelemetry/otelhttp/internal/request"
	"github.com/karelbilek/opentelemetry/otelhttp/internal/semconv"
)

// Transport implements the http.RoundTripper interface and wraps
// outbound HTTP(S) requests with a span and enriches it with metrics.
type Transport struct {
	rt http.RoundTripper

	tracer *sdktrace.Tracer

	semconv semconv.HTTPClient
}

var _ http.RoundTripper = &Transport{}

// NewTransport wraps the provided http.RoundTripper with one that
// starts a span, injects the span context into the outbound request headers,
// and enriches it with metrics.
//
// If the provided http.RoundTripper is nil, http.DefaultTransport will be used
// as the base http.RoundTripper.
func NewTransport(base http.RoundTripper,
	eh otel.ErrorHandler,
	tracerProvider *sdktrace.TracerProvider,
	meterProvider *sdkmetric.MeterProvider,
) *Transport {
	if base == nil {
		base = http.DefaultTransport
	}

	t := Transport{
		rt: base,
	}

	t.tracer = newTracer(tracerProvider)
	meter := meterProvider.Meter(
		ScopeName,
	)
	t.semconv = semconv.NewHTTPClient(meter, eh)

	return &t
}

// RoundTrip creates a Span and propagates its context via the provided request's headers
// before handing the request to the configured base RoundTripper. The created span will
// end when the response body is closed or when a read from the body returns io.EOF.
func (t *Transport) RoundTrip(r *http.Request) (*http.Response, error) {
	requestStartTime := time.Now()

	tracer := t.tracer

	spanName := semconv.SpanName(r)
	ctx, span := tracer.Start(r.Context(), spanName, trace.WithSpanKind(trace.SpanKindClient))

	r = r.Clone(ctx) // According to RoundTripper spec, we shouldn't modify the origin request.

	var lastBW *request.BodyWrapper // Records the last body wrapper. Can be nil.
	maybeWrapBody := func(body io.ReadCloser) io.ReadCloser {
		if body == nil || body == http.NoBody {
			return body
		}
		bw := request.NewBodyWrapper(body)
		lastBW = bw
		return bw
	}
	r.Body = maybeWrapBody(r.Body)
	if r.GetBody != nil {
		originalGetBody := r.GetBody
		r.GetBody = func() (io.ReadCloser, error) {
			b, err := originalGetBody()
			if err != nil {
				lastBW = nil // The underlying transport will fail to make a retry request, hence, record no data.
				return nil, err
			}
			return maybeWrapBody(b), nil
		}
	}

	span.SetAttributes(t.semconv.RequestTraceAttrs(r)...)
	propagation.Inject(ctx, r.Header)

	res, err := t.rt.RoundTrip(r)
	if err == nil {
		res, err = ensureResponseBody(t.rt, r, res)
	}

	// Record the metrics on error or no error.
	statusCode := 0
	if err == nil {
		statusCode = res.StatusCode
	}
	var requestSize int64
	if lastBW != nil {
		requestSize = lastBW.BytesRead()
	}
	t.semconv.RecordMetrics(
		ctx,
		semconv.MetricData{
			RequestSize:     requestSize,
			RequestDuration: time.Since(requestStartTime),
		},
		t.semconv.MetricOptions(semconv.MetricAttributes{
			Req:        r,
			Resp:       res,
			StatusCode: statusCode,
			Err:        err,
		}),
	)

	if err != nil {
		span.SetAttributes(otelsemconv.ErrorType(err))
		span.SetStatus(codes.Error, err.Error())
		span.End()

		return res, err
	}

	res.Body = newWrappedBody(span, res.Body)
	// traces
	span.SetAttributes(t.semconv.ResponseTraceAttrs(res)...)
	span.SetStatus(t.semconv.Status(res.StatusCode))

	return res, nil
}

func ensureResponseBody(rt http.RoundTripper, r *http.Request, res *http.Response) (*http.Response, error) {
	switch {
	case res == nil:
		return nil, fmt.Errorf("http: RoundTripper implementation (%T) returned a nil *Response with a nil error", rt)
	case res.Body != nil:
		return res, nil
	case res.ContentLength > 0 && r.Method != http.MethodHead:
		return nil, fmt.Errorf("http: RoundTripper implementation (%T) returned a *Response with content length %d but a nil Body", rt, res.ContentLength)
	default:
		res.Body = http.NoBody
		return res, nil
	}
}

// newWrappedBody returns a new and appropriately scoped *wrappedBody as an
// io.ReadCloser. If the passed body implements io.Writer, the returned value
// will implement io.ReadWriteCloser.
func newWrappedBody(span *sdktrace.Span, body io.ReadCloser) io.ReadCloser {
	// The successful protocol switch responses will have a body that
	// implement an io.ReadWriteCloser. Ensure this interface type continues
	// to be satisfied if that is the case.
	if _, ok := body.(io.ReadWriteCloser); ok {
		return &wrappedBody{span: span, body: body}
	}

	// Remove the implementation of the io.ReadWriteCloser and only implement
	// the io.ReadCloser.
	return struct{ io.ReadCloser }{&wrappedBody{span: span, body: body}}
}

// wrappedBody is the response body type returned by the transport
// instrumentation to complete a span. Errors encountered when using the
// response body are recorded in span tracking the response.
//
// The span tracking the response is ended when this body is closed.
//
// If the response body implements the io.Writer interface (i.e. for
// successful protocol switches), the wrapped body also will.
type wrappedBody struct {
	span *sdktrace.Span
	body io.ReadCloser
}

var _ io.ReadWriteCloser = &wrappedBody{}

func (wb *wrappedBody) Write(p []byte) (int, error) {
	// This will not panic given the guard in newWrappedBody.
	n, err := wb.body.(io.Writer).Write(p)
	if err != nil {
		wb.span.SetAttributes(otelsemconv.ErrorType(err))
		wb.span.SetStatus(codes.Error, err.Error())
	}
	return n, err
}

func (wb *wrappedBody) Read(b []byte) (int, error) {
	n, err := wb.body.Read(b)

	switch err {
	case nil:
		// nothing to do here but fall through to the return
	case io.EOF:
		wb.span.End()
	default:
		wb.span.SetAttributes(otelsemconv.ErrorType(err))
		wb.span.SetStatus(codes.Error, err.Error())
	}
	return n, err
}

func (wb *wrappedBody) Close() error {
	wb.span.End()
	if wb.body != nil {
		return wb.body.Close()
	}
	return nil
}
