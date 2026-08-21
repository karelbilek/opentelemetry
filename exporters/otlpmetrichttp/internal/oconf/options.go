// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package oconf provides configuration for the otlpmetric exporters.
package oconf

import (
	"time"

	"github.com/karelbilek/opentelemetry/retry"
)

type (
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
