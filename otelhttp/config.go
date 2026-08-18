// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelhttp

import (
	"net/http"

	"github.com/karelbilek/opentelemetry/attribute"
	"github.com/karelbilek/opentelemetry/metric"
	sdktrace "github.com/karelbilek/opentelemetry/sdk/trace"
	"github.com/karelbilek/opentelemetry/trace"
)

// ScopeName is the instrumentation scope name.
const ScopeName = "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

// config represents the configuration options available for the http.Handler
// and http.Transport types.
type config struct {
	ServerName       string
	TracerProvider   *sdktrace.TracerProvider
	SpanStartOptions []trace.SpanStartOption
	PublicEndpointFn func(*http.Request) bool
	Filters          []Filter

	MeterProvider      metric.MeterProvider
	MetricAttributesFn func(*http.Request) []attribute.KeyValue
}

// newConfig creates a new config struct and applies opts to it.
func newConfig(
	serverName string,
	tracerProvider *sdktrace.TracerProvider,
	spanStartOptions []trace.SpanStartOption,
	publicEndpointFn func(*http.Request) bool,
	filters []Filter,
	meterProvider metric.MeterProvider,
	metricAttributesFn func(*http.Request) []attribute.KeyValue,
) *config {
	return &config{
		ServerName:         serverName,
		TracerProvider:     tracerProvider,
		SpanStartOptions:   spanStartOptions,
		PublicEndpointFn:   publicEndpointFn,
		Filters:            filters,
		MeterProvider:      meterProvider,
		MetricAttributesFn: metricAttributesFn,
	}
}
