package main

import (
	"context"
	"errors"
	"fmt"
	"io"
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

	"github.com/karelbilek/opentelemetry/sdk/resource"
	"github.com/karelbilek/opentelemetry/sdk/trace"
)

var resource_name = "server_plus_client"

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
		metric.DefaultTemporalitySelector,
		metric.DefaultAggregationSelector,
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
		metric.DefaultCardinalityLimitSelector,
		oh,
	)
	meterProvider := metric.NewMeterProvider(
		res,
		[]metric.Reader{perReader},
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
	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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
	sm := http.NewServeMux()
	sm.Handle("GET /hello/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		slogger.InfoContext(ctx, "inside handler")
		ctx, span := tracer.Start(r.Context(), "subspan")
		defer span.End()
		slogger.InfoContext(ctx, "inside subspan")

		var body1 string
		var body2 string

		errs := make(chan error, 2)
		s := &sync.WaitGroup{}
		s.Go(func() {
			body, err := get(ctx, cl, "http://127.0.0.1:8091/from/"+r.PathValue("id"))
			if err != nil {
				errs <- err
			} else {
				body2 = body
			}
		})
		s.Go(func() {
			body, err := get(ctx, cl, "http://127.0.0.1:8091/hello")
			if err != nil {
				errs <- err
			} else {
				body1 = body
			}
		})
		s.Wait()
		close(errs)
		var err error
		for serr := range errs {
			err = errors.Join(err, serr)
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "error")
			return
		}
		fmt.Fprint(w, body1, " ", body2)
		go func() {
			// just here to test force flushing
			if err := logProvider.ForceFlush(context.Background()); err != nil {
				panic(err)
			}
			if err := tracerProvider.ForceFlush(context.Background()); err != nil {
				panic(err)
			}
			if err := meterProvider.ForceFlush(context.Background()); err != nil {
				panic(err)
			}
		}()
	}))

	s := otelhttp.NewMiddleware(oh, tracerProvider, meterProvider, nil, nil)(sm)

	server := &http.Server{
		Addr:    ":8090",
		Handler: s,
	}

	go func() {
		fmt.Println("Server starting on :8090.")
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
