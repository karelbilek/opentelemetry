// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metric

import (
	"context"

	"github.com/karelbilek/opentelemetry/sdk/resource"
)

// config contains configuration options for a MeterProvider.
type config struct {
	res              *resource.Resource
	reader           Reader
	cardinalityLimit int
}

const defaultCardinalityLimit = 2000

// readerSignals returns a force-flush and shutdown function for a
// MeterProvider to call in their respective options. All Readers c contains
// will have their force-flush and shutdown methods unified into returned
// single functions.
func (c config) readerSignals() (forceFlush, shutdown func(context.Context) error) {
	var fFunc, sFunc func(context.Context) error
	sFunc = c.reader.Shutdown
	if f, ok := c.reader.(interface{ ForceFlush(context.Context) error }); ok {
		fFunc = f.ForceFlush
	} else {
		fFunc = func(ctx context.Context) error { return nil }
	}

	return fFunc, sFunc
}

// newConfig returns a config configured with options.
func newConfig(
	res *resource.Resource,
	reader Reader,
	cardinalityLimit int,
) config {
	return config{
		res:              res,
		reader:           reader,
		cardinalityLimit: cardinalityLimit,
	}
}
