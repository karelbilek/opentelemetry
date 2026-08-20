// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package log

import (
	"github.com/karelbilek/opentelemetry/log"
	"github.com/karelbilek/opentelemetry/sdk/instrumentation"
)

// EnabledParameters represents payload for [Processor]'s Enabled method.
type EnabledParameters struct {
	InstrumentationScope instrumentation.Scope
	Severity             log.Severity
	EventName            string
}
