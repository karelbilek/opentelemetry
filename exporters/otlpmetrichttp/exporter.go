// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpmetrichttp

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	metricpb "github.com/karelbilek/opentelemetry/proto/metrics/v1"

	"github.com/karelbilek/opentelemetry/exporters/otlpmetrichttp/internal/oconf"
	"github.com/karelbilek/opentelemetry/exporters/otlpmetrichttp/internal/transform"
	"github.com/karelbilek/opentelemetry/internal/global"
	"github.com/karelbilek/opentelemetry/retry"
	"github.com/karelbilek/opentelemetry/sdk/metric/metricdata"
)

// Exporter is a OpenTelemetry metric Exporter using protobufs over HTTP.
type Exporter struct {
	// Ensure synchronous access to the client across all functionality.
	clientMu sync.Mutex
	client   interface {
		UploadMetrics(context.Context, *metricpb.ResourceMetrics) error
		Shutdown(context.Context) error
	}

	shutdownOnce sync.Once
}

func newExporter(c *client, cfg oconf.Config) (*Exporter, error) {
	return &Exporter{
		client: c,
	}, nil
}

// Export transforms and transmits metric data to an OTLP receiver.
//
// This method returns an error if called after Shutdown.
// This method returns an error if the method is canceled by the passed context.
func (e *Exporter) Export(ctx context.Context, rm *metricdata.ResourceMetrics) error {
	defer global.Debug("OTLP/HTTP exporter export", "Data", rm)

	otlpRm, err := transform.ResourceMetrics(rm)
	// Best effort upload of transformable metrics.
	e.clientMu.Lock()
	upErr := e.client.UploadMetrics(ctx, otlpRm)
	e.clientMu.Unlock()
	if upErr != nil {
		if err == nil {
			return fmt.Errorf("failed to upload metrics: %w", upErr)
		}
		// Merge the two errors.
		return fmt.Errorf("failed to upload incomplete metrics (%w): %w", err, upErr)
	}
	return err
}

// ForceFlush flushes any metric data held by an exporter.
//
// This method returns an error if called after Shutdown.
// This method returns an error if the method is canceled by the passed context.
//
// This method is safe to call concurrently.
func (*Exporter) ForceFlush(ctx context.Context) error {
	// The exporter and client hold no state, nothing to flush.
	return ctx.Err()
}

// Shutdown flushes all metric data held by an exporter and releases any held
// computational resources.
//
// This method returns an error if called after Shutdown.
// This method returns an error if the method is canceled by the passed context.
//
// This method is safe to call concurrently.
func (e *Exporter) Shutdown(ctx context.Context) error {
	err := errShutdown
	e.shutdownOnce.Do(func() {
		e.clientMu.Lock()
		client := e.client
		e.client = shutdownClient{}
		e.clientMu.Unlock()
		err = client.Shutdown(ctx)
	})
	return err
}

var errShutdown = errors.New("HTTP exporter is shutdown")

type shutdownClient struct{}

func (shutdownClient) err(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return errShutdown
}

func (c shutdownClient) UploadMetrics(ctx context.Context, _ *metricpb.ResourceMetrics) error {
	return c.err(ctx)
}

func (c shutdownClient) Shutdown(ctx context.Context) error {
	return c.err(ctx)
}

// MarshalLog returns logging data about the Exporter.
func (*Exporter) MarshalLog() any {
	return struct{ Type string }{Type: "OTLP/HTTP"}
}

// New returns an OpenTelemetry metric Exporter. The Exporter can be used with
// a PeriodicReader to export OpenTelemetry metric data to an OTLP receiving
// endpoint using protobufs over HTTP.
func New(_ context.Context, endpoint string, urlPath string, insecure bool, headers map[string]string, maxRequestSize int, timeout time.Duration, retry retry.Config) (*Exporter, error) {
	cfg := oconf.NewHTTPConfig(endpoint, urlPath, insecure, headers, maxRequestSize, timeout, retry)
	c, err := newClient(cfg)
	if err != nil {
		return nil, err
	}
	return newExporter(c, cfg)
}
