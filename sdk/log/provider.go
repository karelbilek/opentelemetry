// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package log

import (
	"context"
	"sync"
	"sync/atomic"

	otel "github.com/karelbilek/opentelemetry"
	"github.com/karelbilek/opentelemetry/internal/global"
	"github.com/karelbilek/opentelemetry/sdk/instrumentation"
	"github.com/karelbilek/opentelemetry/sdk/resource"
)

// LoggerProvider handles the creation and coordination of Loggers. All Loggers
// it creates are associated with the same Resource.
//
// After [LoggerProvider.Shutdown] starts, calls to [log.Logger.Enabled] on
// Loggers created by the LoggerProvider return false, and calls to
// [log.Logger.Emit] perform no operation.
type LoggerProvider struct {
	eh otel.ErrorHandler

	resource                  *resource.Resource
	processor                 *BatchProcessor
	attributeCountLimit       int
	attributeValueLengthLimit int
	allowDupKeys              bool

	loggersMu sync.Mutex
	loggers   map[instrumentation.Scope]*Logger

	stopped                   atomic.Bool
	processorOperationsMu     sync.Mutex
	processorOperationsActive int
	processorOperationsDone   chan struct{}

	noCmp [0]func() //nolint: unused  // This is indeed used.
}

// NewLoggerProvider returns a new and configured LoggerProvider.
//
// By default, the returned LoggerProvider is configured with the default
// Resource and no Processors. Processors cannot be added after a LoggerProvider is
// created. This means the returned LoggerProvider, one created with no
// Processors, will perform no operations.
func NewLoggerProvider(eh otel.ErrorHandler, resource *resource.Resource, processor *BatchProcessor, attrCntLim int, attrValLenLim int, allowDupKeys bool) *LoggerProvider {
	return &LoggerProvider{
		eh:                        eh,
		resource:                  resource,
		processor:                 processor,
		attributeCountLimit:       attrCntLim,
		attributeValueLengthLimit: attrValLenLim,
		allowDupKeys:              allowDupKeys,
	}
}

// Logger returns a new [log.Logger] with the provided name and configuration.
//
// Calls made after [LoggerProvider.Shutdown] starts return a [noop.Logger].
//
// This method can be called concurrently.
func (p *LoggerProvider) Logger(name string) *Logger {
	if p == nil {
		return nil
	}
	if name == "" {
		global.Warn("Invalid Logger name.", "name", name)
	}

	if p.stopped.Load() {
		return nil
	}

	scope := instrumentation.Scope{
		Name: name,
	}

	p.loggersMu.Lock()
	defer p.loggersMu.Unlock()

	if p.loggers == nil {
		l := newLogger(p, scope)
		p.loggers = map[instrumentation.Scope]*Logger{scope: l}
		return l
	}

	l, ok := p.loggers[scope]
	if !ok {
		l = newLogger(p, scope)
		p.loggers[scope] = l
	}

	return l
}

// Shutdown shuts down the provider and all processors in the order they were
// registered.
//
// The first call stops admitting new operations that invoke processor Enabled,
// OnEmit, or ForceFlush methods. It waits for operations already admitted to
// complete before synchronously invoking each processor's Shutdown method. If
// ctx is canceled before the admitted operations complete, Shutdown returns
// ctx.Err() without invoking processor Shutdown.
//
// Concurrent or subsequent Shutdown calls return nil without invoking
// processor Shutdown.
//
// Shutdown must not be called directly or indirectly from a Processor method.
//
// This method can be called concurrently.
func (p *LoggerProvider) Shutdown(ctx context.Context) error {
	p.processorOperationsMu.Lock()
	if p.stopped.Load() {
		p.processorOperationsMu.Unlock()
		return nil
	}
	p.processorOperationsDone = make(chan struct{})
	p.stopped.Store(true)
	if p.processorOperationsActive == 0 {
		close(p.processorOperationsDone)
	}
	p.processorOperationsMu.Unlock()

	// All count updates happen while processorOperationsMu is held. Therefore,
	// either Shutdown observes zero operations and closes the channel above, or
	// the final operation observes stopped and closes it synchronously.
	if err := p.waitForProcessorOperations(ctx); err != nil {
		return err
	}

	if p.processor == nil {
		return nil
	}
	err := p.processor.Shutdown(ctx)
	return err
}

// ForceFlush flushes all processors.
//
// Once Shutdown starts, ForceFlush performs no operation and returns nil.
//
// This method can be called concurrently.
func (p *LoggerProvider) ForceFlush(ctx context.Context) error {
	if p.processor == nil {
		return nil
	}
	if !p.beginProcessorOperation() {
		return nil
	}
	defer p.endProcessorOperation()

	err := p.processor.ForceFlush(ctx)
	return err
}

func (p *LoggerProvider) beginProcessorOperation() bool {
	p.processorOperationsMu.Lock()
	defer p.processorOperationsMu.Unlock()
	if p.stopped.Load() {
		return false
	}
	p.processorOperationsActive++
	return true
}

func (p *LoggerProvider) endProcessorOperation() {
	p.processorOperationsMu.Lock()
	defer p.processorOperationsMu.Unlock()
	p.processorOperationsActive--
	if p.processorOperationsActive == 0 && p.stopped.Load() {
		close(p.processorOperationsDone)
	}
}

func (p *LoggerProvider) waitForProcessorOperations(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	return waitForProcessorOperationsCompletion(ctx, p.processorOperationsDone)
}

func waitForProcessorOperationsCompletion(ctx context.Context, done <-chan struct{}) error {
	select {
	case <-done:
	case <-ctx.Done():
		// Prefer a completed drain when it races with cancellation.
		select {
		case <-done:
		default:
			return ctx.Err()
		}
	}
	return nil
}
