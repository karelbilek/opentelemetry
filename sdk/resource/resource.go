// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package resource

import (
	"context"
	"sync"

	otel "github.com/karelbilek/opentelemetry"
	"github.com/karelbilek/opentelemetry/attribute"
	"github.com/karelbilek/opentelemetry/sdk/internal/attrnorm"
)

// Resource describes an entity about which identifying information
// and metadata is exposed.  Resource is an immutable object,
// equivalent to a map from key to unique value.
//
// Resources should be passed and stored as pointers
// (`*resource.Resource`).  The `nil` value is equivalent to an empty
// Resource.
//
// Note that the Go == operator compares not just the resource attributes but
// also all other internals of the Resource type. Therefore, Resource values
// should not be used as map or database keys. In general, the [Resource.Equal]
// method should be used instead of direct comparison with ==, since that
// method ensures the correct comparison of resource attributes, and the
// [attribute.Distinct] returned from [Resource.Equivalent] should be used for
// map and database keys instead.
type Resource struct {
	attrs attribute.Set
}

// Compile-time check that the Resource remains comparable.
var _ map[Resource]struct{} = nil

var (
	defaultResource     *Resource
	defaultResourceOnce sync.Once
)

// New returns a [Resource] built using opts.
// Duplicate top-level attribute keys and duplicate keys inside map
// values are resolved using last-value-wins semantics.
//
// This may return a partial Resource along with an error containing
// [ErrPartialResource] if options that provide a [Detector] are used and that
// error is returned from one or more of the Detectors.
func New(ctx context.Context, opts ...Option) (*Resource, error) {
	cfg := config{}
	for _, opt := range opts {
		cfg = opt.apply(cfg)
	}

	r := &Resource{}
	return r, detect(ctx, r, cfg.detectors)
}

// NewWithAttributes creates a resource from attrs. If attrs contains duplicate
// top-level attribute keys or duplicate keys inside map values, the last
// value will be used. If attrs contains any invalid items those items will be
// dropped.
func NewWithAttributes(attrs ...attribute.KeyValue) *Resource {
	if len(attrs) == 0 {
		return &Resource{}
	}

	attrs, _ = attrnorm.KeyValues(attrs)

	// Ensure attributes comply with the specification:
	// https://github.com/open-telemetry/opentelemetry-specification/blob/v1.20.0/specification/common/README.md#attribute
	s, _ := attribute.NewSetWithFiltered(attrs, func(kv attribute.KeyValue) bool {
		return kv.Valid()
	})

	// If attrs only contains invalid entries do not allocate a new resource.
	if s.Len() == 0 {
		return &Resource{}
	}

	return &Resource{attrs: s} //nolint
}

// String implements the Stringer interface and provides a
// human-readable form of the resource.
//
// Avoid using this representation as the key in a map of resources,
// use Equivalent() as the key instead.
func (r *Resource) String() string {
	if r == nil {
		return ""
	}
	return r.attrs.Encoded(attribute.DefaultEncoder())
}

// MarshalLog is the marshaling function used by the logging system to represent this Resource.
func (r *Resource) MarshalLog() any {
	return struct {
		Attributes attribute.Set
	}{
		Attributes: r.attrs,
	}
}

// Attributes returns a copy of attributes from the resource in a sorted order.
// To avoid allocating a new slice, use an iterator.
func (r *Resource) Attributes() []attribute.KeyValue {
	if r == nil {
		r = Empty()
	}
	return r.attrs.ToSlice()
}

// Iter returns an iterator of the Resource attributes.
// This is ideal to use if you do not want a copy of the attributes.
func (r *Resource) Iter() attribute.Iterator {
	if r == nil {
		r = Empty()
	}
	return r.attrs.Iter()
}

// Equal reports whether r and o represent the same resource. Two resources can
// be equal even if they have different schema URLs.
//
// See the documentation on the [Resource] type for the pitfalls of using ==
// with Resource values; most code should use Equal instead.
func (r *Resource) Equal(o *Resource) bool {
	if r == nil {
		r = Empty()
	}
	if o == nil {
		o = Empty()
	}
	return r.Equivalent() == o.Equivalent()
}

// Merge creates a new [Resource] by merging a and b.
//
// If there are common keys between a and b, then the value from b will
// overwrite the value from a, even if b's value is empty.
func Merge(a, b *Resource) *Resource {
	if a == nil && b == nil {
		return Empty()
	}
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}

	// Note: 'b' attributes will overwrite 'a' with last-value-wins in attribute.Key()
	// Meaning this is equivalent to: append(a.Attributes(), b.Attributes()...)
	mi := attribute.NewMergeIterator(b.Set(), a.Set())
	combine := make([]attribute.KeyValue, 0, a.Len()+b.Len())
	for mi.Next() {
		combine = append(combine, mi.Attribute())
	}

	return NewWithAttributes(combine...)
}

// Empty returns an instance of Resource with no attributes. It is
// equivalent to a `nil` Resource.
func Empty() *Resource {
	return &Resource{}
}

// Default returns an instance of Resource with a default
// "service.name" and OpenTelemetrySDK attributes.
func Default(errorHandler otel.ErrorHandler, name string) *Resource {
	return DefaultWithContext(context.Background(), errorHandler, name)
}

// DefaultWithContext returns an instance of Resource with a default
// "service.name" and OpenTelemetrySDK attributes.
//
// If the default resource has already been initialized, the provided ctx
// is ignored and the cached resource is returned.
func DefaultWithContext(ctx context.Context, errorHandler otel.ErrorHandler, name string) *Resource {
	defaultResourceOnce.Do(func() {
		var err error
		defaultDetectors := []Detector{
			defaultServiceNameDetector{},
			telemetrySDK{},
		}
		if name != "" {
			defaultDetectors = []Detector{
				fixedServiceNameDetector{s: name},
				telemetrySDK{},
			}
		}

		defaultResource, err = Detect(
			ctx,
			defaultDetectors...,
		)
		if err != nil {
			otel.Handle(errorHandler, err)
		}
		// If Detect did not return a valid resource, fall back to emptyResource.
		if defaultResource == nil {
			defaultResource = &Resource{}
		}
	})
	return defaultResource
}

// Equivalent returns an object that can be compared for equality
// between two resources. This value is suitable for use as a key in
// a map.
func (r *Resource) Equivalent() attribute.Distinct {
	return r.Set().Equivalent()
}

// Set returns the equivalent *attribute.Set of this resource's attributes.
func (r *Resource) Set() *attribute.Set {
	if r == nil {
		r = Empty()
	}
	return &r.attrs
}

// MarshalJSON encodes the resource attributes as a JSON list of { "Key":
// "...", "Value": ... } pairs in order sorted by key.
func (r *Resource) MarshalJSON() ([]byte, error) {
	if r == nil {
		r = Empty()
	}
	return r.attrs.MarshalJSON()
}

// Len returns the number of unique key-values in this Resource.
func (r *Resource) Len() int {
	if r == nil {
		return 0
	}
	return r.attrs.Len()
}

// Encoded returns an encoded representation of the resource.
func (r *Resource) Encoded(enc attribute.Encoder) string {
	if r == nil {
		return ""
	}
	return r.attrs.Encoded(enc)
}
