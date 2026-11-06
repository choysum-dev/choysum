// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"

	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	"github.com/choysum-dev/choysum/internal/module/plan"
	"github.com/choysum-dev/choysum/pkg/bus"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
)

// Action values persisted on meta_module_operation.action.
const (
	ActionInstall   = "install"
	ActionUpgrade   = "upgrade"
	ActionUninstall = "uninstall"
	ActionSync      = "sync"
	ActionLoadDemo  = "load-demo"
)

// Status values persisted on meta_module_operation.status.
const (
	StatusDraft     = "draft"
	StatusQueued    = "queued"
	StatusRunning   = "running"
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
)

// Phase values persisted on meta_module_operation.phase.
const (
	PhaseQueued    = "queued"
	PhaseExecuting = "executing"
	PhaseReloading = "reloading"
	PhaseDone      = "done"
)

const (
	originLocal    = "local"
	originRegistry = "registry"
)

// Overridable defaults for SyncSourceSnapshotsThenMaterialize (tests).
var (
	defaultSyncLocal    = lifecycle.SyncLocalModuleSourceSnapshot
	defaultSyncRegistry = lifecycle.SyncRegistryModuleSourceSnapshot
)

// SnapshotSyncFunc syncs one origin's module source snapshots (local or registry).
type SnapshotSyncFunc func(ctx context.Context, runtimeScope scope.Scope, lockerFactory statepkg.LockerFactory) (lifecycle.ModuleSourceSnapshotSyncStats, error)

// Ops is the module control-plane host for Plan / Apply / Run and catalog materialize.
type Ops struct {
	Scope         scope.Scope
	Lifecycle     lifecycle.Service
	Resolver      plan.Resolver
	LockerFactory statepkg.LockerFactory
	Events        bus.EventBus

	// Optional sync hooks; nil uses lifecycle.Sync*ModuleSourceSnapshot.
	SyncLocalSnapshot    SnapshotSyncFunc
	SyncRegistrySnapshot SnapshotSyncFunc
}
