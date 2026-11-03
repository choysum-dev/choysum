// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package provision

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	"github.com/choysum-dev/choysum/internal/module/plan"
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

func TestProvisionMinimalRejectsReadyAfterInstall(t *testing.T) {
	runtimeScope, db := newSetupStateScope(t)
	createMetaModuleTable(t, db)
	createAuthAppSettingTable(t, db)
	_, err := ProvisionMinimal(context.Background(), runtimeScope, Options{
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
	if err == nil || !strings.Contains(err.Error(), "must not mark setup complete") {
		t.Fatalf("error=%v", err)
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

type okLocker struct{}

func (o *okLocker) Acquire(context.Context, string, string, time.Duration) error {
	return nil
}
func (o *okLocker) Renew(context.Context, string, string, time.Duration) error { return nil }
func (o *okLocker) Release(context.Context, string, string) error              { return nil }

type busyLocker struct{}

func (b *busyLocker) Acquire(context.Context, string, string, time.Duration) error {
	return ErrLeaseBusy
}
func (b *busyLocker) Renew(context.Context, string, string, time.Duration) error { return nil }
func (b *busyLocker) Release(context.Context, string, string) error              { return nil }
