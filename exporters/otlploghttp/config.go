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
	var c config
	c.endpoint = endpoint
	c.path = path
	c.insecure = insecure
	// c.compression=compression
	c.timeout = c.timeout
	c.retryCfg = retryCfg

	return c
}
