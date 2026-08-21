// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package attribute

import (
	"cmp"
	"encoding/json"
	"reflect"
	"slices"
	"sort"

	"github.com/karelbilek/opentelemetry/attribute/internal/xxhash"
)

type (
	// Set is the representation for a distinct attribute set. It manages an
	// immutable set of attributes, with an internal cache for storing
	// attribute encodings.
	//
	// This type will remain comparable for backwards compatibility. The
	// equivalence of Sets across versions is not guaranteed to be stable.
	// Prior versions may find two Sets to be equal or not when compared
	// directly (i.e. ==), but subsequent versions may not. Users should use
	// the Equals method to ensure stable equivalence checking.
	//
	// Users should also use the Distinct returned from Equivalent as a map key
	// instead of a Set directly. Set has relatively poor performance when used
	// as a map key compared to Distinct.
	Set struct {
		hash uint64
		data any
	}

	// Distinct is an identifier of a Set which is very likely to be unique.
	//
	// Distinct should be used as a map key instead of a Set for to provide better
	// performance for map operations.
	Distinct struct {
		hash uint64
	}

	// Sortable implements sort.Interface, used for sorting KeyValue.
	//
	// Deprecated: This type is no longer used. It was added as a performance
	// optimization for Go < 1.21 that is no longer needed (Go < 1.21 is no
	// longer supported by the module).
	Sortable []KeyValue
)

// Compile time check these types remain comparable.
var (
	_ = isComparable(Set{})
	_ = isComparable(Distinct{})
)

func isComparable[T comparable](t T) T { return t }

var (
	// keyValueType is used in computeDistinctReflect.
	keyValueType = reflect.TypeFor[KeyValue]()

	// emptyHash is the hash of an empty set.
	emptyHash = xxhash.New().Sum64()

	// userDefinedEmptySet is an empty set. It was mistakenly exposed to users
	// as something they can assign to, so it must remain addressable and
	// mutable.
	//
	// This is kept for backwards compatibility, but should not be used in new code.
	userDefinedEmptySet = &Set{
		hash: emptyHash,
		data: [0]KeyValue{},
	}

	emptySet = Set{
		hash: emptyHash,
		data: [0]KeyValue{},
	}
)

// EmptySet returns a reference to a Set with no elements.
//
// This is a convenience provided for optimized calling utility.
func EmptySet() *Set {
	// Continue to return the pointer to the user-defined empty set for
	// backwards-compatibility.
	//
	// New code should not use this, instead use emptySet.
	return userDefinedEmptySet
}

// Valid reports whether this value refers to a valid Set.
func (d Distinct) Valid() bool { return d.hash != 0 }

// reflectValue abbreviates reflect.ValueOf(d).
func (l Set) reflectValue() reflect.Value {
	return reflect.ValueOf(l.data)
}

// Len returns the number of attributes in this set.
func (l *Set) Len() int {
	if l == nil || l.hash == 0 {
		return 0
	}
	return l.reflectValue().Len()
}

// Get returns the KeyValue at ordered position idx in this set.
func (l *Set) Get(idx int) (KeyValue, bool) {
	if l == nil || l.hash == 0 {
		return KeyValue{}, false
	}
	value := l.reflectValue()

	if idx >= 0 && idx < value.Len() {
		// Note: The Go compiler successfully avoids an allocation for
		// the interface{} conversion here:
		return value.Index(idx).Interface().(KeyValue), true
	}

	return KeyValue{}, false
}

// Value returns the value of a specified key in this set.
func (l *Set) Value(k Key) (Value, bool) {
	if l == nil || l.hash == 0 {
		return Value{}, false
	}
	rValue := l.reflectValue()
	vlen := rValue.Len()

	idx := sort.Search(vlen, func(idx int) bool {
		return rValue.Index(idx).Interface().(KeyValue).Key >= k
	})
	if idx >= vlen {
		return Value{}, false
	}
	keyValue := rValue.Index(idx).Interface().(KeyValue)
	if k == keyValue.Key {
		return keyValue.Value, true
	}
	return Value{}, false
}

// HasValue reports whether a key is defined in this set.
func (l *Set) HasValue(k Key) bool {
	if l == nil {
		return false
	}
	_, ok := l.Value(k)
	return ok
}

// Iter returns an iterator for visiting the attributes in this set.
func (l *Set) Iter() Iterator {
	return Iterator{
		storage: l,
		idx:     -1,
	}
}

// ToSlice returns the set of attributes belonging to this set, sorted, where
// keys appear no more than once.
func (l *Set) ToSlice() []KeyValue {
	iter := l.Iter()
	return iter.ToSlice()
}

// Equivalent returns a value that may be used as a map key. Equal Distinct
// values are very likely to be equivalent attribute Sets. Distinct value of any
// attribute set with the same elements as this, where sets are made unique by
// choosing the last value in the input for any given key.
func (l *Set) Equivalent() Distinct {
	if l == nil || l.hash == 0 {
		return Distinct{hash: emptySet.hash}
	}
	return Distinct{hash: l.hash}
}

// Equals reports whether the argument set is equivalent to this set.
func (l *Set) Equals(o *Set) bool {
	if l.Equivalent() != o.Equivalent() {
		return false
	}
	if l == nil || l.hash == 0 {
		l = &emptySet
	}
	if o == nil || o.hash == 0 {
		o = &emptySet
	}
	return l.data == o.data
}

// Encoded returns the encoded form of this set, according to encoder.
func (l *Set) Encoded() string {
	if l == nil {
		return ""
	}
	return EncodeAttributes(l.Iter())
}

func NewSet(kvs ...KeyValue) Set {
	// Check for empty set.
	if len(kvs) == 0 {
		return emptySet
	}

	// Stable sort so the following de-duplication can implement
	// last-value-wins semantics.
	slices.SortStableFunc(kvs, func(a, b KeyValue) int {
		return cmp.Compare(a.Key, b.Key)
	})

	position := len(kvs) - 1
	offset := position - 1

	// The requirements stated above require that the stable
	// result be placed in the end of the input slice, while
	// overwritten values are swapped to the beginning.
	//
	// De-duplicate with last-value-wins semantics.  Preserve
	// duplicate values at the beginning of the input slice.
	for ; offset >= 0; offset-- {
		if kvs[offset].Key == kvs[position].Key {
			continue
		}
		position--
		kvs[offset], kvs[position] = kvs[position], kvs[offset]
	}
	kvs = kvs[position:]

	return newSet(kvs)
}

// Filter returns a filtered copy of this Set. See the documentation for
// NewSetWithSortableFiltered for more details.
func (l *Set) Filter() (Set, []KeyValue) {
	return *l, nil
}

// newSet returns a new set based on the sorted and uniqued kvs.
func newSet(kvs []KeyValue) Set {
	s := Set{
		hash: hashKVs(kvs),
		data: computeDataFixed(kvs),
	}
	if s.data == nil {
		s.data = computeDataReflect(kvs)
	}
	return s
}

// computeDataFixed computes a Set data for small slices. It returns nil if the
// input is too large for this code path.
func computeDataFixed(kvs []KeyValue) any {
	switch len(kvs) {
	case 1:
		return [1]KeyValue(kvs)
	case 2:
		return [2]KeyValue(kvs)
	case 3:
		return [3]KeyValue(kvs)
	case 4:
		return [4]KeyValue(kvs)
	case 5:
		return [5]KeyValue(kvs)
	case 6:
		return [6]KeyValue(kvs)
	case 7:
		return [7]KeyValue(kvs)
	case 8:
		return [8]KeyValue(kvs)
	case 9:
		return [9]KeyValue(kvs)
	case 10:
		return [10]KeyValue(kvs)
	default:
		return nil
	}
}

// computeDataReflect computes a Set data using reflection, works for any size
// input.
func computeDataReflect(kvs []KeyValue) any {
	at := reflect.New(reflect.ArrayOf(len(kvs), keyValueType)).Elem()
	for i, keyValue := range kvs {
		*at.Index(i).Addr().Interface().(*KeyValue) = keyValue
	}
	return at.Interface()
}

// MarshalJSON returns the JSON encoding of the Set.
func (l *Set) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.data)
}

// MarshalLog is the marshaling function used by the logging system to represent this Set.
func (l Set) MarshalLog() any {
	kvs := make(map[string]string)
	for _, kv := range l.ToSlice() {
		kvs[string(kv.Key)] = kv.Value.String()
	}
	return kvs
}

// Len implements sort.Interface.
func (l *Sortable) Len() int {
	return len(*l)
}

// Swap implements sort.Interface.
func (l *Sortable) Swap(i, j int) {
	(*l)[i], (*l)[j] = (*l)[j], (*l)[i]
}

// Less implements sort.Interface.
func (l *Sortable) Less(i, j int) bool {
	return (*l)[i].Key < (*l)[j].Key
}
