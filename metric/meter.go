// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metric

import (
	"context"
)

// Callback is a function registered with a Meter that makes observations for
// the set of instruments it is registered with. The Observer parameter is used
// to record measurement observations for these instruments.
//
// The function needs to complete in a finite amount of time and the deadline
// of the passed context is expected to be honored.
//
// The function needs to make unique observations across all registered
// Callbacks. Meaning, it should not report measurements for an instrument with
// the same attributes as another Callback will report.
//
// The function needs to be reentrant and concurrent safe.
//
// Note that Go's mutexes are not reentrant, and locking a mutex takes
// an indefinite amount of time. It is therefore advised to avoid
// using mutexes inside callbacks.
type Callback func(context.Context, Observer) error

// Observer records measurements for multiple instruments in a Callback.
//
// Warning: Methods may be added to this interface in minor releases. See
// package documentation on API implementation for information on how to set
// default behavior for unimplemented methods.
type Observer interface {
	// ObserveFloat64 records the float64 value for obsrv.
	//
	// Implementations of this method need to be safe for a user to call
	// concurrently.
	ObserveFloat64(obsrv Float64Observable, value float64, opts ...ObserveOption)

	// ObserveInt64 records the int64 value for obsrv.
	//
	// Implementations of this method need to be safe for a user to call
	// concurrently.
	ObserveInt64(obsrv Int64Observable, value int64, opts ...ObserveOption)
}

// Registration is an token representing the unique registration of a callback
// for a set of instruments with a Meter.
//
// Warning: Methods may be added to this interface in minor releases. See
// package documentation on API implementation for information on how to set
// default behavior for unimplemented methods.
type Registration interface {

	// Unregister removes the callback registration from a Meter.
	//
	// Implementations of this method need to be idempotent and safe for a user
	//  to call concurrently.
	Unregister() error
}
