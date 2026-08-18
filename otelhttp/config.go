// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelhttp

import (
	"context"
	"net/http"
	"net/http/httptrace"

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
	ServerName        string
	TracerProvider    *sdktrace.TracerProvider
	SpanStartOptions  []trace.SpanStartOption
	PublicEndpointFn  func(*http.Request) bool
	ReadEvent         bool
	WriteEvent        bool
	Filters           []Filter
	SpanNameFormatter func(string, *http.Request) string
	ClientTrace       func(context.Context) *httptrace.ClientTrace

	MeterProvider      metric.MeterProvider
	MetricAttributesFn func(*http.Request) []attribute.KeyValue
}

// // Option interface used for setting optional config properties.
// type Option interface {
// 	apply(*config)
// }

// type optionFunc func(*config)

// func (o optionFunc) apply(c *config) {
// 	o(c)
// }

// newConfig creates a new config struct and applies opts to it.
func newConfig(
	serverName string,
	tracerProvider *sdktrace.TracerProvider,
	spanStartOptions []trace.SpanStartOption,
	PublicEndpointFn func(*http.Request) bool,
	readEvent bool,
	writeEvent bool,
	filters []Filter,
	spanNameFormatter func(string, *http.Request) string,
	clientTrace func(context.Context) *httptrace.ClientTrace,
	meterProvider metric.MeterProvider,
	metricAttributesFn func(*http.Request) []attribute.KeyValue,
) *config {
	return &config{
		ServerName:         serverName,
		TracerProvider:     tracerProvider,
		SpanStartOptions:   spanStartOptions,
		PublicEndpointFn:   PublicEndpointFn,
		ReadEvent:          readEvent,
		WriteEvent:         writeEvent,
		Filters:            filters,
		SpanNameFormatter:  spanNameFormatter,
		ClientTrace:        clientTrace,
		MeterProvider:      meterProvider,
		MetricAttributesFn: metricAttributesFn,
	}
}
