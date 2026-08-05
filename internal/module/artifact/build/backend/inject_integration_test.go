// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendbuilder

import (
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/module/artifact/build/injectappmodel"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestInjectAppModels_LifecycleReleasesSchedule(t *testing.T) {
	injectappmodel.ResetScheduledForTest()
	dsn := filepath.Join(t.TempDir(), "inject-lifecycle.sqlite")
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := meta.EnsureDualStoreTables(db); err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&meta.Application{}, &meta.Module{}); err != nil {
		t.Fatal(err)
	}

	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	buildPlugin := &stubEsbPlugin{name: "build"}
	prebuildPlugin := &stubEsbPlugin{name: "prebuild"}
	testScope := newBuilderTestScope()
	testScope.session = &scope.Session{DB: db}
	builder := &ModuleBuilder{
		runtimeScope:   testScope,
		module:         mod,
		buildPlugin:    buildPlugin,
		prebuildPlugin: prebuildPlugin,
		entryPoint:     "",
	}

	if err := builder.injectAppModels(nil); err != nil {
		t.Fatalf("injectAppModels: %v", err)
	}
	sess := builder.ensureInjectSession()
	plan := sess.Plan("FieldDefault")
	if !plan.NeedInject || plan.ScheduledApp != "partner" {
		t.Fatalf("expected NeedInject plan, got %+v", plan)
	}
	wantPath := filepath.ToSlash(filepath.Join(mod.Path, "service/models/__generated__/field_default.ts"))
	if sess.LastInjectPath("FieldDefault") != wantPath {
		t.Fatalf("inject path = %q want %q", sess.LastInjectPath("FieldDefault"), wantPath)
	}
	if _, ok := buildPlugin.virtualSources[wantPath]; !ok {
		t.Fatal("expected virtual source on build plugin")
	}

	builder.releaseInjectSchedules()
	if plan := sess.Plan("FieldDefault"); plan.ScheduledApp != "" {
		t.Fatalf("expected schedule cleared, got %+v", plan)
	}
	if _, loaded := injectappmodel.ScheduledApps("FieldDefault").Load("partner"); loaded {
		t.Fatal("expected process claim cleared")
	}
}
