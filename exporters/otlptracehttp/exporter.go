// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlptracehttp

import (
	"context"
	"time"

	"github.com/karelbilek/opentelemetry/exporters/otlptrace"
	"github.com/karelbilek/opentelemetry/metric"
	"github.com/karelbilek/opentelemetry/retry"
)

// New constructs a new Exporter and starts it.
func New(ctx context.Context, mp metric.MeterProvider, endpoint string, insecure bool, headers map[string]string, maxRequestSize int, timeout time.Duration, urlPath string, retry retry.Config) (*otlptrace.Exporter, error) {
	return otlptrace.New(ctx, NewClient(mp, endpoint, insecure, headers, maxRequestSize, timeout, urlPath, retry))
}

// NewUnstarted constructs a new Exporter and does not start it.
func NewUnstarted(mp metric.MeterProvider, endpoint string, insecure bool, headers map[string]string, maxRequestSize int, timeout time.Duration, urlPath string, retry retry.Config) *otlptrace.Exporter {
	return otlptrace.NewUnstarted(NewClient(mp, endpoint, insecure, headers, maxRequestSize, timeout, urlPath, retry))
}
