package metricinternals

// InstrumentKind is the identifier of a group of instruments that all
// performing the same function.
type InstrumentKind uint8

const (
	// instrumentKindUndefined is an undefined instrument kind, it should not
	// be used by any initialized type.
	InstrumentKindUndefined InstrumentKind = 0 // nolint:unused
	// InstrumentKindCounter identifies a group of instruments that record
	// increasing values synchronously with the code path they are measuring.
	InstrumentKindCounter InstrumentKind = 1
	// InstrumentKindUpDownCounter identifies a group of instruments that
	// record increasing and decreasing values synchronously with the code path
	// they are measuring.
	InstrumentKindUpDownCounter InstrumentKind = 2
	// InstrumentKindHistogram identifies a group of instruments that record a
	// distribution of values synchronously with the code path they are
	// measuring.
	InstrumentKindHistogram InstrumentKind = 3
	// InstrumentKindObservableCounter identifies a group of instruments that
	// record increasing values in an asynchronous callback.
	InstrumentKindObservableCounter InstrumentKind = 4
	// InstrumentKindObservableUpDownCounter identifies a group of instruments
	// that record increasing and decreasing values in an asynchronous
	// callback.
	InstrumentKindObservableUpDownCounter InstrumentKind = 5
	// InstrumentKindObservableGauge identifies a group of instruments that
	// record current values in an asynchronous callback.
	InstrumentKindObservableGauge InstrumentKind = 6
	// InstrumentKindGauge identifies a group of instruments that record
	// instantaneous values synchronously with the code path they are
	// measuring.
	InstrumentKindGauge InstrumentKind = 7
)
