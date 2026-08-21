package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"

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
		otlploghttp.DefaultLogsPath,
		true,
		nil,
		otlploghttp.DefaultMaxRequestSize,
		otlploghttp.DefaultTimeout,
		retry.DefaultConfig)

	if err != nil {
		return nil, nil, nil, nil, err
	}
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
		return nil, nil, nil, nil, err
	}

	// from now on nothing can return error

	res := resource.Default(context.Background(), oh, resource_name)

	logProcessor := log.NewBatchProcessor(
		logExporter,
		oh,
		log.DefaultMaxQueueSize,
		log.DefaultExpInterval,
		log.DefaultExpTimeout,
		log.DefaultExpMaxBatchSize)
	logProvider := log.NewLoggerProvider(
		oh,
		res,
		logProcessor,
		log.DefaultAttributeCountLimit,
		log.DefaultAttributeValueLengthLimit,
	)
	slogger := otelslog.NewLogger("mylogger", logProvider, true)

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
