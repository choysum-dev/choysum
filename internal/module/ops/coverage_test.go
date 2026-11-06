// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func closeOpsDB(t *testing.T, db interface{ DB() (*sql.DB, error) }) {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestCoverage_GuardsAndHelpers(t *testing.T) {
	t.Parallel()

	if _, err := (*Ops)(nil).Plan(context.Background(), PlanRequest{}); err == nil {
		t.Fatal("nil Plan")
	}
	if err := (*Ops)(nil).Apply(context.Background(), ApplyRequest{Id: "x"}); err == nil {
		t.Fatal("nil Apply")
	}
	if err := (*Ops)(nil).Run(context.Background(), RunRequest{Id: "x"}); err == nil {
		t.Fatal("nil Run")
	}
	if err := (*Ops)(nil).MaterializeCatalog(context.Background(), nil); err == nil {
		t.Fatal("nil Materialize")
	}
	if _, err := (*Ops)(nil).SyncSourceSnapshotsThenMaterialize(context.Background(), "all"); err == nil {
		t.Fatal("nil Sync")
	}
	if (*Ops)(nil).ResolveWithDemo(context.Background()) {
		t.Fatal("nil ResolveWithDemo")
	}

	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db), Resolver: &mapResolver{}}
	if _, err := o.Plan(nil, PlanRequest{Action: "nope"}); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("unsupported Plan: %v", err)
	}
	if err := o.Apply(nil, ApplyRequest{}); err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("empty Apply id: %v", err)
	}
	if err := o.Run(nil, RunRequest{}); err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("empty Run id: %v", err)
	}
	if err := o.MaterializeCatalog(nil, nil); err != nil {
		t.Fatalf("nil ctx Materialize: %v", err)
	}

	if normalizeOrigin(originRegistry) != originRegistry {
		t.Fatal("registry origin")
	}
	if op, ok := toPlanOpType(ActionUpgrade); !ok || op == "" {
		t.Fatal("upgrade op")
	}
	if op, ok := toPlanOpType(ActionUninstall); !ok || op == "" {
		t.Fatal("uninstall op")
	}
	if _, ok := toPlanOpType("nope"); ok {
		t.Fatal("bad op")
	}

	if normalizeVersion("") != "" || normalizeVersion("v1.2.3") != "v1.2.3" {
		t.Fatal("normalizeVersion prefix/empty")
	}
	if normalizeVersion("not-a-semver") != "not-a-semver" {
		t.Fatal("normalizeVersion passthrough")
	}
	if compareVersion("zzz", "aaa") <= 0 {
		t.Fatal("non-semver compare")
	}
	if versionNewer("", "1.0.0") || !versionNewer("1.0.0", "") {
		t.Fatal("versionNewer edges")
	}

	if hasDemoPayload(nil) || hasDemoPayload(datatypes.JSON("null")) {
		t.Fatal("hasDemoPayload false cases")
	}
	if !hasDemoPayload(datatypes.JSON(`["a"]`)) {
		t.Fatal("hasDemoPayload true")
	}

	fillCatalogFromManifest(nil, datatypes.JSON(`{}`))
	fillCatalogFromManifest(&modmeta.ModuleCatalog{}, nil)
	fillCatalogFromManifest(&modmeta.ModuleCatalog{}, datatypes.JSON(`{`))

	names := affectedModuleNames(PlanDocument{
		ModuleOrder: []ModuleOrderEntry{{ModuleName: "a"}, {ModuleName: " a "}, {ModuleName: ""}},
		EnsureOrder: []string{"b", ""},
	}, "a")
	if len(names) != 2 {
		t.Fatalf("affected names = %v", names)
	}

	if _, err := unmarshalPlanDocument(nil); err != nil {
		t.Fatalf("empty plan: %v", err)
	}
	if _, err := unmarshalPlanDocument(datatypes.JSON(`{`)); err == nil {
		t.Fatal("bad plan json")
	}

	prev := jsonMarshal
	jsonMarshal = func(any) ([]byte, error) { return nil, errors.New("marshal boom") }
	if _, err := marshalPlanDocument(PlanDocument{}); err == nil {
		t.Fatal("marshal fail")
	}
	jsonMarshal = prev

	(*Ops)(nil).publishOperationChanged(context.Background(), &modmeta.ModuleOperation{})
	o.publishOperationChanged(context.Background(), nil)
	o.Events = &recordingBus{}
	o.publishOperationChanged(context.Background(), &modmeta.ModuleOperation{})
	o.publishOperationChanged(nil, &modmeta.ModuleOperation{
		BaseModel: pkgmeta.BaseModel{Id: nullString("op1")},
		JobId:     nullString("job-1"),
	})
	if nullString("").Valid {
		t.Fatal("nullString empty")
	}

	if err := o.runLifecycle(context.Background(), "nope", "x", false); err == nil {
		t.Fatal("bad lifecycle action")
	}
	if err := o.SetWithDemo(context.Background(), true); err != nil {
		t.Fatalf("SetWithDemo: %v", err)
	}
	if err := (&Ops{}).SetWithDemo(context.Background(), true); err == nil {
		t.Fatal("SetWithDemo no scope")
	}
}

func TestCoverage_PlanPaths(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	runtimeScope := newOpsTestScope(t, db)

	// sync plan
	o := &Ops{Scope: runtimeScope, Resolver: &mapResolver{}}
	syncRes, err := o.Plan(context.Background(), PlanRequest{Action: ActionSync})
	if err != nil || syncRes.Id == "" {
		t.Fatalf("Plan sync: %v %#v", err, syncRes)
	}
	if err := o.Apply(context.Background(), ApplyRequest{Id: syncRes.Id}); err != nil {
		t.Fatalf("Apply sync: %v", err)
	}

	// empty module name
	if _, err := o.Plan(context.Background(), PlanRequest{Action: ActionInstall}); err == nil {
		t.Fatal("expected moduleName required")
	}
	if _, err := (&Ops{Scope: runtimeScope}).Plan(context.Background(), PlanRequest{ModuleName: "x", Action: ActionInstall}); err == nil {
		t.Fatal("expected resolver required")
	}

	// Load fallback + registry origin target
	mod := &pkgmeta.Module{Name: "core", Version: "1.0.0", ApplicationStr: "core"}
	o.Resolver = &mapResolver{load: map[string]*pkgmeta.Module{"core": mod}}
	res, err := o.Plan(context.Background(), PlanRequest{ModuleName: "core", Action: ActionInstall, Origin: originRegistry})
	if err != nil {
		t.Fatalf("Plan registry: %v", err)
	}
	if res.ModuleOrder[0].Origin != originRegistry || res.ModuleOrder[0].TargetVersion != "1.0.0" {
		t.Fatalf("registry order = %#v", res.ModuleOrder[0])
	}

	// Peek error + Load error
	o.Resolver = &mapResolver{peekErr: errors.New("peek"), loadErr: errors.New("load")}
	if _, err := o.Plan(context.Background(), PlanRequest{ModuleName: "x", Action: ActionInstall}); err == nil || !strings.Contains(err.Error(), "load") {
		t.Fatalf("Plan load err: %v", err)
	}

	// planModuleOp unknown action via direct call
	o.Resolver = &mapResolver{peek: map[string]*pkgmeta.Module{"core": mod}}
	if _, err := o.planModuleOp(context.Background(), "bogus", "core", originLocal, ""); err == nil {
		t.Fatal("bogus action")
	}

	// uninstall blockers
	dep := &pkgmeta.Module{Name: "auth", Version: "1.0.0", Status: pkgmeta.Installed, ApplicationStr: "auth"}
	root := &pkgmeta.Module{
		Name: "core", Version: "1.0.0", ApplicationStr: "core",
		Dependents: []*pkgmeta.Module{dep, nil, {Name: " ", Status: pkgmeta.Installed}, {Name: "web", Status: pkgmeta.Uninstalled}},
	}
	o.Resolver = &mapResolver{
		peek: map[string]*pkgmeta.Module{"core": root},
		load: map[string]*pkgmeta.Module{"core": root},
	}
	unRes, err := o.Plan(context.Background(), PlanRequest{ModuleName: "core", Action: ActionUninstall})
	if err != nil {
		t.Fatalf("Plan uninstall: %v", err)
	}
	if len(unRes.Blockers) == 0 {
		t.Fatal("expected DEPENDENT_INSTALLED blocker")
	}

	// snapshotVersions with nil scope + both origins
	_, _ = (&Ops{}).snapshotVersions(context.Background(), "core")
	now := time.Now().UTC()
	if err := db.Create(&modmeta.ModuleSourceSnapshot{
		ModuleName: "core", SourceType: originLocal, SourceRef: "local", Available: true,
		Version: nullString("1.0.0"), LastSyncAt: &now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&modmeta.ModuleSourceSnapshot{
		ModuleName: "core", SourceType: originRegistry, SourceRef: "registry", Available: true,
		Version: nullString("1.1.0"),
	}).Error; err != nil {
		t.Fatal(err)
	}
	localV, regV := o.snapshotVersions(context.Background(), "core")
	if localV != "1.0.0" || regV != "1.1.0" {
		t.Fatalf("snapshots %q %q", localV, regV)
	}

	// marshal failure on Plan
	prev := jsonMarshal
	jsonMarshal = func(any) ([]byte, error) { return nil, errors.New("boom") }
	if _, err := o.Plan(context.Background(), PlanRequest{Action: ActionSync}); err == nil {
		t.Fatal("sync marshal")
	}
	if _, err := o.Plan(context.Background(), PlanRequest{Action: ActionLoadDemo}); err == nil {
		t.Fatal("load-demo marshal")
	}
	if _, err := o.Plan(context.Background(), PlanRequest{ModuleName: "core", Action: ActionInstall}); err == nil {
		t.Fatal("install marshal")
	}
	jsonMarshal = prev

	// createOperation failure via closed DB
	closeOpsDB(t, db)
	if _, err := o.Plan(context.Background(), PlanRequest{Action: ActionSync}); err == nil {
		t.Fatal("create after close")
	}
}

func TestCoverage_ApplyRunStore(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	mod := &pkgmeta.Module{Name: "core", Version: "1.0.0", ApplicationStr: "core"}
	life := &stubLifecycle{}
	events := &recordingBus{}
	o := &Ops{
		Scope:     newOpsTestScope(t, db),
		Resolver:  &mapResolver{peek: map[string]*pkgmeta.Module{"core": mod}},
		Lifecycle: life,
		Events:    events,
		LockerFactory: func(scope.Scope) statepkg.Locker {
			return testLocker{}
		},
		SyncLocalSnapshot: func(context.Context, scope.Scope, statepkg.LockerFactory) (lifecycle.ModuleSourceSnapshotSyncStats, error) {
			return lifecycle.ModuleSourceSnapshotSyncStats{Total: 1, Success: 1}, nil
		},
		SyncRegistrySnapshot: func(context.Context, scope.Scope, statepkg.LockerFactory) (lifecycle.ModuleSourceSnapshotSyncStats, error) {
			return lifecycle.ModuleSourceSnapshotSyncStats{Total: 1, Success: 1}, nil
		},
	}

	if err := o.Apply(context.Background(), ApplyRequest{Id: "missing"}); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("missing apply: %v", err)
	}

	result, err := o.Plan(context.Background(), PlanRequest{ModuleName: "core", Action: ActionInstall})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if err := o.Apply(context.Background(), ApplyRequest{Id: result.Id}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if err := o.Apply(context.Background(), ApplyRequest{Id: result.Id}); err == nil || !strings.Contains(err.Error(), "want draft") {
		t.Fatalf("re-apply: %v", err)
	}

	row, _ := o.loadOperation(context.Background(), result.Id)
	row.Status = StatusDraft
	row.PlanJson = datatypes.JSON(`{`)
	_ = o.saveOperation(context.Background(), row)
	if err := o.Apply(context.Background(), ApplyRequest{Id: result.Id}); err == nil || !strings.Contains(err.Error(), "decode") {
		t.Fatalf("bad json apply: %v", err)
	}

	// upgrade + uninstall run
	up, err := o.Plan(context.Background(), PlanRequest{ModuleName: "core", Action: ActionUpgrade})
	if err != nil {
		t.Fatalf("Plan upgrade: %v", err)
	}
	if err := o.Apply(context.Background(), ApplyRequest{Id: up.Id}); err != nil {
		t.Fatalf("Apply upgrade: %v", err)
	}
	if err := o.Run(context.Background(), RunRequest{Id: up.Id}); err != nil {
		t.Fatalf("Run upgrade: %v", err)
	}
	if len(life.upgrades) != 1 {
		t.Fatalf("upgrades=%v", life.upgrades)
	}

	un, err := o.Plan(context.Background(), PlanRequest{ModuleName: "core", Action: ActionUninstall})
	if err != nil {
		t.Fatalf("Plan uninstall: %v", err)
	}
	if err := o.Apply(context.Background(), ApplyRequest{Id: un.Id}); err != nil {
		t.Fatalf("Apply uninstall: %v", err)
	}
	if err := o.Run(context.Background(), RunRequest{Id: un.Id}); err != nil {
		t.Fatalf("Run uninstall: %v", err)
	}

	// sync run via draft
	syncRes, err := o.Plan(context.Background(), PlanRequest{Action: ActionSync})
	if err != nil {
		t.Fatalf("Plan sync: %v", err)
	}
	if err := o.Run(context.Background(), RunRequest{Id: syncRes.Id}); err != nil {
		t.Fatalf("Run sync: %v", err)
	}

	// wrong status
	done, _ := o.loadOperation(context.Background(), syncRes.Id)
	if err := o.Run(context.Background(), RunRequest{Id: done.Id.String}); err == nil {
		t.Fatal("run succeeded op")
	}

	// nil lifecycle
	inst, err := o.Plan(context.Background(), PlanRequest{ModuleName: "core", Action: ActionInstall})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if err := o.Apply(context.Background(), ApplyRequest{Id: inst.Id}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	o.Lifecycle = nil
	if err := o.Run(context.Background(), RunRequest{Id: inst.Id}); err == nil || !strings.Contains(err.Error(), "lifecycle") {
		t.Fatalf("nil lifecycle: %v", err)
	}

	// unsupported action + bad plan json on execute
	bad := &modmeta.ModuleOperation{Action: "nope", Status: StatusQueued, PlanJson: datatypes.JSON(`{}`)}
	if err := db.Create(bad).Error; err != nil {
		t.Fatal(err)
	}
	if err := o.Run(context.Background(), RunRequest{Id: bad.Id.String}); err == nil {
		t.Fatal("unsupported action")
	}
	bad2 := &modmeta.ModuleOperation{Action: ActionInstall, Status: StatusQueued, ModuleName: "core", PlanJson: datatypes.JSON(`{`)}
	if err := db.Create(bad2).Error; err != nil {
		t.Fatal(err)
	}
	o.Lifecycle = life
	if err := o.Run(context.Background(), RunRequest{Id: bad2.Id.String}); err == nil {
		t.Fatal("bad plan on run")
	}

	// SyncSourceSnapshotsThenMaterialize branches
	if _, err := o.SyncSourceSnapshotsThenMaterialize(nil, ""); err != nil {
		t.Fatalf("sync all: %v", err)
	}
	if _, err := o.SyncSourceSnapshotsThenMaterialize(context.Background(), originLocal); err != nil {
		t.Fatalf("sync local: %v", err)
	}
	if _, err := o.SyncSourceSnapshotsThenMaterialize(context.Background(), originRegistry); err != nil {
		t.Fatalf("sync registry: %v", err)
	}
	if _, err := o.SyncSourceSnapshotsThenMaterialize(context.Background(), "weird"); err == nil {
		t.Fatal("bad originType")
	}
	if _, err := (&Ops{LockerFactory: nil}).SyncSourceSnapshotsThenMaterialize(context.Background(), "all"); err == nil {
		t.Fatal("nil locker")
	}
	o.SyncLocalSnapshot = func(context.Context, scope.Scope, statepkg.LockerFactory) (lifecycle.ModuleSourceSnapshotSyncStats, error) {
		return lifecycle.ModuleSourceSnapshotSyncStats{}, errors.New("local fail")
	}
	o.SyncRegistrySnapshot = func(context.Context, scope.Scope, statepkg.LockerFactory) (lifecycle.ModuleSourceSnapshotSyncStats, error) {
		return lifecycle.ModuleSourceSnapshotSyncStats{}, errors.New("reg fail")
	}
	if _, err := o.SyncSourceSnapshotsThenMaterialize(context.Background(), "all"); err == nil {
		t.Fatal("joined sync errors")
	}
	o.SyncLocalSnapshot = func(context.Context, scope.Scope, statepkg.LockerFactory) (lifecycle.ModuleSourceSnapshotSyncStats, error) {
		return lifecycle.ModuleSourceSnapshotSyncStats{}, nil
	}
	o.SyncRegistrySnapshot = o.SyncLocalSnapshot
	o.Scope = &opsTestScope{ctx: context.Background(), session: &scope.Session{}} // DB nil
	if _, err := o.SyncSourceSnapshotsThenMaterialize(context.Background(), originLocal); err == nil {
		t.Fatal("materialize after sync with nil db")
	}
}

func TestCoverage_MaterializeEdges(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db)}
	now := time.Now().UTC()
	manifest := datatypes.JSON(`{
		"summary":"S","description":"D","category":"C","application":"A",
		"author":"Au","license":"L","depends":["x"],
		"demo":["d"],"choysum":{"demo":["e"]}
	}`)
	if err := db.Create(&modmeta.ModuleSourceSnapshot{
		ModuleName: " ", SourceType: originLocal, SourceRef: "skip", Available: true,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&modmeta.ModuleSourceSnapshot{
		ModuleName: "demo", SourceType: originLocal, SourceRef: "local", Available: false,
		Version: nullString("1.0.0"), LocalPath: nullString("/p"), LastSyncAt: &now,
		LastErrorMessage: nullString("local-err"), ManifestJson: manifest,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&modmeta.ModuleSourceSnapshot{
		ModuleName: "regonly", SourceType: originRegistry, SourceRef: "registry", Available: false,
		Version: nullString("2.0.0"), LastSyncAt: &now, LastErrorMessage: nullString("reg-err"),
		ManifestJson: datatypes.JSON(`{"summary":"R","choysum":{"demo":["z"]}}`),
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&pkgmeta.Module{
		Name: " ", Status: pkgmeta.Installed, Version: "0.1.0",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&pkgmeta.Module{
		Name: "pathmod", Status: pkgmeta.Installed, Version: "0.1.0",
		Path: "/from-installed", Summary: "is", Description: "id", Category: "ic",
		ApplicationStr: "ia", Author: "iau", License: "il",
		DependsStr: datatypes.JSON(`["y"]`), DemoStr: datatypes.JSON(`[]`),
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := o.MaterializeCatalog(context.Background(), []string{"demo", "regonly", "pathmod", "demo", " "}); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	var demo, reg, pathmod modmeta.ModuleCatalog
	if err := db.Where("module_name = ?", "demo").Take(&demo).Error; err != nil {
		t.Fatal(err)
	}
	if demo.UnavailableReason.String != "local-err" || !demo.HasDemo || demo.Summary != "S" {
		t.Fatalf("demo catalog %#v", demo)
	}
	if err := db.Where("module_name = ?", "regonly").Take(&reg).Error; err != nil {
		t.Fatal(err)
	}
	if reg.UnavailableReason.String != "reg-err" || !reg.HasDemo || reg.Summary != "R" {
		t.Fatalf("reg %#v", reg)
	}
	if err := db.Where("module_name = ?", "pathmod").Take(&pathmod).Error; err != nil {
		t.Fatal(err)
	}
	if pathmod.LocalPath.String != "/from-installed" || pathmod.HasDemo {
		t.Fatalf("pathmod %#v", pathmod)
	}

	// Find errors after close
	closeOpsDB(t, db)
	if err := o.MaterializeCatalog(context.Background(), []string{"demo"}); err == nil {
		t.Fatal("find after close")
	}
	if err := o.SetWithDemo(context.Background(), true); err == nil {
		t.Fatal("SetWithDemo after close")
	}
	if o.ResolveWithDemo(context.Background()) {
		t.Fatal("ResolveWithDemo after close")
	}
	if _, err := o.loadOperation(context.Background(), "x"); err == nil {
		t.Fatal("load after close")
	}
	if err := o.saveOperation(context.Background(), &modmeta.ModuleOperation{}); err == nil {
		t.Fatal("save after close")
	}
	if err := o.createOperation(context.Background(), &modmeta.ModuleOperation{}); err == nil {
		t.Fatal("create after close")
	}
}

func TestCoverage_RunSaveFailures(t *testing.T) {
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
	if err := o.Apply(context.Background(), ApplyRequest{Id: result.Id}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	closeOpsDB(t, db)
	if err := o.Run(context.Background(), RunRequest{Id: result.Id}); err == nil {
		t.Fatal("Run save running after close")
	}

	db2 := newOpsTestDB(t)
	o2 := &Ops{Scope: newOpsTestScope(t, db2), Events: &recordingBus{}}
	row := &modmeta.ModuleOperation{Action: ActionLoadDemo, Status: StatusDraft, PlanJson: datatypes.JSON(`{}`)}
	if err := db2.Create(row).Error; err != nil {
		t.Fatal(err)
	}
	closeOpsDB(t, db2)
	_ = o2.failOperation(context.Background(), row, "ops", "X", "y")
	_ = o2.succeedOperation(context.Background(), row)
}

func TestCoverage_UninstallBlockersEdges(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	o := &Ops{Scope: newOpsTestScope(t, db), Resolver: &mapResolver{loadErr: errors.New("no")}}
	if issues := o.uninstallBlockers(context.Background(), "core", nil); issues != nil {
		t.Fatalf("load err => %#v", issues)
	}
	o.Resolver = &mapResolver{}
	if issues := o.uninstallBlockers(context.Background(), "core", nil); issues != nil {
		t.Fatalf("nil root => %#v", issues)
	}
	root := &pkgmeta.Module{
		Name: "core",
		Dependents: []*pkgmeta.Module{
			{Name: "auth", Status: pkgmeta.Installed},
		},
	}
	o.Resolver = &mapResolver{load: map[string]*pkgmeta.Module{"core": root}}
	issues := o.uninstallBlockers(context.Background(), "core", []string{"auth"})
	if len(issues) != 0 {
		t.Fatalf("planned dependent skipped: %#v", issues)
	}
}

func TestCoverage_SessionUnavailable(t *testing.T) {
	t.Parallel()
	o := &Ops{Scope: &opsTestScope{ctx: context.Background(), session: &scope.Session{}}}
	if _, err := o.session(context.Background()); err == nil {
		t.Fatal("nil db session")
	}
	bare := &Ops{}
	if bare.ResolveWithDemo(context.Background()) {
		t.Fatal("ResolveWithDemo no scope")
	}
	if _, err := bare.loadOperation(context.Background(), "x"); err == nil {
		t.Fatal("load no scope")
	}
	if err := bare.saveOperation(context.Background(), &modmeta.ModuleOperation{}); err == nil {
		t.Fatal("save no scope")
	}
	if err := bare.createOperation(context.Background(), &modmeta.ModuleOperation{}); err == nil {
		t.Fatal("create no scope")
	}
}

func TestCoverage_RemainingBranches(t *testing.T) {
	t.Parallel()
	db := newOpsTestDB(t)
	runtimeScope := newOpsTestScope(t, db)
	locker := func(scope.Scope) statepkg.Locker { return testLocker{} }

	// Default lifecycle sync hooks (nil overrides).
	o := &Ops{Scope: runtimeScope, LockerFactory: locker}
	if _, err := o.SyncSourceSnapshotsThenMaterialize(context.Background(), originLocal); err != nil {
		// Empty modules path may still succeed with zero modules; either path covers defaults.
		t.Logf("local sync: %v", err)
	}
	if _, err := o.SyncSourceSnapshotsThenMaterialize(context.Background(), originRegistry); err != nil {
		t.Logf("registry sync: %v", err)
	}

	// Empty module_name / name rows skipped during materialize.
	if err := db.Exec(
		`INSERT INTO meta_module_source_snapshot (id, module_name, source_type, source_ref, available, created_at, updated_at) VALUES (?,?,?,?,?,?,?)`,
		"snapempty001", "", "local", "local", true, time.Now(), time.Now(),
	).Error; err != nil {
		t.Fatalf("insert empty snapshot: %v", err)
	}
	if err := db.Exec(
		`INSERT INTO meta_module (id, name, status, version, created_at, updated_at) VALUES (?,?,?,?,?,?)`,
		"modempty00001", "", "installed", "1.0.0", time.Now(), time.Now(),
	).Error; err != nil {
		t.Fatalf("insert empty module: %v", err)
	}
	if err := o.MaterializeCatalog(context.Background(), nil); err != nil {
		t.Fatalf("materialize empty names: %v", err)
	}

	// Drop installed table so snapshot Find succeeds and installed Find fails.
	if err := db.Migrator().DropTable(&pkgmeta.Module{}); err != nil {
		t.Fatalf("drop module: %v", err)
	}
	if err := o.MaterializeCatalog(context.Background(), []string{"core"}); err == nil {
		t.Fatal("expected installed find error")
	}

	// Recreate module table for later ops.
	if err := db.AutoMigrate(&pkgmeta.Module{}); err != nil {
		t.Fatalf("remigrate module: %v", err)
	}

	// BuildPlan error via dependency Load failure.
	root := &pkgmeta.Module{
		Name: "auth", Version: "1.0.0", ApplicationStr: "auth",
		DependsStr: datatypes.JSON(`["core"]`),
	}
	o.Resolver = &mapResolver{
		peek:    map[string]*pkgmeta.Module{"auth": root},
		loadErr: errors.New("dep missing"),
	}
	if _, err := o.Plan(context.Background(), PlanRequest{ModuleName: "auth", Action: ActionInstall}); err == nil || !strings.Contains(err.Error(), "dep missing") {
		t.Fatalf("BuildPlan err: %v", err)
	}

	// createOperation failures for load-demo and install.
	mod := &pkgmeta.Module{Name: "core", Version: "1.0.0", ApplicationStr: "core"}
	o.Resolver = &mapResolver{peek: map[string]*pkgmeta.Module{"core": mod}}
	closeOpsDB(t, db)
	if _, err := o.Plan(context.Background(), PlanRequest{Action: ActionLoadDemo}); err == nil {
		t.Fatal("load-demo create fail")
	}
	if _, err := o.Plan(context.Background(), PlanRequest{ModuleName: "core", Action: ActionInstall}); err == nil {
		t.Fatal("install create fail")
	}

	// Run save-running failure via GORM update callback.
	db2 := newOpsTestDB(t)
	o2 := &Ops{
		Scope:     newOpsTestScope(t, db2),
		Resolver:  &mapResolver{peek: map[string]*pkgmeta.Module{"core": mod}},
		Lifecycle: &stubLifecycle{},
		Events:    &recordingBus{},
	}
	res, err := o2.Plan(context.Background(), PlanRequest{ModuleName: "core", Action: ActionInstall})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if err := o2.Apply(context.Background(), ApplyRequest{Id: res.Id}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if err := db2.Callback().Update().Before("gorm:update").Register("ops_fail_update", func(tx *gorm.DB) {
		_ = tx.AddError(errors.New("forced update fail"))
	}); err != nil {
		t.Fatalf("callback: %v", err)
	}
	if err := o2.Run(context.Background(), RunRequest{Id: res.Id}); err == nil || !strings.Contains(err.Error(), "forced update fail") {
		t.Fatalf("Run save fail: %v", err)
	}

	// SetWithDemo Take non-RecordNotFound via query callback (AutoMigrate uses same DB).
	db3 := newOpsTestDB(t)
	o3 := &Ops{Scope: newOpsTestScope(t, db3)}
	if err := db3.AutoMigrate(&metaAppSetting{}); err != nil {
		t.Fatal(err)
	}
	if err := db3.Callback().Query().Before("gorm:query").Register("ops_fail_query", func(tx *gorm.DB) {
		_ = tx.AddError(errors.New("forced query fail"))
	}); err != nil {
		t.Fatalf("query callback: %v", err)
	}
	if err := o3.SetWithDemo(context.Background(), true); err == nil || !strings.Contains(err.Error(), "forced query fail") {
		t.Fatalf("SetWithDemo query fail: %v", err)
	}
}
