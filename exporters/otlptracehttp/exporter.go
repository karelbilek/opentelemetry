// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlptracehttp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/karelbilek/opentelemetry/exporters/otlptracehttp/internal/tracetransform"
	"github.com/karelbilek/opentelemetry/retry"
	"github.com/karelbilek/opentelemetry/sdk/trace/tracedata"
)

// New constructs a new Exporter.
func New(endpoint string, urlPath string, insecure bool, headers map[string]string, maxRequestSize int, timeout time.Duration, retry retry.Config) *Exporter {
	client := NewClient(endpoint, urlPath, insecure, headers, maxRequestSize, timeout, retry)
	return &Exporter{
		client: client,
	}
}

// Exporter exports trace data in the OTLP wire format.
type Exporter struct {
	client *Client

	stopOnce sync.Once
}

// ExportSpans exports a batch of spans.
func (e *Exporter) ExportSpans(ctx context.Context, ss []*tracedata.Snapshot) error {
	protoSpans := tracetransform.Spans(ss)
	if len(protoSpans) == 0 {
		return nil
	}

	err := e.client.UploadTraces(ctx, protoSpans)
	if err != nil {
		return fmt.Errorf("traces export: %w", err)
	}
	return nil
}

// Shutdown flushes all exports and closes all connections to the receiving endpoint.
func (e *Exporter) Shutdown(ctx context.Context) error {
	var err error
	e.stopOnce.Do(func() {
		err = e.client.Stop(ctx)
	})
	return err
}

// MarshalLog is the marshaling function used by the logging system to represent this Exporter.
func (e *Exporter) MarshalLog() any {
	return struct {
		Type   string
		Client string
	}{
		Type:   "otlptracehttp",
		Client: fmt.Sprintf("%T", e.client),
	}
}
