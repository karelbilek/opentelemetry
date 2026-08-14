// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlptracehttp

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/karelbilek/opentelemetry/exporters/otlptracehttp/internal/tracetransform"
	"github.com/karelbilek/opentelemetry/retry"
	tracesdk "github.com/karelbilek/opentelemetry/sdk/trace"
)

// New constructs a new Exporter and starts it.
func New(ctx context.Context, endpoint string, urlPath string, insecure bool, headers map[string]string, maxRequestSize int, timeout time.Duration, retry retry.Config) (*Exporter, error) {
	exp := NewUnstarted(endpoint, urlPath, insecure, headers, maxRequestSize, timeout, retry)
	if err := exp.Start(ctx); err != nil {
		return nil, err
	}
	return exp, nil
}

// NewUnstarted constructs a new Exporter and does not start it.
func NewUnstarted(endpoint string, urlPath string, insecure bool, headers map[string]string, maxRequestSize int, timeout time.Duration, retry retry.Config) *Exporter {
	client := NewClient(endpoint, urlPath, insecure, headers, maxRequestSize, timeout, retry)
	return &Exporter{
		client: client,
	}
}

var errAlreadyStarted = errors.New("already started")

// Exporter exports trace data in the OTLP wire format.
type Exporter struct {
	client *Client

	mu      sync.RWMutex
	started bool

	startOnce sync.Once
	stopOnce  sync.Once
}

// ExportSpans exports a batch of spans.
func (e *Exporter) ExportSpans(ctx context.Context, ss []*tracesdk.Snapshot) error {
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

// Start establishes a connection to the receiving endpoint.
func (e *Exporter) Start(ctx context.Context) error {
	err := errAlreadyStarted
	e.startOnce.Do(func() {
		e.mu.Lock()
		e.started = true
		e.mu.Unlock()
		err = e.client.Start(ctx)
	})

	return err
}

// Shutdown flushes all exports and closes all connections to the receiving endpoint.
func (e *Exporter) Shutdown(ctx context.Context) error {
	e.mu.RLock()
	started := e.started
	e.mu.RUnlock()

	if !started {
		return nil
	}

	var err error

	e.stopOnce.Do(func() {
		err = e.client.Stop(ctx)
		e.mu.Lock()
		e.started = false
		e.mu.Unlock()
	})

	return err
}

var _ tracesdk.SpanExporter = (*Exporter)(nil)

// MarshalLog is the marshaling function used by the logging system to represent this Exporter.
func (e *Exporter) MarshalLog() any {
	return struct {
		Type   string
		Client string
	}{
		Type:   "otlptrace",
		Client: fmt.Sprintf("%T", e.client),
	}
}
