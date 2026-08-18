// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package trace

import (
	"context"
	"encoding/binary"
	"math/rand/v2"

	"github.com/karelbilek/opentelemetry/trace"
)

// newSpanID returns a non-zero span ID from a randomly-chosen sequence.
func newSpanID(context.Context, trace.TraceID) trace.SpanID {
	sid := trace.SpanID{}
	for {
		binary.NativeEndian.PutUint64(sid[:], rand.Uint64())
		if sid.IsValid() {
			break
		}
	}
	return sid
}

// newIDs returns a non-zero trace ID and a non-zero span ID from a
// randomly-chosen sequence.
func newIDs(context.Context) (trace.TraceID, trace.SpanID) {
	tid := trace.TraceID{}
	sid := trace.SpanID{}
	for {
		binary.NativeEndian.PutUint64(tid[:8], rand.Uint64())
		binary.NativeEndian.PutUint64(tid[8:], rand.Uint64())
		if tid.IsValid() {
			break
		}
	}
	for {
		binary.NativeEndian.PutUint64(sid[:], rand.Uint64())
		if sid.IsValid() {
			break
		}
	}
	return tid, sid
}
