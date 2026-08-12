// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package log

import (
	"context"

	"github.com/karelbilek/opentelemetry/attribute"
	"github.com/karelbilek/opentelemetry/log/embedded"
)

// Logger emits log records.
//
// Warning: Methods may be added to this interface in minor releases. See
// package documentation on API implementation for information on how to set
// default behavior for unimplemented methods.
type Logger interface {
	// Users of the interface can ignore this. This embedded type is only used
	// by implementations of this interface. See the "API Implementations"
	// section of the package documentation for more information.
	embedded.Logger

	// Emit emits a log record.
	//
	// The record may be held by the implementation. Callers should not mutate
	// the record after passed.
	//
	// Implementations of this method need to be safe for a user to call
	// concurrently.
	Emit(ctx context.Context, record Record)

	// Enabled reports whether the Logger emits for the given context and
	// param.
	//
	// This is useful for users that want to know if a [Record]
	// will be processed or dropped before they perform complex operations to
	// construct the [Record]. Callers should invoke Enabled before each call
	// to [Logger.Emit] because the enabled state may change over time.
	//
	// The passed param is likely to be a partial record information being
	// provided (e.g a param with only the Severity set).
	// If a Logger needs more information than is provided, it
	// is said to be in an indeterminate state (see below).
	//
	// The returned value will be true when the Logger will emit for the
	// provided context and param, and will be false if the Logger will not
	// emit. The returned value may be true or false in an indeterminate state.
	// An implementation should default to returning true for an indeterminate
	// state, but may return false if valid reasons in particular circumstances
	// exist (e.g. performance, correctness).
	//
	// The param should not be held by the implementation. A copy should be
	// made if the param needs to be held after the call returns.
	//
	// Implementations of this method need to be safe for a user to call
	// concurrently.
	Enabled(ctx context.Context, param EnabledParameters) bool
}

// LoggerConfig contains options for a [Logger].
type LoggerConfig struct {
	// Ensure forward compatibility by explicitly making this not comparable.
	noCmp [0]func() //nolint: unused  // This is indeed used.

	version   string
	schemaURL string
	attrs     attribute.Set
}


// NewLoggerConfig returns a new [LoggerConfig] with all the options applied.
func NewLoggerConfig(version string, schemaURL string, attrs attribute.Set) LoggerConfig {

	return LoggerConfig{
		version:   version,
		schemaURL: schemaURL,
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

// SchemaURL returns the schema URL of the library providing instrumentation.
func (cfg LoggerConfig) SchemaURL() string {
	return cfg.schemaURL
}

type loggerOptionFunc func(LoggerConfig) LoggerConfig

func (fn loggerOptionFunc) applyLogger(cfg LoggerConfig) LoggerConfig {
	return fn(cfg)
}

// mergeSets returns the union of keys between a and b. Any duplicate keys will
// use the value associated with b.
func mergeSets(a, b attribute.Set) attribute.Set {
	// NewMergeIterator uses the first value for any duplicates.
	iter := attribute.NewMergeIterator(&b, &a)
	merged := make([]attribute.KeyValue, 0, a.Len()+b.Len())
	for iter.Next() {
		merged = append(merged, iter.Attribute())
	}
	return attribute.NewSet(merged...)
}

// EnabledParameters represents payload for [Logger]'s Enabled method.
type EnabledParameters struct {
	Severity  Severity
	EventName string
}
