// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metric

// Int64CounterConfig contains options for synchronous counter instruments that
// record int64 values.
type Int64CounterConfig struct {
	description string
	unit        string
}

// NewInt64CounterConfig returns a new [Int64CounterConfig] with all opts
// applied.
func NewInt64CounterConfig(opts ...Int64CounterOption) Int64CounterConfig {
	var config Int64CounterConfig
	for _, o := range opts {
		config = o.applyInt64Counter(config)
	}
	return config
}

// Description returns the configured description.
func (c Int64CounterConfig) Description() string {
	return c.description
}

// Unit returns the configured unit.
func (c Int64CounterConfig) Unit() string {
	return c.unit
}

// Int64CounterOption applies options to a [Int64CounterConfig]. See
// [InstrumentOption] for other options that can be used as an
// Int64CounterOption.
type Int64CounterOption interface {
	applyInt64Counter(Int64CounterConfig) Int64CounterConfig
}

// Int64UpDownCounterConfig contains options for synchronous counter
// instruments that record int64 values.
type Int64UpDownCounterConfig struct {
	description string
	unit        string
}

// NewInt64UpDownCounterConfig returns a new [Int64UpDownCounterConfig] with
// all opts applied.
func NewInt64UpDownCounterConfig(opts ...Int64UpDownCounterOption) Int64UpDownCounterConfig {
	var config Int64UpDownCounterConfig
	for _, o := range opts {
		config = o.applyInt64UpDownCounter(config)
	}
	return config
}

// Description returns the configured description.
func (c Int64UpDownCounterConfig) Description() string {
	return c.description
}

// Unit returns the configured unit.
func (c Int64UpDownCounterConfig) Unit() string {
	return c.unit
}

// Int64UpDownCounterOption applies options to a [Int64UpDownCounterConfig].
// See [InstrumentOption] for other options that can be used as an
// Int64UpDownCounterOption.
type Int64UpDownCounterOption interface {
	applyInt64UpDownCounter(Int64UpDownCounterConfig) Int64UpDownCounterConfig
}

// Int64HistogramConfig contains options for synchronous histogram instruments
// that record int64 values.
type Int64HistogramConfig struct {
	description              string
	unit                     string
	explicitBucketBoundaries []float64
}

// NewInt64HistogramConfig returns a new [Int64HistogramConfig] with all opts
// applied.
func NewInt64HistogramConfig(opts ...Int64HistogramOption) Int64HistogramConfig {
	var config Int64HistogramConfig
	for _, o := range opts {
		config = o.applyInt64Histogram(config)
	}
	return config
}

// Description returns the configured description.
func (c Int64HistogramConfig) Description() string {
	return c.description
}

// Unit returns the configured unit.
func (c Int64HistogramConfig) Unit() string {
	return c.unit
}

// ExplicitBucketBoundaries returns the configured explicit bucket boundaries.
func (c Int64HistogramConfig) ExplicitBucketBoundaries() []float64 {
	return c.explicitBucketBoundaries
}

// Int64HistogramOption applies options to a [Int64HistogramConfig]. See
// [InstrumentOption] for other options that can be used as an
// Int64HistogramOption.
type Int64HistogramOption interface {
	applyInt64Histogram(Int64HistogramConfig) Int64HistogramConfig
}

// Int64GaugeConfig contains options for synchronous gauge instruments that
// record int64 values.
type Int64GaugeConfig struct {
	description string
	unit        string
}

// NewInt64GaugeConfig returns a new [Int64GaugeConfig] with all opts
// applied.
func NewInt64GaugeConfig(opts ...Int64GaugeOption) Int64GaugeConfig {
	var config Int64GaugeConfig
	for _, o := range opts {
		config = o.applyInt64Gauge(config)
	}
	return config
}

// Description returns the configured description.
func (c Int64GaugeConfig) Description() string {
	return c.description
}

// Unit returns the configured unit.
func (c Int64GaugeConfig) Unit() string {
	return c.unit
}

// Int64GaugeOption applies options to a [Int64GaugeConfig]. See
// [InstrumentOption] for other options that can be used as a
// Int64GaugeOption.
type Int64GaugeOption interface {
	applyInt64Gauge(Int64GaugeConfig) Int64GaugeConfig
}
