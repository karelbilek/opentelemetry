package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	otel "github.com/karelbilek/opentelemetry"
	"github.com/karelbilek/opentelemetry/exporters/otlploghttp"
	"github.com/karelbilek/opentelemetry/exporters/otlpmetrichttp"
	"github.com/karelbilek/opentelemetry/exporters/otlptracehttp"
	"github.com/karelbilek/opentelemetry/otelhttp"
	"github.com/karelbilek/opentelemetry/otelslog"
	"github.com/karelbilek/opentelemetry/retry"
	"github.com/karelbilek/opentelemetry/sdk/log"
	"github.com/karelbilek/opentelemetry/sdk/metric"

	"github.com/karelbilek/opentelemetry/sdk/resource"
	"github.com/karelbilek/opentelemetry/sdk/trace"
)

var resource_name = "client"

func stopOtlp(ctx context.Context, logProvider *log.LoggerProvider, tracerProvider *trace.TracerProvider, metricProvider *metric.MeterProvider) error {
	fmt.Println("stopping otlp")
	errs := make(chan error, 3)
	s := &sync.WaitGroup{}
	s.Go(func() {
		if err := logProvider.Shutdown(ctx); err != nil {
			errs <- err
		}
	})
	s.Go(func() {
		if err := tracerProvider.Shutdown(ctx); err != nil {
			errs <- err
		}
	})
	s.Go(func() {
		if err := metricProvider.Shutdown(ctx); err != nil {
			errs <- err
		}
	})
	s.Wait()
	close(errs)
	var err error
	for serr := range errs {
		err = errors.Join(err, serr)
	}
	fmt.Println("stopping otlp done")

	return err
}

func startOtlp(oh otel.ErrorHandler) (*slog.Logger, *log.LoggerProvider, *trace.TracerProvider, *metric.MeterProvider, error) {
	logExporter, err := otlploghttp.New(context.Background(),
		"127.0.0.1:4318",
		"/v1/logs",
		true,
		64*1024*1024,
		10*time.Second,
		retry.DefaultConfig)

	if err != nil {
		return nil, nil, nil, nil, err
	}
	metricExporter, err := otlpmetrichttp.New(
		context.Background(),
		"127.0.0.1:4318",
		"/v1/metrics",
		true,
		nil,
		64*1024*1024,
		10*time.Second,
		retry.DefaultConfig,
	)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// from now on nothing can return error

	res := resource.Default(context.Background(), oh, resource_name)

	logProcessor := log.NewBatchProcessor(logExporter, oh, 2048, time.Second, 30*time.Second, 512)
	logProvider := log.NewLoggerProvider(
		oh,
		res,
		logProcessor,
		128,
		-1,
	)
	slogger := otelslog.NewLogger("mylogger", logProvider, true)

	traceExporter := otlptracehttp.New(
		"127.0.0.1:4318",
		"/v1/traces",
		true,
		nil,
		64*1024*1024,
		10*time.Second,
		retry.DefaultConfig,
	)
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
		res,
		false,
	)

	perReader := metric.NewPeriodicReader(
		metricExporter,
		time.Millisecond*60000,
		time.Millisecond*30000,
		oh,
	)
	meterProvider := metric.NewMeterProvider(
		res,
		perReader,
		2000,
	)

	return slogger, logProvider, tracerProvider, meterProvider, nil
}

func get(ctx context.Context, cl http.Client, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	res, err := cl.Do(req)
	if err != nil {
		return "", err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	err = res.Body.Close()
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func main() {

	oh := otel.BasicErrorHandler{}

	slogger, logProvider, tracerProvider, meterProvider, err := startOtlp(oh)
	if err != nil {
		panic(err)
	}
	defer stopOtlp(context.Background(), logProvider, tracerProvider, meterProvider)

	cl := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport, oh, tracerProvider, meterProvider),
	}

	tracer := tracerProvider.Tracer("main")
	ctx, span := tracer.Start(context.Background(), "main()")
	defer span.End()
	slogger.InfoContext(ctx, "in main!")
	body, err := get(ctx, cl, "http://127.0.0.1:8090/hello/5")
	if err != nil {
		panic(err)
	}
	slogger.InfoContext(ctx, "got the message", "message", string(body))
}
