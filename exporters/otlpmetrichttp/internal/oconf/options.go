// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package oconf provides configuration for the otlpmetric exporters.
package oconf

import (
	"net/http"
	"net/url"
	"time"

	"github.com/karelbilek/opentelemetry/retry"
)

type (
	// HTTPTransportProxyFunc is a function that resolves which URL to use as proxy for a given request.
	// This type is compatible with `http.Transport.Proxy` and can be used to set a custom proxy function to the OTLP HTTP client.
	HTTPTransportProxyFunc func(*http.Request) (*url.URL, error)

	SignalConfig struct {
		Endpoint       string
		Insecure       bool
		Headers        map[string]string
		MaxRequestSize int
		Timeout        time.Duration
		URLPath        string
	}

	Config struct {
		// Signal specific configurations
		Metrics SignalConfig

		RetryConfig retry.Config

		// gRPC configurations
		ReconnectionPeriod time.Duration
		ServiceConfig      string
	}
)

// NewHTTPConfig returns a new Config with all settings applied from opts and
// any unset setting using the default HTTP config values.
func NewHTTPConfig(
	endpoint string,
	urlPath string,
	insecure bool,
	headers map[string]string,
	maxRequestSize int,
	timeout time.Duration,
	retry retry.Config,
) Config {
	return Config{
		Metrics: SignalConfig{
			Endpoint:       endpoint,
			URLPath:        urlPath,
			Insecure:       insecure,
			Headers:        headers,
			MaxRequestSize: maxRequestSize,
			Timeout:        timeout,
		},
		RetryConfig: retry,
	}
}
