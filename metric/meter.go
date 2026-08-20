// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metric

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
