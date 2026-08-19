// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metric

import (
	"github.com/karelbilek/opentelemetry/sdk/resource"
)

// config contains configuration options for a MeterProvider.
type config struct {
	res              *resource.Resource
	reader           *PeriodicReader
	cardinalityLimit int
}

const defaultCardinalityLimit = 2000

// newConfig returns a config configured with options.
func newConfig(
	res *resource.Resource,
	reader *PeriodicReader,
	cardinalityLimit int,
) config {
	return config{
		res:              res,
		reader:           reader,
		cardinalityLimit: cardinalityLimit,
	}
}
