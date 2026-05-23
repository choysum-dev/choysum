// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package otelmw

import (
	"context"
	"time"

	"github.com/choysum-dev/choysum/pkg/trace"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc"
)

type otelTelemetry struct {
	opts           []grpc.ServerOption
	shutdown       func(ctx context.Context) error
	metricShutdown func(ctx context.Context) error
}

func (t *otelTelemetry) ServerOptions() []grpc.ServerOption { return t.opts }
func (t *otelTelemetry) Shutdown(ctx context.Context) error { return t.shutdown(ctx) }
func (t *otelTelemetry) MetricShutdown(ctx context.Context) error {
	if t.metricShutdown == nil {
		return nil
	}
	return t.metricShutdown(ctx)
}

// NewTelemetry initializes OTel; prepareOnly=true means prepare only without exporting or sampling while still producing valid traces/spans.
func NewTelemetry(serviceName string, prepareOnly bool) (trace.Telemetry, error) {
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	var tp *sdktrace.TracerProvider
	var mp *sdkmetric.MeterProvider
	if prepareOnly {
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.NeverSample()),
			sdktrace.WithResource(res),
		)
		mp = sdkmetric.NewMeterProvider()
	} else {
		exp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, err
		}
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exp,
				sdktrace.WithMaxExportBatchSize(512),
				sdktrace.WithBatchTimeout(2*time.Second),
			),
			sdktrace.WithResource(res),
		)

		metricExp, err := stdoutmetric.New()
		if err != nil {
			return nil, err
		}
		reader := sdkmetric.NewPeriodicReader(metricExp, sdkmetric.WithInterval(15*time.Second))
		mp = sdkmetric.NewMeterProvider(sdkmetric.WithResource(res), sdkmetric.WithReader(reader))
	}

	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	serverHandler := otelgrpc.NewServerHandler()

	return &otelTelemetry{
		opts: []grpc.ServerOption{
			grpc.StatsHandler(serverHandler),
		},
		shutdown: func(ctx context.Context) error {
			return tp.Shutdown(ctx)
		},
		metricShutdown: func(ctx context.Context) error {
			return mp.Shutdown(ctx)
		},
	}, nil
}
