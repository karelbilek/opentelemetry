package main

import (
	"context"
	"time"

	otel "github.com/karelbilek/opentelemetry"
	"github.com/karelbilek/opentelemetry/exporters/otlploghttp"
	"github.com/karelbilek/opentelemetry/exporters/otlptracehttp"
	"github.com/karelbilek/opentelemetry/metric/noop"
	"github.com/karelbilek/opentelemetry/otelslog"
	"github.com/karelbilek/opentelemetry/retry"
	"github.com/karelbilek/opentelemetry/sdk/log"
	"github.com/karelbilek/opentelemetry/sdk/resource"
	"github.com/karelbilek/opentelemetry/sdk/trace"
)

func main() {
	exporter, err := otlploghttp.New(context.Background(),
		"127.0.0.1:4318",
		"/v1/logs",
		true,
		64*1024*1024,
		10*time.Second,
		retry.DefaultConfig)
	if err != nil {
		panic(err)
	}
	mp := noop.MeterProvider{}
	oh := otel.BasicErrorHandler{}

	res := resource.Default(oh, "my_service")

	processor := log.NewBatchProcessor(exporter, mp, oh, 2048, time.Second, 30*time.Second, 512)
	provider := log.NewLoggerProvider(
		res,
		[]log.Processor{processor},
		128,
		-1,
		false,
	)
	slogger := otelslog.NewLogger("mylogger", provider, mp, oh, "nevim", "nevim2", nil, true)
	defer func() {
		time.Sleep(30 * time.Second)
		if err := provider.Shutdown(context.Background()); err != nil {
			panic(err)
		}
	}()

	exp, err := otlptracehttp.New(
		context.Background(),
		mp,
		"127.0.0.1:4318",
		true,
		nil,
		64*1024*1024,
		10*time.Second,
		"/v1/traces",
		retry.DefaultConfig,
	)
	if err != nil {
		panic(err)
	}
	tracerProvider := trace.NewTracerProvider(
		mp,
		oh,
		-1,
		128,
		128,
		128,
		128,
		128,
		[]trace.SpanProcessor{trace.NewBatchSpanProcessor(
			exp,
			mp,
			oh,
			2048,
			5000*time.Millisecond,
			30000*time.Millisecond,
			512,
			false,
		)},
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
	t := tracerProvider.Tracer("mine2")
	newCtx, newCpan := t.Start(context.Background(), "mujspan")
	time.Sleep(1 * time.Second)
	newCpan.AddEvent("olold2")
	slogger.With("foo", "bar").InfoContext(newCtx, "HELLO")

	time.Sleep(1 * time.Second)

	newCpan.End()
}
