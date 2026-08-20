// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package log

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	otel "github.com/karelbilek/opentelemetry"
	"github.com/karelbilek/opentelemetry/internal/global"
)

// BatchProcessor is a processor that exports batches of log records.
//
// Use [NewBatchProcessor] to create a BatchProcessor. An empty BatchProcessor
// is shut down by default, no records will be batched or exported.
type BatchProcessor struct {
	// A single goroutine owns dequeueing and all exporter calls. OnEmit only
	// writes to the bounded queue and signals that goroutine. Consequently,
	// exporter backpressure blocks the exporter goroutine instead of causing
	// another goroutine to retry without making progress.
	exporter Exporter

	// q is the active queue of records that have not yet been exported.
	q *queue
	// batchSize is the maximum number of records in a scheduled export.
	batchSize int

	// exportTrigger is a coalesced signal that records are ready to export.
	exportTrigger chan struct{}
	// flush serializes ForceFlush requests through the worker.
	flush chan batchProcessorRequest
	// shutdown accepts the single Shutdown request. It is separate from flush
	// so shutdown cannot be blocked behind concurrent ForceFlush callers.
	shutdown chan batchProcessorRequest
	// done is closed by the exporter goroutine after exporter shutdown.
	done chan struct{}

	// stopped holds the stopped state of the BatchProcessor.
	stopped atomic.Bool

	noCmp [0]func() //nolint: unused  // This is indeed used.
}

type batchProcessorRequest struct {
	ctx  context.Context
	resp chan<- error
}

func (r batchProcessorRequest) respond(err error) {
	r.resp <- err
}

// NewBatchProcessor decorates the provided exporter
// so that the log records are batched before exporting.
//
// Calls to the exporter's Export, ForceFlush, and Shutdown methods are
// synchronized and never invoked concurrently.
func NewBatchProcessor(exporter Exporter, errHandler otel.ErrorHandler, maxQSize int, expInterval time.Duration, expTimeout time.Duration, expMaxBatchSize int) *BatchProcessor {
	cfg := newBatchConfig(maxQSize, expInterval, expTimeout, expMaxBatchSize)

	b := &BatchProcessor{
		q:             newQueue(cfg.maxQSize),
		batchSize:     cfg.expMaxBatchSize,
		exportTrigger: make(chan struct{}, 1),
		flush:         make(chan batchProcessorRequest),
		shutdown:      make(chan batchProcessorRequest, 1),
		done:          make(chan struct{}),
	}

	// Order is important here. Wrap the timeoutExporter with the chunkExporter
	// to ensure each export completes in timeout (instead of all chunked
	// exports).
	exporter = newTimeoutExporter(exporter, cfg.expTimeout)
	// Use a chunkExporter to ensure ForceFlush and Shutdown calls are batched
	// appropriately on export.
	exporter = newChunkExporter(exporter, cfg.expMaxBatchSize)

	b.exporter = exporter
	b.process(cfg.expInterval, errHandler)
	return b
}

// process starts the goroutine that owns dequeueing and all exporter calls.
func (b *BatchProcessor) process(interval time.Duration, errHandler otel.ErrorHandler) {
	go func() {
		timer := time.NewTimer(interval)
		defer timer.Stop()
		// The worker owns and reuses buf. Exporters must not retain the slice
		// passed to them, so it is safe to refill after Export returns.
		buf := make([]Record, b.batchSize)

		for {
			// Probe shutdown by itself first. This makes an already queued terminal
			// request win over every other ready case. Closing done before replying
			// also means a successful Shutdown response observes a stopped worker.
			select {
			case req := <-b.shutdown:
				err := b.shutdownExporter(req.ctx)
				close(b.done)
				req.respond(err)
				return
			default:
			}

			// With no queued shutdown, service a waiting ForceFlush before ordinary
			// export wakes. The default keeps this priority check non-blocking.
			// Shutdown remains selectable in case it arrived after the first probe.
			select {
			case req := <-b.shutdown:
				err := b.shutdownExporter(req.ctx)
				close(b.done)
				req.respond(err)
				return
			case req := <-b.flush:
				err := b.flushExporter(req.ctx)
				req.respond(err)
				continue
			default:
			}

			// No lifecycle request was waiting, so block on the complete event set.
			// Both timer and size-triggered exports start a new interval window.
			select {
			case req := <-b.shutdown:
				err := b.shutdownExporter(req.ctx)
				close(b.done)
				req.respond(err)
				return
			case req := <-b.flush:
				err := b.flushExporter(req.ctx)
				req.respond(err)
			case <-timer.C:
				resetTimer(timer, interval)
				b.exportBatch(buf, errHandler)
			case <-b.exportTrigger:
				resetTimer(timer, interval)
				b.exportBatch(buf, errHandler)
			}
		}
	}()
}

func resetTimer(timer *time.Timer, interval time.Duration) {
	// Handle both GODEBUG=asynctimerchan=[0|1] properly.
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(interval)
}

func (b *BatchProcessor) exportBatch(buf []Record, errHandler otel.ErrorHandler) {
	b.logDroppedRecords()
	n, remaining := b.q.Dequeue(buf)
	if n == 0 {
		return
	}

	err := b.exporter.Export(context.Background(), buf[:n])
	clear(buf[:n])
	if err != nil {
		otel.Handle(errHandler, err)
	}
	if remaining >= b.batchSize {
		b.triggerExport()
	}
}

func (b *BatchProcessor) flushExporter(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	b.logDroppedRecords()
	records := b.q.Flush()
	err := b.exporter.Export(ctx, records)
	clear(records)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return errors.Join(err, ctxErr)
	}
	return errors.Join(err, b.exporter.ForceFlush(ctx))
}

func (b *BatchProcessor) shutdownExporter(ctx context.Context) error {
	b.logDroppedRecords()
	records := b.q.Flush()
	err := b.exporter.Export(ctx, records)
	clear(records)
	if ctxErr := ctx.Err(); ctxErr != nil {
		err = errors.Join(err, ctxErr)
	} else {
		err = errors.Join(err, b.exporter.ForceFlush(ctx))
	}
	err = errors.Join(err, b.exporter.Shutdown(ctx))
	return err
}

func (b *BatchProcessor) logDroppedRecords() {
	if d := b.q.Dropped(); d > 0 {
		global.Warn(fmt.Sprintf("dropped log records - dropped %d", d))
	}
}

func (b *BatchProcessor) triggerExport() {
	select {
	case b.exportTrigger <- struct{}{}:
	default:
	}
}

// Enabled returns true, indicating this Processor will process all records.
func (*BatchProcessor) Enabled(context.Context, EnabledParameters) bool {
	return true
}

// OnEmit batches provided log record.
func (b *BatchProcessor) OnEmit(_ context.Context, r *Record) error {
	if b.stopped.Load() || b.q == nil {
		return nil
	}
	// The record is cloned so that changes done by subsequent processors
	// are not going to lead to a data race.
	if n, accepted := b.q.Enqueue(r.Clone()); accepted && n >= b.batchSize {
		b.triggerExport()
	}
	return nil
}

// Shutdown flushes queued log records and the decorated exporter before
// shutting it down.
func (b *BatchProcessor) Shutdown(ctx context.Context) error {
	if b.stopped.Swap(true) || b.q == nil {
		return nil
	}

	b.q.Close()
	resp := make(chan error, 1)
	b.shutdown <- batchProcessorRequest{ctx: ctx, resp: resp}
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case err := <-resp:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ForceFlush flushes queued log records and flushes the decorated exporter.
func (b *BatchProcessor) ForceFlush(ctx context.Context) error {
	if b.stopped.Load() || b.q == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	resp := make(chan error, 1)
	req := batchProcessorRequest{ctx: ctx, resp: resp}
	select {
	case b.flush <- req:
	case <-b.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}

	select {
	case err := <-resp:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// queue holds a queue of logging records.
//
// When the queue becomes full, the oldest records in the queue are
// overwritten.
type queue struct {
	sync.Mutex

	dropped     atomic.Uint64
	cap, len    int
	read, write *ring
	closed      bool
}

func newQueue(size int) *queue {
	r := newRing(size)
	return &queue{
		cap:   size,
		read:  r,
		write: r,
	}
}

func (q *queue) Len() int {
	q.Lock()
	defer q.Unlock()

	return q.len
}

// Dropped returns the number of Records dropped during enqueueing since the
// last time Dropped was called.
func (q *queue) Dropped() uint64 {
	return q.dropped.Swap(0)
}

// Enqueue adds r to the queue. The queue size, including the addition of r, is
// returned.
//
// If enqueueing r will exceed the capacity of q, the oldest Record held in q
// will be dropped and r retained.
func (q *queue) Enqueue(r Record) (int, bool) {
	q.Lock()
	defer q.Unlock()

	if q.closed {
		return q.len, false
	}

	q.write.Value = r
	q.write = q.write.Next()

	q.len++
	if q.len > q.cap {
		// Overflow. Advance read to be the new "oldest".
		q.len = q.cap
		q.read = q.read.Next()
		q.dropped.Add(1)
	}
	return q.len, true
}

// Dequeue removes up to len(buf) records from the queue and copies them into
// buf. The number copied and the number remaining are returned.
func (q *queue) Dequeue(buf []Record) (int, int) {
	q.Lock()
	defer q.Unlock()

	n := min(len(buf), q.len)
	for i := range n {
		buf[i] = q.read.Value // nolint:gosec // n is bounded by len(buf)
		q.read.Value = Record{}
		q.read = q.read.Next()
	}
	q.len -= n
	return n, q.len
}

// Flush returns all the Records held in the queue and resets it to be
// empty.
func (q *queue) Flush() []Record {
	q.Lock()
	defer q.Unlock()

	return q.flush()
}

// Close stops the queue from accepting records.
func (q *queue) Close() {
	q.Lock()
	defer q.Unlock()

	q.closed = true
}

func (q *queue) flush() []Record {
	out := make([]Record, q.len)
	for i := range out {
		out[i] = q.read.Value
		q.read.Value = Record{}
		q.read = q.read.Next()
	}
	q.len = 0

	return out
}

type batchConfig struct {
	maxQSize        int
	expInterval     time.Duration
	expTimeout      time.Duration
	expMaxBatchSize int
}

func newBatchConfig(maxQSize int, expInterval time.Duration, expTimeout time.Duration, expMaxBatchSize int) batchConfig {
	return batchConfig{
		maxQSize:        maxQSize,
		expInterval:     expInterval,
		expTimeout:      expTimeout,
		expMaxBatchSize: expMaxBatchSize,
	}
}
