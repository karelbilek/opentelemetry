// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package transform provides transformation functionality from the
// sdk/metric/metricdata data-types into OTLP data-types.
package transform

import (
	"fmt"
	"time"

	cpb "github.com/karelbilek/opentelemetry/proto/common/v1"
	mpb "github.com/karelbilek/opentelemetry/proto/metrics/v1"
	rpb "github.com/karelbilek/opentelemetry/proto/resource/v1"

	"github.com/karelbilek/opentelemetry/sdk/metric/metricdata"
)

// ResourceMetrics returns an OTLP ResourceMetrics generated from rm. If rm
// contains invalid ScopeMetrics, an error will be returned along with an OTLP
// ResourceMetrics that contains partial OTLP ScopeMetrics.
func ResourceMetrics(rm *metricdata.ResourceMetrics) (*mpb.ResourceMetrics, error) {
	sms, err := ScopeMetrics(rm.ScopeMetrics)
	return &mpb.ResourceMetrics{
		Resource: &rpb.Resource{
			Attributes: AttrIter(rm.Resource.Iter()),
		},
		ScopeMetrics: sms,
	}, err
}

// ScopeMetrics returns a slice of OTLP ScopeMetrics generated from sms. If
// sms contains invalid metric values, an error will be returned along with a
// slice that contains partial OTLP ScopeMetrics.
func ScopeMetrics(sms []metricdata.ScopeMetrics) ([]*mpb.ScopeMetrics, error) {
	errs := &multiErr{datatype: "ScopeMetrics"}
	out := make([]*mpb.ScopeMetrics, 0, len(sms))
	for _, sm := range sms {
		ms, err := Metrics(sm.Metrics)
		if err != nil {
			errs.append(err)
		}

		out = append(out, &mpb.ScopeMetrics{
			Scope: &cpb.InstrumentationScope{
				Name: sm.Scope.Name,
			},
			Metrics: ms,
		})
	}
	return out, errs.errOrNil()
}

// Metrics returns a slice of OTLP Metric generated from ms. If ms contains
// invalid metric values, an error will be returned along with a slice that
// contains partial OTLP Metrics.
func Metrics(ms []metricdata.Metrics) ([]*mpb.Metric, error) {
	errs := &multiErr{datatype: "Metrics"}
	out := make([]*mpb.Metric, 0, len(ms))
	for _, m := range ms {
		o, err := metric(m)
		if err != nil {
			// Do not include invalid data. Drop the metric, report the error.
			errs.append(errMetric{m: o, err: err})
			continue
		}
		out = append(out, o)
	}
	return out, errs.errOrNil()
}

func metric(m metricdata.Metrics) (*mpb.Metric, error) {
	var err error
	out := &mpb.Metric{
		Name:        m.Name,
		Description: m.Description,
		Unit:        m.Unit,
	}
	switch a := m.Data.(type) {
	case metricdata.Gauge[int64]:
		out.Data = Gauge(a)
	case metricdata.Gauge[float64]:
		out.Data = Gauge(a)
	case metricdata.Sum[int64]:
		out.Data, err = Sum(a)
	case metricdata.Sum[float64]:
		out.Data, err = Sum(a)
	case metricdata.Histogram[int64]:
		out.Data, err = Histogram(a)
	case metricdata.Histogram[float64]:
		out.Data, err = Histogram(a)
	default:
		return out, fmt.Errorf("%w: %T", errUnknownAggregation, a)
	}
	return out, err
}

// Gauge returns an OTLP Metric_Gauge generated from g.
func Gauge[N int64 | float64](g metricdata.Gauge[N]) *mpb.Metric_Gauge {
	return &mpb.Metric_Gauge{
		Gauge: &mpb.Gauge{
			DataPoints: DataPoints(g.DataPoints),
		},
	}
}

// Sum returns an OTLP Metric_Sum generated from s.
func Sum[N int64 | float64](s metricdata.Sum[N]) (*mpb.Metric_Sum, error) {
	return &mpb.Metric_Sum{
		Sum: &mpb.Sum{
			AggregationTemporality: mpb.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
			IsMonotonic:            s.IsMonotonic,
			DataPoints:             DataPoints(s.DataPoints),
		},
	}, nil
}

// DataPoints returns a slice of OTLP NumberDataPoint generated from dPts.
func DataPoints[N int64 | float64](dPts []metricdata.DataPoint[N]) []*mpb.NumberDataPoint {
	out := make([]*mpb.NumberDataPoint, 0, len(dPts))
	for _, dPt := range dPts {
		ndp := &mpb.NumberDataPoint{
			Attributes:        AttrIter(dPt.Attributes.Iter()),
			StartTimeUnixNano: timeUnixNano(dPt.StartTime),
			TimeUnixNano:      timeUnixNano(dPt.Time),
		}
		switch v := any(dPt.Value).(type) {
		case int64:
			ndp.Value = &mpb.NumberDataPoint_AsInt{
				AsInt: v,
			}
		case float64:
			ndp.Value = &mpb.NumberDataPoint_AsDouble{
				AsDouble: v,
			}
		}
		out = append(out, ndp)
	}
	return out
}

// Histogram returns an OTLP Metric_Histogram generated from h. An error is
// returned if the temporality of h is unknown.
func Histogram[N int64 | float64](h metricdata.Histogram[N]) (*mpb.Metric_Histogram, error) {
	return &mpb.Metric_Histogram{
		Histogram: &mpb.Histogram{
			AggregationTemporality: mpb.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
			DataPoints:             HistogramDataPoints(h.DataPoints),
		},
	}, nil
}

// HistogramDataPoints returns a slice of OTLP HistogramDataPoint generated
// from dPts.
func HistogramDataPoints[N int64 | float64](dPts []metricdata.HistogramDataPoint[N]) []*mpb.HistogramDataPoint {
	out := make([]*mpb.HistogramDataPoint, 0, len(dPts))
	for _, dPt := range dPts {
		sum := float64(dPt.Sum)
		hdp := &mpb.HistogramDataPoint{
			Attributes:        AttrIter(dPt.Attributes.Iter()),
			StartTimeUnixNano: timeUnixNano(dPt.StartTime),
			TimeUnixNano:      timeUnixNano(dPt.Time),
			Count:             dPt.Count,
			Sum:               &sum,
			BucketCounts:      dPt.BucketCounts,
			ExplicitBounds:    dPt.Bounds,
		}
		if v, ok := dPt.Min.Value(); ok {
			vF64 := float64(v)
			hdp.Min = &vF64
		}
		if v, ok := dPt.Max.Value(); ok {
			vF64 := float64(v)
			hdp.Max = &vF64
		}
		out = append(out, hdp)
	}
	return out
}

// timeUnixNano returns t as a Unix time, the number of nanoseconds elapsed
// since January 1, 1970 UTC as uint64.
// The result is undefined if the Unix time
// in nanoseconds cannot be represented by an int64
// (a date before the year 1678 or after 2262).
// timeUnixNano on the zero Time returns 0.
// The result does not depend on the location associated with t.
func timeUnixNano(t time.Time) uint64 {
	return uint64(max(0, t.UnixNano())) // nolint:gosec // Overflow checked.
}
