package metricinternals

import (
	"github.com/karelbilek/opentelemetry/sdk/metric/metricdata"
)

// TemporalitySelector selects the temporality to use based on the InstrumentKind.
type TemporalitySelector func(InstrumentKind) metricdata.Temporality

// DefaultTemporalitySelector is the default TemporalitySelector used if
// WithTemporalitySelector is not provided. CumulativeTemporality will be used
// for all instrument kinds if this TemporalitySelector is used.
func DefaultTemporalitySelector(k InstrumentKind) metricdata.Temporality {
	return CumulativeTemporalitySelector(k)
}

// CumulativeTemporalitySelector is the TemporalitySelector that uses
// a cumulative temporality for all instrument kinds.
func CumulativeTemporalitySelector(InstrumentKind) metricdata.Temporality {
	return metricdata.CumulativeTemporality
}

// DeltaTemporalitySelector is the TemporalitySelector that uses
// a delta temporality for instrument kinds: counter, histogram, observable counter
// All other instruments use cumulative temporality.
func DeltaTemporalitySelector(k InstrumentKind) metricdata.Temporality {
	switch k {
	case InstrumentKindCounter, InstrumentKindHistogram, InstrumentKindObservableCounter:
		return metricdata.DeltaTemporality
	default:
		return metricdata.CumulativeTemporality
	}
}

// LowMemoryTemporalitySelector is the TemporalitySelector that uses
// delta temporality for counters and histograms. All other instruments use
// cumulative temporality.
func LowMemoryTemporalitySelector(k InstrumentKind) metricdata.Temporality {
	switch k {
	case InstrumentKindCounter, InstrumentKindHistogram:
		return metricdata.DeltaTemporality
	default:
		return metricdata.CumulativeTemporality
	}
}

// AggregationSelector selects the aggregation and the parameters to use for
// that aggregation based on the InstrumentKind.
//
// If the Aggregation returned is nil or DefaultAggregation, the selection from
// DefaultAggregationSelector will be used.
type AggregationSelector func(InstrumentKind) Aggregation

// DefaultAggregationSelector returns the default aggregation and parameters
// that will be used to summarize measurement made from an instrument of
// InstrumentKind. This AggregationSelector using the following selection
// mapping: Counter ⇨ Sum, Observable Counter ⇨ Sum, UpDownCounter ⇨ Sum,
// Observable UpDownCounter ⇨ Sum, Observable Gauge ⇨ LastValue,
// Histogram ⇨ ExplicitBucketHistogram.
//
// The default ExplicitBucketHistogram boundaries are designed for
// millisecond-scale latency values. Boundaries are interpreted relative to the
// values recorded for the instrument and are not rescaled when an instrument is
// created with a different unit (e.g. via
// [github.com/karelbilek/opentelemetry/metric.WithUnit]). Instrumentation authors should
// supply appropriate boundaries per instrument via
// [github.com/karelbilek/opentelemetry/metric.WithExplicitBucketBoundaries]; end users
// can also override boundaries for a specific instrument with a [View].
func DefaultAggregationSelector(ik InstrumentKind) Aggregation {
	switch ik {
	case InstrumentKindCounter,
		InstrumentKindUpDownCounter,
		InstrumentKindObservableCounter,
		InstrumentKindObservableUpDownCounter:
		return AggregationSum{}
	case InstrumentKindObservableGauge, InstrumentKindGauge:
		return AggregationLastValue{}
	case InstrumentKindHistogram:
		return AggregationExplicitBucketHistogram{
			Boundaries: []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
			NoMinMax:   false,
		}
	}
	panic("unknown instrument kind")
}

// CardinalityLimitSelector selects the cardinality limit to use based on the
// InstrumentKind. The cardinality limit is the maximum number of distinct
// attribute sets that can be recorded for a single instrument.
//
// The selector returns (limit, fallback). When fallback is true, the pipeline
// falls back to the provider's global cardinality limit.
// When fallback is false, the limit is applied: a value of 0 or less means
// no limit, and a positive value is the limit for that kind.
// To avoid overriding the provider's global limit, return (0, true).
type CardinalityLimitSelector func(InstrumentKind) (limit int, fallback bool)

// defaultCardinalityLimitSelector is the default CardinalityLimitSelector used
// if WithCardinalityLimitSelector is not provided. It returns (0, true) for all
// instrument kinds, allowing the pipeline to fall back to the provider's global
// limit.
func DefaultCardinalityLimitSelector(_ InstrumentKind) (int, bool) {
	return 0, true
}
