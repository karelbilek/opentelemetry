package main

import (
	"context"
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
		"/v1/logs",
		true,
		64*1024*1024,
		10*time.Second,
		retry.DefaultConfig)
	if err != nil {
		panic(err)
	}
	oh := otel.BasicErrorHandler{}

	res := resource.Default(oh, "my_service")

	processor := log.NewBatchProcessor(logExporter, oh, 2048, time.Second, 30*time.Second, 512)
	provider := log.NewLoggerProvider(
		res,
		[]log.Processor{processor},
		128,
		-1,
		false,
	)
	slogger := otelslog.NewLogger("mylogger", provider, "nevim", "nevim2", nil, true)
	defer func() {
		time.Sleep(30 * time.Second)
		if err := provider.Shutdown(context.Background()); err != nil {
			panic(err)
		}
	}()

	traceExporter, err := otlptracehttp.New(
		context.Background(),
		"127.0.0.1:4318",
		"/v1/traces",
		true,
		nil,
		64*1024*1024,
		10*time.Second,
		retry.DefaultConfig,
	)
	if err != nil {
		panic(err)
	}
	tracerProvider := trace.NewTracerProvider(
		oh,
		-1,
		128,
		128,
		128,
		128,
		128,
		trace.NewBatchSpanProcessor(
			traceExporter,
			oh,
			2048,
			5000*time.Millisecond,
			30000*time.Millisecond,
			512,
			false,
		),
		nil,
		nil,
		res,
		false,
	)
	defer func() {
		if err := tracerProvider.Shutdown(context.Background()); err != nil {
			panic(err)
		}
	}()

	metricExporter, err := otlpmetrichttp.New(
		context.Background(),
		"127.0.0.1:4318",
		"/v1/metrics",
		true,
		nil,
		64*1024*1024,
		10*time.Second,
		metric.DefaultTemporalitySelector,
		metric.DefaultAggregationSelector,
		retry.DefaultConfig,
	)
	if err != nil {
		panic(err)
	}

	perReader := metric.NewPeriodicReader(
		metricExporter,
		time.Millisecond*60000,
		time.Millisecond*30000,
		metric.DefaultCardinalityLimitSelector,
		oh,
	)
	meterProvider := metric.NewMeterProvider(
		res,
		[]metric.Reader{perReader},
		nil,
		2000,
	)
	defer func() {
		if err := meterProvider.Shutdown(context.Background()); err != nil {
			panic(err)
		}
	}()

	meter := meterProvider.Meter("metroo")
	gaug, err := meter.Int64Gauge("gaugoo", oh, ametric.WithDescription("popokatepetl"))

	if err != nil {
		panic(err)
	}

	t := tracerProvider.Tracer("mine2")
	newCtx, newCpan := t.Start(context.Background(), "mujspan")
	time.Sleep(1 * time.Second)
	newCpan.AddEvent("olold2")
	slogger.With("foo", "bar").InfoContext(newCtx, "HELLO")

	gaug.Record(newCtx, 500)

	time.Sleep(1 * time.Second)
	gaug.Record(newCtx, 600)

	newCpan.End()
}
