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

	_ metric.Int64Observer   = Int64Observer{}
	_ metric.Float64Observer = Float64Observer{}
)

// Int64Observer is a recorder of int64 measurements that performs no operation.
type Int64Observer struct{}

// Observe performs no operation.
func (Int64Observer) Observe(int64, ...metric.ObserveOption) {}

// Float64Observer is a recorder of float64 measurements that performs no
// operation.
type Float64Observer struct{}

// Observe performs no operation.
func (Float64Observer) Observe(float64, ...metric.ObserveOption) {}
