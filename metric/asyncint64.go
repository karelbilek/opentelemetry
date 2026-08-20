// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metric

// Int64ObservableCounterConfig contains options for asynchronous counter
// instruments that record int64 values.
type Int64ObservableCounterConfig struct {
	description string
	unit        string
}

// NewInt64ObservableCounterConfig returns a new [Int64ObservableCounterConfig]
// with all opts applied.
func NewInt64ObservableCounterConfig(opts ...Int64ObservableCounterOption) Int64ObservableCounterConfig {
	var config Int64ObservableCounterConfig
	for _, o := range opts {
		config = o.applyInt64ObservableCounter(config)
	}
	return config
}

// Description returns the configured description.
func (c Int64ObservableCounterConfig) Description() string {
	return c.description
}

// Unit returns the configured unit.
func (c Int64ObservableCounterConfig) Unit() string {
	return c.unit
}

// Int64ObservableCounterOption applies options to a
// [Int64ObservableCounterConfig]. See [Int64ObservableOption] and
// [InstrumentOption] for other options that can be used as an
// Int64ObservableCounterOption.
type Int64ObservableCounterOption interface {
	applyInt64ObservableCounter(Int64ObservableCounterConfig) Int64ObservableCounterConfig
}

// Int64ObservableUpDownCounterConfig contains options for asynchronous counter
// instruments that record int64 values.
type Int64ObservableUpDownCounterConfig struct {
	description string
	unit        string
}

// NewInt64ObservableUpDownCounterConfig returns a new
// [Int64ObservableUpDownCounterConfig] with all opts applied.
func NewInt64ObservableUpDownCounterConfig(
	opts ...Int64ObservableUpDownCounterOption,
) Int64ObservableUpDownCounterConfig {
	var config Int64ObservableUpDownCounterConfig
	for _, o := range opts {
		config = o.applyInt64ObservableUpDownCounter(config)
	}
	return config
}

// Description returns the configured description.
func (c Int64ObservableUpDownCounterConfig) Description() string {
	return c.description
}

// Unit returns the configured unit.
func (c Int64ObservableUpDownCounterConfig) Unit() string {
	return c.unit
}

// Int64ObservableUpDownCounterOption applies options to a
// [Int64ObservableUpDownCounterConfig]. See [Int64ObservableOption] and
// [InstrumentOption] for other options that can be used as an
// Int64ObservableUpDownCounterOption.
type Int64ObservableUpDownCounterOption interface {
	applyInt64ObservableUpDownCounter(Int64ObservableUpDownCounterConfig) Int64ObservableUpDownCounterConfig
}

// Int64ObservableGaugeConfig contains options for asynchronous counter
// instruments that record int64 values.
type Int64ObservableGaugeConfig struct {
	description string
	unit        string
}

// NewInt64ObservableGaugeConfig returns a new [Int64ObservableGaugeConfig]
// with all opts applied.
func NewInt64ObservableGaugeConfig(opts ...Int64ObservableGaugeOption) Int64ObservableGaugeConfig {
	var config Int64ObservableGaugeConfig
	for _, o := range opts {
		config = o.applyInt64ObservableGauge(config)
	}
	return config
}

// Description returns the configured description.
func (c Int64ObservableGaugeConfig) Description() string {
	return c.description
}

// Unit returns the configured unit.
func (c Int64ObservableGaugeConfig) Unit() string {
	return c.unit
}

// Int64ObservableGaugeOption applies options to a
// [Int64ObservableGaugeConfig]. See [Int64ObservableOption] and
// [InstrumentOption] for other options that can be used as an
// Int64ObservableGaugeOption.
type Int64ObservableGaugeOption interface {
	applyInt64ObservableGauge(Int64ObservableGaugeConfig) Int64ObservableGaugeConfig
}

// Int64ObservableOption applies options to int64 Observer instruments.
type Int64ObservableOption interface {
	Int64ObservableCounterOption
	Int64ObservableUpDownCounterOption
	Int64ObservableGaugeOption
}
