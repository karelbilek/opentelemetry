// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package trace

// SpanLimits represents the limits of a span.
type SpanLimits struct {
	// AttributeValueLengthLimit is the maximum allowed attribute value length.
	//
	// This limit only applies to string, string slice, byte slice, slice, and
	// map attribute values. Any string and byte slice longer than this value
	// will be truncated to this length. For slice and map attribute values,
	// the limit is applied recursively to contained values.
	//
	// Setting this to a negative value means no limit is applied.
	AttributeValueLengthLimit int

	// AttributeCountLimit is the maximum allowed span attribute count. Any
	// attribute added to a span once this limit is reached will be dropped.
	//
	// Setting this to zero means no attributes will be recorded.
	//
	// Setting this to a negative value means no limit is applied.
	AttributeCountLimit int

	// EventCountLimit is the maximum allowed span event count. Any event
	// added to a span once this limit is reached means it will be added but
	// the oldest event will be dropped.
	//
	// Setting this to zero means no events we be recorded.
	//
	// Setting this to a negative value means no limit is applied.
	EventCountLimit int

	// LinkCountLimit is the maximum allowed span link count. Any link added
	// to a span once this limit is reached means it will be added but the
	// oldest link will be dropped.
	//
	// Setting this to zero means no links we be recorded.
	//
	// Setting this to a negative value means no limit is applied.
	LinkCountLimit int

	// AttributePerEventCountLimit is the maximum number of attributes allowed
	// per span event. Any attribute added after this limit reached will be
	// dropped.
	//
	// Setting this to zero means no attributes will be recorded for events.
	//
	// Setting this to a negative value means no limit is applied.
	AttributePerEventCountLimit int

	// AttributePerLinkCountLimit is the maximum number of attributes allowed
	// per span link. Any attribute added after this limit reached will be
	// dropped.
	//
	// Setting this to zero means no attributes will be recorded for links.
	//
	// Setting this to a negative value means no limit is applied.
	AttributePerLinkCountLimit int
}

// NewSpanLimits returns a SpanLimits with all limits set to the value their
// corresponding environment variable holds, or the default if unset.
//
// • AttributeValueLengthLimit: OTEL_SPAN_ATTRIBUTE_VALUE_LENGTH_LIMIT
// (default: unlimited)
//
// • AttributeCountLimit: OTEL_SPAN_ATTRIBUTE_COUNT_LIMIT (default: 128)
//
// • EventCountLimit: OTEL_SPAN_EVENT_COUNT_LIMIT (default: 128)
//
// • AttributePerEventCountLimit: OTEL_EVENT_ATTRIBUTE_COUNT_LIMIT (default:
// 128)
//
// • LinkCountLimit: OTEL_SPAN_LINK_COUNT_LIMIT (default: 128)
//
// • AttributePerLinkCountLimit: OTEL_LINK_ATTRIBUTE_COUNT_LIMIT (default: 128)
func NewSpanLimits(
	attributeValueLengthLimit int,
	attributeCountLimit int,
	eventCountLimit int,
	linkCountLimit int,
	attributePerEventCountLimit int,
	attributePerLinkCountLimit int,
) SpanLimits {
	return SpanLimits{
		AttributeValueLengthLimit:   attributeValueLengthLimit,
		AttributeCountLimit:         attributeCountLimit,
		EventCountLimit:             eventCountLimit,
		LinkCountLimit:              linkCountLimit,
		AttributePerEventCountLimit: attributePerEventCountLimit,
		AttributePerLinkCountLimit:  attributePerLinkCountLimit,
	}
}
