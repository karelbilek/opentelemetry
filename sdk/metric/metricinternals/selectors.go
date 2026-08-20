package metricinternals


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
