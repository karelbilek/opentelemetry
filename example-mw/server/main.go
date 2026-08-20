package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
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
	"github.com/karelbilek/opentelemetry/sdk/metric/metricinternals"

	"github.com/karelbilek/opentelemetry/sdk/resource"
	"github.com/karelbilek/opentelemetry/sdk/trace"
)

var resource_name = "server"

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
		metricinternals.DefaultAggregationSelector,
		retry.DefaultConfig,
	)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// from now on nothing can return error

	res := resource.Default(oh, resource_name)

	logProcessor := log.NewBatchProcessor(logExporter, oh, 2048, time.Second, 30*time.Second, 512)
	logProvider := log.NewLoggerProvider(
		res,
		[]log.Processor{logProcessor},
		128,
		-1,
		false,
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
		metricinternals.DefaultCardinalityLimitSelector,
		oh,
	)
	meterProvider := metric.NewMeterProvider(
		res,
		perReader,
		2000,
	)

	return slogger, logProvider, tracerProvider, meterProvider, nil
}

func main() {
	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	oh := otel.BasicErrorHandler{}

	_, logProvider, tracerProvider, meterProvider, err := startOtlp(oh)
	if err != nil {
		panic(err)
	}
	defer stopOtlp(context.Background(), logProvider, tracerProvider, meterProvider)

	sm := http.NewServeMux()
	sm.Handle("GET /from/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "from %s", r.PathValue("id"))
	}))

	sm.Handle("GET /hello", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello")
	}))

	s := otelhttp.NewMiddleware(oh, tracerProvider, meterProvider, nil, nil)(sm)

	server := &http.Server{
		Addr:    ":8091",
		Handler: s,
	}

	go func() {
		fmt.Println("Server starting on :8091.")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	<-rootCtx.Done()
	stop()
	err = server.Shutdown(rootCtx)
	if err != nil {
		fmt.Println("err", err)
	}
}
