// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlptracehttp

import (
	"github.com/karelbilek/opentelemetry/retry"
)

// RetryConfig defines configuration for retrying batches in case of export
// failure using an exponential backoff.
type RetryConfig retry.Config
