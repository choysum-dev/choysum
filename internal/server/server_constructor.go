// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"github.com/choysum-dev/choysum/internal/server/reload"
	"github.com/choysum-dev/choysum/internal/server/runplan"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/registry"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/choysum-dev/choysum/pkg/server"
	"github.com/choysum-dev/choysum/pkg/trace"
)

type ConstructorOption interface {
	apply(*GRPCWebServer)
}

type constructorOptionFunc func(*GRPCWebServer)

func (f constructorOptionFunc) apply(s *GRPCWebServer) {
	if f == nil {
		return
	}
	f(s)
}

// WithRegistry injects a custom registry implementation.
func WithRegistry(r registry.Registry) ConstructorOption {
	return constructorOptionFunc(func(s *GRPCWebServer) { s.registry = r })
}

// WithExecutor injects a custom JS executor instance.
func WithExecutor(exec jsexecutor.JsExecutor) ConstructorOption {
	return constructorOptionFunc(func(s *GRPCWebServer) { s.jsExecutor = exec })
}

// WithTelemetry injects the telemetry implementation.
func WithTelemetry(tel trace.Telemetry) ConstructorOption {
	return constructorOptionFunc(func(s *GRPCWebServer) { s.telemetry = tel })
}

func NewServer(runtimeScope scope.Scope, opts ...ConstructorOption) server.Server {
	s := &GRPCWebServer{
		runtimeScope:   runtimeScope,
		runtimeOptions: runtimeOptionsFromScope(runtimeScope),
		runState:       runState{runMode: runplan.RunModeApplication},
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt.apply(s)
	}

	if s.registry == nil {
		s.registry = registry.NewRegistry(runtimeScope)
	}
	reload.Register(s.Restart)

	return s
}
