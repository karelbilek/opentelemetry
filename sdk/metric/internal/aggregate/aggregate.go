// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package aggregate

import (
	"context"
	"time"

	otel "github.com/karelbilek/opentelemetry"
	"github.com/karelbilek/opentelemetry/attribute"
	"github.com/karelbilek/opentelemetry/sdk/metric/metricdata"
)

// now is used to return the current local time while allowing tests to
// override the default time.Now function.
var now = time.Now

// Measure receives measurements to be aggregated.
type Measure[N int64 | float64] func(context.Context, N, attribute.Set)

// ComputeAggregation stores the aggregate of measurements into dest and
// returns the number of aggregate data-points output.
type ComputeAggregation func(dest *metricdata.Aggregation) int

// Builder builds an aggregate function.
type Builder[N int64 | float64] struct {
	// Filter is the attribute filter the aggregate function will use on the
	// input of measurements.
	Filter attribute.Filter
	// AggregationLimit is the cardinality limit of measurement attributes. Any
	// measurement for new attributes once the limit has been reached will be
	// aggregated into a single aggregate for the "otel.metric.overflow"
	// attribute.
	//
	// If AggregationLimit is less than or equal to zero there will not be an
	// aggregation limit imposed (i.e. unlimited attribute sets).
	AggregationLimit int
}

type fltrMeasure[N int64 | float64] func(ctx context.Context, value N, lazy lazyFilteredAttributes, eh otel.ErrorHandler)

func (b Builder[N]) filter(f fltrMeasure[N], eh otel.ErrorHandler) Measure[N] {
	if b.Filter != nil {
		fltr := b.Filter // Copy to make it immutable after assignment.
		return func(ctx context.Context, n N, a attribute.Set) {
			f(ctx, n, newLazyFilteredAttributes(a, fltr), eh)
		}
	}
	return func(ctx context.Context, n N, a attribute.Set) {
		f(ctx, n, newLazyFilteredAttributes(a, nil), eh)
	}
}

// LastValue returns a last-value aggregate function input and output.
func (b Builder[N]) LastValue(eh otel.ErrorHandler) (Measure[N], ComputeAggregation) {
	lv := newCumulativeLastValue[N](b.AggregationLimit)
	return b.filter(lv.measure, eh), lv.collect
}

// PrecomputedLastValue returns a last-value aggregate function input and
// output. The aggregation returned from the returned ComputeAggregation
// function will always only return values from the previous collection cycle.
func (b Builder[N]) PrecomputedLastValue(eh otel.ErrorHandler) (Measure[N], ComputeAggregation) {
	lv := newPrecomputedLastValue[N](b.AggregationLimit)
	return b.filter(lv.measure, eh), lv.cumulative
}

// PrecomputedSum returns a sum aggregate function input and output. The
// arguments passed to the input are expected to be the precomputed sum values.
func (b Builder[N]) PrecomputedSum(monotonic bool, eh otel.ErrorHandler) (Measure[N], ComputeAggregation) {
	s := newPrecomputedSum[N](monotonic, b.AggregationLimit)
	return b.filter(s.measure, eh), s.cumulative
}

// Sum returns a sum aggregate function input and output.
func (b Builder[N]) Sum(monotonic bool, eh otel.ErrorHandler) (Measure[N], ComputeAggregation) {
	s := newCumulativeSum[N](monotonic, b.AggregationLimit)
	return b.filter(s.measure, eh), s.collect
}

// ExplicitBucketHistogram returns a histogram aggregate function input and
// output.
func (b Builder[N]) ExplicitBucketHistogram(
	boundaries []float64,
	noMinMax, noSum bool,
	eh otel.ErrorHandler,
) (Measure[N], ComputeAggregation) {
	h := newCumulativeHistogram[N](boundaries, noMinMax, noSum, b.AggregationLimit)
	return b.filter(h.measure, eh), h.collect
}

// reset ensures s has capacity and sets it length. If the capacity of s too
// small, a new slice is returned with the specified capacity and length.
func reset[T any](s []T, length, capacity int) []T {
	if cap(s) < capacity {
		return make([]T, length, capacity)
	}
	return s[:length]
}
