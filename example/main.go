package main

import (
	"context"
	"runtime"
	"time"

	otel "github.com/karelbilek/opentelemetry"
	"github.com/karelbilek/opentelemetry/exporters/otlploghttp"
	"github.com/karelbilek/opentelemetry/exporters/otlpmetrichttp"
	"github.com/karelbilek/opentelemetry/exporters/otlptracehttp"
	ametric "github.com/karelbilek/opentelemetry/metric"
	"github.com/karelbilek/opentelemetry/otelslog"
	"github.com/karelbilek/opentelemetry/retry"
	"github.com/karelbilek/opentelemetry/sdk/log"
	"github.com/karelbilek/opentelemetry/sdk/metric"

	"github.com/karelbilek/opentelemetry/sdk/resource"
	"github.com/karelbilek/opentelemetry/sdk/trace"
)

func main() {
	logExporter, err := otlploghttp.New(context.Background(),
		"127.0.0.1:4318",
		otlploghttp.DefaultLogsPath,
		true,
		nil,
		otlploghttp.DefaultMaxRequestSize,
		otlploghttp.DefaultTimeout,
		retry.DefaultConfig)
	if err != nil {
		panic(err)
	}
	oh := otel.BasicErrorHandler{}

	res := resource.Default(context.Background(), oh, "my_service")

	processor := log.NewBatchProcessor(
		logExporter,
		oh,
		log.DefaultMaxQueueSize,
		log.DefaultExpInterval,
		log.DefaultExpTimeout,
		log.DefaultExpMaxBatchSize)
	provider := log.NewLoggerProvider(
		oh,
		res,
		processor,
		log.DefaultAttributeCountLimit,
		log.DefaultAttributeValueLengthLimit,
	)
	slogger := otelslog.NewLogger("mylogger", provider, true)
	defer func() {
		time.Sleep(30 * time.Second)
		if err := provider.Shutdown(context.Background()); err != nil {
			panic(err)
		}
	}()

	traceExporter := otlptracehttp.New(
		"127.0.0.1:4318",
		otlptracehttp.DefaultTracesPath,
		true,
		nil,
		otlptracehttp.DefaultMaxRequestSize,
		otlptracehttp.DefaultTimeout,
		retry.DefaultConfig,
	)
	tracerProvider := trace.NewTracerProvider(
		oh,
		trace.DefaultAttributeValueLengthLimit,
		trace.DefaultAttributeCountLimit,
		trace.DefaultEventCountLimit,
		trace.DefaultLinkCountLimit,
		trace.DefaultAttributePerEventCountLimit,
		trace.DefaultAttributePerLinkCountLimit,
		trace.NewBatchSpanProcessor(
			traceExporter,
			oh,
			trace.DefaultMaxQueueSize,
			trace.DefaultBatchTimeout,
			trace.DefaultExportTimeout,
			trace.DefaultMaxExportBatchSize,
		),
		res,
	)
	defer func() {
		if err := tracerProvider.Shutdown(context.Background()); err != nil {
			panic(err)
		}
	}()

	metricExporter, err := otlpmetrichttp.New(
		context.Background(),
		"127.0.0.1:4318",
		otlpmetrichttp.DefaultMetricsPath,
		true,
		nil,
		otlpmetrichttp.DefaultMaxRequestSize,
		otlpmetrichttp.DefaultTimeout,
		retry.DefaultConfig,
	)
	if err != nil {
		panic(err)
	}

	perReader := metric.NewPeriodicReader(
		metricExporter,
		metric.DefaultPeriodicInterval,
		metric.DefaultPeriodicTimeout,
		oh,
	)
	meterProvider := metric.NewMeterProvider(
		res,
		perReader,
		metric.DefaultCardinalityLimit,
	)
	defer func() {
		if err := meterProvider.Shutdown(context.Background()); err != nil {
			panic(err)
		}
	}()

	meter := meterProvider.Meter("metroo")
	count, err := meter.Int64Counter("count", oh, ametric.WithDescription("counter"))
	if err != nil {
		panic(err)
	}

	gaug, err := meter.Int64Gauge("gauge", oh, ametric.WithDescription("popokatepetl"))

	if err != nil {
		panic(err)
	}

	// Observable gauge: no Record calls anywhere. The callback is invoked by
	// the SDK itself once per PeriodicReader collection cycle, right before
	// each export; it reports the current value at that moment.
	_, err = meter.Int64ObservableGauge("goroutines", oh,
		[]metric.Int64Callback{
			func(ctx context.Context, o metric.Int64Observer) error {
				o.Observe(int64(runtime.NumGoroutine()))
				return nil
			},
		},
		ametric.WithDescription("number of goroutines, sampled at export time"),
	)
	if err != nil {
		panic(err)
	}

	// RegisterCallback: the other observable style. Instruments are created
	// WITHOUT callbacks; one batch callback observes several of them at once.
	// Useful when one expensive read (here ReadMemStats) feeds many instruments.
	heapAlloc, err := meter.Int64ObservableGauge("heap_alloc", oh, nil,
		ametric.WithDescription("bytes of allocated heap objects"))
	if err != nil {
		panic(err)
	}
	heapObjects, err := meter.Int64ObservableGauge("heap_objects", oh, nil,
		ametric.WithDescription("number of allocated heap objects"))
	if err != nil {
		panic(err)
	}
	_, err = meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)                          // one read...
		o.ObserveInt64(heapAlloc, int64(ms.HeapAlloc))     // ...feeds
		o.ObserveInt64(heapObjects, int64(ms.HeapObjects)) // ...both
		return nil
	}, heapAlloc, heapObjects)
	if err != nil {
		panic(err)
	}

	t := tracerProvider.Tracer("mine2")
	newCtx, newCpan := t.Start(context.Background(), "mujspan")
	time.Sleep(1 * time.Second)
	newCpan.AddEvent("olold2")
	slogger.With("foo", "bar").InfoContext(newCtx, "HELLO")

	count.Add(newCtx, 4)

	gaug.Record(newCtx, 500)
	time.Sleep(1 * time.Second)
	count.Add(newCtx, 4)

	gaug.Record(newCtx, 600)

	newCpan.End()
}
