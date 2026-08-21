// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package aggregate

import (
	"context"
	"time"

	"github.com/karelbilek/opentelemetry/attribute"
	"github.com/karelbilek/opentelemetry/sdk/metric/metricdata"
)

type sumValue[N int64 | float64] struct {
	n         atomicCounter[N]
	attrs     attribute.Set
	startTime time.Time
}

type sumValueMap[N int64 | float64] struct {
	values limitedSyncMap[*sumValue[N]]
}

func (s *sumValueMap[N]) measure(
	ctx context.Context,
	value N,
	a attribute.Set,
) {
	sv := s.values.LoadOrStoreAttr(a, func(attr attribute.Set) *sumValue[N] {
		return &sumValue[N]{
			// res:           r,
			attrs:     attr,
			startTime: now(),
		}
	})
	sv.n.add(value)
	// It is possible for collection to race with measurement and observe the
	// exemplar in the batch of metrics after the add() for cumulative sums.
	// This is an accepted tradeoff to avoid locking during measurement.
}

// newDeltaSum returns an aggregator that summarizes a set of measurements as
// their arithmetic sum. Each sum is scoped by attributes and the aggregation
// cycle the measurements were made in.
func newDeltaSum[N int64 | float64](
	monotonic bool,
	limit int,
) *deltaSum[N] {
	return &deltaSum[N]{
		monotonic: monotonic,
		start:     now(),
		hotColdValMap: [2]sumValueMap[N]{
			{
				// newRes: r,
				values: limitedSyncMap[*sumValue[N]]{aggLimit: limit},
			},
			{
				// newRes: r,
				values: limitedSyncMap[*sumValue[N]]{aggLimit: limit},
			},
		},
	}
}

// deltaSum is the storage for sums which resets every collection interval.
type deltaSum[N int64 | float64] struct {
	monotonic bool
	start     time.Time

	hcwg          hotColdWaitGroup
	hotColdValMap [2]sumValueMap[N]
}

func (s *deltaSum[N]) measure(ctx context.Context, value N, a attribute.Set) {
	hotIdx := s.hcwg.start()
	defer s.hcwg.done(hotIdx)
	s.hotColdValMap[hotIdx].measure(ctx, value, a)
}

// newCumulativeSum returns an aggregator that summarizes a set of measurements
// as their arithmetic sum. Each sum is scoped by attributes and the
// aggregation cycle the measurements were made in.
func newCumulativeSum[N int64 | float64](
	monotonic bool,
	limit int,
) *cumulativeSum[N] {
	return &cumulativeSum[N]{
		monotonic: monotonic,
		start:     now(),
		sumValueMap: sumValueMap[N]{
			// newRes: r,
			values: limitedSyncMap[*sumValue[N]]{aggLimit: limit},
		},
	}
}

type cumulativeSum[N int64 | float64] struct {
	monotonic bool
	start     time.Time

	sumValueMap[N]
}

func (s *cumulativeSum[N]) collect(
	dest *metricdata.Aggregation, //nolint:gocritic // The pointer is needed for the ComputeAggregation interface
) int {
	t := now()

	// If *dest is not a metricdata.Sum, memory reuse is missed. In that case,
	// use the zero-value sData and hope for better alignment next cycle.
	sData, _ := (*dest).(metricdata.Sum[N])
	sData.IsMonotonic = s.monotonic

	// Values are being concurrently written while we iterate, so only use the
	// current length for capacity.
	dPts := reset(sData.DataPoints, 0, s.values.Len())

	var i int
	s.values.Range(func(_, value any) bool {
		val := value.(*sumValue[N])

		startTime := s.start

		newPt := metricdata.DataPoint[N]{
			Attributes: val.attrs,
			StartTime:  startTime,
			Time:       t,
			Value:      val.n.load(),
		}
		// collectExemplars(&newPt.Exemplars, val.res.Collect)
		dPts = append(dPts, newPt)
		// TODO (#3006): This will use an unbounded amount of memory if there
		// are unbounded number of attribute sets being aggregated. Attribute
		// sets that become "stale" need to be forgotten so this will not
		// overload the system.
		i++
		return true
	})

	sData.DataPoints = dPts
	*dest = sData

	return i
}

// newPrecomputedSum returns an aggregator that summarizes a set of
// observations as their arithmetic sum. Each sum is scoped by attributes and
// the aggregation cycle the measurements were made in.
func newPrecomputedSum[N int64 | float64](
	monotonic bool,
	limit int,
) *precomputedSum[N] {
	return &precomputedSum[N]{
		deltaSum: newDeltaSum[N](monotonic, limit),
	}
}

// precomputedSum summarizes a set of observations as their arithmetic sum.
type precomputedSum[N int64 | float64] struct {
	*deltaSum[N]
}

func (s *precomputedSum[N]) cumulative(
	dest *metricdata.Aggregation, //nolint:gocritic // The pointer is needed for the ComputeAggregation interface
) int {
	t := now()

	// If *dest is not a metricdata.Sum, memory reuse is missed. In that case,
	// use the zero-value sData and hope for better alignment next cycle.
	sData, _ := (*dest).(metricdata.Sum[N])
	sData.IsMonotonic = s.monotonic

	// cumulative precomputed always clears values on collection
	readIdx := s.hcwg.swapHotAndWait()
	// The len will not change while we iterate over values, since we waited
	// for all writes to finish to the cold values and len.
	n := s.hotColdValMap[readIdx].values.Len()
	dPts := reset(sData.DataPoints, n, n)

	var i int
	s.hotColdValMap[readIdx].values.Range(func(_, value any) bool {
		val := value.(*sumValue[N])
		// collectExemplars(&dPts[i].Exemplars, val.res.Collect)
		dPts[i].Attributes = val.attrs
		dPts[i].StartTime = s.start
		dPts[i].Time = t
		dPts[i].Value = val.n.load()
		i++
		return true
	})
	s.hotColdValMap[readIdx].values.Clear()

	sData.DataPoints = dPts
	*dest = sData

	return i
}
