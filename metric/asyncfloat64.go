// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metric

// Float64ObservableCounterConfig contains options for asynchronous counter
// instruments that record float64 values.
type Float64ObservableCounterConfig struct {
	description string
	unit        string
}

// NewFloat64ObservableCounterConfig returns a new
// [Float64ObservableCounterConfig] with all opts applied.
func NewFloat64ObservableCounterConfig(opts ...Float64ObservableCounterOption) Float64ObservableCounterConfig {
	var config Float64ObservableCounterConfig
	for _, o := range opts {
		config = o.applyFloat64ObservableCounter(config)
	}
	return config
}

// Description returns the configured description.
func (c Float64ObservableCounterConfig) Description() string {
	return c.description
}

// Unit returns the configured unit.
func (c Float64ObservableCounterConfig) Unit() string {
	return c.unit
}

// Float64ObservableCounterOption applies options to a
// [Float64ObservableCounterConfig]. See [Float64ObservableOption] and
// [InstrumentOption] for other options that can be used as a
// Float64ObservableCounterOption.
type Float64ObservableCounterOption interface {
	applyFloat64ObservableCounter(Float64ObservableCounterConfig) Float64ObservableCounterConfig
}

// Float64ObservableUpDownCounterConfig contains options for asynchronous
// counter instruments that record float64 values.
type Float64ObservableUpDownCounterConfig struct {
	description string
	unit        string
}

// NewFloat64ObservableUpDownCounterConfig returns a new
// [Float64ObservableUpDownCounterConfig] with all opts applied.
func NewFloat64ObservableUpDownCounterConfig(
	opts ...Float64ObservableUpDownCounterOption,
) Float64ObservableUpDownCounterConfig {
	var config Float64ObservableUpDownCounterConfig
	for _, o := range opts {
		config = o.applyFloat64ObservableUpDownCounter(config)
	}
	return config
}

// Description returns the configured description.
func (c Float64ObservableUpDownCounterConfig) Description() string {
	return c.description
}

// Unit returns the configured unit.
func (c Float64ObservableUpDownCounterConfig) Unit() string {
	return c.unit
}

// Float64ObservableUpDownCounterOption applies options to a
// [Float64ObservableUpDownCounterConfig]. See [Float64ObservableOption] and
// [InstrumentOption] for other options that can be used as a
// Float64ObservableUpDownCounterOption.
type Float64ObservableUpDownCounterOption interface {
	applyFloat64ObservableUpDownCounter(Float64ObservableUpDownCounterConfig) Float64ObservableUpDownCounterConfig
}

// Float64ObservableGaugeConfig contains options for asynchronous counter
// instruments that record float64 values.
type Float64ObservableGaugeConfig struct {
	description string
	unit        string
}

// NewFloat64ObservableGaugeConfig returns a new [Float64ObservableGaugeConfig]
// with all opts applied.
func NewFloat64ObservableGaugeConfig(opts ...Float64ObservableGaugeOption) Float64ObservableGaugeConfig {
	var config Float64ObservableGaugeConfig
	for _, o := range opts {
		config = o.applyFloat64ObservableGauge(config)
	}
	return config
}

// Description returns the configured description.
func (c Float64ObservableGaugeConfig) Description() string {
	return c.description
}

// Unit returns the configured unit.
func (c Float64ObservableGaugeConfig) Unit() string {
	return c.unit
}

// Float64ObservableGaugeOption applies options to a
// [Float64ObservableGaugeConfig]. See [Float64ObservableOption] and
// [InstrumentOption] for other options that can be used as a
// Float64ObservableGaugeOption.
type Float64ObservableGaugeOption interface {
	applyFloat64ObservableGauge(Float64ObservableGaugeConfig) Float64ObservableGaugeConfig
}

// Float64ObservableOption applies options to float64 Observer instruments.
type Float64ObservableOption interface {
	Float64ObservableCounterOption
	Float64ObservableUpDownCounterOption
	Float64ObservableGaugeOption
}
