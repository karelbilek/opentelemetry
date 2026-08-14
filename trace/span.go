// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package trace

import (
	"github.com/karelbilek/opentelemetry/attribute"
)

// Link is the relationship between two Spans. The relationship can be within
// the same Trace or across different Traces.
//
// For example, a Link is used in the following situations:
//
//  1. Batch Processing: A batch of operations may contain operations
//     associated with one or more traces/spans. Since there can only be one
//     parent SpanContext, a Link is used to keep reference to the
//     SpanContext of all operations in the batch.
//  2. Public Endpoint: A SpanContext for an in incoming client request on a
//     public endpoint should be considered untrusted. In such a case, a new
//     trace with its own identity and sampling decision needs to be created,
//     but this new trace needs to be related to the original trace in some
//     form. A Link is used to keep reference to the original SpanContext and
//     track the relationship.
type Link struct {
	// SpanContext of the linked Span.
	SpanContext SpanContext

	// Attributes describe the aspects of the link.
	Attributes []attribute.KeyValue
}

// SpanKind is the role a Span plays in a Trace.
type SpanKind int

// As a convenience, these match the proto definition, see
// https://github.com/open-telemetry/opentelemetry-proto/blob/30d237e1ff3ab7aa50e0922b5bebdd93505090af/opentelemetry/proto/trace/v1/trace.proto#L101-L129
//
// The unspecified value is not a valid `SpanKind`. Use `ValidateSpanKind()`
// to coerce a span kind to a valid value.
const (
	// SpanKindUnspecified is an unspecified SpanKind and is not a valid
	// SpanKind. SpanKindUnspecified should be replaced with SpanKindInternal
	// if it is received.
	SpanKindUnspecified SpanKind = 0
	// SpanKindInternal is a SpanKind for a Span that represents an internal
	// operation within an application.
	SpanKindInternal SpanKind = 1
	// SpanKindServer is a SpanKind for a Span that represents the operation
	// of handling a request from a client.
	SpanKindServer SpanKind = 2
	// SpanKindClient is a SpanKind for a Span that represents the operation
	// of client making a request to a server.
	SpanKindClient SpanKind = 3
	// SpanKindProducer is a SpanKind for a Span that represents the operation
	// of a producer sending a message to a message broker. Unlike
	// SpanKindClient and SpanKindServer, there is often no direct
	// relationship between this kind of Span and a SpanKindConsumer kind. A
	// SpanKindProducer Span will end once the message is accepted by the
	// message broker which might not overlap with the processing of that
	// message.
	SpanKindProducer SpanKind = 4
	// SpanKindConsumer is a SpanKind for a Span that represents the operation
	// of a consumer receiving a message from a message broker. Like
	// SpanKindProducer Spans, there is often no direct relationship between
	// this Span and the Span that produced the message.
	SpanKindConsumer SpanKind = 5
)

// ValidateSpanKind returns a valid span kind value.  This will coerce
// invalid values into the default value, SpanKindInternal.
func ValidateSpanKind(spanKind SpanKind) SpanKind {
	switch spanKind {
	case SpanKindInternal,
		SpanKindServer,
		SpanKindClient,
		SpanKindProducer,
		SpanKindConsumer:
		// valid
		return spanKind
	default:
		return SpanKindInternal
	}
}

// String returns the specified name of the SpanKind in lower-case.
func (sk SpanKind) String() string {
	switch sk {
	case SpanKindInternal:
		return "internal"
	case SpanKindServer:
		return "server"
	case SpanKindClient:
		return "client"
	case SpanKindProducer:
		return "producer"
	case SpanKindConsumer:
		return "consumer"
	default:
		return "unspecified"
	}
}
