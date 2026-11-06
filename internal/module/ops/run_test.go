// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/internal/module/plan"
	"github.com/choysum-dev/choysum/pkg/bus"
	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
	"gorm.io/gorm"
)

// seedLifecycle installs into meta_module so MaterializeCatalog sees Installed.
type seedLifecycle struct {
	*stubLifecycle
	db *gorm.DB
}

func (s *seedLifecycle) Install(ctx context.Context, req lifecycle.InstallRequest) error {
	if err := s.stubLifecycle.Install(ctx, req); err != nil {
		return err
	}
	return s.db.Create(&pkgmeta.Module{
		Name:           req.Name,
		Version:        "1.0.0",
		Status:         pkgmeta.Installed,
		ApplicationStr: req.Name,
	}).Error
}

func TestRun_InstallSuccessPublishesTipAndMaterializes(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	runtimeScope := newOpsTestScope(t, db)
	mod := &pkgmeta.Module{Name: "core", Version: "1.0.0", ApplicationStr: "core"}
	events := &recordingBus{}
	life := &seedLifecycle{stubLifecycle: &stubLifecycle{}, db: db}
	o := &Ops{
		Scope:     runtimeScope,
		Resolver:  &mapResolver{peek: map[string]*pkgmeta.Module{"core": mod}},
		Lifecycle: life,
		Events:    events,
		LockerFactory: func(scope.Scope) statepkg.Locker {
			return testLocker{}
		},
	}
	if err := db.Create(&modmeta.ModuleSourceSnapshot{
		ModuleName: "core",
		SourceType: originLocal,
		SourceRef:  "local",
		Available:  true,
		Version:    nullString("1.0.0"),
	}).Error; err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}

	result, err := o.Plan(context.Background(), PlanRequest{ModuleName: "core", Action: ActionInstall})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if err := o.Apply(context.Background(), ApplyRequest{Id: result.Id}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if err := o.Run(context.Background(), RunRequest{Id: result.Id}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	row, err := o.loadOperation(context.Background(), result.Id)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if row.Status != StatusSucceeded || row.Phase != PhaseDone {
		t.Fatalf("status/phase = %q/%q", row.Status, row.Phase)
	}
	if len(life.installs) != 1 || life.installs[0] != "core" {
		t.Fatalf("installs = %#v", life.installs)
	}
	if len(events.events) < 2 {
		t.Fatalf("expected tip publishes, got %d", len(events.events))
	}
	last := events.events[len(events.events)-1]
	if last.Topic != bus.TopicMetaModuleOpChanged {
		t.Fatalf("topic = %q", last.Topic)
	}
	if last.Payload["operationId"] != result.Id {
		t.Fatalf("payload = %#v", last.Payload)
	}
	var catalog modmeta.ModuleCatalog
	if err := db.Where("module_name = ?", "core").Take(&catalog).Error; err != nil {
		t.Fatalf("catalog: %v", err)
	}
	if !catalog.Installed || catalog.InstalledVersion.String != "1.0.0" {
		t.Fatalf("catalog installed=%v version=%q", catalog.Installed, catalog.InstalledVersion.String)
	}
}

func TestRun_InstallFailureWritesFailedStatus(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	mod := &pkgmeta.Module{Name: "core", Version: "1.0.0", ApplicationStr: "core"}
	o := &Ops{
		Scope:     newOpsTestScope(t, db),
		Resolver:  &mapResolver{peek: map[string]*pkgmeta.Module{"core": mod}},
		Lifecycle: &stubLifecycle{installErr: errors.New("boom")},
		Events:    &recordingBus{},
	}
	result, err := o.Plan(context.Background(), PlanRequest{ModuleName: "core", Action: ActionInstall})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if err := o.Apply(context.Background(), ApplyRequest{Id: result.Id}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	err = o.Run(context.Background(), RunRequest{Id: result.Id})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("Run error = %v, want boom", err)
	}
	row, err := o.loadOperation(context.Background(), result.Id)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if row.Status != StatusFailed {
		t.Fatalf("status = %q, want failed", row.Status)
	}
}

func TestRun_LoadDemoNotImplemented(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db), Events: &recordingBus{}}
	result, err := o.Plan(context.Background(), PlanRequest{Action: ActionLoadDemo})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	err = o.Run(context.Background(), RunRequest{Id: result.Id})
	if err == nil || !strings.Contains(err.Error(), "not implemented") {
		t.Fatalf("Run error = %v, want not implemented", err)
	}
	row, err := o.loadOperation(context.Background(), result.Id)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if row.Status != StatusFailed || row.ErrorCode != "LOAD_DEMO_NOT_IMPLEMENTED" {
		t.Fatalf("status=%q code=%q", row.Status, row.ErrorCode)
	}
}

func TestRun_RejectsDraftInstallWithoutApply(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	mod := &pkgmeta.Module{Name: "core", Version: "1.0.0", ApplicationStr: "core"}
	o := &Ops{
		Scope:     newOpsTestScope(t, db),
		Resolver:  &mapResolver{peek: map[string]*pkgmeta.Module{"core": mod}},
		Lifecycle: &stubLifecycle{},
		Events:    &recordingBus{},
	}
	result, err := o.Plan(context.Background(), PlanRequest{ModuleName: "core", Action: ActionInstall})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	err = o.Run(context.Background(), RunRequest{Id: result.Id})
	if err == nil || !strings.Contains(err.Error(), "must be applied") {
		t.Fatalf("Run error = %v, want must be applied", err)
	}
}

func TestRun_WithDemoRequestSetsLatch(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	mod := &pkgmeta.Module{Name: "core", Version: "1.0.0", ApplicationStr: "core"}
	life := &stubLifecycle{}
	o := &Ops{
		Scope:     newOpsTestScope(t, db),
		Resolver:  &mapResolver{peek: map[string]*pkgmeta.Module{"core": mod}},
		Lifecycle: life,
		Events:    &recordingBus{},
	}
	result, err := o.Plan(context.Background(), PlanRequest{ModuleName: "core", Action: ActionInstall})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if err := o.Apply(context.Background(), ApplyRequest{Id: result.Id}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if err := o.Run(context.Background(), RunRequest{Id: result.Id, WithDemo: true}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !life.lastWithDemo {
		t.Fatal("expected WithDemo=true on lifecycle install")
	}
	if !o.ResolveWithDemo(context.Background()) {
		t.Fatal("expected modules.with_demo latch set")
	}
}

func TestRun_PassesFrozenLifecycleOptions(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	mod := &pkgmeta.Module{Name: "core", Version: "1.0.0", ApplicationStr: "core"}
	life := &stubLifecycle{}
	o := &Ops{
		Scope:     newOpsTestScope(t, db),
		Resolver:  &mapResolver{peek: map[string]*pkgmeta.Module{"core": mod}},
		Lifecycle: life,
		Events:    &recordingBus{},
	}
	result, err := o.Plan(context.Background(), PlanRequest{
		ModuleName:      "core",
		Action:          ActionUpgrade,
		SkipWebShell:    true,
		WebShellProfile: string(plan.WebShellProfileMinimal),
		LifecycleInput:  "core@9.9.9",
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if err := o.Apply(context.Background(), ApplyRequest{Id: result.Id}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if err := o.Run(context.Background(), RunRequest{Id: result.Id}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(life.upgrades) != 1 || life.upgrades[0] != "core@9.9.9" {
		t.Fatalf("upgrades = %#v", life.upgrades)
	}
	if !life.lastSkipWebShell || life.lastWebShellProfile != plan.WebShellProfileMinimal {
		t.Fatalf("skip=%v profile=%q", life.lastSkipWebShell, life.lastWebShellProfile)
	}
}
