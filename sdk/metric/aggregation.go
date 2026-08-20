// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metric

import (
	"errors"
	"fmt"
	"slices"
)

var errAgg = errors.New("aggregation")

// aggregation is the aggregation used to summarize recorded measurements.
type aggregation interface {
	// copy returns a deep copy of the Aggregation.
	Copy() aggregation

	// err returns an error for any misconfigured Aggregation.
	Err() error
}

// aggregationSum is an Aggregation that summarizes a set of measurements as their
// arithmetic sum.
type aggregationSum struct{} // AggregationSum has no parameters.

var _ aggregation = aggregationSum{}

// copy returns a deep copy of s.
func (s aggregationSum) Copy() aggregation { return s }

// err returns an error for any misconfiguration. A sum aggregation has no
// parameters and cannot be misconfigured, therefore this always returns nil.
func (aggregationSum) Err() error { return nil }

// aggregationLastValue is an Aggregation that summarizes a set of measurements as the
// last one made.
type aggregationLastValue struct{} // AggregationLastValue has no parameters.

var _ aggregation = aggregationLastValue{}

// copy returns a deep copy of l.
func (l aggregationLastValue) Copy() aggregation { return l }

// err returns an error for any misconfiguration. A last-value aggregation has
// no parameters and cannot be misconfigured, therefore this always returns
// nil.
func (aggregationLastValue) Err() error { return nil }

// aggregationExplicitBucketHistogram is an Aggregation that summarizes a set of
// measurements as an histogram with explicitly defined buckets.
type aggregationExplicitBucketHistogram struct {
	// Boundaries are the increasing bucket boundary values. Boundary values
	// define bucket upper bounds. Buckets are exclusive of their lower
	// boundary and inclusive of their upper bound (except at positive
	// infinity). A measurement is defined to fall into the greatest-numbered
	// bucket with a boundary that is greater than or equal to the
	// measurement. As an example, boundaries defined as:
	//
	// []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 1000}
	//
	// Will define these buckets:
	//
	// (-∞, 0], (0, 5.0], (5.0, 10.0], (10.0, 25.0], (25.0, 50.0],
	// (50.0, 75.0], (75.0, 100.0], (100.0, 250.0], (250.0, 500.0],
	// (500.0, 1000.0], (1000.0, +∞)
	Boundaries []float64
	// NoMinMax indicates whether to not record the min and max of the
	// distribution. By default, these extrema are recorded.
	//
	// Recording these extrema for cumulative data is expected to have little
	// value, they will represent the entire life of the instrument instead of
	// just the current collection cycle. It is recommended to set this to true
	// for that type of data to avoid computing the low-value extrema.
	NoMinMax bool
}

var _ aggregation = aggregationExplicitBucketHistogram{}

// errHist is returned by misconfigured ExplicitBucketHistograms.
var errHist = fmt.Errorf("%w: explicit bucket histogram", errAgg)

// err returns an error for any misconfiguration.
func (h aggregationExplicitBucketHistogram) Err() error {
	if len(h.Boundaries) <= 1 {
		return nil
	}

	// Check boundaries are monotonic.
	i := h.Boundaries[0]
	for _, j := range h.Boundaries[1:] {
		if i >= j {
			return fmt.Errorf("%w: non-monotonic boundaries: %v", errHist, h.Boundaries)
		}
		i = j
	}

	return nil
}

// copy returns a deep copy of h.
func (h aggregationExplicitBucketHistogram) Copy() aggregation {
	return aggregationExplicitBucketHistogram{
		Boundaries: slices.Clone(h.Boundaries),
		NoMinMax:   h.NoMinMax,
	}
}

func selectAggregation(ik InstrumentKind) aggregation {
	switch ik {
	case InstrumentKindCounter,
		InstrumentKindUpDownCounter,
		InstrumentKindObservableCounter,
		InstrumentKindObservableUpDownCounter:
		return aggregationSum{}
	case InstrumentKindObservableGauge, InstrumentKindGauge:
		return aggregationLastValue{}
	case InstrumentKindHistogram:
		return aggregationExplicitBucketHistogram{
			Boundaries: []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
			NoMinMax:   false,
		}
	}
	panic("unknown instrument kind")
}
