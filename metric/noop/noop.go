// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package noop provides an implementation of the OpenTelemetry metric API that
// produces no telemetry and minimizes used computation resources.
//
// Using this package to implement the OpenTelemetry metric API will
// effectively disable OpenTelemetry.
//
// This implementation can be embedded in other implementations of the
// OpenTelemetry metric API. Doing so will mean the implementation defaults to
// no operation for methods it does not implement.
package noop

import (
	"github.com/karelbilek/opentelemetry/metric"
)

var (
	// Compile-time check this implements the OpenTelemetry API.

	_ metric.Observer                       = Observer{}
	_ metric.Registration                   = Registration{}
	_ metric.Int64ObservableCounter         = Int64ObservableCounter{}
	_ metric.Float64ObservableCounter       = Float64ObservableCounter{}
	_ metric.Int64ObservableGauge           = Int64ObservableGauge{}
	_ metric.Float64ObservableGauge         = Float64ObservableGauge{}
	_ metric.Int64ObservableUpDownCounter   = Int64ObservableUpDownCounter{}
	_ metric.Float64ObservableUpDownCounter = Float64ObservableUpDownCounter{}
	_ metric.Int64Observer                  = Int64Observer{}
	_ metric.Float64Observer                = Float64Observer{}
)

// Observer acts as a recorder of measurements for multiple instruments in a
// Callback, it performing no operation.
type Observer struct{}

// ObserveFloat64 performs no operation.
func (Observer) ObserveFloat64(metric.Float64Observable, float64, ...metric.ObserveOption) {
}

// ObserveInt64 performs no operation.
func (Observer) ObserveInt64(metric.Int64Observable, int64, ...metric.ObserveOption) {
}

// Registration is the registration of a Callback with a No-Op Meter.
type Registration struct{}

// Unregister unregisters the Callback the Registration represents with the
// No-Op Meter. This will always return nil because the No-Op Meter performs no
// operation, including hold any record of registrations.
func (Registration) Unregister() error { return nil }

// Int64ObservableCounter is an OpenTelemetry ObservableCounter used to record
// int64 measurements. It produces no telemetry.
type Int64ObservableCounter struct {
	metric.Int64Observable
}

// Float64ObservableCounter is an OpenTelemetry ObservableCounter used to record
// float64 measurements. It produces no telemetry.
type Float64ObservableCounter struct {
	metric.Float64Observable
}

// Int64ObservableGauge is an OpenTelemetry ObservableGauge used to record
// int64 measurements. It produces no telemetry.
type Int64ObservableGauge struct {
	metric.Int64Observable
}

// Float64ObservableGauge is an OpenTelemetry ObservableGauge used to record
// float64 measurements. It produces no telemetry.
type Float64ObservableGauge struct {
	metric.Float64Observable
}

// Int64ObservableUpDownCounter is an OpenTelemetry ObservableUpDownCounter
// used to record int64 measurements. It produces no telemetry.
type Int64ObservableUpDownCounter struct {
	metric.Int64Observable
}

// Float64ObservableUpDownCounter is an OpenTelemetry ObservableUpDownCounter
// used to record float64 measurements. It produces no telemetry.
type Float64ObservableUpDownCounter struct {
	metric.Float64Observable
}

// Int64Observer is a recorder of int64 measurements that performs no operation.
type Int64Observer struct{}

// Observe performs no operation.
func (Int64Observer) Observe(int64, ...metric.ObserveOption) {}

// Float64Observer is a recorder of float64 measurements that performs no
// operation.
type Float64Observer struct{}

// Observe performs no operation.
func (Float64Observer) Observe(float64, ...metric.ObserveOption) {}
