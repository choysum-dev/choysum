// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package otelmw

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

func TestOtelTelemetryMethods(t *testing.T) {
	wantErr := errors.New("shutdown")
	telemetry := &otelTelemetry{
		opts: []grpc.ServerOption{},
		shutdown: func(context.Context) error {
			return wantErr
		},
	}

	if len(telemetry.ServerOptions()) != 0 {
		t.Fatalf("ServerOptions len = %d, want 0", len(telemetry.ServerOptions()))
	}
	if err := telemetry.Shutdown(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("Shutdown() error = %v, want %v", err, wantErr)
	}
	if err := telemetry.MetricShutdown(context.Background()); err != nil {
		t.Fatalf("MetricShutdown() error = %v, want nil", err)
	}

	telemetry.metricShutdown = func(context.Context) error { return wantErr }
	if err := telemetry.MetricShutdown(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("MetricShutdown() error = %v, want %v", err, wantErr)
	}
}

func TestNewTelemetryPrepareOnly(t *testing.T) {
	telemetry, err := NewTelemetry("prepare-only-service", true)
	if err != nil {
		t.Fatalf("NewTelemetry(prepareOnly) error = %v", err)
	}

	concrete, ok := telemetry.(*otelTelemetry)
	if !ok {
		t.Fatalf("telemetry type = %T, want *otelTelemetry", telemetry)
	}
	if len(concrete.ServerOptions()) != 1 {
		t.Fatalf("ServerOptions len = %d, want 1", len(concrete.ServerOptions()))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := concrete.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if err := concrete.MetricShutdown(ctx); err != nil {
		t.Fatalf("MetricShutdown() error = %v", err)
	}

	propagator := otel.GetTextMapPropagator()
	carrier := propagation.MapCarrier{}
	spanCtx := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    oteltrace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SpanID:     oteltrace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
		TraceFlags: oteltrace.FlagsSampled,
		Remote:     true,
	})
	ctx = oteltrace.ContextWithSpanContext(context.Background(), spanCtx)
	propagator.Inject(ctx, carrier)
	if _, ok := carrier["traceparent"]; !ok {
		t.Fatalf("carrier = %#v, want traceparent header", carrier)
	}
}

func TestNewTelemetryWithExporters(t *testing.T) {
	telemetry, err := NewTelemetry("exporting-service", false)
	if err != nil {
		t.Fatalf("NewTelemetry(exporting) error = %v", err)
	}

	concrete, ok := telemetry.(*otelTelemetry)
	if !ok {
		t.Fatalf("telemetry type = %T, want *otelTelemetry", telemetry)
	}
	if len(concrete.ServerOptions()) != 1 {
		t.Fatalf("ServerOptions len = %d, want 1", len(concrete.ServerOptions()))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := concrete.MetricShutdown(ctx); err != nil {
		t.Fatalf("MetricShutdown() error = %v", err)
	}
	if err := concrete.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}
