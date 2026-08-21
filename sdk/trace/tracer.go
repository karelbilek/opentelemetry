// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package trace

import (
	"context"
	"time"

	"github.com/karelbilek/opentelemetry/sdk/instrumentation"
	"github.com/karelbilek/opentelemetry/trace"
)

type Tracer struct {
	noop                 bool
	provider             *TracerProvider
	instrumentationScope instrumentation.Scope
}

func (tr *Tracer) noopStart(ctx context.Context) (context.Context, *Span) {
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
		span = &Span{noop: true, spanContext: sc}
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
func (tr *Tracer) Start(
	ctx context.Context,
	name string,
	options ...trace.SpanStartOption,
) (context.Context, *Span) {
	if tr.noop {
		return tr.noopStart(ctx)
	}
	config := trace.NewSpanStartConfig(options...)

	if ctx == nil {
		// Prevent trace.ContextWithSpan from panicking.
		ctx = context.Background()
	}

	s := tr.newSpan(ctx, name, &config)
	newCtx := ContextWithSpan(ctx, s)

	if !s.noop {
		newCtx = s.runtimeTrace(newCtx)
	}

	return newCtx, s
}

// newSpan returns a new configured span.
func (tr *Tracer) newSpan(ctx context.Context, name string, config *trace.SpanConfig) *Span {
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
		tid, sid = newIDs(ctx)
	} else {
		tid = psc.TraceID()
		sid = newSpanID(ctx, tid)
	}

	samplingResult := shouldSample(samplingParameters{
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
	return tr.newRecordingSpan(psc, sc, name, samplingResult, config)
}

// newRecordingSpan returns a new configured recordingSpan.
func (tr *Tracer) newRecordingSpan(
	psc, sc trace.SpanContext,
	name string,
	sr SamplingResult,
	config *trace.SpanConfig,
) *Span {
	startTime := config.Timestamp()
	if startTime.IsZero() {
		startTime = time.Now()
	}

	s := &Span{
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
func (tr *Tracer) newNonRecordingSpan(sc trace.SpanContext) *Span {
	return &Span{tracer: tr, spanContext: sc, noop: true}
}
