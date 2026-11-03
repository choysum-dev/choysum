// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package provision

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	"github.com/choysum-dev/choysum/internal/module/plan"
	"github.com/choysum-dev/choysum/internal/state/lease"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
)

type noopProvisionExecutor struct{}

func (e *noopProvisionExecutor) Execute(context.Context, *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	return &jsengine.JsResponse{}, nil
}
func (e *noopProvisionExecutor) GetJsScripts() []*jsengine.JsScript            { return nil }
func (e *noopProvisionExecutor) SetJsScripts(scripts []*jsengine.JsScript)     {}
func (e *noopProvisionExecutor) AppendJsScripts(scripts ...*jsengine.JsScript) {}
func (e *noopProvisionExecutor) Reload(scripts ...*jsengine.JsScript) error    { return nil }
func (e *noopProvisionExecutor) Start() error                                  { return nil }
func (e *noopProvisionExecutor) Stop() error                                   { return nil }

func TestMinimalInstallRequest(t *testing.T) {
	req := MinimalInstallRequest()
	if req.Input != MinimalRootModule || req.WithDemo || req.WebShellProfile != plan.WebShellProfileMinimal {
		t.Fatalf("req=%+v", req)
	}
}

func TestMinimalClosureModules(t *testing.T) {
	got := MinimalClosureModules()
	want := []string{MinimalRootModule, "core", "auth", "meta", "base", "task"}
	if len(got) != len(want) {
		t.Fatalf("got=%v want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got=%v want=%v", got, want)
		}
	}
}

func TestProvisionMinimalSkipsWhenSetupPending(t *testing.T) {
	runtimeScope, db := newSetupStateScope(t)
	createMetaModuleTable(t, db)
	createAuthAppSettingTable(t, db)
	insertAuthModule(t, db, meta.Installed)
	called := false
	result, err := ProvisionMinimal(context.Background(), runtimeScope, Options{
		InstallModule: func(context.Context, scope.Scope, jsexecutor.ScriptExecutor, lifecycle.InstallModuleRequest, ...lifecycle.Option) error {
			called = true
			return nil
		},
	})
	if err != nil {
		t.Fatalf("ProvisionMinimal: %v", err)
	}
	if !result.Skipped || result.Phase != PhaseSetupPending {
		t.Fatalf("result=%+v", result)
	}
	if called {
		t.Fatal("InstallModule must not run when already SetupPending")
	}
}

func TestProvisionMinimalInstallsWebMinimalAndStaysSetupPending(t *testing.T) {
	runtimeScope, db := newSetupStateScope(t)
	createMetaModuleTable(t, db)
	createAuthAppSettingTable(t, db)
	var sawReq lifecycle.InstallModuleRequest
	result, err := ProvisionMinimal(context.Background(), runtimeScope, Options{
		InstallTimeout: time.Minute,
		LockerFactory:  func(scope.Scope) statepkg.Locker { return &okLocker{} },
		NewExecutor: func(scope.Scope) (jsexecutor.JsExecutor, error) {
			return &noopProvisionExecutor{}, nil
		},
		InstallModule: func(ctx context.Context, runtimeScope scope.Scope, jsExecutor jsexecutor.ScriptExecutor, req lifecycle.InstallModuleRequest, opts ...lifecycle.Option) error {
			sawReq = req
			insertAuthModule(t, db, meta.Installed)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("ProvisionMinimal: %v", err)
	}
	if result.Skipped || result.Phase != PhaseSetupPending {
		t.Fatalf("result=%+v", result)
	}
	if sawReq.Input != "web" || sawReq.WithDemo || sawReq.WebShellProfile != plan.WebShellProfileMinimal {
		t.Fatalf("InstallModule request=%+v", sawReq)
	}

	var count int64
	if err := db.Table(authAppSettingTable).Where(map[string]any{"key": authSetupCompletedAtKey}).Count(&count).Error; err != nil {
		t.Fatalf("count setup marker: %v", err)
	}
	if count != 0 {
		t.Fatalf("setup.completed_at rows=%d want 0", count)
	}
}

func TestProvisionMinimalAcceptsLegacyReadyAfterInstall(t *testing.T) {
	runtimeScope, db := newSetupStateScope(t)
	createMetaModuleTable(t, db)
	createAuthAppSettingTable(t, db)
	result, err := ProvisionMinimal(context.Background(), runtimeScope, Options{
		LockerFactory: func(scope.Scope) statepkg.Locker { return &okLocker{} },
		NewExecutor: func(scope.Scope) (jsexecutor.JsExecutor, error) {
			return &noopProvisionExecutor{}, nil
		},
		InstallModule: func(context.Context, scope.Scope, jsexecutor.ScriptExecutor, lifecycle.InstallModuleRequest, ...lifecycle.Option) error {
			insertAuthModule(t, db, meta.Installed)
			mustExec(t, db, `INSERT INTO auth_app_setting (id, key, value, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
				"seed", authSetupCompletedAtKey, time.Now().UTC().Format(time.RFC3339), time.Now(), time.Now())
			return nil
		},
	})
	if err != nil {
		t.Fatalf("ProvisionMinimal: %v", err)
	}
	if result.Skipped || result.Phase != PhaseReady {
		t.Fatalf("result=%+v want Ready without skip", result)
	}
}

func TestProvisionMinimalSkipsWhenPeerFinishesUnderLease(t *testing.T) {
	runtimeScope, db := newSetupStateScope(t)
	createMetaModuleTable(t, db)
	createAuthAppSettingTable(t, db)
	called := false
	result, err := ProvisionMinimal(context.Background(), runtimeScope, Options{
		LockerFactory: func(scope.Scope) statepkg.Locker {
			return &okLocker{onAcquire: func() {
				insertAuthModule(t, db, meta.Installed)
			}}
		},
		NewExecutor: func(scope.Scope) (jsexecutor.JsExecutor, error) {
			t.Fatal("executor must not start when phase is no longer Empty")
			return nil, nil
		},
		InstallModule: func(context.Context, scope.Scope, jsexecutor.ScriptExecutor, lifecycle.InstallModuleRequest, ...lifecycle.Option) error {
			called = true
			return nil
		},
	})
	if err != nil {
		t.Fatalf("ProvisionMinimal: %v", err)
	}
	if !result.Skipped || result.Phase != PhaseSetupPending {
		t.Fatalf("result=%+v", result)
	}
	if called {
		t.Fatal("InstallModule must not run after peer provision")
	}
}

func TestProvisionMinimalLeaseBusy(t *testing.T) {
	runtimeScope, _ := newSetupStateScope(t)
	busy := &busyLocker{}
	_, err := ProvisionMinimal(context.Background(), runtimeScope, Options{
		LockerFactory: func(scope.Scope) statepkg.Locker { return busy },
		NewExecutor: func(scope.Scope) (jsexecutor.JsExecutor, error) {
			t.Fatal("executor must not start when lease is busy")
			return nil, nil
		},
	})
	if err == nil || !errors.Is(err, ErrLeaseBusy) {
		t.Fatalf("error=%v", err)
	}
}

func TestProvisionMinimalNilScopeAndNilContext(t *testing.T) {
	if _, err := ProvisionMinimal(context.Background(), nil, Options{}); err == nil {
		t.Fatal("expected nil scope error")
	}
	runtimeScope, db := newSetupStateScope(t)
	createMetaModuleTable(t, db)
	createAuthAppSettingTable(t, db)
	result, err := ProvisionMinimal(nil, runtimeScope, Options{
		LockerFactory: func(scope.Scope) statepkg.Locker { return &okLocker{} },
		NewExecutor: func(scope.Scope) (jsexecutor.JsExecutor, error) {
			return &noopProvisionExecutor{}, nil
		},
		InstallModule: func(context.Context, scope.Scope, jsexecutor.ScriptExecutor, lifecycle.InstallModuleRequest, ...lifecycle.Option) error {
			insertAuthModule(t, db, meta.Installed)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("nil ctx: %v", err)
	}
	if result.Phase != PhaseSetupPending {
		t.Fatalf("result=%+v", result)
	}
}

func TestProvisionMinimalInstallFailureAndStillEmpty(t *testing.T) {
	runtimeScope, db := newSetupStateScope(t)
	createMetaModuleTable(t, db)
	createAuthAppSettingTable(t, db)
	_, err := ProvisionMinimal(context.Background(), runtimeScope, Options{
		LockerFactory: func(scope.Scope) statepkg.Locker { return &okLocker{} },
		NewExecutor: func(scope.Scope) (jsexecutor.JsExecutor, error) {
			return &noopProvisionExecutor{}, nil
		},
		InstallModule: func(context.Context, scope.Scope, jsexecutor.ScriptExecutor, lifecycle.InstallModuleRequest, ...lifecycle.Option) error {
			return errors.New("install boom")
		},
	})
	if err == nil || !strings.Contains(err.Error(), "provision install web") || !strings.Contains(err.Error(), "install boom") {
		t.Fatalf("error=%v", err)
	}

	_, err = ProvisionMinimal(context.Background(), runtimeScope, Options{
		LockerFactory: func(scope.Scope) statepkg.Locker { return &okLocker{} },
		NewExecutor: func(scope.Scope) (jsexecutor.JsExecutor, error) {
			return &noopProvisionExecutor{}, nil
		},
		InstallModule: func(context.Context, scope.Scope, jsexecutor.ScriptExecutor, lifecycle.InstallModuleRequest, ...lifecycle.Option) error {
			return nil // auth never installed
		},
	})
	if err == nil || !strings.Contains(err.Error(), "still empty") {
		t.Fatalf("error=%v", err)
	}
}

func TestProvisionMinimalExecutorErrors(t *testing.T) {
	runtimeScope, _ := newSetupStateScope(t)
	_, err := ProvisionMinimal(context.Background(), runtimeScope, Options{
		LockerFactory: func(scope.Scope) statepkg.Locker { return &okLocker{} },
		NewExecutor: func(scope.Scope) (jsexecutor.JsExecutor, error) {
			return nil, errors.New("executor create failed")
		},
	})
	if err == nil || !strings.Contains(err.Error(), "create provision executor") {
		t.Fatalf("error=%v", err)
	}

	_, err = ProvisionMinimal(context.Background(), runtimeScope, Options{
		LockerFactory: func(scope.Scope) statepkg.Locker { return &okLocker{} },
		NewExecutor: func(scope.Scope) (jsexecutor.JsExecutor, error) {
			return &failStartExecutor{}, nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "start provision executor") {
		t.Fatalf("error=%v", err)
	}
}

func TestProvisionMinimalLeaseLostDuringInstall(t *testing.T) {
	prev := leaseRenewInterval
	leaseRenewInterval = time.Millisecond
	t.Cleanup(func() { leaseRenewInterval = prev })

	runtimeScope, _ := newSetupStateScope(t)
	started := make(chan struct{})
	_, err := ProvisionMinimal(context.Background(), runtimeScope, Options{
		LockerFactory: func(scope.Scope) statepkg.Locker {
			return &failRenewLocker{started: started, renewErr: lease.ErrLeaseNotOwner}
		},
		NewExecutor: func(scope.Scope) (jsexecutor.JsExecutor, error) {
			return &noopProvisionExecutor{}, nil
		},
		InstallModule: func(ctx context.Context, runtimeScope scope.Scope, jsExecutor jsexecutor.ScriptExecutor, req lifecycle.InstallModuleRequest, opts ...lifecycle.Option) error {
			close(started)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(2 * time.Second):
				return errors.New("install should have been canceled")
			}
		},
	})
	if err == nil || !strings.Contains(err.Error(), "lease lost") {
		t.Fatalf("error=%v", err)
	}
}

func TestProvisionMinimalTransientRenewRetriesThenSucceeds(t *testing.T) {
	prevInterval := leaseRenewInterval
	prevTTL := leaseTTL
	leaseRenewInterval = 5 * time.Millisecond
	leaseTTL = time.Second
	t.Cleanup(func() {
		leaseRenewInterval = prevInterval
		leaseTTL = prevTTL
	})

	runtimeScope, db := newSetupStateScope(t)
	createMetaModuleTable(t, db)
	createAuthAppSettingTable(t, db)
	runtimeScope.config = &config.Config{Db: &config.DbConfig{Dialect: "postgres"}}

	locker := &failRenewLocker{renewErr: errors.New("network blip"), failTimes: 3}
	result, err := ProvisionMinimal(context.Background(), runtimeScope, Options{
		InstallTimeout: time.Minute,
		LockerFactory:  func(scope.Scope) statepkg.Locker { return locker },
		NewExecutor: func(scope.Scope) (jsexecutor.JsExecutor, error) {
			return &noopProvisionExecutor{}, nil
		},
		InstallModule: func(ctx context.Context, runtimeScope scope.Scope, jsExecutor jsexecutor.ScriptExecutor, req lifecycle.InstallModuleRequest, opts ...lifecycle.Option) error {
			time.Sleep(40 * time.Millisecond) // allow a few renew ticks
			insertAuthModule(t, db, meta.Installed)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("ProvisionMinimal: %v", err)
	}
	if result.Phase != PhaseSetupPending {
		t.Fatalf("result=%+v", result)
	}
	if locker.renewCalls.Load() < 3 {
		t.Fatalf("renewCalls=%d want >= 3", locker.renewCalls.Load())
	}
}

func TestProvisionMinimalTransientRenewBoundCancels(t *testing.T) {
	prevInterval := leaseRenewInterval
	prevTTL := leaseTTL
	leaseRenewInterval = 5 * time.Millisecond
	leaseTTL = 30 * time.Millisecond
	t.Cleanup(func() {
		leaseRenewInterval = prevInterval
		leaseTTL = prevTTL
	})

	runtimeScope, _ := newSetupStateScope(t)
	runtimeScope.config = &config.Config{Db: &config.DbConfig{Dialect: "postgres"}}
	started := make(chan struct{})
	_, err := ProvisionMinimal(context.Background(), runtimeScope, Options{
		LockerFactory: func(scope.Scope) statepkg.Locker {
			return &failRenewLocker{started: started, renewErr: errors.New("network blip")}
		},
		NewExecutor: func(scope.Scope) (jsexecutor.JsExecutor, error) {
			return &noopProvisionExecutor{}, nil
		},
		InstallModule: func(ctx context.Context, runtimeScope scope.Scope, jsExecutor jsexecutor.ScriptExecutor, req lifecycle.InstallModuleRequest, opts ...lifecycle.Option) error {
			close(started)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(2 * time.Second):
				return errors.New("install should have been canceled")
			}
		},
	})
	if err == nil || !strings.Contains(err.Error(), "lease lost") {
		t.Fatalf("error=%v", err)
	}
}

func TestProvisionMinimalSqliteLockRenewRecovers(t *testing.T) {
	prevInterval := leaseRenewInterval
	prevTTL := leaseTTL
	leaseRenewInterval = 5 * time.Millisecond
	leaseTTL = time.Second
	t.Cleanup(func() {
		leaseRenewInterval = prevInterval
		leaseTTL = prevTTL
	})

	runtimeScope, db := newSetupStateScope(t)
	createMetaModuleTable(t, db)
	createAuthAppSettingTable(t, db)
	runtimeScope.config = &config.Config{Db: &config.DbConfig{Dialect: "sqlite"}}

	locker := &failRenewLocker{renewErr: errors.New("database is locked"), failTimes: 2}
	result, err := ProvisionMinimal(context.Background(), runtimeScope, Options{
		LockerFactory: func(scope.Scope) statepkg.Locker { return locker },
		NewExecutor: func(scope.Scope) (jsexecutor.JsExecutor, error) {
			return &noopProvisionExecutor{}, nil
		},
		InstallModule: func(ctx context.Context, runtimeScope scope.Scope, jsExecutor jsexecutor.ScriptExecutor, req lifecycle.InstallModuleRequest, opts ...lifecycle.Option) error {
			time.Sleep(40 * time.Millisecond)
			insertAuthModule(t, db, meta.Installed)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("ProvisionMinimal: %v", err)
	}
	if result.Phase != PhaseSetupPending {
		t.Fatalf("result=%+v", result)
	}
}

func TestProvisionMinimalResolveAndAcquireErrors(t *testing.T) {
	runtimeScope, db := newSetupStateScope(t)
	// Soft-delete query fails on incomplete meta_module schema.
	mustExec(t, db, `CREATE TABLE meta_module (id TEXT PRIMARY KEY, name TEXT, status TEXT)`)
	mustExec(t, db, `INSERT INTO meta_module (id, name, status) VALUES (?, ?, ?)`,
		"mod-auth", "auth", string(meta.Installed))
	if _, err := ProvisionMinimal(context.Background(), runtimeScope, Options{}); err == nil {
		t.Fatal("expected Resolve error")
	}

	runtimeScope2, db2 := newSetupStateScope(t)
	createMetaModuleTable(t, db2)
	createAuthAppSettingTable(t, db2)
	_, err := ProvisionMinimal(context.Background(), runtimeScope2, Options{
		LockerFactory: func(scope.Scope) statepkg.Locker { return &errAcquireLocker{err: errors.New("acquire boom")} },
	})
	if err == nil || !strings.Contains(err.Error(), "acquire provision lease") {
		t.Fatalf("error=%v", err)
	}
}

func TestProvisionMinimalRecheckResolveError(t *testing.T) {
	runtimeScope, db := newSetupStateScope(t)
	createMetaModuleTable(t, db)
	createAuthAppSettingTable(t, db)
	_, err := ProvisionMinimal(context.Background(), runtimeScope, Options{
		LockerFactory: func(scope.Scope) statepkg.Locker {
			return &okLocker{onAcquire: func() {
				mustExec(t, db, `DROP TABLE meta_module`)
				mustExec(t, db, `CREATE TABLE meta_module (id TEXT PRIMARY KEY, name TEXT, status TEXT)`)
				mustExec(t, db, `INSERT INTO meta_module (id, name, status) VALUES (?, ?, ?)`,
					"mod-auth", "auth", string(meta.Installed))
			}}
		},
	})
	if err == nil {
		t.Fatal("expected re-check Resolve error")
	}
}

func TestProvisionMinimalNilWithContextFallsBack(t *testing.T) {
	base, db := newSetupStateScope(t)
	createMetaModuleTable(t, db)
	createAuthAppSettingTable(t, db)
	runtimeScope := &nilWithContextScope{setupStateTestScope: *base}
	result, err := ProvisionMinimal(context.Background(), runtimeScope, Options{
		LockerFactory: func(scope.Scope) statepkg.Locker { return &okLocker{} },
		NewExecutor: func(scope.Scope) (jsexecutor.JsExecutor, error) {
			return &noopProvisionExecutor{}, nil
		},
		InstallModule: func(context.Context, scope.Scope, jsexecutor.ScriptExecutor, lifecycle.InstallModuleRequest, ...lifecycle.Option) error {
			insertAuthModule(t, db, meta.Installed)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("ProvisionMinimal: %v", err)
	}
	if result.Phase != PhaseSetupPending {
		t.Fatalf("result=%+v", result)
	}
}

func TestProvisionMinimalLeaseLostBeforeInstall(t *testing.T) {
	runtimeScope, db := newSetupStateScope(t)
	createMetaModuleTable(t, db)
	createAuthAppSettingTable(t, db)
	_, err := ProvisionMinimal(context.Background(), runtimeScope, Options{
		LockerFactory: func(scope.Scope) statepkg.Locker {
			return &failRenewLocker{renewErr: lease.ErrLeaseNotOwner, failImmediate: true}
		},
		NewExecutor: func(scope.Scope) (jsexecutor.JsExecutor, error) {
			return &noopProvisionExecutor{}, nil
		},
		InstallModule: func(context.Context, scope.Scope, jsexecutor.ScriptExecutor, lifecycle.InstallModuleRequest, ...lifecycle.Option) error {
			t.Fatal("InstallModule must not run after lease lost")
			return nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "lease lost") {
		t.Fatalf("error=%v", err)
	}
}

func TestProvisionMinimalLeaseLostAfterInstall(t *testing.T) {
	prev := leaseRenewInterval
	leaseRenewInterval = time.Millisecond
	t.Cleanup(func() { leaseRenewInterval = prev })

	runtimeScope, db := newSetupStateScope(t)
	createMetaModuleTable(t, db)
	createAuthAppSettingTable(t, db)
	started := make(chan struct{})
	_, err := ProvisionMinimal(context.Background(), runtimeScope, Options{
		LockerFactory: func(scope.Scope) statepkg.Locker {
			return &countingRenewLocker{
				started: started,
				onRenew: func(n int32) error {
					if n == 1 {
						return nil // immediate probe after acquire
					}
					return lease.ErrLeaseNotOwner
				},
			}
		},
		NewExecutor: func(scope.Scope) (jsexecutor.JsExecutor, error) {
			return &noopProvisionExecutor{}, nil
		},
		InstallModule: func(ctx context.Context, runtimeScope scope.Scope, jsExecutor jsexecutor.ScriptExecutor, req lifecycle.InstallModuleRequest, opts ...lifecycle.Option) error {
			insertAuthModule(t, db, meta.Installed)
			close(started)
			time.Sleep(30 * time.Millisecond) // let renew fire after success
			return nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "lease lost") {
		t.Fatalf("error=%v", err)
	}
}

func TestProvisionMinimalFinalResolveError(t *testing.T) {
	runtimeScope, db := newSetupStateScope(t)
	createMetaModuleTable(t, db)
	createAuthAppSettingTable(t, db)
	_, err := ProvisionMinimal(context.Background(), runtimeScope, Options{
		LockerFactory: func(scope.Scope) statepkg.Locker { return &okLocker{} },
		NewExecutor: func(scope.Scope) (jsexecutor.JsExecutor, error) {
			return &noopProvisionExecutor{}, nil
		},
		InstallModule: func(context.Context, scope.Scope, jsexecutor.ScriptExecutor, lifecycle.InstallModuleRequest, ...lifecycle.Option) error {
			mustExec(t, db, `DROP TABLE meta_module`)
			mustExec(t, db, `CREATE TABLE meta_module (id TEXT PRIMARY KEY, name TEXT, status TEXT)`)
			mustExec(t, db, `INSERT INTO meta_module (id, name, status) VALUES (?, ?, ?)`,
				"mod-auth", "auth", string(meta.Installed))
			return nil
		},
	})
	if err == nil {
		t.Fatal("expected final Resolve error")
	}
}

func TestModuleInstallTimeoutFromConfig(t *testing.T) {
	runtimeScope, _ := newSetupStateScope(t)
	runtimeScope.config = &config.Config{BootstrapModuleInstallTimeoutSeconds: 42}
	if got := moduleInstallTimeout(runtimeScope); got != 42*time.Second {
		t.Fatalf("timeout=%v want 42s", got)
	}
}

func TestProvisionMinimalDefaultExecutorAndInstallWiring(t *testing.T) {
	runtimeScope, db := newSetupStateScope(t)
	createMetaModuleTable(t, db)
	createAuthAppSettingTable(t, db)
	// Nil NewExecutor uses production default (may fail create).
	_, err := ProvisionMinimal(context.Background(), runtimeScope, Options{
		InstallTimeout: 5 * time.Second,
		LockerFactory:  func(scope.Scope) statepkg.Locker { return &okLocker{} },
	})
	if err == nil {
		t.Fatal("expected default executor/install path to fail without modules")
	}

	// Nil InstallModule with a working executor covers lifecycle.InstallModule wiring.
	_, err = ProvisionMinimal(context.Background(), runtimeScope, Options{
		InstallTimeout: 5 * time.Second,
		LockerFactory:  func(scope.Scope) statepkg.Locker { return &okLocker{} },
		NewExecutor: func(scope.Scope) (jsexecutor.JsExecutor, error) {
			return &noopProvisionExecutor{}, nil
		},
	})
	if err == nil {
		t.Fatal("expected default InstallModule to fail without modules")
	}
	if !strings.Contains(err.Error(), "provision install web") {
		t.Fatalf("error=%v", err)
	}
}

func TestCheckProvisionLeaseRenewNil(t *testing.T) {
	if err := checkProvisionLeaseRenew(nil); err != nil {
		t.Fatalf("nil handle: %v", err)
	}
}

func TestAcquireProvisionLeaseNilFactoryAndMigrate(t *testing.T) {
	runtimeScope, _ := newSetupStateScope(t)
	// Migrates LockLease, then nil factory selects defaultLockerFactory.
	// Passthrough test transactor cannot run requires-new Acquire.
	_, err := acquireProvisionLease(context.Background(), runtimeScope, nil, nil)
	if err == nil {
		t.Fatal("expected default locker Acquire to fail under test scope")
	}
	if defaultLockerFactory(runtimeScope) == nil {
		t.Fatal("defaultLockerFactory returned nil")
	}
}

func TestAcquireProvisionLeaseAutoMigrateError(t *testing.T) {
	runtimeScope, db := newSetupStateScope(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	_, err = acquireProvisionLease(context.Background(), runtimeScope, func(scope.Scope) statepkg.Locker {
		return &okLocker{}
	}, nil)
	if err == nil {
		t.Fatal("expected AutoMigrate/HasTable error on closed db")
	}
}

func TestContinueAfterLeaseMigrateErr(t *testing.T) {
	if err := continueAfterLeaseMigrateErr(nil, errors.New("migrate boom"), false); err == nil {
		t.Fatal("expected error when table missing")
	}
	runtimeScope, _ := newSetupStateScope(t)
	if err := continueAfterLeaseMigrateErr(runtimeScope, errors.New("migrate boom"), true); err != nil {
		t.Fatalf("expected continue when table exists: %v", err)
	}
}

func TestNotifyLeaseRenewErr(t *testing.T) {
	notifyLeaseRenewErr(nil, errors.New("x"))
	var canceled bool
	handle := &leaseHandle{
		renewErr: make(chan error, 1),
		cancel:   func() { canceled = true },
	}
	notifyLeaseRenewErr(handle, errors.New("first"))
	notifyLeaseRenewErr(handle, errors.New("second")) // default branch: buffer full
	if !canceled {
		t.Fatal("expected cancel")
	}
	if err := <-handle.renewErr; err == nil || err.Error() != "first" {
		t.Fatalf("renewErr=%v", err)
	}
}

func TestReleaseProvisionLeaseWaitsForRenewDone(t *testing.T) {
	prevInterval := leaseRenewInterval
	prevWait := leaseReleaseWait
	leaseRenewInterval = time.Hour
	leaseReleaseWait = time.Second
	t.Cleanup(func() {
		leaseRenewInterval = prevInterval
		leaseReleaseWait = prevWait
	})

	released := make(chan struct{})
	locker := &blockingRenewLocker{renewStarted: make(chan struct{}), releaseCh: released}
	handle := &leaseHandle{
		ownerID:  "o",
		locker:   locker,
		stopCh:   make(chan struct{}),
		done:     make(chan struct{}),
		renewErr: make(chan error, 1),
	}
	go renewProvisionLease(context.Background(), locker, handle, nil)
	<-locker.renewStarted // renew is blocked inside Renew
	runtimeScope, _ := newSetupStateScope(t)
	go releaseProvisionLease(runtimeScope, nil, handle)
	select {
	case <-released:
		t.Fatal("Release must wait until renew exits")
	case <-time.After(30 * time.Millisecond):
	}
	close(locker.unblockRenew)
	select {
	case <-released:
	case <-time.After(time.Second):
		t.Fatal("Release did not finish after renew unblocked")
	}
}

func TestReleaseProvisionLeaseBoundsWaitOnHungRenew(t *testing.T) {
	prevInterval := leaseRenewInterval
	prevWait := leaseReleaseWait
	leaseRenewInterval = time.Hour
	leaseReleaseWait = 20 * time.Millisecond
	t.Cleanup(func() {
		leaseRenewInterval = prevInterval
		leaseReleaseWait = prevWait
	})

	released := make(chan struct{})
	locker := &blockingRenewLocker{renewStarted: make(chan struct{}), releaseCh: released}
	handle := &leaseHandle{
		ownerID:  "o",
		locker:   locker,
		stopCh:   make(chan struct{}),
		done:     make(chan struct{}),
		renewErr: make(chan error, 1),
	}
	go renewProvisionLease(context.Background(), locker, handle, nil)
	<-locker.renewStarted
	runtimeScope, _ := newSetupStateScope(t)
	done := make(chan struct{})
	go func() {
		releaseProvisionLease(runtimeScope, nil, handle)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("release must time out waiting for hung renew")
	}
	// Unblock renew so the background goroutine can exit cleanly.
	close(locker.unblockRenew)
	select {
	case <-handle.done:
	case <-time.After(time.Second):
		t.Fatal("renew did not exit after unblock")
	}
}

func TestAcquireProvisionLeaseMigratesThenFakeLocker(t *testing.T) {
	runtimeScope, _ := newSetupStateScope(t)
	handle, err := acquireProvisionLease(context.Background(), runtimeScope, func(scope.Scope) statepkg.Locker {
		return &okLocker{}
	}, nil)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	// nil factory on release covers defaultLockerFactory Release path.
	releaseProvisionLease(runtimeScope, nil, handle)
}

func TestReleaseProvisionLeaseNilHandle(t *testing.T) {
	releaseProvisionLease(nil, nil, nil)
}

func TestReleaseProvisionLeaseLogsFailure(t *testing.T) {
	done := make(chan struct{})
	close(done)
	runtimeScope, _ := newSetupStateScope(t)
	handle := &leaseHandle{
		ownerID:  "o",
		locker:   &errReleaseLocker{err: errors.New("release boom")},
		stopCh:   make(chan struct{}),
		done:     done,
		renewErr: make(chan error, 1),
	}
	releaseProvisionLease(runtimeScope, nil, handle)
}

func TestReleaseProvisionLeaseFallsBackToFactory(t *testing.T) {
	done := make(chan struct{})
	close(done)
	runtimeScope, _ := newSetupStateScope(t)
	released := make(chan struct{}, 1)
	handle := &leaseHandle{
		ownerID:  "o",
		// locker intentionally nil — release must rebuild via factory.
		stopCh:   make(chan struct{}),
		done:     done,
		renewErr: make(chan error, 1),
	}
	releaseProvisionLease(runtimeScope, func(scope.Scope) statepkg.Locker {
		return &okLocker{onRelease: func() { released <- struct{}{} }}
	}, handle)
	select {
	case <-released:
	default:
		t.Fatal("expected factory locker Release")
	}

	// nil locker + nil factory selects defaultLockerFactory.
	handle2 := &leaseHandle{
		ownerID:  "o2",
		stopCh:   make(chan struct{}),
		done:     done,
		renewErr: make(chan error, 1),
	}
	releaseProvisionLease(runtimeScope, nil, handle2)
}

type errReleaseLocker struct{ err error }

func (e *errReleaseLocker) Acquire(context.Context, string, string, time.Duration) error {
	return nil
}
func (e *errReleaseLocker) Renew(context.Context, string, string, time.Duration) error {
	return nil
}
func (e *errReleaseLocker) Release(context.Context, string, string) error { return e.err }

func TestCheckProvisionLeaseRenewPendingError(t *testing.T) {
	handle := &leaseHandle{stopCh: make(chan struct{}), renewErr: make(chan error, 1)}
	handle.renewErr <- errors.New("renew boom")
	if err := checkProvisionLeaseRenew(handle); err == nil || !strings.Contains(err.Error(), "renew boom") {
		t.Fatalf("error=%v", err)
	}
}

func TestRenewProvisionLeaseStopsOnContextCancel(t *testing.T) {
	prev := leaseRenewInterval
	leaseRenewInterval = time.Hour // avoid ticking
	t.Cleanup(func() { leaseRenewInterval = prev })

	ctx, cancel := context.WithCancel(context.Background())
	handle := &leaseHandle{
		stopCh:   make(chan struct{}),
		done:     make(chan struct{}),
		renewErr: make(chan error, 1),
		ownerID:  "o",
	}
	go renewProvisionLease(ctx, &okLocker{}, handle, nil)
	cancel()
	select {
	case <-handle.done:
	case <-time.After(time.Second):
		t.Fatal("renew goroutine did not stop on ctx cancel")
	}
}

func TestReleaseProvisionLeaseIdempotent(t *testing.T) {
	done := make(chan struct{})
	close(done)
	handle := &leaseHandle{stopCh: make(chan struct{}), done: done, renewErr: make(chan error, 1)}
	releaseProvisionLease(nil, nil, handle)
	releaseProvisionLease(nil, nil, handle) // must not panic
}

func TestAcquireProvisionLeaseErrors(t *testing.T) {
	if _, err := acquireProvisionLease(context.Background(), nil, nil, nil); err == nil {
		t.Fatal("expected nil scope error")
	}
	runtimeScope := &setupStateTestScope{ctx: context.Background(), db: nil}
	if _, err := acquireProvisionLease(context.Background(), runtimeScope, nil, nil); err == nil {
		t.Fatal("expected nil session error")
	}
}

type failStartExecutor struct{ noopProvisionExecutor }

func (e *failStartExecutor) Start() error { return errors.New("start failed") }

type failRenewLocker struct {
	started       chan struct{}
	renewErr      error
	failTimes     int32
	failImmediate bool
	renewCalls    atomic.Int32
	failures      atomic.Int32
}

func (f *failRenewLocker) Acquire(context.Context, string, string, time.Duration) error {
	return nil
}
func (f *failRenewLocker) Renew(context.Context, string, string, time.Duration) error {
	f.renewCalls.Add(1)
	err := f.renewErr
	if err == nil {
		err = lease.ErrLeaseNotOwner
	}
	if f.failImmediate {
		return err
	}
	if f.failTimes > 0 {
		n := f.failures.Add(1)
		if n <= f.failTimes {
			return err
		}
		return nil
	}
	if f.started != nil {
		select {
		case <-f.started:
			return err
		default:
			return nil
		}
	}
	return err
}
func (f *failRenewLocker) Release(context.Context, string, string) error { return nil }

type blockingRenewLocker struct {
	renewStarted chan struct{}
	unblockRenew chan struct{}
	releaseCh    chan struct{}
	once         sync.Once
}

func (b *blockingRenewLocker) Acquire(context.Context, string, string, time.Duration) error {
	return nil
}
func (b *blockingRenewLocker) Renew(context.Context, string, string, time.Duration) error {
	b.once.Do(func() {
		if b.unblockRenew == nil {
			b.unblockRenew = make(chan struct{})
		}
		close(b.renewStarted)
		<-b.unblockRenew
	})
	return nil
}
func (b *blockingRenewLocker) Release(context.Context, string, string) error {
	close(b.releaseCh)
	return nil
}

type countingRenewLocker struct {
	started chan struct{}
	onRenew func(n int32) error
	calls   atomic.Int32
}

func (c *countingRenewLocker) Acquire(context.Context, string, string, time.Duration) error {
	return nil
}
func (c *countingRenewLocker) Renew(context.Context, string, string, time.Duration) error {
	n := c.calls.Add(1)
	if c.onRenew != nil {
		return c.onRenew(n)
	}
	if c.started != nil {
		select {
		case <-c.started:
			return lease.ErrLeaseNotOwner
		default:
			return nil
		}
	}
	return nil
}
func (c *countingRenewLocker) Release(context.Context, string, string) error { return nil }

type errAcquireLocker struct{ err error }

func (e *errAcquireLocker) Acquire(context.Context, string, string, time.Duration) error {
	return e.err
}
func (e *errAcquireLocker) Renew(context.Context, string, string, time.Duration) error {
	return nil
}
func (e *errAcquireLocker) Release(context.Context, string, string) error { return nil }

type nilWithContextScope struct {
	setupStateTestScope
}

func (e *nilWithContextScope) WithContext(context.Context) scope.Scope { return nil }
func (e *nilWithContextScope) Session() *scope.Session {
	return e.setupStateTestScope.Session()
}
func (e *nilWithContextScope) Config() *config.Config {
	return e.setupStateTestScope.Config()
}
func (e *nilWithContextScope) FactoryInput() scope.FactoryInput {
	return e.setupStateTestScope.FactoryInput()
}
func (e *nilWithContextScope) Logger() *slog.Logger {
	return e.setupStateTestScope.Logger()
}
func (e *nilWithContextScope) Context() context.Context {
	return e.setupStateTestScope.Context()
}
func (e *nilWithContextScope) Run(fn func(scope.Scope) error) error { return fn(e) }
func (e *nilWithContextScope) Transactor() scope.Transactor {
	return e.setupStateTestScope.Transactor()
}

type okLocker struct {
	onAcquire func()
	onRelease func()
}

func (o *okLocker) Acquire(context.Context, string, string, time.Duration) error {
	if o != nil && o.onAcquire != nil {
		o.onAcquire()
	}
	return nil
}
func (o *okLocker) Renew(context.Context, string, string, time.Duration) error { return nil }
func (o *okLocker) Release(context.Context, string, string) error {
	if o != nil && o.onRelease != nil {
		o.onRelease()
	}
	return nil
}

type busyLocker struct{}

func (b *busyLocker) Acquire(context.Context, string, string, time.Duration) error {
	return ErrLeaseBusy
}
func (b *busyLocker) Renew(context.Context, string, string, time.Duration) error { return nil }
func (b *busyLocker) Release(context.Context, string, string) error              { return nil }
