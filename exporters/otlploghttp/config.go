// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlploghttp

import (
	"time"

	"github.com/karelbilek/opentelemetry/retry"
)

type config struct {
	endpoint string
	path     string
	insecure bool
	// tlsCfg         *tls.Config
	// headers        map[string]string
	// compression    Compression
	maxRequestSize int
	timeout        time.Duration
	retryCfg       retry.Config
	// httpClient     *http.Client
}

func newConfig(
	endpoint string,
	path string,
	insecure bool,
	// headers map[string]string,
	// compression Compression,
	maxRequestSize int,
	timeout time.Duration,
	retryCfg retry.Config,
) config {
	return config{
		endpoint:       endpoint,
		path:           path,
		insecure:       insecure,
		maxRequestSize: maxRequestSize,
		timeout:        timeout,
		retryCfg:       retryCfg,
	}
}
