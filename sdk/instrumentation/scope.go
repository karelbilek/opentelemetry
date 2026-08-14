// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

// Scope represents the instrumentation scope.
type Scope struct {
	// Name is the name of the instrumentation scope. This should be the
	// Go package name of that scope.
	Name string
}
