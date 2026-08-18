// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelhttp

import (
	"net/http"

	sdktrace "github.com/karelbilek/opentelemetry/sdk/trace"
)

const ScopeName = "go.github.com/karelbilek/opentelemetry/otelhttp"

// Filter is a predicate used to determine whether a given http.request should
// be traced. A Filter must return true if the request should be traced.
type Filter func(*http.Request) bool

func newTracer(tp *sdktrace.TracerProvider) *sdktrace.Tracer {
	return tp.Tracer(ScopeName)
}
