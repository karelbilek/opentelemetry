// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package trace

import (
	"context"

	"github.com/karelbilek/opentelemetry/attribute"
	"github.com/karelbilek/opentelemetry/trace"
)

// samplingParameters contains the values passed to a Sampler.
type samplingParameters struct {
	ParentContext context.Context
	TraceID       trace.TraceID
	Name          string
	Kind          trace.SpanKind
	Attributes    []attribute.KeyValue
	Links         []trace.Link
}

// SamplingDecision indicates whether a span is dropped, recorded and/or sampled.
type SamplingDecision uint8

// Valid sampling decisions.
const (
	// Drop will not record the span and all attributes/events will be dropped.
	Drop SamplingDecision = iota

	// RecordOnly indicates the span's IsRecording method returns true, but trace.FlagsSampled flag
	// must not be set.
	RecordOnly

	// RecordAndSample indicates the span's IsRecording method returns true and trace.FlagsSampled flag
	// must be set.
	RecordAndSample
)

// SamplingResult conveys a SamplingDecision, set of Attributes and a Tracestate.
type SamplingResult struct {
	Decision   SamplingDecision
	Attributes []attribute.KeyValue
	Tracestate trace.TraceState
}

func shouldSample(p samplingParameters) SamplingResult {
	return SamplingResult{
		Decision:   RecordAndSample,
		Tracestate: SpanContextFromContext(p.ParentContext).TraceState(),
	}
}
