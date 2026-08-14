// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metric

import (
	"context"
	"errors"
	"sync"

	"github.com/karelbilek/opentelemetry/sdk/resource"
)

// config contains configuration options for a MeterProvider.
type config struct {
	res              *resource.Resource
	readers          []Reader
	views            []View
	cardinalityLimit int
}

const defaultCardinalityLimit = 2000

// readerSignals returns a force-flush and shutdown function for a
// MeterProvider to call in their respective options. All Readers c contains
// will have their force-flush and shutdown methods unified into returned
// single functions.
func (c config) readerSignals() (forceFlush, shutdown func(context.Context) error) {
	var fFuncs, sFuncs []func(context.Context) error
	for _, r := range c.readers {
		sFuncs = append(sFuncs, r.Shutdown)
		if f, ok := r.(interface{ ForceFlush(context.Context) error }); ok {
			fFuncs = append(fFuncs, f.ForceFlush)
		}
	}

	return unify(fFuncs), unifyShutdown(sFuncs)
}

// unify unifies calling all of funcs into a single function call. All errors
// returned from calls to funcs will be unify into a single error return
// value.
func unify(funcs []func(context.Context) error) func(context.Context) error {
	return func(ctx context.Context) error {
		var err error
		for _, f := range funcs {
			if e := f(ctx); e != nil {
				err = errors.Join(err, e)
			}
		}
		return err
	}
}

// unifyShutdown unifies calling all of funcs once for a shutdown. If called
// more than once, an ErrReaderShutdown error is returned.
func unifyShutdown(funcs []func(context.Context) error) func(context.Context) error {
	f := unify(funcs)
	var once sync.Once
	return func(ctx context.Context) error {
		err := ErrReaderShutdown
		once.Do(func() { err = f(ctx) })
		return err
	}
}


// newConfig returns a config configured with options.
func newConfig(
	res *resource.Resource,
	readers []Reader,
	views []View,
	cardinalityLimit int,
) config {
	return config{
		res:              res,
		readers:          readers,
		views:            views,
		cardinalityLimit: cardinalityLimit,
	}
}
