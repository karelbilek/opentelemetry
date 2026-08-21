// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package aggregate

import "github.com/karelbilek/opentelemetry/attribute"

// overflowSet is the attribute set used to record a measurement when adding
// another distinct attribute set to the aggregate would exceed the aggregate
// limit.
var overflowSet = attribute.NewSet(attribute.Bool("otel.metric.overflow", true))
