package trace

import (
	"context"

	"github.com/karelbilek/opentelemetry/attribute"
	"github.com/karelbilek/opentelemetry/trace"
)

// ContextWithSpanContext returns a copy of parent with sc as the current
// Span. The Span implementation that wraps sc is non-recording and performs
// no operations other than to return sc as the SpanContext from the
// SpanContext method.
func ContextWithSpanContext(parent context.Context, sc trace.SpanContext) context.Context {
	return ContextWithSpan(parent, &recordingSpan{noop: true, spanContext: sc})
}

// ContextWithRemoteSpanContext returns a copy of parent with rsc set explicitly
// as a remote SpanContext and as the current Span. The Span implementation
// that wraps rsc is non-recording and performs no operations other than to
// return rsc as the SpanContext from the SpanContext method.
func ContextWithRemoteSpanContext(parent context.Context, rsc trace.SpanContext) context.Context {
	return ContextWithSpanContext(parent, rsc.WithRemote(true))
}

var noopSpanInstance trace.Span = &recordingSpan{noop: true}

// SpanFromContext returns the current Span from ctx.
//
// If no Span is currently set in ctx an implementation of a Span that
// performs no operations is returned.
func SpanFromContext(ctx context.Context) trace.Span {
	if ctx == nil {
		return noopSpanInstance
	}
	if span, ok := ctx.Value(currentSpanKey).(trace.Span); ok {
		return span
	}
	return noopSpanInstance
}

type traceContextKeyType int

const currentSpanKey traceContextKeyType = iota

// ContextWithSpan returns a copy of parent with span set as the current Span.
func ContextWithSpan(parent context.Context, span trace.Span) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, currentSpanKey, span)
}

// SpanContextFromContext returns the current Span's SpanContext.
func SpanContextFromContext(ctx context.Context) trace.SpanContext {
	return SpanFromContext(ctx).SpanContext()
}

// LinkFromContext returns a link encapsulating the SpanContext in the provided
// ctx.
func LinkFromContext(ctx context.Context, attrs ...attribute.KeyValue) trace.Link {
	return trace.Link{
		SpanContext: SpanContextFromContext(ctx),
		Attributes:  attrs,
	}
}
