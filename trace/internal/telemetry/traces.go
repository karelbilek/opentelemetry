// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Traces represents the traces data that can be stored in a persistent storage,
// OR can be embedded by other protocols that transfer OTLP traces data but do
// not implement the OTLP protocol.
//
// The main difference between this message and collector protocol is that
// in this message there will not be any "control" or "metadata" specific to
// OTLP protocol.
//
// When new fields are added into this message, the OTLP request MUST be updated
// as well.
type Traces struct {
	// An array of ResourceSpans.
	// For data coming from a single resource this array will typically contain
	// one element. Intermediary nodes that receive data from multiple origins
	// typically batch the data before forwarding further and in that case this
	// array will contain multiple elements.
	ResourceSpans []*ResourceSpans `json:"resourceSpans,omitempty"`
}

// UnmarshalJSON decodes the OTLP formatted JSON contained in data into td.
func (td *Traces) UnmarshalJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))

	t, err := decoder.Token()
	if err != nil {
		return err
	}
	if t != json.Delim('{') {
		return errors.New("invalid TracesData type")
	}

	for decoder.More() {
		keyIface, err := decoder.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				// Empty.
				return nil
			}
			return err
		}

		key, ok := keyIface.(string)
		if !ok {
			return fmt.Errorf("invalid TracesData field: %#v", keyIface)
		}

		switch key {
		case "resourceSpans", "resource_spans":
			err = decoder.Decode(&td.ResourceSpans)
		default:
			// Skip unknown.
		}

		if err != nil {
			return err
		}
	}
	return nil
}

// ResourceSpans is a collection of ScopeSpans from a Resource.
type ResourceSpans struct {
	// The resource for the spans in this message.
	// If this field is not set then no resource info is known.
	Resource Resource `json:"resource"`
	// A list of ScopeSpans that originate from a resource.
	ScopeSpans []*ScopeSpans `json:"scopeSpans,omitempty"`
}

// UnmarshalJSON decodes the OTLP formatted JSON contained in data into rs.
func (rs *ResourceSpans) UnmarshalJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))

	t, err := decoder.Token()
	if err != nil {
		return err
	}
	if t != json.Delim('{') {
		return errors.New("invalid ResourceSpans type")
	}

	for decoder.More() {
		keyIface, err := decoder.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				// Empty.
				return nil
			}
			return err
		}

		key, ok := keyIface.(string)
		if !ok {
			return fmt.Errorf("invalid ResourceSpans field: %#v", keyIface)
		}

		switch key {
		case "resource":
			err = decoder.Decode(&rs.Resource)
		case "scopeSpans", "scope_spans":
			err = decoder.Decode(&rs.ScopeSpans)
		default:
			// Skip unknown.
		}

		if err != nil {
			return err
		}
	}
	return nil
}

// ScopeSpans is a collection of Spans produced by an InstrumentationScope.
type ScopeSpans struct {
	// The instrumentation scope information for the spans in this message.
	// Semantically when InstrumentationScope isn't set, it is equivalent with
	// an empty instrumentation scope name (unknown).
	Scope *Scope `json:"scope"`
	// A list of Spans that originate from an instrumentation scope.
	Spans []*Span `json:"spans,omitempty"`
}

// UnmarshalJSON decodes the OTLP formatted JSON contained in data into ss.
func (ss *ScopeSpans) UnmarshalJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))

	t, err := decoder.Token()
	if err != nil {
		return err
	}
	if t != json.Delim('{') {
		return errors.New("invalid ScopeSpans type")
	}

	for decoder.More() {
		keyIface, err := decoder.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				// Empty.
				return nil
			}
			return err
		}

		key, ok := keyIface.(string)
		if !ok {
			return fmt.Errorf("invalid ScopeSpans field: %#v", keyIface)
		}

		switch key {
		case "scope":
			err = decoder.Decode(&ss.Scope)
		case "spans":
			err = decoder.Decode(&ss.Spans)
		default:
			// Skip unknown.
		}

		if err != nil {
			return err
		}
	}
	return nil
}
