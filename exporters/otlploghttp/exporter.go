package otlploghttp

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/karelbilek/opentelemetry/exporters/otlploghttp/internal/transform"
	"github.com/karelbilek/opentelemetry/retry"
	"github.com/karelbilek/opentelemetry/sdk/log/logdata"
)

// Exporter is a OpenTelemetry log Exporter. It transports log data encoded as
// OTLP protobufs using HTTP.
// Exporter must be created with [New].
type Exporter struct {
	client  atomic.Pointer[client]
	stopped atomic.Bool
}

// New returns a new [Exporter].
//
// It is recommended to use it with a [BatchProcessor]
// or other processor exporting records asynchronously.
func New(ctx context.Context, endpoint string,
	path string,
	insecure bool,
	maxRequestSize int,
	timeout time.Duration,
	retryCfg retry.Config,
) (*Exporter, error) {
	cfg := newConfig(endpoint, path, insecure, maxRequestSize, timeout, retryCfg)
	c, err := newHTTPClient(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return newExporter(c, cfg)
}

func newExporter(c *client, _ config) (*Exporter, error) {
	e := &Exporter{}
	e.client.Store(c)
	return e, nil
}

// Used for testing.
var transformResourceLogs = transform.ResourceLogs

// Export transforms and transmits log records to an OTLP receiver.
func (e *Exporter) Export(ctx context.Context, records []logdata.Record) error {
	if e.stopped.Load() {
		return nil
	}
	otlp := transformResourceLogs(records)
	if otlp == nil {
		return nil
	}
	return e.client.Load().UploadLogs(ctx, otlp)
}

// Shutdown shuts down the Exporter. Calls to Export or ForceFlush will perform
// no operation after this is called.
func (e *Exporter) Shutdown(context.Context) error {
	if e.stopped.Swap(true) {
		return nil
	}

	e.client.Store(newNoopClient())
	return nil
}

// ForceFlush does nothing. The Exporter holds no state.
func (*Exporter) ForceFlush(context.Context) error {
	return nil
}
