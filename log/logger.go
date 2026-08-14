// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package log

import (
	"github.com/karelbilek/opentelemetry/attribute"
)

// LoggerConfig contains options for a [Logger].
type LoggerConfig struct {
	// Ensure forward compatibility by explicitly making this not comparable.
	noCmp [0]func() //nolint: unused  // This is indeed used.

	version   string
	attrs     attribute.Set
}

// NewLoggerConfig returns a new [LoggerConfig] with all the options applied.
func NewLoggerConfig(version string, attrs attribute.Set) LoggerConfig {
	return LoggerConfig{
		version:   version,
		attrs:     attrs,
	}
}

// InstrumentationVersion returns the version of the library providing
// instrumentation.
func (cfg LoggerConfig) InstrumentationVersion() string {
	return cfg.version
}

// InstrumentationAttributes returns the attributes associated with the library
// providing instrumentation.
func (cfg LoggerConfig) InstrumentationAttributes() attribute.Set {
	return cfg.attrs
}


// EnabledParameters represents payload for [Logger]'s Enabled method.
type EnabledParameters struct {
	Severity  Severity
	EventName string
}
