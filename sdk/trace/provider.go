// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package trace

import (
	"context"
	"sync"
	"sync/atomic"

	otel "github.com/karelbilek/opentelemetry"
	"github.com/karelbilek/opentelemetry/internal/global"
	"github.com/karelbilek/opentelemetry/sdk/instrumentation"
	"github.com/karelbilek/opentelemetry/sdk/resource"
)

const defaultTracerName = "github.com/karelbilek/opentelemetry/sdk/tracer"

// tracerProviderConfig.
type tracerProviderConfig struct {
	// processors contains collection of SpanProcessors that are processing pipeline
	// for spans in the trace signal.
	// SpanProcessors registered with a TracerProvider and are called at the start
	// and end of a Span's lifecycle, and are called in the order they are
	// registered.
	processor *BatchSpanProcessor

	// spanLimits defines the attribute, event, and link limits for spans.
	spanLimits SpanLimits

	// resource contains attributes representing an entity that produces telemetry.
	resource *resource.Resource

	// panicRecordingDisabled disables recording exception events from panics.
	panicRecordingDisabled bool
}

// MarshalLog is the marshaling function used by the logging system to represent this Provider.
func (cfg tracerProviderConfig) MarshalLog() any {
	return struct {
		SpanProcessors         *BatchSpanProcessor
		IDGeneratorType        string
		SpanLimits             SpanLimits
		Resource               *resource.Resource
		PanicRecordingDisabled bool
	}{
		SpanProcessors:         cfg.processor,
		SpanLimits:             cfg.spanLimits,
		Resource:               cfg.resource,
		PanicRecordingDisabled: cfg.panicRecordingDisabled,
	}
}

// TracerProvider is an OpenTelemetry TracerProvider. It provides Tracers to
// instrumentation so it can trace operational flow through a system.
type TracerProvider struct {
	noop bool

	mu          sync.Mutex
	namedTracer map[instrumentation.Scope]*Tracer

	isShutdown atomic.Bool

	// These fields are not protected by the lock mu. They are assumed to be
	// immutable after creation of the TracerProvider.
	processor              *BatchSpanProcessor
	spanLimits             SpanLimits
	resource               *resource.Resource
	panicRecordingDisabled bool

	h otel.ErrorHandler
}

// NewTracerProvider returns a new and configured TracerProvider.
//
// By default the returned TracerProvider is configured with:
//   - a ParentBased(AlwaysSample) Sampler
//   - a random number IDGenerator
//   - the resource.Default() Resource
//   - the default SpanLimits.
//
// The passed opts are used to override these default values and configure the
// returned TracerProvider appropriately.
func NewTracerProvider(
	h otel.ErrorHandler,
	attributeValueLengthLimit int,
	attributeCountLimit int,
	eventCountLimit int,
	linkCountLimit int,
	attributePerEventCountLimit int,
	attributePerLinkCountLimit int,
	processor *BatchSpanProcessor,
	resource *resource.Resource,
	panicRecordingDisabled bool) *TracerProvider {
	o := tracerProviderConfig{
		spanLimits: NewSpanLimits(
			attributeValueLengthLimit,
			attributeCountLimit,
			eventCountLimit,
			linkCountLimit, attributePerEventCountLimit,
			attributePerLinkCountLimit,
		),
		processor:              processor,
		resource:               resource,
		panicRecordingDisabled: panicRecordingDisabled,
	}

	o = ensureValidTracerProviderConfig(o, h)

	tp := &TracerProvider{
		namedTracer:            make(map[instrumentation.Scope]*Tracer),
		processor:              o.processor,
		spanLimits:             o.spanLimits,
		resource:               o.resource,
		panicRecordingDisabled: o.panicRecordingDisabled,
		h:                      h,
	}
	global.Info("TracerProvider created", "config", o)

	return tp
}

// Tracer returns a Tracer with the given name and options. If a Tracer for
// the given name and options does not exist it is created, otherwise the
// existing Tracer is returned.
//
// If name is empty, DefaultTracerName is used instead.
//
// This method is safe to be called concurrently.
func (p *TracerProvider) Tracer(name string) *Tracer {
	if p.noop {
		return &Tracer{noop: true}
	}
	// This check happens before the mutex is acquired to avoid deadlocking if Tracer() is called from within Shutdown().
	if p.isShutdown.Load() {
		return &Tracer{noop: true}
	}
	if name == "" {
		name = defaultTracerName
	}
	is := instrumentation.Scope{
		Name: name,
	}

	t, ok := func() (*Tracer, bool) {
		p.mu.Lock()
		defer p.mu.Unlock()
		// Must check the flag after acquiring the mutex to avoid returning a valid tracer if Shutdown() ran
		// after the first check above but before we acquired the mutex.
		if p.isShutdown.Load() {
			return &Tracer{noop: true}, true
		}
		t, ok := p.namedTracer[is]
		if !ok {
			t = &Tracer{
				provider:             p,
				instrumentationScope: is,
			}

			p.namedTracer[is] = t
		}
		return t, ok
	}()
	if !ok {
		// This code is outside the mutex to not hold the lock while calling third party logging code:
		// - That code may do slow things like I/O, which would prolong the duration the lock is held,
		//   slowing down all tracing consumers.
		// - Logging code may be instrumented with tracing and deadlock because it could try
		//   acquiring the same non-reentrant mutex.
		global.Info(
			"Tracer created",
			"name",
			name,
		)
	}
	return t
}

// ForceFlush immediately exports all spans that have not yet been exported for
// the span processor.
func (p *TracerProvider) ForceFlush(ctx context.Context) error {
	if p.noop || p.processor == nil {
		return nil
	}
	return p.processor.ForceFlush(ctx)
}

// Shutdown shuts down TracerProvider. All registered span processors are shut down
// in the order they were registered and any held computational resources are released.
// After Shutdown is called, all methods are no-ops.
func (p *TracerProvider) Shutdown(ctx context.Context) error {
	if p.noop {
		return nil
	}
	// This check prevents deadlocks in case of recursive shutdown.
	if p.isShutdown.Load() {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	// This check prevents calls after a shutdown has already been done concurrently.
	if !p.isShutdown.CompareAndSwap(false, true) { // did toggle?
		return nil
	}

	if p.processor == nil {
		return nil
	}
	return p.processor.Shutdown(ctx)
}

// 		cfg.sampler = ParentBased(AlwaysSample())

// ensureValidTracerProviderConfig ensures that given TracerProviderConfig is valid.
func ensureValidTracerProviderConfig(cfg tracerProviderConfig, errorHandler otel.ErrorHandler) tracerProviderConfig {
	if cfg.resource == nil {
		cfg.resource = resource.Default(errorHandler, "")
	}
	return cfg
}
