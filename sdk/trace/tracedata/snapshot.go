// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package tracedata

import (
	"time"

	"github.com/karelbilek/opentelemetry/attribute"
	"github.com/karelbilek/opentelemetry/codes"
	"github.com/karelbilek/opentelemetry/sdk/instrumentation"
	"github.com/karelbilek/opentelemetry/sdk/resource"
	"github.com/karelbilek/opentelemetry/trace"
)

// Snapshot is an record of a spans state at a particular checkpointed time.
// It is used as a read-only representation of that state.
type Snapshot struct {
	Name                  string
	SpanContext           trace.SpanContext
	Parent                trace.SpanContext
	SpanKind              trace.SpanKind
	StartTime             time.Time
	EndTime               time.Time
	Attributes            []attribute.KeyValue
	Events                []Event
	Links                 []Link
	Status                Status
	DroppedAttributeCount int
	DroppedEventCount     int
	DroppedLinkCount      int
	Resource              *resource.Resource
	InstrumentationScope  instrumentation.Scope
}

// Status is the classified state of a Span.
type Status struct {
	// Code is an identifier of a Spans state classification.
	Code codes.Code
	// Description is a user hint about why that status was set. It is only
	// applicable when Code is Error.
	Description string
}
