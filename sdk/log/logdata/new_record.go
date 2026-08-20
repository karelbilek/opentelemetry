package logdata

import (
	"time"

	"github.com/karelbilek/opentelemetry/log"
	"github.com/karelbilek/opentelemetry/sdk/instrumentation"
	"github.com/karelbilek/opentelemetry/sdk/resource"
	"github.com/karelbilek/opentelemetry/trace"
)

func NewRecord(
	eventName string,
	timestamp time.Time,
	observedTimestamp time.Time,
	severity log.Severity,
	severityText string,
	traceID trace.TraceID,
	spanID trace.SpanID,
	traceFlags trace.TraceFlags,
	resource *resource.Resource,
	scope *instrumentation.Scope,
	attributeValueLengthLimit int,
	attributeCountLimit int,
	allowDupKeys bool,
) Record {
	return Record{
		eventName:         eventName,
		timestamp:         timestamp,
		observedTimestamp: observedTimestamp,
		severity:          severity,
		severityText:      severityText,

		traceID:    traceID,
		spanID:     spanID,
		traceFlags: traceFlags,

		resource:                  resource,
		scope:                     scope,
		attributeValueLengthLimit: attributeValueLengthLimit,
		attributeCountLimit:       attributeCountLimit,
		allowDupKeys:              allowDupKeys,
	}
}
