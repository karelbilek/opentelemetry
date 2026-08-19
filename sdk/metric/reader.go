// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metric

import (
	"context"
	"errors"

	"github.com/karelbilek/opentelemetry/sdk/metric/metricdata"
)

// errDuplicateRegister is logged by a Reader when an attempt to registered it
// more than once occurs.
var errDuplicateRegister = errors.New("duplicate reader registration")

// ErrReaderNotRegistered is returned if Collect or Shutdown are called before
// the reader is registered with a MeterProvider.
var ErrReaderNotRegistered = errors.New("reader is not registered")

// ErrReaderShutdown is returned if Collect or Shutdown are called after a
// reader has been Shutdown once.
var ErrReaderShutdown = errors.New("reader is shutdown")

// sdkProducer produces metrics for a Reader.
type sdkProducer interface {
	// produce returns aggregated metrics from a single collection.
	//
	// This method is safe to call concurrently.
	produce(context.Context, *metricdata.ResourceMetrics) error
}

// produceHolder is used as an atomic.Value to wrap the non-concrete producer
// type.
type produceHolder struct {
	produce func(context.Context, *metricdata.ResourceMetrics) error
}

// shutdownProducer produces an ErrReaderShutdown error always.
type shutdownProducer struct{}

// produce returns an ErrReaderShutdown error.
func (shutdownProducer) produce(context.Context, *metricdata.ResourceMetrics) error {
	return ErrReaderShutdown
}
