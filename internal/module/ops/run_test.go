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
	if row.Status != StatusSucceeded || row.Phase != PhaseReloading || !row.ReloadTriggered || row.ReloadFailed {
		t.Fatalf("status/phase/reload = %q/%q triggered=%v failed=%v", row.Status, row.Phase, row.ReloadTriggered, row.ReloadFailed)
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

func TestRun_LoadDemoEmptyOrderSucceedsWithoutLatch(t *testing.T) {
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db), Events: &recordingBus{}}
	result, err := o.Plan(context.Background(), PlanRequest{Action: ActionLoadDemo})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	// Empty pending plans carry NO_DEMO_PENDING blockers; CLI/bridge Run draft path skips Apply.
	if err := o.Run(context.Background(), RunRequest{Id: result.Id}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if o.ResolveWithDemo(context.Background()) {
		t.Fatal("empty load-demo must not set modules.with_demo")
	}
	row, err := o.loadOperation(context.Background(), result.Id)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if row.Status != StatusSucceeded {
		t.Fatalf("status=%q", row.Status)
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

func TestAbandonDraft_CancelsDraftRow(t *testing.T) {
	t.Parallel()
	if err := (*Ops)(nil).AbandonDraft(context.Background(), "x", ""); err == nil {
		t.Fatal("expected nil host error")
	}
	db := newOpsTestDB(t)
	mod := &pkgmeta.Module{Name: "core", Version: "1.0.0", ApplicationStr: "core"}
	o := &Ops{
		Scope:     newOpsTestScope(t, db),
		Resolver:  &mapResolver{peek: map[string]*pkgmeta.Module{"core": mod}},
		Lifecycle: &stubLifecycle{},
		Events:    &recordingBus{},
	}
	if err := o.AbandonDraft(nil, "  ", ""); err == nil || !strings.Contains(err.Error(), "id is required") {
		t.Fatalf("empty id = %v", err)
	}
	if err := o.AbandonDraft(context.Background(), "missing-op", ""); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("missing id = %v", err)
	}
	result, err := o.Plan(context.Background(), PlanRequest{ModuleName: "core", Action: ActionInstall})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if err := o.AbandonDraft(nil, result.Id, "plan blocked"); err != nil {
		t.Fatalf("AbandonDraft: %v", err)
	}
	row, err := o.loadOperation(context.Background(), result.Id)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if row.Status != StatusCancelled || row.ErrorCode != "ABANDONED" {
		t.Fatalf("status=%q code=%q", row.Status, row.ErrorCode)
	}
	// Non-draft is a no-op.
	if err := o.AbandonDraft(context.Background(), result.Id, "again"); err != nil {
		t.Fatalf("AbandonDraft non-draft: %v", err)
	}

	// CAS/update failure while still draft.
	db2 := newOpsTestDB(t)
	o2 := &Ops{
		Scope:     newOpsTestScope(t, db2),
		Resolver:  &mapResolver{peek: map[string]*pkgmeta.Module{"core": mod}},
		Lifecycle: &stubLifecycle{},
		Events:    &recordingBus{},
	}
	res2, err := o2.Plan(context.Background(), PlanRequest{ModuleName: "core", Action: ActionInstall})
	if err != nil {
		t.Fatalf("Plan2: %v", err)
	}
	if err := db2.Callback().Update().Before("gorm:update").Register("ops_fail_abandon_update", func(tx *gorm.DB) {
		if tx.Statement == nil {
			return
		}
		if st, ok := tx.Statement.Dest.(map[string]any); ok {
			if status, _ := st["status"].(string); status == StatusCancelled {
				_ = tx.AddError(errors.New("abandon save fail"))
			}
		}
	}); err != nil {
		t.Fatalf("callback: %v", err)
	}
	if err := o2.AbandonDraft(context.Background(), res2.Id, "blocked"); err == nil || !strings.Contains(err.Error(), "abandon save fail") {
		t.Fatalf("AbandonDraft CAS err = %v", err)
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

func TestForceFailTerminalAndMarkReloadFailed(t *testing.T) {
	t.Parallel()

	var nilOps *Ops
	if err := nilOps.ForceFailTerminal(nil, "x", "d", "c", "m"); err == nil || !strings.Contains(err.Error(), "nil host") {
		t.Fatalf("ForceFailTerminal(nil) = %v", err)
	}
	if err := nilOps.MarkReloadFailed(nil, "x"); err == nil || !strings.Contains(err.Error(), "nil host") {
		t.Fatalf("MarkReloadFailed(nil) = %v", err)
	}

	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db), Events: &recordingBus{}}
	if err := o.ForceFailTerminal(nil, " ", "d", "c", "m"); err == nil || !strings.Contains(err.Error(), "operation id") {
		t.Fatalf("ForceFailTerminal(empty id) = %v", err)
	}
	if err := o.MarkReloadFailed(context.Background(), ""); err == nil || !strings.Contains(err.Error(), "operation id") {
		t.Fatalf("MarkReloadFailed(empty id) = %v", err)
	}
	if err := o.ForceFailTerminal(context.Background(), "missing", "d", "c", "m"); err == nil {
		t.Fatal("ForceFailTerminal(missing) expected load error")
	}
	if err := o.MarkReloadFailed(context.Background(), "missing"); err == nil {
		t.Fatal("MarkReloadFailed(missing) expected load error")
	}

	mod := &pkgmeta.Module{Name: "core", Version: "1.0.0", ApplicationStr: "core"}
	life := &seedLifecycle{stubLifecycle: &stubLifecycle{}, db: db}
	o = &Ops{
		Scope:     newOpsTestScope(t, db),
		Resolver:  &mapResolver{peek: map[string]*pkgmeta.Module{"core": mod}},
		Lifecycle: life,
		Events:    &recordingBus{},
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

	if err := o.ForceFailTerminal(nil, result.Id, "MODULE_MANAGEMENT", "OP_FAILED", "forced"); err != nil {
		t.Fatalf("ForceFailTerminal: %v", err)
	}
	row, err := o.loadOperation(context.Background(), result.Id)
	if err != nil {
		t.Fatalf("load after ForceFailTerminal: %v", err)
	}
	if row.Status != StatusFailed || row.ErrorCode != "OP_FAILED" || row.ErrorMessage != "forced" {
		t.Fatalf("forced row = %#v", row)
	}

	// Succeed again via direct CAS so MarkReloadFailed has a succeeded row.
	row.Status = StatusSucceeded
	row.Phase = PhaseDone
	row.ErrorDomain = ""
	row.ErrorCode = ""
	row.ErrorMessage = ""
	if err := o.casSaveOperation(context.Background(), row, StatusFailed); err != nil {
		t.Fatalf("restore succeeded: %v", err)
	}
	if err := o.MarkReloadFailed(nil, result.Id); err != nil {
		t.Fatalf("MarkReloadFailed: %v", err)
	}
	row, err = o.loadOperation(context.Background(), result.Id)
	if err != nil {
		t.Fatalf("load after MarkReloadFailed: %v", err)
	}
	if !row.ReloadTriggered || !row.ReloadFailed || row.Status != StatusSucceeded {
		t.Fatalf("reload-failed row = %#v", row)
	}

	const failCAS = "block-ops-cas"
	if err := db.Callback().Update().Before("gorm:update").Register(failCAS, func(tx *gorm.DB) {
		_ = tx.AddError(errors.New("cas blocked"))
	}); err != nil {
		t.Fatalf("register cas block callback: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Callback().Update().Remove(failCAS)
	})
	if err := o.ForceFailTerminal(context.Background(), result.Id, "d", "c", "m"); err == nil || !strings.Contains(err.Error(), "cas blocked") {
		t.Fatalf("ForceFailTerminal CAS err = %v", err)
	}
	if err := o.MarkReloadFailed(context.Background(), result.Id); err == nil || !strings.Contains(err.Error(), "cas blocked") {
		t.Fatalf("MarkReloadFailed CAS err = %v", err)
	}
}
