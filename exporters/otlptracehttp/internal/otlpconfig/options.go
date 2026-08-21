// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package otlpconfig provides configuration for the otlptrace exporters.
package otlpconfig

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
		Traces SignalConfig

		RetryConfig retry.Config
	}
)

// NewHTTPConfig returns a new Config with all settings applied from opts and
// any unset setting using the default HTTP config values.
func NewHTTPConfig(endpoint string, urlPath string, insecure bool, headers map[string]string, maxRequestSize int, timeout time.Duration, retry retry.Config) Config {
	cfg := Config{
		Traces: SignalConfig{
			Endpoint:       endpoint,
			Insecure:       insecure,
			Headers:        headers,
			MaxRequestSize: maxRequestSize,
			Timeout:        timeout,
			URLPath:        urlPath,
		}, RetryConfig: retry,
	}
	return cfg
}
