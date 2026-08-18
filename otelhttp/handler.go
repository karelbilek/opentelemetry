// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelhttp

import (
	"context"
	"net/http"
	"net/http/httptrace"
	"time"

	"github.com/felixge/httpsnoop"
	otel "github.com/karelbilek/opentelemetry"
	"github.com/karelbilek/opentelemetry/attribute"
	"github.com/karelbilek/opentelemetry/metric"
	"github.com/karelbilek/opentelemetry/otelhttp/internal/request"
	"github.com/karelbilek/opentelemetry/otelhttp/internal/semconv"
	"github.com/karelbilek/opentelemetry/propagation"
	sdktrace "github.com/karelbilek/opentelemetry/sdk/trace"
	"github.com/karelbilek/opentelemetry/trace"
)

// middleware is an http middleware which wraps the next handler in a span.
type middleware struct {
	operation string
	server    string

	tracer             *sdktrace.Tracer
	spanStartOptions   []trace.SpanStartOption
	readEvent          bool
	writeEvent         bool
	filters            []Filter
	spanNameFormatter  func(string, *http.Request) string
	publicEndpointFn   func(*http.Request) bool
	metricAttributesFn func(*http.Request) []attribute.KeyValue

	semconv semconv.HTTPServer
}

// NewHandler wraps the passed handler in a span named after the operation and
// enriches it with metrics.
func NewHandler(handler http.Handler, eh otel.ErrorHandler, operation string,
	serverName string,
	tracerProvider *sdktrace.TracerProvider,
	spanStartOptions []trace.SpanStartOption,
	PublicEndpointFn func(*http.Request) bool,
	readEvent bool,
	writeEvent bool,
	filters []Filter,
	spanNameFormatter func(string, *http.Request) string,
	clientTrace func(context.Context) *httptrace.ClientTrace,
	meterProvider metric.MeterProvider,
	metricAttributesFn func(*http.Request) []attribute.KeyValue,
) http.Handler {
	return NewMiddleware(operation, eh, serverName, tracerProvider, spanStartOptions, PublicEndpointFn, readEvent, writeEvent, filters, spanNameFormatter, clientTrace, meterProvider, metricAttributesFn)(handler)
}

// NewMiddleware returns a tracing and metrics instrumentation middleware.
// The handler returned by the middleware wraps a handler
// in a span named after the operation and enriches it with metrics.
func NewMiddleware(operation string, eh otel.ErrorHandler, serverName string,
	tracerProvider *sdktrace.TracerProvider,
	spanStartOptions []trace.SpanStartOption,
	PublicEndpointFn func(*http.Request) bool,
	readEvent bool,
	writeEvent bool,
	filters []Filter,
	spanNameFormatter func(string, *http.Request) string,
	clientTrace func(context.Context) *httptrace.ClientTrace,
	meterProvider metric.MeterProvider,
	metricAttributesFn func(*http.Request) []attribute.KeyValue,
) func(http.Handler) http.Handler {
	h := middleware{
		operation: operation,
	}

	c := newConfig(serverName, tracerProvider, spanStartOptions, PublicEndpointFn, readEvent, writeEvent, filters, spanNameFormatter, clientTrace, meterProvider, metricAttributesFn)
	c.SpanStartOptions = append([]trace.SpanStartOption{trace.WithSpanKind(trace.SpanKindServer)}, c.SpanStartOptions...)

	h.configure(c, eh)

	if h.spanNameFormatter == nil {
		h.spanNameFormatter = func(_ string, r *http.Request) string {
			return h.semconv.SpanName(r)
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.serveHTTP(w, r, next)
		})
	}
}

func (h *middleware) configure(c *config, eh otel.ErrorHandler) {
	h.tracer = newTracer(c.TracerProvider)
	h.spanStartOptions = c.SpanStartOptions
	h.readEvent = c.ReadEvent
	h.writeEvent = c.WriteEvent
	h.filters = c.Filters
	h.spanNameFormatter = c.SpanNameFormatter
	h.publicEndpointFn = c.PublicEndpointFn
	h.server = c.ServerName
	meter := c.MeterProvider.Meter(
		ScopeName,
	)
	h.semconv = semconv.NewHTTPServer(meter, eh)
	h.metricAttributesFn = c.MetricAttributesFn
}

// serveHTTP sets up tracing and calls the given next http.Handler with the span
// context injected into the request context.
func (h *middleware) serveHTTP(w http.ResponseWriter, r *http.Request, next http.Handler) {
	requestStartTime := time.Now()
	for _, f := range h.filters {
		if !f(r) {
			// Simply pass through to the handler if a filter rejects the request
			next.ServeHTTP(w, r)
			return
		}
	}

	ctx := propagation.Extract(r.Context(), r.Header)
	opts := []trace.SpanStartOption{
		trace.WithAttributes(h.semconv.RequestTraceAttrs(h.server, r, semconv.RequestTraceAttrsOpts{})...),
	}

	opts = append(opts, h.spanStartOptions...)
	if h.publicEndpointFn != nil && h.publicEndpointFn(r.WithContext(ctx)) {
		opts = append(opts, trace.WithNewRoot())
		// Linking incoming span context if any for public endpoint.
		if s := sdktrace.SpanContextFromContext(ctx); s.IsValid() && s.IsRemote() {
			opts = append(opts, trace.WithLinks(trace.Link{SpanContext: s}))
		}
	}

	tracer := h.tracer

	if startTime := StartTimeFromContext(ctx); !startTime.IsZero() {
		opts = append(opts, trace.WithTimestamp(startTime))
		requestStartTime = startTime
	}

	ctx, span := tracer.Start(ctx, h.spanNameFormatter(h.operation, r), opts...)
	defer span.End()

	readRecordFunc := func(int64) {}
	if h.readEvent {
		readRecordFunc = func(n int64) {
			span.AddEvent("read", trace.WithAttributes(ReadBytesKey.Int64(n)))
		}
	}

	// if request body is nil or NoBody, we don't want to mutate the body as it
	// will affect the identity of it in an unforeseeable way because we assert
	// ReadCloser fulfills a certain interface and it is indeed nil or NoBody.
	bw := request.NewBodyWrapper(r.Body, readRecordFunc)
	if r.Body != nil && r.Body != http.NoBody {
		origReq := r
		prevBody := r.Body
		r.Body = bw

		// Restore the original body after the request is processed to avoid issues
		// with extra wrapper since `http/server.go` later checks type of `r.Body`.
		defer func() { origReq.Body = prevBody }()
	}

	writeRecordFunc := func(int64) {}
	if h.writeEvent {
		writeRecordFunc = func(n int64) {
			span.AddEvent("write", trace.WithAttributes(WroteBytesKey.Int64(n)))
		}
	}

	rww := request.NewRespWriterWrapper(w, writeRecordFunc)

	// Wrap w to use our ResponseWriter methods while also exposing
	// other interfaces that w may implement (http.CloseNotifier,
	// http.Flusher, http.Hijacker, http.Pusher, io.ReaderFrom).

	w = httpsnoop.Wrap(w, httpsnoop.Hooks{
		Header: func(httpsnoop.HeaderFunc) httpsnoop.HeaderFunc {
			return rww.Header
		},
		Write: func(httpsnoop.WriteFunc) httpsnoop.WriteFunc {
			return rww.Write
		},
		WriteHeader: func(httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
			return rww.WriteHeader
		},
		Flush: func(httpsnoop.FlushFunc) httpsnoop.FlushFunc {
			return rww.Flush
		},
	})

	labeler, found := LabelerFromContext(ctx)
	if !found {
		ctx = ContextWithLabeler(ctx, labeler)
	}

	r = r.WithContext(ctx)
	next.ServeHTTP(w, r)

	if r.Pattern != "" {
		span.SetName(h.spanNameFormatter(h.operation, r))
	}

	statusCode := rww.StatusCode()
	bytesWritten := rww.BytesWritten()
	span.SetStatus(h.semconv.Status(statusCode))
	bytesRead := bw.BytesRead()
	span.SetAttributes(h.semconv.ResponseTraceAttrs(semconv.ResponseTelemetry{
		StatusCode: statusCode,
		ReadBytes:  bytesRead,
		ReadError:  bw.Error(),
		WriteBytes: bytesWritten,
		WriteError: rww.Error(),
	})...)

	h.semconv.RecordMetrics(ctx, semconv.ServerMetricData{
		ServerName:   h.server,
		ResponseSize: bytesWritten,
		MetricAttributes: semconv.MetricAttributes{
			Req:                  r,
			StatusCode:           statusCode,
			AdditionalAttributes: append(labeler.Get(), h.metricAttributesFromRequest(r)...),
		},
		MetricData: semconv.MetricData{
			RequestSize:     bytesRead,
			RequestDuration: time.Since(requestStartTime),
		},
	})
}

func (h *middleware) metricAttributesFromRequest(r *http.Request) []attribute.KeyValue {
	var attributeForRequest []attribute.KeyValue
	if h.metricAttributesFn != nil {
		attributeForRequest = h.metricAttributesFn(r)
	}
	return attributeForRequest
}
