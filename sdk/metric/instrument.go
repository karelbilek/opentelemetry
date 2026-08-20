// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:generate stringer -type=InstrumentKind -trimprefix=InstrumentKind

package metric

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/karelbilek/opentelemetry/attribute"
	"github.com/karelbilek/opentelemetry/metric"
	"github.com/karelbilek/opentelemetry/sdk/instrumentation"
	"github.com/karelbilek/opentelemetry/sdk/metric/internal/aggregate"
	"github.com/karelbilek/opentelemetry/sdk/metric/internal/attrnorm"
)

var zeroScope instrumentation.Scope

type nonComparable [0]func() // nolint: unused  // This is indeed used.

// Instrument describes properties an instrument is created with.
type Instrument struct {
	// Name is the human-readable identifier of the instrument.
	Name string
	// Description describes the purpose of the instrument.
	Description string
	// Kind defines the functional group of the instrument.
	Kind InstrumentKind
	// Unit is the unit of measurement recorded by the instrument.
	Unit string
	// Scope identifies the instrumentation that created the instrument.
	Scope instrumentation.Scope

	// Ensure forward compatibility if non-comparable fields need to be added.
	nonComparable // nolint: unused
}

// IsEmpty reports whether all Instrument fields are their zero-value.
func (i Instrument) IsEmpty() bool {
	return i.Name == "" &&
		i.Description == "" &&
		i.Kind == InstrumentKindUndefined &&
		i.Unit == "" &&
		i.Scope == zeroScope
}

// matches returns whether all the non-zero-value fields of i match the
// corresponding fields of other. If i is empty it will match all other, and
// true will always be returned.
func (i Instrument) matches(other Instrument) bool {
	return i.matchesName(other) &&
		i.matchesDescription(other) &&
		i.matchesKind(other) &&
		i.matchesUnit(other) &&
		i.matchesScope(other)
}

// matchesName returns true if the Name of i is "" or it equals the Name of
// other, otherwise false.
func (i Instrument) matchesName(other Instrument) bool {
	return i.Name == "" || i.Name == other.Name
}

// matchesDescription returns true if the Description of i is "" or it equals
// the Description of other, otherwise false.
func (i Instrument) matchesDescription(other Instrument) bool {
	return i.Description == "" || i.Description == other.Description
}

// matchesKind returns true if the Kind of i is its zero-value or it equals the
// Kind of other, otherwise false.
func (i Instrument) matchesKind(other Instrument) bool {
	return i.Kind == InstrumentKindUndefined || i.Kind == other.Kind
}

// matchesUnit returns true if the Unit of i is its zero-value or it equals the
// Unit of other, otherwise false.
func (i Instrument) matchesUnit(other Instrument) bool {
	return i.Unit == "" || i.Unit == other.Unit
}

// matchesScope returns true if the Scope of i is its zero-value or it equals
// the Scope of other, otherwise false.
func (i Instrument) matchesScope(other Instrument) bool {
	return (i.Scope.Name == "" || i.Scope.Name == other.Scope.Name)
}

// Stream describes the stream of data an instrument produces.
type Stream struct {
	// Name is the human-readable identifier of the stream.
	Name string
	// Description describes the purpose of the data.
	Description string
	// Unit is the unit of measurement recorded.
	Unit string
	// Aggregation the stream uses for an instrument.
	Aggregation aggregation
	// AttributeFilter is an attribute Filter applied to the attributes
	// recorded for an instrument's measurement. If the filter returns false
	// the attribute will not be recorded, otherwise, if it returns true, it
	// will record the attribute.
	//
	// Note that attributes filtered out by a View may still appear on Exemplars,
	// because Exemplars are recorded with the dropped measurement attributes
	// when View attribute filtering is applied.
	//
	// Use NewAllowKeysFilter from "github.com/karelbilek/opentelemetry/attribute" to
	// provide an allow-list of attribute keys here.
	AttributeFilter attribute.Filter
	// ExemplarReservoirProvider selects the
	// [github.com/karelbilek/opentelemetry/sdk/metric/exemplar.ReservoirProvider] based
	// on the [Aggregation].
	//
	// If unspecified, [DefaultExemplarReservoirProviderSelector] is used.
	// ExemplarReservoirProviderSelector ExemplarReservoirProviderSelector
}

// instID are the identifying properties of a instrument.
type instID struct {
	// Name is the name of the stream.
	Name string
	// Description is the description of the stream.
	Description string
	// Kind defines the functional group of the instrument.
	Kind InstrumentKind
	// Unit is the unit of the stream.
	Unit string
	// Number is the number type of the stream.
	Number string
}

// Returns a normalized copy of the instID i.
//
// Instrument names are considered case-insensitive. Standardize the instrument
// name to always be lowercase for the returned instID so it can be compared
// without the name casing affecting the comparison.
func (i instID) normalize() instID {
	i.Name = strings.ToLower(i.Name)
	return i
}

type rawAttributesOption interface {
	RawAttributes() []attribute.KeyValue
	Experimental()
}

func extractRawKVs[T any](opts []T) []attribute.KeyValue {
	var rawKVs []attribute.KeyValue
	var count int
	for _, opt := range opts {
		if r, ok := any(opt).(rawAttributesOption); ok {
			count++
			if count == 1 {
				rawKVs = r.RawAttributes()
			} else {
				if count == 2 {
					// Create a new slice to avoid modifying the original slice from the first option.
					rawKVs = append([]attribute.KeyValue(nil), rawKVs...)
				}
				rawKVs = append(rawKVs, r.RawAttributes()...)
			}
		}
	}
	return rawKVs
}

func resolveAttributes(configAttrs attribute.Set, rawKVs []attribute.KeyValue) attribute.Set {
	configAttrs, _ = attrnorm.Set(configAttrs)
	if len(rawKVs) == 0 {
		return configAttrs
	}
	rawKVs, _ = attrnorm.KeyValues(rawKVs)
	merged := make([]attribute.KeyValue, 0, configAttrs.Len()+len(rawKVs))
	merged = append(merged, configAttrs.ToSlice()...)
	// rawKVs are appended after configAttrs, meaning they will override any duplicate keys in configAttrs.
	// This behavior is documented in WithUnsafeAttributes.
	merged = append(merged, rawKVs...)
	// TODO(#7743): Defer computing the full attribute.NewSet.
	return attribute.NewSet(merged...)
}

type int64Inst struct {
	measures []aggregate.Measure[int64]
}

type Int64Adder struct {
	inst *int64Inst
}

type Int64Recorder struct {
	inst *int64Inst
}

func (i Int64Adder) Add(ctx context.Context, val int64, opts ...metric.AddOption) {
	i.inst.Add(ctx, val, opts...)
}

func (i Int64Adder) Enabled(ctx context.Context) bool {
	return i.inst.Enabled(ctx)
}

func (i Int64Recorder) Record(ctx context.Context, val int64, opts ...metric.RecordOption) {
	i.inst.Record(ctx, val, opts...)
}

func (i Int64Recorder) Enabled(ctx context.Context) bool {
	return i.inst.Enabled(ctx)
}

func (i *int64Inst) Add(ctx context.Context, val int64, opts ...metric.AddOption) {
	if i == nil {
		return
	}
	c := metric.NewAddConfig(opts)
	rawKVs := extractRawKVs(opts)
	i.aggregate(ctx, val, resolveAttributes(c.Attributes(), rawKVs))
}

func (i *int64Inst) Record(ctx context.Context, val int64, opts ...metric.RecordOption) {
	if i == nil {
		return
	}
	c := metric.NewRecordConfig(opts)
	rawKVs := extractRawKVs(opts)
	i.aggregate(ctx, val, resolveAttributes(c.Attributes(), rawKVs))
}

func (i *int64Inst) Enabled(context.Context) bool {
	if i == nil {
		return false
	}
	return len(i.measures) != 0
}

func (i *int64Inst) aggregate(
	ctx context.Context,
	val int64,
	s attribute.Set,
) { // nolint:revive  // okay to shadow pkg with method.
	for _, in := range i.measures {
		in(ctx, val, s)
	}
}

type float64Inst struct {
	measures []aggregate.Measure[float64]
}

type Float64Adder struct {
	inst *float64Inst
}

type Float64Recorder struct {
	inst *float64Inst
}

func (i Float64Adder) Add(ctx context.Context, val float64, opts ...metric.AddOption) {
	i.inst.Add(ctx, val, opts...)
}

func (i Float64Adder) Enabled(ctx context.Context) bool {
	return i.inst.Enabled(ctx)
}

func (i Float64Recorder) Record(ctx context.Context, val float64, opts ...metric.RecordOption) {
	i.inst.Record(ctx, val, opts...)
}

func (i Float64Recorder) Enabled(ctx context.Context) bool {
	return i.inst.Enabled(ctx)
}

func (i *float64Inst) Add(ctx context.Context, val float64, opts ...metric.AddOption) {
	if i == nil {
		return
	}
	c := metric.NewAddConfig(opts)
	rawKVs := extractRawKVs(opts)
	i.aggregate(ctx, val, resolveAttributes(c.Attributes(), rawKVs))
}

func (i *float64Inst) Record(ctx context.Context, val float64, opts ...metric.RecordOption) {
	if i == nil {
		return
	}
	c := metric.NewRecordConfig(opts)
	rawKVs := extractRawKVs(opts)
	i.aggregate(ctx, val, resolveAttributes(c.Attributes(), rawKVs))
}

func (i *float64Inst) Enabled(context.Context) bool {
	if i == nil {
		return false
	}
	return len(i.measures) != 0
}

func (i *float64Inst) aggregate(ctx context.Context, val float64, s attribute.Set) {
	for _, in := range i.measures {
		in(ctx, val, s)
	}
}

// observableID is a comparable unique identifier of an observable.
type observableID[N int64 | float64] struct {
	name        string
	description string
	kind        InstrumentKind
	unit        string
	scope       instrumentation.Scope
}

type Float64Observable struct {
	*observable[float64]
}

func (Float64Observable) observableMarker() {}

func newFloat64Observable(m *Meter, kind InstrumentKind, name, desc, u string) Float64Observable {
	return Float64Observable{
		observable: newObservable[float64](m, kind, name, desc, u),
	}
}

type Int64Observable struct {
	*observable[int64]
}

func (Int64Observable) observableMarker() {}

func newInt64Observable(m *Meter, kind InstrumentKind, name, desc, u string) Int64Observable {
	return Int64Observable{
		observable: newObservable[int64](m, kind, name, desc, u),
	}
}

type observable[N int64 | float64] struct {
	observableID[N]

	meter           *Meter
	measures        measures[N]
	dropAggregation bool
}

func newObservable[N int64 | float64](m *Meter, kind InstrumentKind, name, desc, u string) *observable[N] {
	return &observable[N]{
		observableID: observableID[N]{
			name:        name,
			description: desc,
			kind:        kind,
			unit:        u,
			scope:       m.scope,
		},
		meter: m,
	}
}

// observe records the val for the set of attrs.
func (o *observable[N]) observe(val N, s attribute.Set) {
	o.measures.observe(val, s)
}

func (o *observable[N]) appendMeasures(meas []aggregate.Measure[N]) {
	o.measures = append(o.measures, meas...)
}

type measures[N int64 | float64] []aggregate.Measure[N]

// observe records the val for the set of attrs.
func (m measures[N]) observe(val N, s attribute.Set) {
	for _, in := range m {
		in(context.Background(), val, s)
	}
}

var errEmptyAgg = errors.New("no aggregators for observable instrument")

// registerable returns an error if the observable o should not be registered,
// and nil if it should. An errEmptyAgg error is returned if o is effectively a
// no-op because it does not have any aggregators. Also, an error is returned
// if scope defines a Meter other than the one o was created by.
func (o *observable[N]) registerable(m *Meter) error {
	if o == nil {
		return errEmptyAgg
	}
	if len(o.measures) == 0 {
		return errEmptyAgg
	}
	if m != o.meter {
		return fmt.Errorf(
			"invalid registration: observable %q from Meter %q, registered with Meter %q",
			o.name,
			o.scope.Name,
			m.scope.Name,
		)
	}
	return nil
}
