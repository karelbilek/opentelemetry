// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package retry

import (
	"net/http"
	"strconv"
	"time"
)

// retryableError represents a request failure that can be retried.
type retryableError struct {
	throttle time.Duration
	err      error
}

// NewResponseError returns a retry-able error and will extract any explicit
// throttle delay contained in headers. The returned error wraps wrapped
// if it is not nil.
func NewResponseError(header http.Header, wrapped error) error {
	var rErr retryableError
	if v := header.Get("Retry-After"); v != "" {
		rErr.throttle = retryAfterDuration(v)
	}

	rErr.err = wrapped
	return rErr
}

func retryAfterDuration(v string) time.Duration {
	if t, err := strconv.ParseInt(v, 10, 64); err == nil && t >= 0 {
		const maxRetryAfterSeconds = int64(1<<63-1) / int64(time.Second)
		if t > maxRetryAfterSeconds {
			return time.Duration(1<<63 - 1)
		}
		return time.Duration(t) * time.Second
	}

	if date, err := http.ParseTime(v); err == nil {
		return max(time.Until(date), 0)
	}

	return 0
}

func (e retryableError) Error() string {
	if e.err != nil {
		return "retry-able request failure: " + e.err.Error()
	}

	return "retry-able request failure"
}

func (e retryableError) Unwrap() error {
	return e.err
}

func (e retryableError) As(target any) bool {
	if e.err == nil {
		return false
	}

	switch v := target.(type) {
	case **retryableError:
		*v = &e
		return true
	default:
		return false
	}
}

// Evaluate returns if err is retry-able. If it is and it includes an explicit
// throttling delay, that delay is also returned.
func Evaluate(err error) (bool, time.Duration) {
	if err == nil {
		return false, 0
	}

	// Do not use errors.As here, this should only be flattened one layer. If
	// there are several chained errors, all the errors above it will be
	// discarded if errors.As is used instead.
	rErr, ok := err.(retryableError) //nolint:errorlint
	if !ok {
		return false, 0
	}

	return true, rErr.throttle
}
