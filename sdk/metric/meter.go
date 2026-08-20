// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metric

import (
	"context"
	"errors"
	"fmt"

	otel "github.com/karelbilek/opentelemetry"
	"github.com/karelbilek/opentelemetry/attribute"
	"github.com/karelbilek/opentelemetry/internal/global"
	"github.com/karelbilek/opentelemetry/metric"
	"github.com/karelbilek/opentelemetry/sdk/instrumentation"
	"github.com/karelbilek/opentelemetry/sdk/metric/internal/aggregate"
)

// ErrInstrumentName indicates the created instrument has an invalid name.
// Valid names must consist of 255 or fewer characters including alphanumeric, _, ., -, / and start with a letter.
var ErrInstrumentName = errors.New("invalid instrument name")

// Meter handles the creation and coordination of all metric instruments. A
// Meter represents a single instrumentation scope; all metric telemetry
// produced by an instrumentation scope will use metric instruments from a
// single Meter.
type Meter struct {
	noop bool

	scope instrumentation.Scope
	pipe  *pipeline

	int64Insts             *cacheWithErr[instID, *int64Inst]
	float64Insts           *cacheWithErr[instID, *float64Inst]
	int64ObservableInsts   *cacheWithErr[instID, Int64Observable]
	float64ObservableInsts *cacheWithErr[instID, Float64Observable]

	int64Resolver   resolver[int64]
	float64Resolver resolver[float64]
}

func newMeter(s instrumentation.Scope, p *pipeline) *Meter {
	// viewCache ensures instrument conflicts, including number conflicts, this
	// meter is asked to create are logged to the user.
	var viewCache cache[string, instID]

	var int64Insts cacheWithErr[instID, *int64Inst]
	var float64Insts cacheWithErr[instID, *float64Inst]
	var int64ObservableInsts cacheWithErr[instID, Int64Observable]
	var float64ObservableInsts cacheWithErr[instID, Float64Observable]

	return &Meter{
		scope:                  s,
		pipe:                   p,
		int64Insts:             &int64Insts,
		float64Insts:           &float64Insts,
		int64ObservableInsts:   &int64ObservableInsts,
		float64ObservableInsts: &float64ObservableInsts,
		int64Resolver:          newResolver[int64](p, &viewCache),
		float64Resolver:        newResolver[float64](p, &viewCache),
	}
}

// Int64Counter returns a new instrument identified by name and configured with
// options. The instrument is used to synchronously record increasing int64
// measurements during a computational operation.
func (m *Meter) Int64Counter(name string, h otel.ErrorHandler, options ...metric.Int64CounterOption) (Int64Adder, error) {
	if m.noop {
		return Int64Adder{nil}, nil
	}
	cfg := metric.NewInt64CounterConfig(options...)
	const kind = InstrumentKindCounter
	p := int64InstProvider{m}
	i, err := p.lookup(kind, name, cfg.Description(), cfg.Unit(), defaultAttributes(options), h)
	if err != nil {
		return Int64Adder{i}, err
	}

	return Int64Adder{i}, validateInstrumentName(name)
}

// Int64UpDownCounter returns a new instrument identified by name and
// configured with options. The instrument is used to synchronously record
// int64 measurements during a computational operation.
func (m *Meter) Int64UpDownCounter(
	name string,
	h otel.ErrorHandler,
	options ...metric.Int64UpDownCounterOption,
) (Int64Adder, error) {
	if m.noop {
		return Int64Adder{nil}, nil
	}
	cfg := metric.NewInt64UpDownCounterConfig(options...)
	const kind = InstrumentKindUpDownCounter
	p := int64InstProvider{m}
	i, err := p.lookup(kind, name, cfg.Description(), cfg.Unit(), defaultAttributes(options), h)
	if err != nil {
		return Int64Adder{i}, err
	}

	return Int64Adder{i}, validateInstrumentName(name)
}

// Int64Histogram returns a new instrument identified by name and configured
// with options. The instrument is used to synchronously record the
// distribution of int64 measurements during a computational operation.
func (m *Meter) Int64Histogram(name string, h otel.ErrorHandler, options ...metric.Int64HistogramOption) (Int64Recorder, error) {
	if m.noop {
		return Int64Recorder{nil}, nil
	}
	cfg := metric.NewInt64HistogramConfig(options...)
	p := int64InstProvider{m}
	i, err := p.lookupHistogram(name, cfg, defaultAttributes(options), h)
	if err != nil {
		return Int64Recorder{i}, err
	}

	return Int64Recorder{i}, validateInstrumentName(name)
}

// Int64Gauge returns a new instrument identified by name and configured
// with options. The instrument is used to synchronously record the
// distribution of int64 measurements during a computational operation.
func (m *Meter) Int64Gauge(name string, h otel.ErrorHandler, options ...metric.Int64GaugeOption) (Int64Recorder, error) {
	if m.noop {
		return Int64Recorder{}, nil
	}
	cfg := metric.NewInt64GaugeConfig(options...)
	const kind = InstrumentKindGauge
	p := int64InstProvider{m}
	i, err := p.lookup(kind, name, cfg.Description(), cfg.Unit(), defaultAttributes(options), h)
	if err != nil {
		return Int64Recorder{i}, err
	}

	return Int64Recorder{i}, validateInstrumentName(name)
}

// int64ObservableInstrument returns a new observable identified by the Instrument.
// It registers callbacks for each reader's pipeline.
func (m *Meter) int64ObservableInstrument(
	id Instrument,
	allowedKeys []attribute.Key,
	callbacks []metric.Int64Callback,
	h otel.ErrorHandler,
) (Int64Observable, error) {
	key := instID{
		Name:        id.Name,
		Description: id.Description,
		Unit:        id.Unit,
		Kind:        id.Kind,
	}
	if m.int64ObservableInsts.HasKey(key) && len(callbacks) > 0 {
		warnRepeatedObservableCallbacks(id)
	}
	return m.int64ObservableInsts.Lookup(key, func() (Int64Observable, error) {
		inst := newInt64Observable(m, id.Kind, id.Name, id.Description, id.Unit)
		insert := m.int64Resolver.inserter
		// Connect the measure functions for instruments in this pipeline with the
		// callbacks for this pipeline.
		in, err := insert.Instrument(id, allowedKeys, selectAggregation(id.Kind), h)
		if err != nil {
			return inst, err
		}
		// Drop aggregation
		if len(in) == 0 {
			inst.dropAggregation = true
			return inst, validateInstrumentName(id.Name)
		}
		inst.appendMeasures(in)

		// Add the measures to the pipeline. It is required to maintain
		// measures per pipeline to avoid calling the measure that
		// is not part of the pipeline.
		insert.pipeline.addInt64Measure(inst.observableID, in)
		for _, cback := range callbacks {
			inst := int64Observer{measures: in}
			fn := cback
			insert.addCallback(func(ctx context.Context) error { return fn(ctx, inst) })
		}
		return inst, validateInstrumentName(id.Name)
	})
}

// Int64ObservableCounter returns a new instrument identified by name and
// configured with options. The instrument is used to asynchronously record
// increasing int64 measurements once per a measurement collection cycle.
// Only the measurements recorded during the collection cycle are exported.
//
// If Int64ObservableCounter is invoked repeatedly with the same Name,
// Description, and Unit, only the first set of callbacks provided are used.
// Use meter.RegisterCallback and Registration.Unregister to manage callbacks
// if instrumentation can be created multiple times with different callbacks.
func (m *Meter) Int64ObservableCounter(
	name string,
	h otel.ErrorHandler,
	options ...metric.Int64ObservableCounterOption,
) (Int64Observable, error) {
	if m.noop {
		return Int64Observable{}, nil
	}
	cfg := metric.NewInt64ObservableCounterConfig(options...)
	id := Instrument{
		Name:        name,
		Description: cfg.Description(),
		Unit:        cfg.Unit(),
		Kind:        InstrumentKindObservableCounter,
		Scope:       m.scope,
	}
	return m.int64ObservableInstrument(id, defaultAttributes(options), cfg.Callbacks(), h)
}

// Int64ObservableUpDownCounter returns a new instrument identified by name and
// configured with options. The instrument is used to asynchronously record
// int64 measurements once per a measurement collection cycle. Only the
// measurements recorded during the collection cycle are exported.
//
// If Int64ObservableUpDownCounter is invoked repeatedly with the same Name,
// Description, and Unit, only the first set of callbacks provided are used.
// Use meter.RegisterCallback and Registration.Unregister to manage callbacks
// if instrumentation can be created multiple times with different callbacks.
func (m *Meter) Int64ObservableUpDownCounter(
	name string,
	h otel.ErrorHandler,
	options ...metric.Int64ObservableUpDownCounterOption,
) (Int64Observable, error) {
	if m.noop {
		return Int64Observable{}, nil
	}
	cfg := metric.NewInt64ObservableUpDownCounterConfig(options...)
	id := Instrument{
		Name:        name,
		Description: cfg.Description(),
		Unit:        cfg.Unit(),
		Kind:        InstrumentKindObservableUpDownCounter,
		Scope:       m.scope,
	}
	return m.int64ObservableInstrument(id, defaultAttributes(options), cfg.Callbacks(), h)
}

// Int64ObservableGauge returns a new instrument identified by name and
// configured with options. The instrument is used to asynchronously record
// instantaneous int64 measurements once per a measurement collection cycle.
// Only the measurements recorded during the collection cycle are exported.
//
// If Int64ObservableGauge is invoked repeatedly with the same Name,
// Description, and Unit, only the first set of callbacks provided are used.
// Use meter.RegisterCallback and Registration.Unregister to manage callbacks
// if instrumentation can be created multiple times with different callbacks.
func (m *Meter) Int64ObservableGauge(
	name string,
	h otel.ErrorHandler,
	options ...metric.Int64ObservableGaugeOption,
) (Int64Observable, error) {
	if m.noop {
		return Int64Observable{}, nil
	}
	cfg := metric.NewInt64ObservableGaugeConfig(options...)
	id := Instrument{
		Name:        name,
		Description: cfg.Description(),
		Unit:        cfg.Unit(),
		Kind:        InstrumentKindObservableGauge,
		Scope:       m.scope,
	}
	return m.int64ObservableInstrument(id, defaultAttributes(options), cfg.Callbacks(), h)
}

// Float64Counter returns a new instrument identified by name and configured
// with options. The instrument is used to synchronously record increasing
// float64 measurements during a computational operation.
func (m *Meter) Float64Counter(name string, h otel.ErrorHandler, options ...metric.Float64CounterOption) (Float64Adder, error) {
	if m.noop {
		return Float64Adder{}, nil
	}
	cfg := metric.NewFloat64CounterConfig(options...)
	const kind = InstrumentKindCounter
	p := float64InstProvider{m}
	i, err := p.lookup(kind, name, cfg.Description(), cfg.Unit(), defaultAttributes(options), h)
	if err != nil {
		return Float64Adder{i}, err
	}

	return Float64Adder{i}, validateInstrumentName(name)
}

// Float64UpDownCounter returns a new instrument identified by name and
// configured with options. The instrument is used to synchronously record
// float64 measurements during a computational operation.
func (m *Meter) Float64UpDownCounter(
	name string,
	h otel.ErrorHandler,
	options ...metric.Float64UpDownCounterOption,
) (Float64Adder, error) {
	if m.noop {
		return Float64Adder{}, nil
	}
	cfg := metric.NewFloat64UpDownCounterConfig(options...)
	const kind = InstrumentKindUpDownCounter
	p := float64InstProvider{m}
	i, err := p.lookup(kind, name, cfg.Description(), cfg.Unit(), defaultAttributes(options), h)
	if err != nil {
		return Float64Adder{i}, err
	}

	return Float64Adder{i}, validateInstrumentName(name)
}

// Float64Histogram returns a new instrument identified by name and configured
// with options. The instrument is used to synchronously record the
// distribution of float64 measurements during a computational operation.
func (m *Meter) Float64Histogram(
	name string,
	h otel.ErrorHandler,
	options ...metric.Float64HistogramOption,
) (Float64Recorder, error) {
	if m.noop {
		return Float64Recorder{}, nil
	}
	cfg := metric.NewFloat64HistogramConfig(options...)
	p := float64InstProvider{m}
	i, err := p.lookupHistogram(name, cfg, defaultAttributes(options), h)
	if err != nil {
		return Float64Recorder{i}, err
	}

	return Float64Recorder{i}, validateInstrumentName(name)
}

// Float64Gauge returns a new instrument identified by name and configured
// with options. The instrument is used to synchronously record the
// distribution of float64 measurements during a computational operation.
func (m *Meter) Float64Gauge(name string, h otel.ErrorHandler, options ...metric.Float64GaugeOption) (Float64Recorder, error) {
	if m.noop {
		return Float64Recorder{}, nil
	}
	cfg := metric.NewFloat64GaugeConfig(options...)
	const kind = InstrumentKindGauge
	p := float64InstProvider{m}
	i, err := p.lookup(kind, name, cfg.Description(), cfg.Unit(), defaultAttributes(options), h)
	if err != nil {
		return Float64Recorder{i}, err
	}

	return Float64Recorder{i}, validateInstrumentName(name)
}

// float64ObservableInstrument returns a new observable identified by the Instrument.
// It registers callbacks for each reader's pipeline.
func (m *Meter) float64ObservableInstrument(
	id Instrument,
	allowedKeys []attribute.Key,
	callbacks []metric.Float64Callback,
	h otel.ErrorHandler,
) (Float64Observable, error) {
	key := instID{
		Name:        id.Name,
		Description: id.Description,
		Unit:        id.Unit,
		Kind:        id.Kind,
	}
	if m.float64ObservableInsts.HasKey(key) && len(callbacks) > 0 {
		warnRepeatedObservableCallbacks(id)
	}
	return m.float64ObservableInsts.Lookup(key, func() (Float64Observable, error) {
		inst := newFloat64Observable(m, id.Kind, id.Name, id.Description, id.Unit)
		insert := m.float64Resolver.inserter
		// Connect the measure functions for instruments in this pipeline with the
		// callbacks for this pipeline.
		in, err := insert.Instrument(id, allowedKeys, selectAggregation(id.Kind), h)
		if err != nil {
			return inst, err
		}
		// Drop aggregation
		if len(in) == 0 {
			inst.dropAggregation = true
			return inst, validateInstrumentName(id.Name)
		}
		inst.appendMeasures(in)

		// Add the measures to the pipeline. It is required to maintain
		// measures per pipeline to avoid calling the measure that
		// is not part of the pipeline.
		insert.pipeline.addFloat64Measure(inst.observableID, in)
		for _, cback := range callbacks {
			inst := float64Observer{measures: in}
			fn := cback
			insert.addCallback(func(ctx context.Context) error { return fn(ctx, inst) })
		}

		return inst, validateInstrumentName(id.Name)
	})
}

// Float64ObservableCounter returns a new instrument identified by name and
// configured with options. The instrument is used to asynchronously record
// increasing float64 measurements once per a measurement collection cycle.
// Only the measurements recorded during the collection cycle are exported.
//
// If Float64ObservableCounter is invoked repeatedly with the same Name,
// Description, and Unit, only the first set of callbacks provided are used.
// Use meter.RegisterCallback and Registration.Unregister to manage callbacks
// if instrumentation can be created multiple times with different callbacks.
func (m *Meter) Float64ObservableCounter(
	name string,
	h otel.ErrorHandler,
	options ...metric.Float64ObservableCounterOption,
) (Float64Observable, error) {
	if m.noop {
		return Float64Observable{}, nil
	}
	cfg := metric.NewFloat64ObservableCounterConfig(options...)
	id := Instrument{
		Name:        name,
		Description: cfg.Description(),
		Unit:        cfg.Unit(),
		Kind:        InstrumentKindObservableCounter,
		Scope:       m.scope,
	}
	return m.float64ObservableInstrument(id, defaultAttributes(options), cfg.Callbacks(), h)
}

// Float64ObservableUpDownCounter returns a new instrument identified by name
// and configured with options. The instrument is used to asynchronously record
// float64 measurements once per a measurement collection cycle. Only the
// measurements recorded during the collection cycle are exported.
//
// If Float64ObservableUpDownCounter is invoked repeatedly with the same Name,
// Description, and Unit, only the first set of callbacks provided are used.
// Use meter.RegisterCallback and Registration.Unregister to manage callbacks
// if instrumentation can be created multiple times with different callbacks.
func (m *Meter) Float64ObservableUpDownCounter(
	name string,
	h otel.ErrorHandler,
	options ...metric.Float64ObservableUpDownCounterOption,
) (Float64Observable, error) {
	if m.noop {
		return Float64Observable{}, nil
	}
	cfg := metric.NewFloat64ObservableUpDownCounterConfig(options...)
	id := Instrument{
		Name:        name,
		Description: cfg.Description(),
		Unit:        cfg.Unit(),
		Kind:        InstrumentKindObservableUpDownCounter,
		Scope:       m.scope,
	}
	return m.float64ObservableInstrument(id, defaultAttributes(options), cfg.Callbacks(), h)
}

// Float64ObservableGauge returns a new instrument identified by name and
// configured with options. The instrument is used to asynchronously record
// instantaneous float64 measurements once per a measurement collection cycle.
// Only the measurements recorded during the collection cycle are exported.
//
// If Float64ObservableGauge is invoked repeatedly with the same Name,
// Description, and Unit, only the first set of callbacks provided are used.
// Use meter.RegisterCallback and Registration.Unregister to manage callbacks
// if instrumentation can be created multiple times with different callbacks.
func (m *Meter) Float64ObservableGauge(
	name string,
	h otel.ErrorHandler,
	options ...metric.Float64ObservableGaugeOption,
) (Float64Observable, error) {
	if m.noop {
		return Float64Observable{}, nil
	}
	cfg := metric.NewFloat64ObservableGaugeConfig(options...)
	id := Instrument{
		Name:        name,
		Description: cfg.Description(),
		Unit:        cfg.Unit(),
		Kind:        InstrumentKindObservableGauge,
		Scope:       m.scope,
	}
	return m.float64ObservableInstrument(id, defaultAttributes(options), cfg.Callbacks(), h)
}

func validateInstrumentName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: %s: is empty", ErrInstrumentName, name)
	}
	if len(name) > 255 {
		return fmt.Errorf("%w: %s: longer than 255 characters", ErrInstrumentName, name)
	}
	if !isAlpha([]rune(name)[0]) {
		return fmt.Errorf("%w: %s: must start with a letter", ErrInstrumentName, name)
	}
	if len(name) == 1 {
		return nil
	}
	for _, c := range name[1:] {
		if !isAlphanumeric(c) && c != '_' && c != '.' && c != '-' && c != '/' {
			return fmt.Errorf("%w: %s: must only contain [A-Za-z0-9_.-/]", ErrInstrumentName, name)
		}
	}
	return nil
}

func isAlpha(c rune) bool {
	return ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z')
}

func isAlphanumeric(c rune) bool {
	return isAlpha(c) || ('0' <= c && c <= '9')
}

func warnRepeatedObservableCallbacks(id Instrument) {
	inst := fmt.Sprintf(
		"Instrument{Name: %q, Description: %q, Kind: %q, Unit: %q}",
		id.Name, id.Description, "InstrumentKind"+id.Kind.String(), id.Unit,
	)
	global.Warn(
		"Repeated observable instrument creation with callbacks. Ignoring new callbacks. Use meter.RegisterCallback and Registration.Unregister to manage callbacks.",
		"instrument",
		inst,
	)
}

type Observable interface {
	observableMarker()
}

type Callback func(context.Context, Observer) error

// RegisterCallback registers f to be called each collection cycle so it will
// make observations for insts during those cycles.
//
// The only instruments f can make observations for are insts. All other
// observations will be dropped and an error will be logged.
//
// Only instruments from this meter can be registered with f, an error is
// returned if other instrument are provided.
//
// Only observations made in the callback will be exported. Unlike synchronous
// instruments, asynchronous callbacks can "forget" attribute sets that are no
// longer relevant by omitting the observation during the callback.
//
// The returned Registration can be used to unregister f.
func (m *Meter) RegisterCallback(f Callback, insts ...Observable) (UnregisterFunc, error) {
	if m.noop {
		return UnregisterFunc{}, nil
	}

	if len(insts) == 0 {
		// Don't allocate a observer if not needed.
		return UnregisterFunc{}, nil
	}

	var err error
	validInstruments := make([]Observable, 0, len(insts))
	for _, inst := range insts {
		switch o := inst.(type) {
		case Int64Observable:
			if e := o.registerable(m); e != nil {
				if !errors.Is(e, errEmptyAgg) {
					err = errors.Join(err, e)
				}
				continue
			}

			validInstruments = append(validInstruments, inst)
		case Float64Observable:
			if e := o.registerable(m); e != nil {
				if !errors.Is(e, errEmptyAgg) {
					err = errors.Join(err, e)
				}
				continue
			}

			validInstruments = append(validInstruments, inst)
		default:
			// Instrument external to the SDK.
			return UnregisterFunc{}, errors.New("invalid observable: from different implementation")
		}
	}

	if len(validInstruments) == 0 {
		// All insts use drop aggregation or are invalid.
		return UnregisterFunc{}, err
	}

	pipe := m.pipe
	reg := newObserver(pipe)
	for _, inst := range validInstruments {
		switch o := inst.(type) {
		case Int64Observable:
			reg.registerInt64(o.observableID)
		case Float64Observable:
			reg.registerFloat64(o.observableID)
		}
	}

	// Some or all instruments were valid.
	cBack := func(ctx context.Context) error { return f(ctx, reg) }

	return UnregisterFunc{f: pipe.addMultiCallback(cBack)}, err
}

type Observer struct {
	pipe    *pipeline
	float64 map[observableID[float64]]struct{}
	int64   map[observableID[int64]]struct{}
}

func newObserver(p *pipeline) Observer {
	return Observer{
		pipe:    p,
		float64: make(map[observableID[float64]]struct{}),
		int64:   make(map[observableID[int64]]struct{}),
	}
}

func (r Observer) registerFloat64(id observableID[float64]) {
	r.float64[id] = struct{}{}
}

func (r Observer) registerInt64(id observableID[int64]) {
	r.int64[id] = struct{}{}
}

var (
	errUnknownObserver = errors.New("unknown observable instrument")
	errUnregObserver   = errors.New("observable instrument not registered for callback")
)

func (r Observer) ObserveFloat64(o Float64Observable, v float64, opts ...metric.ObserveOption) {
	if o.observable == nil {
		return
	}

	if _, registered := r.float64[o.observableID]; !registered {
		if !o.dropAggregation {
			global.Error(
				errUnregObserver, "failed to record",
				"name", o.name,
				"description", o.description,
				"unit", o.unit,
				"number", fmt.Sprintf("%T", float64(0)),
			)
		}
		return
	}
	c := metric.NewObserveConfig(opts)
	rawKVs := extractRawKVs(opts)
	set := resolveAttributes(c.Attributes(), rawKVs)
	// Access to r.pipe.float64Measure is already guarded by a lock in pipeline.produce.
	// TODO (#5946): Refactor pipeline and observable measures.
	measures := r.pipe.float64Measures[o.observableID]
	for _, m := range measures {
		m(context.Background(), v, set)
	}
}

func (r Observer) ObserveInt64(o Int64Observable, v int64, opts ...metric.ObserveOption) {
	if o.observable == nil {
		return
	}

	if _, registered := r.int64[o.observableID]; !registered {
		if !o.dropAggregation {
			global.Error(
				errUnregObserver, "failed to record",
				"name", o.name,
				"description", o.description,
				"unit", o.unit,
				"number", fmt.Sprintf("%T", int64(0)),
			)
		}
		return
	}
	c := metric.NewObserveConfig(opts)
	rawKVs := extractRawKVs(opts)
	set := resolveAttributes(c.Attributes(), rawKVs)
	// Access to r.pipe.int64Measures is already guarded b a lock in pipeline.produce.
	// TODO (#5946): Refactor pipeline and observable measures.
	measures := r.pipe.int64Measures[o.observableID]
	for _, m := range measures {
		m(context.Background(), v, set)
	}
}

type noopRegister struct{}

func (noopRegister) Unregister() error {
	return nil
}

// int64InstProvider provides int64 OpenTelemetry instruments.
type int64InstProvider struct{ *Meter }

func (p int64InstProvider) aggs(
	kind InstrumentKind,
	name, desc, u string,
	allowedKeys []attribute.Key,
	h otel.ErrorHandler,
) ([]aggregate.Measure[int64], error) {
	inst := Instrument{
		Name:        name,
		Description: desc,
		Unit:        u,
		Kind:        kind,
		Scope:       p.scope,
	}
	return p.int64Resolver.Aggregators(inst, allowedKeys, h)
}

func (p int64InstProvider) histogramAggs(
	name string,
	cfg metric.Int64HistogramConfig,
	allowedKeys []attribute.Key,
	h otel.ErrorHandler,
) ([]aggregate.Measure[int64], error) {
	boundaries := cfg.ExplicitBucketBoundaries()
	aggError := aggregationExplicitBucketHistogram{Boundaries: boundaries}.Err()
	if aggError != nil {
		// If boundaries are invalid, ignore them.
		boundaries = nil
	}
	inst := Instrument{
		Name:        name,
		Description: cfg.Description(),
		Unit:        cfg.Unit(),
		Kind:        InstrumentKindHistogram,
		Scope:       p.scope,
	}
	measures, err := p.int64Resolver.HistogramAggregators(inst, allowedKeys, boundaries, h)
	return measures, errors.Join(aggError, err)
}

// lookup returns the resolved instrumentImpl.
func (p int64InstProvider) lookup(
	kind InstrumentKind,
	name, desc, u string,
	allowedKeys []attribute.Key,
	h otel.ErrorHandler,
) (*int64Inst, error) {
	return p.int64Insts.Lookup(instID{
		Name:        name,
		Description: desc,
		Unit:        u,
		Kind:        kind,
	}, func() (*int64Inst, error) {
		aggs, err := p.aggs(kind, name, desc, u, allowedKeys, h)
		return &int64Inst{measures: aggs}, err
	})
}

// lookupHistogram returns the resolved instrumentImpl.
func (p int64InstProvider) lookupHistogram(
	name string,
	cfg metric.Int64HistogramConfig,
	allowedKeys []attribute.Key,
	h otel.ErrorHandler,
) (*int64Inst, error) {
	return p.int64Insts.Lookup(instID{
		Name:        name,
		Description: cfg.Description(),
		Unit:        cfg.Unit(),
		Kind:        InstrumentKindHistogram,
	}, func() (*int64Inst, error) {
		aggs, err := p.histogramAggs(name, cfg, allowedKeys, h)
		return &int64Inst{measures: aggs}, err
	})
}

// float64InstProvider provides float64 OpenTelemetry instruments.
type float64InstProvider struct{ *Meter }

func (p float64InstProvider) aggs(
	kind InstrumentKind,
	name, desc, u string,
	allowedKeys []attribute.Key,
	h otel.ErrorHandler,
) ([]aggregate.Measure[float64], error) {
	inst := Instrument{
		Name:        name,
		Description: desc,
		Unit:        u,
		Kind:        kind,
		Scope:       p.scope,
	}
	return p.float64Resolver.Aggregators(inst, allowedKeys, h)
}

func (p float64InstProvider) histogramAggs(
	name string,
	cfg metric.Float64HistogramConfig,
	allowedKeys []attribute.Key,
	h otel.ErrorHandler,
) ([]aggregate.Measure[float64], error) {
	boundaries := cfg.ExplicitBucketBoundaries()
	aggError := aggregationExplicitBucketHistogram{Boundaries: boundaries}.Err()
	if aggError != nil {
		// If boundaries are invalid, ignore them.
		boundaries = nil
	}
	inst := Instrument{
		Name:        name,
		Description: cfg.Description(),
		Unit:        cfg.Unit(),
		Kind:        InstrumentKindHistogram,
		Scope:       p.scope,
	}
	measures, err := p.float64Resolver.HistogramAggregators(inst, allowedKeys, boundaries, h)
	return measures, errors.Join(aggError, err)
}

// lookup returns the resolved instrumentImpl.
func (p float64InstProvider) lookup(
	kind InstrumentKind,
	name, desc, u string,
	allowedKeys []attribute.Key,
	h otel.ErrorHandler,
) (*float64Inst, error) {
	return p.float64Insts.Lookup(instID{
		Name:        name,
		Description: desc,
		Unit:        u,
		Kind:        kind,
	}, func() (*float64Inst, error) {
		aggs, err := p.aggs(kind, name, desc, u, allowedKeys, h)
		return &float64Inst{measures: aggs}, err
	})
}

// lookupHistogram returns the resolved instrumentImpl.
func (p float64InstProvider) lookupHistogram(
	name string,
	cfg metric.Float64HistogramConfig,
	allowedKeys []attribute.Key,
	h otel.ErrorHandler,
) (*float64Inst, error) {
	return p.float64Insts.Lookup(instID{
		Name:        name,
		Description: cfg.Description(),
		Unit:        cfg.Unit(),
		Kind:        InstrumentKindHistogram,
	}, func() (*float64Inst, error) {
		aggs, err := p.histogramAggs(name, cfg, allowedKeys, h)
		return &float64Inst{measures: aggs}, err
	})
}

type int64Observer struct {
	measures[int64]
}

func (o int64Observer) Observe(val int64, opts ...metric.ObserveOption) {
	c := metric.NewObserveConfig(opts)
	rawKVs := extractRawKVs(opts)
	o.observe(val, resolveAttributes(c.Attributes(), rawKVs))
}

type float64Observer struct {
	measures[float64]
}

func (o float64Observer) Observe(val float64, opts ...metric.ObserveOption) {
	c := metric.NewObserveConfig(opts)
	rawKVs := extractRawKVs(opts)
	o.observe(val, resolveAttributes(c.Attributes(), rawKVs))
}

func defaultAttributes[T any](opts []T) []attribute.Key {
	var keys []attribute.Key
	var found bool
	for _, o := range opts {
		if exp, ok := any(o).(interface{ AllowedKeys() []attribute.Key }); ok {
			found = true
			keys = append(keys, exp.AllowedKeys()...)
		}
	}
	if found && keys == nil {
		return []attribute.Key{}
	}
	return keys
}
