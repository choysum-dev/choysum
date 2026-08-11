// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package injectappmodel

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var testDBSeq atomic.Uint64

func testMemoryDSN(t *testing.T, prefix string) string {
	t.Helper()
	name := strings.ReplaceAll(t.Name(), "/", "_")
	return fmt.Sprintf("file:%s-%s-%d?mode=memory&cache=shared", prefix, name, testDBSeq.Add(1))
}

func newTestSession(t *testing.T, mod *meta.Module) (*Session, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(testMemoryDSN(t, "injectappmodel")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("AutoMigrate CatalogEntities: %v", err)
	}
	reg := NewRegistryWithDefaults()
	sess := NewSession(BuildCtx{
		Module:      mod,
		DB:          db,
		ModulesPath: "/virtual/modules",
	}, reg)
	return sess, db
}

func seedDeclaration(t *testing.T, db *gorm.DB, name, id, path, application string) {
	t.Helper()
	moduleID := "seed-" + application
	m := &meta.Model{
		Name: name, Path: path, Application: application,
		ModuleId: sql.NullString{String: moduleID, Valid: true},
	}
	if id != "" {
		m.Id = sql.NullString{String: id, Valid: true}
	}
	existing, err := modmeta.ListDeclarations(db, modmeta.DeclarationQuery{ModuleID: moduleID, PreloadTree: true})
	if err != nil {
		t.Fatalf("list declarations for seed: %v", err)
	}
	out := make([]*meta.Model, 0, len(existing)+1)
	for _, e := range existing {
		if e != nil && e.Path != path {
			out = append(out, e)
		}
	}
	out = append(out, m)
	if _, err := modmeta.ReplaceModuleDeclarations(db, moduleID, out); err != nil {
		t.Fatalf("seed %s: %v", name, err)
	}
}

func effectsFileMap(fx Effects) map[string]string {
	out := make(map[string]string, len(fx.Files))
	for _, f := range fx.Files {
		out[f.Path] = f.Contents
	}
	return out
}

func TestSpecs_BuiltinsRegistered(t *testing.T) {
	specs := Specs()
	if len(specs) != 4 {
		t.Fatalf("expected 4 builtins, got %d", len(specs))
	}
	if specs[0].ModelName != "TranslationTerm" ||
		specs[1].ModelName != "FieldDefault" ||
		specs[2].ModelName != "AppSetting" ||
		specs[3].ModelName != "PropertyDefinition" {
		t.Fatalf("unexpected order/names: %#v", specs)
	}
	if !specs[0].EnsureServiceEntry {
		t.Fatal("TranslationTerm must EnsureServiceEntry")
	}
	if specs[1].EnsureServiceEntry || specs[2].EnsureServiceEntry || specs[3].EnsureServiceEntry {
		t.Fatal("FieldDefault/AppSetting/PropertyDefinition must leave EnsureServiceEntry false")
	}
}

func TestDecideFieldDefault_NeedInject(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sess, _ := newTestSession(t, mod)
	if _, err := InjectAppModels(sess, nil); err != nil {
		t.Fatalf("inject: %v", err)
	}
	plan := sess.Plan("FieldDefault")
	if !plan.NeedInject || plan.SupersedeInject || plan.ScheduledApp != "partner" {
		t.Fatalf("expected NeedInject with scheduledApp, got %+v", plan)
	}
}

func TestDecideAppSetting_SupersedeInject(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sess, db := newTestSession(t, mod)
	hand := "/virtual/modules/partner/service/models/app_setting.ts"
	seedDeclaration(t, db, "AppSetting", "as-virt", "/virtual/modules/partner/service/models/__generated__/app_setting.ts", "partner")
	if _, err := InjectAppModels(sess, []*parser.ParserResult{
		{Path: hand, Model: &meta.Model{Name: "AppSetting", Path: hand}},
	}); err != nil {
		t.Fatalf("inject: %v", err)
	}
	plan := sess.Plan("AppSetting")
	if plan.NeedInject || !plan.SupersedeInject {
		t.Fatalf("expected SupersedeInject only, got %+v", plan)
	}
}

func TestDecide_OwnerReinjectForeignClaim_NoScheduledApp(t *testing.T) {
	// FieldDefault / AppSetting / PropertyDefinition share the same claim semantics after P1.
	for _, tc := range []struct {
		modelName string
		rel       string
	}{
		{"FieldDefault", "field_default"},
		{"AppSetting", "app_setting"},
		{"PropertyDefinition", "property_definition"},
	} {
		t.Run(tc.modelName, func(t *testing.T) {
			mod := &meta.Module{
				Name: "partner", Path: "/virtual/modules/partner",
				ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
			}
			sess, db := newTestSession(t, mod)
			virt := "/virtual/modules/partner/service/models/__generated__/" + tc.rel + ".ts"
			seedDeclaration(t, db, tc.modelName, "virt1", virt, "partner")

			sess.Registry().TryClaim(tc.modelName, "partner", "other_builder")

			if _, err := InjectAppModels(sess, nil); err != nil {
				t.Fatalf("inject: %v", err)
			}
			plan := sess.Plan(tc.modelName)
			if !plan.NeedInject || plan.ScheduledApp != "" {
				t.Fatalf("expected NeedInject without adopting foreign claim, got %+v", plan)
			}
			if owner, ok := sess.Registry().ClaimOwner(tc.modelName, "partner"); !ok || owner != "other_builder" {
				t.Fatalf("foreign claim must remain, got %#v ok=%v", owner, ok)
			}
		})
	}
}

func TestDecide_EmptyServiceEntry_SkipsFieldDefaultWithoutEnsure(t *testing.T) {
	mod := &meta.Module{
		Name: "web", Path: "/virtual/modules/web",
		ApplicationStr: "web", ServiceEntryPoint: "",
	}
	sess, _ := newTestSession(t, mod)
	fx, err := DecideAndInjectOne(sess, "FieldDefault", nil)
	if err != nil {
		t.Fatalf("DecideAndInjectOne: %v", err)
	}
	plan := sess.Plan("FieldDefault")
	if plan.NeedInject || plan.SupersedeInject || len(fx.Files) != 0 {
		t.Fatalf("FieldDefault without Ensure must skip empty entry, got plan=%+v fx=%+v", plan, fx)
	}
}

func TestResolveServiceEntryPathAndCanEnsure(t *testing.T) {
	mod := &meta.Module{Name: "web", Path: "/virtual/modules/web"}
	if got := resolveServiceEntryPath(mod, "service/index.ts"); got != "/virtual/modules/web/service/index.ts" {
		t.Fatalf("relative resolve = %q", got)
	}
	if got := resolveServiceEntryPath(mod, "/abs/service/index.ts"); got != "/abs/service/index.ts" {
		t.Fatalf("absolute resolve = %q", got)
	}
	if got := resolveServiceEntryPath(nil, "rel/a.ts"); got != "rel/a.ts" {
		t.Fatalf("nil mod resolve = %q", got)
	}
	if got := resolveServiceEntryPath(&meta.Module{Path: ""}, "rel/b.ts"); got != "rel/b.ts" {
		t.Fatalf("empty path resolve = %q", got)
	}
	if got := resolveServiceEntryPath(mod, ""); got != "" {
		t.Fatalf("empty entry = %q", got)
	}
	if canEnsureServiceEntry(nil, &Spec{BaseModelFile: "x"}, "/p") {
		t.Fatal("nil sess")
	}
	if canEnsureServiceEntry(&Session{}, nil, "/p") {
		t.Fatal("nil spec")
	}
	if !canEnsureServiceEntry(&Session{ctx: BuildCtx{ModulesPath: "/any"}}, &Spec{BaseModelFile: ""}, "/p") {
		t.Fatal("empty BaseModelFile should allow")
	}

	dir := t.TempDir()
	sess, _ := newTestSession(t, &meta.Module{Name: "x", Path: filepath.Join(dir, "x"), ApplicationStr: "x"})
	sess.Context().ModulesPath = dir
	spec := Spec{BaseModelFile: "core/service/orm/model/translation_term_base_model.ts"}
	// ModulesPath exists but base model missing → deny Ensure.
	if canEnsureServiceEntry(sess, &spec, filepath.Join(dir, "x")) {
		t.Fatal("expected canEnsure false when base model missing on disk")
	}
	// Non-existent ModulesPath (virtual harness) → allow.
	sess.Context().ModulesPath = filepath.Join(dir, "missing-modules-root")
	if !canEnsureServiceEntry(sess, &spec, filepath.Join(dir, "x")) {
		t.Fatal("expected canEnsure true for missing ModulesPath harness")
	}
	sess.Context().ModulesPath = "."
	if canEnsureServiceEntry(sess, &spec, "x") {
		t.Fatal("ModulesPath=. should deny")
	}
	sess.Context().ModulesPath = ""
	if canEnsureServiceEntry(sess, &spec, ".") {
		t.Fatal("Dir(.) fallback should deny")
	}
}

func TestEnsureServiceEntryPath_Guards(t *testing.T) {
	mod := &meta.Module{Name: "web", Path: "/v/web", ApplicationStr: "web", ServiceEntryPoint: ""}
	sess, _ := newTestSession(t, mod)
	sess.ensureServiceEntryPath("", true)
	if mod.ServiceEntryPoint != "" || sess.ensuredServiceEntry {
		t.Fatal("empty path must not ensure")
	}
	sess.Context().Module = nil
	sess.ensureServiceEntryPath("/v/web/service/index.ts", true)
	sess.Context().Module = mod
	sess.ensureServiceEntryPath("/v/web/service/index.ts", true)
	if mod.ServiceEntryPoint != "/v/web/service/index.ts" || !sess.ensuredServiceEntry || !sess.ensuredVirtual {
		t.Fatal("expected ensure to set virtual entry")
	}
	prior := mod.ServiceEntryPoint
	sess.ensureServiceEntryPath("/other", false)
	if sess.priorServiceEntry != "" && sess.priorServiceEntry != prior {
		// prior remembered only once — first prior was empty
	}
	if mod.ServiceEntryPoint != "/other" {
		t.Fatalf("second ensure should update path, got %q", mod.ServiceEntryPoint)
	}
	if !sess.ensuredVirtual {
		t.Fatal("virtual flag must stick once set for this build")
	}
	(*Session)(nil).ensureServiceEntryPath("/x", true)
}

func TestDecide_EmptyModulePathAndEnsureDenied(t *testing.T) {
	mod := &meta.Module{
		Name: "web", Path: "",
		ApplicationStr: "web", ServiceEntryPoint: "",
	}
	sess, _ := newTestSession(t, mod)
	fx, err := DecideAndInjectOne(sess, "TranslationTerm", nil)
	if err != nil {
		t.Fatal(err)
	}
	if sess.Plan("TranslationTerm").NeedInject || len(fx.Files) != 0 {
		t.Fatalf("empty Path must skip, got plan=%+v fx=%+v", sess.Plan("TranslationTerm"), fx)
	}

	dir := t.TempDir()
	mod2 := &meta.Module{
		Name: "web", Path: filepath.Join(dir, "web"),
		ApplicationStr: "web", ServiceEntryPoint: "",
	}
	sess2, _ := newTestSession(t, mod2)
	sess2.Context().ModulesPath = dir // exists, base model missing
	fx2, err := DecideAndInjectOne(sess2, "TranslationTerm", nil)
	if err != nil {
		t.Fatal(err)
	}
	if sess2.Plan("TranslationTerm").NeedInject || len(fx2.Files) != 0 {
		t.Fatalf("canEnsure deny must skip, got plan=%+v", sess2.Plan("TranslationTerm"))
	}
}

func TestApplyInjectOne_MaterializeErrorReleases(t *testing.T) {
	mod := &meta.Module{
		Name: "web", Path: "",
		ApplicationStr: "web", ServiceEntryPoint: "service/index.ts",
	}
	sess, _ := newTestSession(t, mod)
	sess.SetPlan("FieldDefault", Plan{NeedInject: true, ScheduledApp: "web"})
	if _, err := ApplyInjectOne(sess, "FieldDefault"); err == nil {
		t.Fatal("expected materialize error for empty module path")
	}
}

func TestMaterialize_EmptyEntryWithoutEnsureReturnsEmpty(t *testing.T) {
	mod := &meta.Module{
		Name: "web", Path: "/virtual/modules/web",
		ApplicationStr: "web", ServiceEntryPoint: "",
	}
	sess, _ := newTestSession(t, mod)
	sess.SetPlan("FieldDefault", Plan{NeedInject: true, ScheduledApp: "web"})
	fx, err := materializeInject(sess, specByNameOrPanic("FieldDefault"), Plan{NeedInject: true, ScheduledApp: "web"})
	if err != nil {
		t.Fatal(err)
	}
	if len(fx.Files) != 0 || len(fx.Imports) != 0 || fx.ServiceEntryPath != "" {
		t.Fatalf("expected empty effects, got %+v", fx)
	}
}

func TestBundleSpec_EnsureEmptyEntry_EmitsVirtualService(t *testing.T) {
	mod := &meta.Module{
		Name: "web", Path: "/virtual/modules/web",
		ApplicationStr: "web", ServiceEntryPoint: "",
	}
	sess, _ := newTestSession(t, mod)
	sess.Context().ModulesPath = "/virtual/modules-missing-harness"
	fx, err := BundleOne(sess, "TranslationTerm", []*meta.Module{mod})
	if err != nil {
		t.Fatal(err)
	}
	if mod.ServiceEntryPoint != "" {
		t.Fatalf("Bundle must not mutate ServiceEntryPoint, got %q", mod.ServiceEntryPoint)
	}
	wantEntry := virtualServiceEntryPath(mod.Path)
	found := false
	for _, f := range fx.Files {
		if f.Path == wantEntry {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected virtual service at %q in %#v", wantEntry, fx.Files)
	}
}

func TestBundleSpec_EnsureEmptyEntry_SkipsWhenDiskExists(t *testing.T) {
	dir := t.TempDir()
	modPath := filepath.Join(dir, "web")
	svc := filepath.Join(modPath, "service")
	if err := os.MkdirAll(svc, 0o755); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(svc, "index.ts")
	if err := os.WriteFile(entry, []byte("export {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mod := &meta.Module{
		Name: "web", Path: modPath,
		ApplicationStr: "web", ServiceEntryPoint: "",
	}
	sess, _ := newTestSession(t, mod)
	sess.Context().ModulesPath = filepath.Join(dir, "missing-modules")
	fx, err := BundleOne(sess, "TranslationTerm", []*meta.Module{mod})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fx.Files {
		if strings.HasSuffix(filepath.ToSlash(f.Path), "service/index.ts") {
			t.Fatalf("must not virtualize existing disk entry: %q", f.Path)
		}
	}
}

func TestBundleSpec_EnsureRelativeEntry_MissingEmitsVirtual(t *testing.T) {
	mod := &meta.Module{
		Name: "web", Path: "/virtual/modules/web",
		ApplicationStr: "web", ServiceEntryPoint: "service/index.ts",
	}
	sess, _ := newTestSession(t, mod)
	fx, err := BundleOne(sess, "TranslationTerm", []*meta.Module{mod})
	if err != nil {
		t.Fatal(err)
	}
	abs := resolveServiceEntryPath(mod, "service/index.ts")
	found := false
	for _, f := range fx.Files {
		if filepath.Clean(f.Path) == filepath.Clean(abs) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected virtual at resolved %q, files=%#v", abs, fx.Files)
	}
}

func TestInject_EnsureServiceEntry_AdoptsExistingDiskEntry(t *testing.T) {
	dir := t.TempDir()
	modPath := filepath.Join(dir, "auth")
	svcDir := filepath.Join(modPath, "service")
	if err := os.MkdirAll(svcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	entryPath := filepath.Join(svcDir, "index.ts")
	if err := os.WriteFile(entryPath, []byte("export * from './models';\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mod := &meta.Module{
		Name: "auth", Path: modPath,
		ApplicationStr: "auth", ServiceEntryPoint: "",
	}
	sess, _ := newTestSession(t, mod)
	fx, err := InjectAppModels(sess, nil)
	if err != nil {
		t.Fatalf("inject: %v", err)
	}
	if mod.ServiceEntryPoint != entryPath && filepath.Clean(mod.ServiceEntryPoint) != filepath.Clean(entryPath) {
		t.Fatalf("ServiceEntryPoint = %q want disk entry %q", mod.ServiceEntryPoint, entryPath)
	}
	if fx.ServiceEntryPath != entryPath && filepath.Clean(fx.ServiceEntryPath) != filepath.Clean(entryPath) {
		t.Fatalf("ServiceEntryPath = %q, want disk entry so builder can adopt it", fx.ServiceEntryPath)
	}
	for _, f := range fx.Files {
		if strings.HasSuffix(filepath.ToSlash(f.Path), "service/index.ts") {
			t.Fatalf("must not register virtual service stub over disk entry: %q", f.Path)
		}
	}
}

func TestInject_WebEmptyEntry_EnsureThenSiblingSpecs(t *testing.T) {
	mod := &meta.Module{
		Name: "web", Path: "/virtual/modules/web",
		ApplicationStr: "web", ServiceEntryPoint: "",
	}
	sess, _ := newTestSession(t, mod)
	fx, err := InjectAppModels(sess, nil)
	if err != nil {
		t.Fatalf("inject: %v", err)
	}
	tt := sess.Plan("TranslationTerm")
	if !tt.NeedInject || tt.ScheduledApp != "web" {
		t.Fatalf("TranslationTerm should Ensure+NeedInject, got %+v", tt)
	}
	// Virtual Ensure must not unlock FieldDefault / AppSetting / PropertyDefinition
	// (package.json has only entryPoints.web — see per-app-platform-store.md §1).
	for _, name := range []string{"FieldDefault", "AppSetting", "PropertyDefinition"} {
		plan := sess.Plan(name)
		if plan.NeedInject || plan.SupersedeInject {
			t.Fatalf("%s must skip after virtual Ensure, got %+v", name, plan)
		}
	}
	wantEntry := virtualServiceEntryPath(mod.Path)
	if mod.ServiceEntryPoint != wantEntry {
		t.Fatalf("ServiceEntryPoint = %q want %q", mod.ServiceEntryPoint, wantEntry)
	}
	if fx.ServiceEntryPath != wantEntry {
		t.Fatalf("ServiceEntryPath = %q want %q", fx.ServiceEntryPath, wantEntry)
	}
	files := effectsFileMap(fx)
	if _, ok := files[wantEntry]; !ok {
		t.Fatalf("expected virtual service entry at %q", wantEntry)
	}
	wantTT := generatedPath(specByNameOrPanic("TranslationTerm"), mod.Path)
	if src, ok := files[wantTT]; !ok || !strings.Contains(src, "TranslationTermBaseModel") {
		t.Fatalf("expected TranslationTerm thin class at %q, got ok=%v src=%q", wantTT, ok, src)
	}
	if !strings.Contains(generatedSource(specByNameOrPanic("TranslationTerm"), "/virtual/modules", "web"), "softDelete: false") {
		t.Fatal("TranslationTerm thin class should softDelete: false")
	}
	if len(fx.Imports) != 1 {
		t.Fatalf("Imports = %#v, want only TranslationTerm", fx.Imports)
	}
	wantFD := generatedPath(specByNameOrPanic("FieldDefault"), mod.Path)
	wantAS := generatedPath(specByNameOrPanic("AppSetting"), mod.Path)
	wantPD := generatedPath(specByNameOrPanic("PropertyDefinition"), mod.Path)
	if _, ok := files[wantFD]; ok {
		t.Fatalf("must not inject FieldDefault for web-only entry: %q", wantFD)
	}
	if _, ok := files[wantAS]; ok {
		t.Fatalf("must not inject AppSetting for web-only entry: %q", wantAS)
	}
	if _, ok := files[wantPD]; ok {
		t.Fatalf("must not inject PropertyDefinition for web-only entry: %q", wantPD)
	}
}

func TestDecide_EmptyDeclaredEntry_SupersedesStaleGenerated(t *testing.T) {
	mod := &meta.Module{
		Name: "web", Path: "/virtual/modules/web",
		ApplicationStr: "web", ServiceEntryPoint: "",
	}
	sess, db := newTestSession(t, mod)
	virt := generatedPath(specByNameOrPanic("FieldDefault"), mod.Path)
	seedDeclaration(t, db, "FieldDefault", "stale-fd", virt, "web")

	plan, err := DecideOne(sess, "FieldDefault", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.SupersedeInject || plan.NeedInject {
		t.Fatalf("expected SupersedeInject for stale C2 without declared service entry, got %+v", plan)
	}
}

func TestDecide_EmptyDeclaredEntry_EarlyReturnsAndLoadError(t *testing.T) {
	spec := specByNameOrPanic("FieldDefault")

	for _, tc := range []struct {
		name string
		mod  *meta.Module
	}{
		{"empty-app", &meta.Module{Name: "x", Path: "/p", ApplicationStr: "", ServiceEntryPoint: ""}},
		{"core-app", &meta.Module{Name: "x", Path: "/p", ApplicationStr: "core", ServiceEntryPoint: ""}},
		{"empty-path", &meta.Module{Name: "web", Path: "", ApplicationStr: "web", ServiceEntryPoint: ""}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sess, _ := newTestSession(t, tc.mod)
			plan, err := decidePlan(spec, sess, nil)
			if err != nil || plan.NeedInject || plan.SupersedeInject {
				t.Fatalf("expected empty plan, got %+v err=%v", plan, err)
			}
		})
	}

	mod := &meta.Module{
		Name: "web", Path: "/virtual/modules/web",
		ApplicationStr: "web", ServiceEntryPoint: "",
	}
	sess, db := newTestSession(t, mod)
	if err := db.Migrator().DropTable("meta_raw_model"); err != nil {
		t.Fatal(err)
	}
	if _, err := DecideOne(sess, "FieldDefault", nil); err == nil {
		t.Fatal("expected load error when purging stale C2 without declared service entry")
	}
}

func TestDeclaredServiceEntry_GuardsAndVirtualMask(t *testing.T) {
	if (*Session)(nil).declaredServiceEntry(specByNameOrPanic("FieldDefault")) != "" {
		t.Fatal("nil session must return empty")
	}
	sess := &Session{}
	if sess.declaredServiceEntry(specByNameOrPanic("FieldDefault")) != "" {
		t.Fatal("nil module must return empty")
	}

	mod := &meta.Module{
		Name: "web", Path: "/virtual/modules/web",
		ApplicationStr: "web", ServiceEntryPoint: "",
	}
	sess, _ = newTestSession(t, mod)
	// Disk-adopt first, then a virtual rewrite must stick ensuredVirtual so
	// FieldDefault still sees the prior (empty) declared entry.
	sess.ensureServiceEntryPath("/virtual/modules/web/service/index.ts", false)
	if sess.ensuredVirtual {
		t.Fatal("disk adopt must not mark virtual")
	}
	sess.ensureServiceEntryPath("/virtual/modules/web/service/index.ts", true)
	if !sess.ensuredVirtual {
		t.Fatal("later virtual Ensure must set ensuredVirtual")
	}
	if got := sess.declaredServiceEntry(specByNameOrPanic("FieldDefault")); got != "" {
		t.Fatalf("FieldDefault declared entry = %q, want empty prior", got)
	}
	if got := sess.declaredServiceEntry(specByNameOrPanic("TranslationTerm")); got == "" {
		t.Fatal("Ensure Spec should still see the mutated Module entry")
	}
	if got := sess.declaredServiceEntry(nil); got == "" {
		t.Fatal("nil Spec should return Module entry")
	}
}

func TestDecide_EnsureServiceEntry_EmptyEntryAllowsNeedInject(t *testing.T) {
	mod := &meta.Module{
		Name: "web", Path: "/virtual/modules/web",
		ApplicationStr: "web", ServiceEntryPoint: "",
	}
	sess, _ := newTestSession(t, mod)
	sess.Registry().Register(Spec{
		ModelName:          "TempEnsure",
		GeneratedRelPath:   "service/models/__generated__/temp_ensure.ts",
		DuplicateCode:      "TEMP_ENSURE_DUPLICATE",
		BaseModelFile:      "core/service/orm/model/app_setting_base_model.ts",
		EnsureServiceEntry: true,
	})
	fx, err := DecideAndInjectOne(sess, "TempEnsure", nil)
	if err != nil {
		t.Fatalf("DecideAndInjectOne: %v", err)
	}
	plan := sess.Plan("TempEnsure")
	if !plan.NeedInject || plan.ScheduledApp != "web" {
		t.Fatalf("expected NeedInject with scheduledApp, got %+v", plan)
	}
	wantEntry := virtualServiceEntryPath(mod.Path)
	if fx.ServiceEntryPath != wantEntry || mod.ServiceEntryPoint != wantEntry {
		t.Fatalf("Ensure did not set entry: fx=%q mod=%q", fx.ServiceEntryPath, mod.ServiceEntryPoint)
	}
	if len(fx.Files) < 2 || len(fx.Imports) != 1 {
		t.Fatalf("expected virtual service + model Effects, got files=%d imports=%d", len(fx.Files), len(fx.Imports))
	}
}

func TestDecide_TranslationTerm_OwnerReinjectForeignClaim(t *testing.T) {
	mod := &meta.Module{
		Name: "web", Path: "/virtual/modules/web",
		ApplicationStr: "web", ServiceEntryPoint: "service/index.ts",
	}
	sess, db := newTestSession(t, mod)
	virt := "/virtual/modules/web/service/models/__generated__/translation_term.ts"
	seedDeclaration(t, db, "TranslationTerm", "virt1", virt, "web")
	sess.Registry().TryClaim("TranslationTerm", "web", "other_builder")

	if _, err := InjectAppModels(sess, nil); err != nil {
		t.Fatalf("inject: %v", err)
	}
	plan := sess.Plan("TranslationTerm")
	if !plan.NeedInject || plan.ScheduledApp != "" {
		t.Fatalf("expected NeedInject without adopting foreign claim, got %+v", plan)
	}
	if owner, ok := sess.Registry().ClaimOwner("TranslationTerm", "web"); !ok || owner != "other_builder" {
		t.Fatalf("foreign claim must remain, got %#v ok=%v", owner, ok)
	}
}

func TestClearAllInjectPaths_RevertsEnsuredServiceEntry(t *testing.T) {
	mod := &meta.Module{
		Name: "web", Path: "/virtual/modules/web",
		ApplicationStr: "web", ServiceEntryPoint: "",
	}
	sess, _ := newTestSession(t, mod)
	if _, err := DecideAndInjectOne(sess, "TranslationTerm", nil); err != nil {
		t.Fatalf("inject: %v", err)
	}
	if mod.ServiceEntryPoint == "" {
		t.Fatal("expected Ensure to set ServiceEntryPoint")
	}
	sess.ClearAllInjectPaths()
	if mod.ServiceEntryPoint != "" {
		t.Fatalf("expected revert to empty entry, got %q", mod.ServiceEntryPoint)
	}
}

func TestRevertEnsuredServiceEntry_ForPersist(t *testing.T) {
	mod := &meta.Module{
		Name: "web", Path: "/virtual/modules/web",
		ApplicationStr: "web", ServiceEntryPoint: "",
	}
	sess, _ := newTestSession(t, mod)
	if _, err := DecideAndInjectOne(sess, "TranslationTerm", nil); err != nil {
		t.Fatalf("inject: %v", err)
	}
	sess.RevertEnsuredServiceEntry()
	if mod.ServiceEntryPoint != "" {
		t.Fatalf("Persist revert must clear Ensure entry, got %q", mod.ServiceEntryPoint)
	}
}

func TestProcessDedup_SameModuleReclaim(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sess, _ := newTestSession(t, mod)
	if _, err := InjectAppModels(sess, nil); err != nil {
		t.Fatalf("first inject: %v", err)
	}
	if _, err := InjectAppModels(sess, nil); err != nil {
		t.Fatalf("reclaim inject: %v", err)
	}
	plan := sess.Plan("FieldDefault")
	if !plan.NeedInject || plan.ScheduledApp != "partner" {
		t.Fatalf("expected reclaim with scheduledApp, got %+v", plan)
	}
}

func TestReleaseSchedules_ClearsClaim(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sess, _ := newTestSession(t, mod)
	if _, err := InjectAppModels(sess, nil); err != nil {
		t.Fatalf("inject: %v", err)
	}
	if owner, ok := sess.Registry().ClaimOwner("FieldDefault", "partner"); !ok || owner == "" {
		t.Fatalf("expected scheduled claim, got owner=%q ok=%v", owner, ok)
	}
	sess.ReleaseSchedules()
	if _, ok := sess.Registry().ClaimOwner("FieldDefault", "partner"); ok {
		t.Fatal("expected claim cleared")
	}
	if plan := sess.Plan("FieldDefault"); plan.ScheduledApp != "" {
		t.Fatalf("expected plan scheduledApp cleared, got %+v", plan)
	}
}

func TestSupersedeInjectAppModels_DeletesGeneratedOnly(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sess, db := newTestSession(t, mod)
	hand := "/virtual/modules/partner/service/models/app_setting.ts"
	virt := "/virtual/modules/partner/service/models/__generated__/app_setting.ts"
	seedDeclaration(t, db, "AppSetting", "virt-id", virt, "partner")
	seedDeclaration(t, db, "AppSetting", "hand-id", hand, "partner")

	sess.plans["AppSetting"] = Plan{SupersedeInject: true}
	if err := SupersedeInjectAppModels(sess); err != nil {
		t.Fatalf("supersede: %v", err)
	}

	remaining, err := modmeta.ListDeclarations(db, modmeta.DeclarationQuery{
		Application: "partner",
		Name:        "AppSetting",
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(remaining) != 1 {
		t.Fatalf("expected 1 remaining row (handwritten), got %d", len(remaining))
	}
	if isGeneratedPath(specByNameOrPanic("AppSetting"), remaining[0].Path) {
		t.Fatalf("expected handwritten row to remain, path=%q", remaining[0].Path)
	}
}

func specByNameOrPanic(name string) *Spec {
	s, ok := specByName(name)
	if !ok {
		panic("spec missing: " + name)
	}
	return s
}

func TestInjectRegistersVirtualSource(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sess, _ := newTestSession(t, mod)
	fx, err := InjectAppModels(sess, nil)
	if err != nil {
		t.Fatalf("inject: %v", err)
	}
	wantTT := generatedPath(specByNameOrPanic("TranslationTerm"), mod.Path)
	wantFD := generatedPath(specByNameOrPanic("FieldDefault"), mod.Path)
	wantAS := generatedPath(specByNameOrPanic("AppSetting"), mod.Path)
	wantPD := generatedPath(specByNameOrPanic("PropertyDefinition"), mod.Path)
	files := effectsFileMap(fx)
	for _, want := range []string{wantTT, wantFD, wantAS, wantPD} {
		if _, ok := files[want]; !ok {
			t.Fatalf("expected virtual source at %q, got keys %#v", want, files)
		}
	}
	if sess.LastInjectPath("TranslationTerm") != wantTT {
		t.Fatalf("LastInjectPath TranslationTerm = %q want %q", sess.LastInjectPath("TranslationTerm"), wantTT)
	}
	if len(fx.Imports) != 4 {
		t.Fatalf("Imports = %#v, want unique [TranslationTerm, FieldDefault, AppSetting, PropertyDefinition] paths", fx.Imports)
	}
	seen := map[string]int{}
	for _, p := range fx.Imports {
		seen[p]++
	}
	if seen[wantTT] != 1 || seen[wantFD] != 1 || seen[wantAS] != 1 || seen[wantPD] != 1 {
		t.Fatalf("Imports = %#v, want one each of %q, %q, %q, %q", fx.Imports, wantTT, wantFD, wantAS, wantPD)
	}
}

func TestValidateInjectAppModels_Duplicate(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sess, _ := newTestSession(t, mod)
	a := "/virtual/modules/partner/service/models/fd_a.ts"
	b := "/virtual/modules/partner/service/models/fd_b.ts"
	err := ValidateInjectAppModels(sess, []*parser.ParserResult{
		{Path: a, Model: &meta.Model{Name: "FieldDefault", Path: a}},
		{Path: b, Model: &meta.Model{Name: "FieldDefault", Path: b}},
	})
	if err == nil || !strings.Contains(err.Error(), "FIELD_DEFAULT_DUPLICATE") {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestValidateInjectAppModels_PropertyDefinitionDuplicate(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sess, _ := newTestSession(t, mod)
	a := "/virtual/modules/partner/service/models/pd_a.ts"
	b := "/virtual/modules/partner/service/models/pd_b.ts"
	err := ValidateInjectAppModels(sess, []*parser.ParserResult{
		{Path: a, Model: &meta.Model{Name: "PropertyDefinition", Path: a}},
		{Path: b, Model: &meta.Model{Name: "PropertyDefinition", Path: b}},
	})
	if err == nil || !strings.Contains(err.Error(), "PROPERTY_DEFINITION_DUPLICATE") {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestInjectAppModels_ClearsPathsOnPartialFailure(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sess, _ := newTestSession(t, mod)
	// TranslationTerm + FieldDefault materialize first; AppSetting then fails Decide on duplicate handwritten.
	a := "/virtual/modules/partner/service/models/as_a.ts"
	b := "/virtual/modules/partner/service/models/as_b.ts"
	_, err := InjectAppModels(sess, []*parser.ParserResult{
		{Path: a, Model: &meta.Model{Name: "AppSetting", Path: a}},
		{Path: b, Model: &meta.Model{Name: "AppSetting", Path: b}},
	})
	if err == nil || !strings.Contains(err.Error(), "APP_SETTING_DUPLICATE") {
		t.Fatalf("expected AppSetting duplicate, got %v", err)
	}
	if paths := sess.AllInjectPaths(); len(paths) != 0 {
		t.Fatalf("stale inject paths after failed multi-spec inject: %#v", paths)
	}
	if sess.LastInjectPath("TranslationTerm") != "" || sess.LastInjectPath("FieldDefault") != "" {
		t.Fatal("earlier Spec paths should be cleared")
	}
}

func TestDecideSkipsCore(t *testing.T) {
	mod := &meta.Module{
		Name: "core", Path: "/virtual/modules/core",
		ApplicationStr: "core", ServiceEntryPoint: "service/index.ts",
	}
	sess, _ := newTestSession(t, mod)
	if _, err := InjectAppModels(sess, nil); err != nil {
		t.Fatalf("inject: %v", err)
	}
	for _, name := range []string{"TranslationTerm", "FieldDefault", "AppSetting", "PropertyDefinition"} {
		if plan := sess.Plan(name); plan.NeedInject || plan.SupersedeInject {
			t.Fatalf("%s should skip core, got %+v", name, plan)
		}
	}
}

func TestDecide_HandwrittenSkipsInject(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sess, db := newTestSession(t, mod)
	hand := "/virtual/modules/partner/service/models/field_default.ts"
	seedDeclaration(t, db, "FieldDefault", "hand", hand, "partner")
	if _, err := InjectAppModels(sess, nil); err != nil {
		t.Fatalf("inject: %v", err)
	}
	plan := sess.Plan("FieldDefault")
	if plan.NeedInject || plan.SupersedeInject {
		t.Fatalf("expected skip when DB handwritten exists, got %+v", plan)
	}
}

func TestProcessDedup_OtherModuleSkips(t *testing.T) {
	modA := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sessA, db := newTestSession(t, modA)
	if _, err := InjectAppModels(sessA, nil); err != nil {
		t.Fatalf("inject A: %v", err)
	}
	if !sessA.Plan("FieldDefault").NeedInject {
		t.Fatal("A should inject")
	}

	modB := &meta.Module{
		Name: "partner_bank", Path: "/virtual/modules/partner_bank",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sessB := NewSession(BuildCtx{
		Module:      modB,
		DB:          db,
		ModulesPath: "/virtual/modules",
	}, sessA.Registry())
	if _, err := InjectAppModels(sessB, nil); err != nil {
		t.Fatalf("inject B: %v", err)
	}
	if plan := sessB.Plan("FieldDefault"); plan.NeedInject {
		t.Fatalf("B should skip after A claimed app, got %+v", plan)
	}
}

func TestBundleInjectAppModels_SkipsHandwritten(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sess, db := newTestSession(t, mod)
	seedDeclaration(t, db, "FieldDefault", "hand", "/virtual/modules/partner/service/models/field_default.ts", "partner")
	base := &meta.Module{
		Name: "base", Path: "/virtual/modules/base",
		ApplicationStr: "base", ServiceEntryPoint: "service/index.ts",
	}
	fx, err := BundleInjectAppModels(sess, []*meta.Module{mod, base})
	if err != nil {
		t.Fatalf("bundle: %v", err)
	}
	if paths := sess.InjectPaths("FieldDefault"); len(paths) != 1 {
		t.Fatalf("expected only base C2 path, got %#v", paths)
	}
	want := generatedPath(specByNameOrPanic("FieldDefault"), base.Path)
	if paths := sess.InjectPaths("FieldDefault"); paths[0] != want {
		t.Fatalf("path = %q want %q", paths[0], want)
	}
	files := effectsFileMap(fx)
	if _, ok := files[want]; !ok {
		t.Fatalf("expected virtual source for base at %q", want)
	}
	partnerGen := generatedPath(specByNameOrPanic("FieldDefault"), mod.Path)
	if _, ok := files[partnerGen]; ok {
		t.Fatal("must not inject C2 for handwritten-owned app")
	}
}

func TestGeneratedSource_AppSettingSoftDeleteFalse(t *testing.T) {
	src := generatedSource(specByNameOrPanic("AppSetting"), `/tmp/mod"quote`, "app'name")
	if !strings.Contains(src, "softDelete: false") {
		t.Fatalf("AppSetting source must set softDelete: false:\n%s", src)
	}
	if !strings.Contains(src, strconv.Quote("app'name")) {
		t.Fatalf("application must be quoted:\n%s", src)
	}
	fd := generatedSource(specByNameOrPanic("FieldDefault"), "/tmp/mod", "partner")
	if strings.Contains(fd, "softDelete") {
		t.Fatalf("FieldDefault must not set softDelete:\n%s", fd)
	}
}

func TestBundlePrefersExistingGeneratedPath(t *testing.T) {
	mod := &meta.Module{
		Name: "sibling", Path: "/virtual/modules/sibling",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sess, db := newTestSession(t, mod)
	canon := "/virtual/modules/partner/service/models/__generated__/field_default.ts"
	seedDeclaration(t, db, "FieldDefault", "virt", canon, "partner")
	fx, err := BundleInjectAppModels(sess, []*meta.Module{mod})
	if err != nil {
		t.Fatalf("bundle: %v", err)
	}
	if sess.LastInjectPath("FieldDefault") != canon {
		t.Fatalf("path = %q want canonical %q", sess.LastInjectPath("FieldDefault"), canon)
	}
	if _, ok := effectsFileMap(fx)[canon]; !ok {
		t.Fatal("expected virtual source at canonical meta path")
	}
}
