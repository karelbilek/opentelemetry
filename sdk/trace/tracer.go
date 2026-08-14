// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package trace

import (
	"context"
	"time"

	"github.com/karelbilek/opentelemetry/sdk/instrumentation"
	"github.com/karelbilek/opentelemetry/trace"
)

type tracer struct {
	noop                 bool
	provider             *TracerProvider
	instrumentationScope instrumentation.Scope
}

var _ trace.Tracer = &tracer{}

func (tr *tracer) noopStart(ctx context.Context) (context.Context, trace.Span) {
	span := SpanFromContext(ctx)

	// If the parent context contains a non-zero span context, that span
	// context needs to be returned as a non-recording span
	// (https://github.com/open-telemetry/opentelemetry-specification/blob/3a1dde966a4ce87cce5adf464359fe369741bbea/specification/trace/api.md#behavior-of-the-api-in-the-absence-of-an-installed-sdk).
	var zeroSC trace.SpanContext
	if sc := span.SpanContext(); !sc.Equal(zeroSC) {
		if !span.IsRecording() {
			// If the span is not recording return it directly.
			return ctx, span
		}
		// Otherwise, return the span context needs in a non-recording span.
		span = &recordingSpan{noop: true, spanContext: sc}
	} else {
		// No parent, return a No-Op span with an empty span context.
		span = noopSpanInstance
	}
	return ContextWithSpan(ctx, span), span
}

// Start starts a Span and returns it along with a context containing it.
//
// The Span is created with the provided name and as a child of any existing
// span context found in the passed context. The created Span will be
// configured appropriately by any SpanOption passed.
func (tr *tracer) Start(
	ctx context.Context,
	name string,
	options ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	if tr.noop {
		return tr.noopStart(ctx)
	}
	config := trace.NewSpanStartConfig(options...)

	if ctx == nil {
		// Prevent trace.ContextWithSpan from panicking.
		ctx = context.Background()
	}

	// For local spans created by this SDK, track child span count.
	if p := SpanFromContext(ctx); p != nil {
		if sdkSpan, ok := p.(*recordingSpan); ok {
			sdkSpan.addChild()
		}
	}

	s := tr.newSpan(ctx, name, &config)
	newCtx := ContextWithSpan(ctx, s)

	if rw, ok := s.(ReadWriteSpan); ok && s.IsRecording() {
		sps := tr.provider.getSpanProcessors()
		for _, sp := range sps {
			// Use original context.
			sp.sp.OnStart(ctx, rw)
		}
	}
	if rtt, ok := s.(runtimeTracer); ok {
		newCtx = rtt.runtimeTrace(newCtx)
	}

	return newCtx, s
}

type runtimeTracer interface {
	// runtimeTrace starts a "runtime/trace".Task for the span and
	// returns a context containing the task.
	runtimeTrace(ctx context.Context) context.Context
}

// newSpan returns a new configured span.
func (tr *tracer) newSpan(ctx context.Context, name string, config *trace.SpanConfig) trace.Span {
	// If told explicitly to make this a new root use a zero value SpanContext
	// as a parent which contains an invalid trace ID and is not remote.
	var psc trace.SpanContext
	if config.NewRoot() {
		ctx = ContextWithSpanContext(ctx, psc)
	} else {
		psc = SpanContextFromContext(ctx)
	}

	// If there is a valid parent trace ID, use it to ensure the continuity of
	// the trace. Always generate a new span ID so other components can rely
	// on a unique span ID, even if the Span is non-recording.
	var tid trace.TraceID
	var sid trace.SpanID
	if !psc.TraceID().IsValid() {
		tid, sid = tr.provider.idGenerator.NewIDs(ctx)
	} else {
		tid = psc.TraceID()
		sid = tr.provider.idGenerator.NewSpanID(ctx, tid)
	}

	samplingResult := tr.provider.sampler.ShouldSample(SamplingParameters{
		ParentContext: ctx,
		TraceID:       tid,
		Name:          name,
		Kind:          config.SpanKind(),
		Attributes:    config.Attributes(),
		Links:         config.Links(),
	})

	scc := trace.SpanContextConfig{
		TraceID:    tid,
		SpanID:     sid,
		TraceState: samplingResult.Tracestate,
	}
	if isSampled(samplingResult) {
		scc.TraceFlags = psc.TraceFlags() | trace.FlagsSampled
	} else {
		scc.TraceFlags = psc.TraceFlags() &^ trace.FlagsSampled
	}
	sc := trace.NewSpanContext(scc)

	if !isRecording(samplingResult) {
		return tr.newNonRecordingSpan(sc)
	}
	return tr.newRecordingSpan(ctx, psc, sc, name, samplingResult, config)
}

// newRecordingSpan returns a new configured recordingSpan.
func (tr *tracer) newRecordingSpan(
	ctx context.Context,
	psc, sc trace.SpanContext,
	name string,
	sr SamplingResult,
	config *trace.SpanConfig,
) *recordingSpan {
	startTime := config.Timestamp()
	if startTime.IsZero() {
		startTime = time.Now()
	}

	s := &recordingSpan{
		// Do not pre-allocate the attributes slice here! Doing so will
		// allocate memory that is likely never going to be used, or if used,
		// will be over-sized. The default Go compiler has been tested to
		// dynamically allocate needed space very well. Benchmarking has shown
		// it to be more performant than what we can predetermine here,
		// especially for the common use case of few to no added
		// attributes.

		parent:      psc,
		spanContext: sc,
		spanKind:    trace.ValidateSpanKind(config.SpanKind()),
		name:        name,
		startTime:   startTime,
		events:      newEvictedQueueEvent(tr.provider.spanLimits.EventCountLimit),
		links:       newEvictedQueueLink(tr.provider.spanLimits.LinkCountLimit),
		tracer:      tr,
	}

	for _, l := range config.Links() {
		s.AddLink(l)
	}

	s.SetAttributes(sr.Attributes...)
	s.SetAttributes(config.Attributes()...)

	return s
}

// newNonRecordingSpan returns a new configured nonRecordingSpan.
func (tr *tracer) newNonRecordingSpan(sc trace.SpanContext) *recordingSpan {
	return &recordingSpan{tracer: tr, spanContext: sc, noop: true}
}
