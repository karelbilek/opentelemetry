// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package metricdata provides types for the metric SDK data model.
package metricdata

import (
	"encoding/json"
	"time"

	"github.com/karelbilek/opentelemetry/attribute"
	"github.com/karelbilek/opentelemetry/sdk/instrumentation"
	"github.com/karelbilek/opentelemetry/sdk/resource"
)

// ResourceMetrics is a collection of ScopeMetrics and the associated Resource
// that created them.
type ResourceMetrics struct {
	// Resource represents the entity that collected the metrics.
	Resource *resource.Resource
	// ScopeMetrics are the collection of metrics with unique Scopes.
	ScopeMetrics []ScopeMetrics
}

// ScopeMetrics is a collection of Metrics Produces by a Meter.
type ScopeMetrics struct {
	// Scope is the Scope that the Meter was created with.
	Scope instrumentation.Scope
	// Metrics are a list of aggregations created by the Meter.
	Metrics []Metrics
}

// Metrics is a collection of one or more aggregated timeseries from an Instrument.
type Metrics struct {
	// Name is the name of the Instrument that created this data.
	Name string
	// Description is the description of the Instrument, which can be used in documentation.
	Description string
	// Unit is the unit in which the Instrument reports.
	Unit string
	// Data is the aggregated data from an Instrument.
	Data Aggregation
}

// Aggregation is the store of data reported by an Instrument.
// It will be one of: Gauge, Sum, Histogram.
type Aggregation interface {
	privateAggregation()
}

// Gauge represents a measurement of the current value of an instrument.
type Gauge[N int64 | float64] struct {
	// DataPoints are the individual aggregated measurements with unique
	// Attributes.
	DataPoints []DataPoint[N]
}

func (Gauge[N]) privateAggregation() {}

// Sum represents the sum of all measurements of values from an instrument.
type Sum[N int64 | float64] struct {
	// DataPoints are the individual aggregated measurements with unique
	// Attributes.
	DataPoints  []DataPoint[N]
	IsMonotonic bool
}

func (Sum[N]) privateAggregation() {}

// DataPoint is a single data point in a timeseries.
type DataPoint[N int64 | float64] struct {
	// Attributes is the set of key value pairs that uniquely identify the
	// timeseries.
	Attributes attribute.Set
	// StartTime is when the timeseries was started. (optional)
	StartTime time.Time `json:",omitempty"`
	// Time is the time when the timeseries was recorded. (optional)
	Time time.Time `json:",omitempty"`
	// Value is the value of this data point.
	Value N
}

// Histogram represents the histogram of all measurements of values from an instrument.
type Histogram[N int64 | float64] struct {
	// DataPoints are the individual aggregated measurements with unique
	// Attributes.
	DataPoints []HistogramDataPoint[N]
}

func (Histogram[N]) privateAggregation() {}

// HistogramDataPoint is a single histogram data point in a timeseries.
type HistogramDataPoint[N int64 | float64] struct {
	// Attributes is the set of key value pairs that uniquely identify the
	// timeseries.
	Attributes attribute.Set
	// StartTime is when the timeseries was started.
	StartTime time.Time
	// Time is the time when the timeseries was recorded.
	Time time.Time

	// Count is the number of updates this histogram has been calculated with.
	Count uint64
	// Bounds are the upper bounds of the buckets of the histogram. Because the
	// last boundary is +infinity this one is implied.
	Bounds []float64
	// BucketCounts is the count of each of the buckets.
	BucketCounts []uint64

	// Min is the minimum value recorded. (optional)
	Min Extrema[N]
	// Max is the maximum value recorded. (optional)
	Max Extrema[N]
	// Sum is the sum of the values recorded.
	Sum N
}

// Extrema is the minimum or maximum value of a dataset.
type Extrema[N int64 | float64] struct {
	value N
	valid bool
}

// MarshalText converts the Extrema value to text.
func (e Extrema[N]) MarshalText() ([]byte, error) {
	if !e.valid {
		return json.Marshal(nil)
	}
	return json.Marshal(e.value)
}

// MarshalJSON converts the Extrema value to JSON number.
func (e *Extrema[N]) MarshalJSON() ([]byte, error) {
	return e.MarshalText()
}

// NewExtrema returns an Extrema set to v.
func NewExtrema[N int64 | float64](v N) Extrema[N] {
	return Extrema[N]{value: v, valid: true}
}

// Value returns the Extrema value and true if the Extrema is defined.
// Otherwise, if the Extrema is its zero-value, defined will be false.
func (e Extrema[N]) Value() (v N, defined bool) {
	return e.value, e.valid
}
