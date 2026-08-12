// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package log

import (
	"context"
	"fmt"

	"github.com/karelbilek/opentelemetry/metric"
	"github.com/karelbilek/opentelemetry/sdk"
	semconv "github.com/karelbilek/opentelemetry/semconv"
	"github.com/karelbilek/opentelemetry/semconv/otelconv"
)

// newRecordCounterIncr returns a function that increments the log record
// counter metric. If observability is disabled, it returns nil.
func newRecordCounterIncr(metricProvider metric.MeterProvider) (func(context.Context), error) {
	m := metricProvider.Meter(
		"github.com/karelbilek/opentelemetry/sdk/log",
		metric.WithInstrumentationVersion(sdk.Version()),
		metric.WithSchemaURL(semconv.SchemaURL),
	)

	created, err := otelconv.NewSDKLogCreated(m)
	if err != nil {
		err = fmt.Errorf("failed to create log created metric: %w", err)
		return nil, err
	}
	inst := created.Inst()
	f := func(ctx context.Context) { inst.Add(ctx, 1) }
	return f, nil
}
