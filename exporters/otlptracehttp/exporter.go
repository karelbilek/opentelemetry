// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlptracehttp

import (
	"context"
	"time"

	"github.com/karelbilek/opentelemetry/exporters/otlptrace"
	"github.com/karelbilek/opentelemetry/retry"
)

// New constructs a new Exporter and starts it.
func New(ctx context.Context, endpoint string, urlPath string, insecure bool, headers map[string]string, maxRequestSize int, timeout time.Duration, retry retry.Config) (*otlptrace.Exporter, error) {
	return otlptrace.New(ctx, NewClient(endpoint, urlPath, insecure, headers, maxRequestSize, timeout, retry))
}

// NewUnstarted constructs a new Exporter and does not start it.
func NewUnstarted(endpoint string, urlPath string, insecure bool, headers map[string]string, maxRequestSize int, timeout time.Duration, retry retry.Config) *otlptrace.Exporter {
	return otlptrace.NewUnstarted(NewClient(endpoint, urlPath, insecure, headers, maxRequestSize, timeout, retry))
}
