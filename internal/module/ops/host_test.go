// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/choysum-dev/choysum/pkg/bus"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
)

type hostTestExecutor struct {
	started int
}

func (e *hostTestExecutor) AppendJsScripts(...*jsengine.JsScript) {}
func (e *hostTestExecutor) Start() error                          { e.started++; return nil }
func (e *hostTestExecutor) Stop() error                           { return nil }
func (e *hostTestExecutor) Execute(context.Context, *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	return &jsengine.JsResponse{}, nil
}
func (e *hostTestExecutor) GetJsScripts() []*jsengine.JsScript { return nil }
func (e *hostTestExecutor) SetJsScripts([]*jsengine.JsScript)  {}
func (e *hostTestExecutor) Reload(...*jsengine.JsScript) error { return nil }

func TestNewHostRequiresScope(t *testing.T) {
	_, err := NewHost(HostOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewHostRequiresExecutorWhenLifecycleMissing(t *testing.T) {
	base := &opsTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    &config.Config{},
	}
	_, err := NewHost(HostOptions{Scope: base})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewHostAssemblesLifecycleAndResolver(t *testing.T) {
	base := &opsTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    &config.Config{ModulesPath: t.TempDir()},
	}
	exec := &hostTestExecutor{}
	host, err := NewHost(HostOptions{
		Scope:    base,
		Executor: exec,
		LockerFactory: func(scope.Scope) statepkg.Locker {
			return testLocker{}
		},
		Events: bus.EnsureHost(base),
	})
	if err != nil {
		t.Fatalf("NewHost: %v", err)
	}
	if host.Lifecycle == nil || host.Resolver == nil {
		t.Fatal("expected Lifecycle and Resolver")
	}
	if host.Scope != base {
		t.Fatal("expected scope preserved")
	}
}

func TestNewHostKeepsInjectedLifecycle(t *testing.T) {
	base := &opsTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    &config.Config{},
	}
	life := &stubLifecycle{}
	host, err := NewHost(HostOptions{
		Scope:     base,
		Lifecycle: life,
		Resolver:  &mapResolver{},
	})
	if err != nil {
		t.Fatalf("NewHost: %v", err)
	}
	if host.Lifecycle != life {
		t.Fatal("expected injected Lifecycle")
	}
}

func TestNewHostDefaultLockerFactory(t *testing.T) {
	base := &opsTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    &config.Config{ModulesPath: t.TempDir()},
	}
	life := &stubLifecycle{}
	host, err := NewHost(HostOptions{
		Scope:     base,
		Lifecycle: life,
		Resolver:  &mapResolver{},
		// LockerFactory nil → default lease.New wrapper
	})
	if err != nil {
		t.Fatalf("NewHost: %v", err)
	}
	if host.LockerFactory == nil {
		t.Fatal("expected default LockerFactory")
	}
	if got := host.LockerFactory(base); got == nil {
		t.Fatal("default LockerFactory returned nil")
	}
}
