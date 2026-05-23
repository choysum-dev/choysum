// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"context"
	"time"

	otelmw "github.com/choysum-dev/choysum/internal/server/middleware/otel"
	"google.golang.org/grpc"
)

func (s *GRPCWebServer) ensureTelemetry(serverOpts *[]grpc.ServerOption) {
	if s.telemetry == nil {
		if tm, err := otelmw.NewTelemetry("choysum", true); err != nil {
			s.runtimeScope.Logger().Warn("opentelemetry initialization failed", "reason", "tracing_disabled", "error", err)
		} else {
			s.telemetry = tm
		}
	}
	if s.telemetry != nil {
		*serverOpts = append(*serverOpts, s.telemetry.ServerOptions()...)
	}
}

func (s *GRPCWebServer) stopTelemetryRuntime() {
	if s.telemetry == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = s.telemetry.Shutdown(ctx)
	if tm, ok := s.telemetry.(interface {
		MetricShutdown(ctx context.Context) error
	}); ok {
		_ = tm.MetricShutdown(ctx)
	}
	s.telemetry = nil
}
