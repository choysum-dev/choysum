// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"errors"

	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	"github.com/choysum-dev/choysum/internal/module/plan"
	"github.com/choysum-dev/choysum/internal/state/lease"
	"github.com/choysum-dev/choysum/pkg/bus"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
)

// HostOptions assembles an Ops control-plane host.
// Executor must already be started when Lifecycle or Resolver is not provided.
type HostOptions struct {
	Scope         scope.Scope
	Executor      jsexecutor.JsExecutor
	LockerFactory statepkg.LockerFactory
	Events        bus.EventBus
	Lifecycle     lifecycle.Service
	Resolver      plan.Resolver
}

// NewHost builds an Ops host with Lifecycle and Resolver ready for Plan/Apply/Run.
func NewHost(opts HostOptions) (*Ops, error) {
	if opts.Scope == nil {
		return nil, errors.New("ops: scope is required")
	}
	lockerFactory := opts.LockerFactory
	if lockerFactory == nil {
		lockerFactory = func(runtimeScope scope.Scope) statepkg.Locker {
			return lease.New(runtimeScope)
		}
	}
	events := opts.Events
	if !bus.IsUsable(events) {
		events = bus.EnsureHost(opts.Scope)
	}

	life := opts.Lifecycle
	resolver := opts.Resolver
	if life == nil || resolver == nil {
		if opts.Executor == nil {
			return nil, errors.New("ops: executor is required when Lifecycle or Resolver is nil")
		}
		if life == nil {
			life = lifecycle.NewService(opts.Scope, opts.Executor, lifecycle.WithLockerFactory(lockerFactory))
		}
		if resolver == nil {
			resolver = lifecycle.NewModuleManager(opts.Scope, opts.Executor, lifecycle.WithLockerFactory(lockerFactory))
		}
	}

	return &Ops{
		Scope:         opts.Scope,
		Lifecycle:     life,
		Resolver:      resolver,
		LockerFactory: lockerFactory,
		Events:        events,
	}, nil
}
