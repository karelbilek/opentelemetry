// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package log

// EnabledParameters represents payload for [Logger]'s Enabled method.
type EnabledParameters struct {
	Severity  Severity
	EventName string
}
