// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package httpconv provides types and functionality for OpenTelemetry semantic
// conventions in the "http" namespace.
package httpconv

import (
	otel "github.com/karelbilek/opentelemetry"
	"github.com/karelbilek/opentelemetry/metric"
	sdkmetric "github.com/karelbilek/opentelemetry/sdk/metric"

	"github.com/karelbilek/opentelemetry/metric/noop"
)

// ClientRequestBodySize is an instrument used to record metric values conforming
// to the "http.client.request.body.size" semantic conventions. It represents the
// size of HTTP client request bodies.
type ClientRequestBodySize struct {
	sdkmetric.Int64Recorder
}

var newClientRequestBodySizeOpts = []metric.Int64HistogramOption{
	metric.WithDescription("Size of HTTP client request bodies."),
	metric.WithUnit("By"),
}

// NewClientRequestBodySize returns a new ClientRequestBodySize instrument.
func NewClientRequestBodySize(
	m *sdkmetric.Meter,
	h otel.ErrorHandler,
	opt ...metric.Int64HistogramOption,
) (ClientRequestBodySize, error) {
	// Check if the meter is nil.
	if m == nil {
		return ClientRequestBodySize{}, nil
	}

	if len(opt) == 0 {
		opt = newClientRequestBodySizeOpts
	} else {
		opt = append(opt, newClientRequestBodySizeOpts...)
	}

	i, err := m.Int64Histogram(
		"http.client.request.body.size",
		h,
		opt...,
	)
	if err != nil {
		return ClientRequestBodySize{}, err
	}
	return ClientRequestBodySize{i}, nil
}

// Inst returns the underlying metric instrument.
func (m ClientRequestBodySize) Inst() sdkmetric.Int64Recorder {
	return m.Int64Recorder
}

// ClientRequestDuration is an instrument used to record metric values conforming
// to the "http.client.request.duration" semantic conventions. It represents the
// duration of HTTP client requests.
type ClientRequestDuration struct {
	metric.Float64Histogram
}

var newClientRequestDurationOpts = []metric.Float64HistogramOption{
	metric.WithDescription("Duration of HTTP client requests."),
	metric.WithUnit("s"),
	metric.WithExplicitBucketBoundaries([]float64{0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10}...),
}

// NewClientRequestDuration returns a new ClientRequestDuration instrument.
func NewClientRequestDuration(
	m *sdkmetric.Meter,
	h otel.ErrorHandler,
	opt ...metric.Float64HistogramOption,
) (ClientRequestDuration, error) {
	// Check if the meter is nil.
	if m == nil {
		return ClientRequestDuration{noop.Float64Histogram{}}, nil
	}

	if len(opt) == 0 {
		opt = newClientRequestDurationOpts
	} else {
		opt = append(opt, newClientRequestDurationOpts...)
	}

	i, err := m.Float64Histogram(
		"http.client.request.duration",
		h,
		opt...,
	)
	if err != nil {
		return ClientRequestDuration{noop.Float64Histogram{}}, err
	}
	return ClientRequestDuration{i}, nil
}

// Inst returns the underlying metric instrument.
func (m ClientRequestDuration) Inst() metric.Float64Histogram {
	return m.Float64Histogram
}

// ServerRequestBodySize is an instrument used to record metric values conforming
// to the "http.server.request.body.size" semantic conventions. It represents the
// size of HTTP server request bodies.
type ServerRequestBodySize struct {
	sdkmetric.Int64Recorder
}

var newServerRequestBodySizeOpts = []metric.Int64HistogramOption{
	metric.WithDescription("Size of HTTP server request bodies."),
	metric.WithUnit("By"),
}

// NewServerRequestBodySize returns a new ServerRequestBodySize instrument.
func NewServerRequestBodySize(
	m *sdkmetric.Meter,
	h otel.ErrorHandler,

	opt ...metric.Int64HistogramOption,
) (ServerRequestBodySize, error) {
	// Check if the meter is nil.
	if m == nil {
		return ServerRequestBodySize{}, nil
	}

	if len(opt) == 0 {
		opt = newServerRequestBodySizeOpts
	} else {
		opt = append(opt, newServerRequestBodySizeOpts...)
	}

	i, err := m.Int64Histogram(
		"http.server.request.body.size",
		h,
		opt...,
	)
	if err != nil {
		return ServerRequestBodySize{}, err
	}
	return ServerRequestBodySize{i}, nil
}

// Inst returns the underlying metric instrument.
func (m ServerRequestBodySize) Inst() sdkmetric.Int64Recorder {
	return m.Int64Recorder
}

// ServerRequestDuration is an instrument used to record metric values conforming
// to the "http.server.request.duration" semantic conventions. It represents the
// duration of HTTP server requests.
type ServerRequestDuration struct {
	metric.Float64Histogram
}

var newServerRequestDurationOpts = []metric.Float64HistogramOption{
	metric.WithDescription("Duration of HTTP server requests."),
	metric.WithUnit("s"),
	metric.WithExplicitBucketBoundaries([]float64{0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10}...),
}

// NewServerRequestDuration returns a new ServerRequestDuration instrument.
func NewServerRequestDuration(
	m *sdkmetric.Meter,
	h otel.ErrorHandler,
	opt ...metric.Float64HistogramOption,
) (ServerRequestDuration, error) {
	// Check if the meter is nil.
	if m == nil {
		return ServerRequestDuration{noop.Float64Histogram{}}, nil
	}

	if len(opt) == 0 {
		opt = newServerRequestDurationOpts
	} else {
		opt = append(opt, newServerRequestDurationOpts...)
	}

	i, err := m.Float64Histogram(
		"http.server.request.duration",
		h,
		opt...,
	)
	if err != nil {
		return ServerRequestDuration{noop.Float64Histogram{}}, err
	}
	return ServerRequestDuration{i}, nil
}

// Inst returns the underlying metric instrument.
func (m ServerRequestDuration) Inst() metric.Float64Histogram {
	return m.Float64Histogram
}

// ServerResponseBodySize is an instrument used to record metric values
// conforming to the "http.server.response.body.size" semantic conventions. It
// represents the size of HTTP server response bodies.
type ServerResponseBodySize struct {
	sdkmetric.Int64Recorder
}

var newServerResponseBodySizeOpts = []metric.Int64HistogramOption{
	metric.WithDescription("Size of HTTP server response bodies."),
	metric.WithUnit("By"),
}

// NewServerResponseBodySize returns a new ServerResponseBodySize instrument.
func NewServerResponseBodySize(
	m *sdkmetric.Meter,
	h otel.ErrorHandler,
	opt ...metric.Int64HistogramOption,
) (ServerResponseBodySize, error) {
	// Check if the meter is nil.
	if m == nil {
		return ServerResponseBodySize{}, nil
	}

	if len(opt) == 0 {
		opt = newServerResponseBodySizeOpts
	} else {
		opt = append(opt, newServerResponseBodySizeOpts...)
	}

	i, err := m.Int64Histogram(
		"http.server.response.body.size",
		h,
		opt...,
	)
	if err != nil {
		return ServerResponseBodySize{}, err
	}
	return ServerResponseBodySize{i}, nil
}

// Inst returns the underlying metric instrument.
func (m ServerResponseBodySize) Inst() sdkmetric.Int64Recorder {
	return m.Int64Recorder
}
