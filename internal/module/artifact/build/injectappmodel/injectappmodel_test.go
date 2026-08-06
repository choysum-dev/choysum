// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package injectappmodel

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

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

type fakeHost struct {
	mod            *meta.Module
	db             *gorm.DB
	modulesPath    string
	entryImports   []string
	virtualPaths   map[string]string
	setImportCalls int
}

func (h *fakeHost) Module() *meta.Module        { return h.mod }
func (h *fakeHost) SessionDB() *gorm.DB         { return h.db }
func (h *fakeHost) ModulesPath() string         { return h.modulesPath }
func (h *fakeHost) EntryPointImports() []string { return h.entryImports }
func (h *fakeHost) SetEntryPointImports(imports []string) {
	h.setImportCalls++
	h.entryImports = append([]string(nil), imports...)
}
func (h *fakeHost) RegisterVirtualSource(path, contents string) { h.virtualPaths[path] = contents }

func newTestSession(t *testing.T, mod *meta.Module) (*Session, *fakeHost, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(testMemoryDSN(t, "injectappmodel")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := meta.EnsureDualStoreTables(db); err != nil {
		t.Fatalf("EnsureDualStoreTables: %v", err)
	}
	host := &fakeHost{
		mod:          mod,
		db:           db,
		modulesPath:  "/virtual/modules",
		virtualPaths: map[string]string{},
	}
	reg := NewRegistryWithDefaults()
	return NewSession(host, reg), host, db
}

func seedDeclaration(t *testing.T, db *gorm.DB, name, id, path, application string) {
	t.Helper()
	m := &meta.Model{Name: name, Path: path, Application: application}
	if id != "" {
		m.Id = sql.NullString{String: id, Valid: true}
	}
	if err := meta.PersistModelTreeAsRaw(db, m); err != nil {
		t.Fatalf("seed %s: %v", name, err)
	}
}

func TestSpecs_BuiltinsRegistered(t *testing.T) {
	specs := Specs()
	if len(specs) != 2 {
		t.Fatalf("expected 2 builtins, got %d", len(specs))
	}
	if specs[0].ModelName != "FieldDefault" || specs[1].ModelName != "AppSetting" {
		t.Fatalf("unexpected order/names: %#v", specs)
	}
}

func TestDecideFieldDefault_NeedInject(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sess, _, _ := newTestSession(t, mod)
	if err := InjectAppModels(sess, nil); err != nil {
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
	sess, _, db := newTestSession(t, mod)
	hand := "/virtual/modules/partner/service/models/app_setting.ts"
	seedDeclaration(t, db, "AppSetting", "as-virt", "/virtual/modules/partner/service/models/__generated__/app_setting.ts", "partner")
	if err := InjectAppModels(sess, []*parser.ParserResult{
		{Path: hand, Model: &meta.Model{Name: "AppSetting", Path: hand}},
	}); err != nil {
		t.Fatalf("inject: %v", err)
	}
	plan := sess.Plan("AppSetting")
	if plan.NeedInject || !plan.SupersedeInject {
		t.Fatalf("expected SupersedeInject only, got %+v", plan)
	}
}

func TestDecideAppSetting_ForeignClaimOnOwnerReinject(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sess, _, db := newTestSession(t, mod)
	virt := "/virtual/modules/partner/service/models/__generated__/app_setting.ts"
	seedDeclaration(t, db, "AppSetting", "as1", virt, "partner")

	sess.Registry().TryClaim("AppSetting", "partner", "other_builder")

	if err := InjectAppModels(sess, nil); err != nil {
		t.Fatalf("inject: %v", err)
	}
	plan := sess.Plan("AppSetting")
	if !plan.NeedInject || plan.ScheduledApp != "" {
		t.Fatalf("expected NeedInject without adopting foreign claim, got %+v", plan)
	}
	if owner, ok := sess.Registry().ClaimOwner("AppSetting", "partner"); !ok || owner != "other_builder" {
		t.Fatalf("foreign claim must remain, got %#v ok=%v", owner, ok)
	}
}

func TestDecideFieldDefault_SameModuleReclaimWithoutForeignBranch(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sess, _, db := newTestSession(t, mod)
	virt := "/virtual/modules/partner/service/models/__generated__/field_default.ts"
	seedDeclaration(t, db, "FieldDefault", "fd1", virt, "partner")

	sess.Registry().TryClaim("FieldDefault", "partner", "other_builder")

	if err := InjectAppModels(sess, nil); err != nil {
		t.Fatalf("inject: %v", err)
	}
	plan := sess.Plan("FieldDefault")
	if !plan.NeedInject || plan.ScheduledApp != "partner" {
		t.Fatalf("FieldDefault re-injects with scheduledApp even under foreign claim, got %+v", plan)
	}
}

func TestProcessDedup_SameModuleReclaim(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sess, _, _ := newTestSession(t, mod)
	if err := InjectAppModels(sess, nil); err != nil {
		t.Fatalf("first inject: %v", err)
	}
	if err := InjectAppModels(sess, nil); err != nil {
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
	sess, _, _ := newTestSession(t, mod)
	if err := InjectAppModels(sess, nil); err != nil {
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
	sess, _, db := newTestSession(t, mod)
	hand := "/virtual/modules/partner/service/models/app_setting.ts"
	virt := "/virtual/modules/partner/service/models/__generated__/app_setting.ts"
	seedDeclaration(t, db, "AppSetting", "virt-id", virt, "partner")
	seedDeclaration(t, db, "AppSetting", "hand-id", hand, "partner")

	sess.plans["AppSetting"] = Plan{SupersedeInject: true}
	if err := SupersedeInjectAppModels(sess); err != nil {
		t.Fatalf("supersede: %v", err)
	}

	remaining, err := meta.ListDeclarations(db, meta.DeclarationQuery{
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
	sess, host, _ := newTestSession(t, mod)
	if err := InjectAppModels(sess, nil); err != nil {
		t.Fatalf("inject: %v", err)
	}
	wantFD := generatedPath(specByNameOrPanic("FieldDefault"), mod.Path)
	wantAS := generatedPath(specByNameOrPanic("AppSetting"), mod.Path)
	if _, ok := host.virtualPaths[wantFD]; !ok {
		t.Fatalf("expected virtual source at %q, got keys %#v", wantFD, host.virtualPaths)
	}
	if sess.LastInjectPath("FieldDefault") != wantFD {
		t.Fatalf("LastInjectPath = %q want %q", sess.LastInjectPath("FieldDefault"), wantFD)
	}
	if len(host.entryImports) != 2 {
		t.Fatalf("entryImports = %#v, want unique [FieldDefault, AppSetting] paths", host.entryImports)
	}
	seen := map[string]int{}
	for _, p := range host.entryImports {
		seen[p]++
	}
	if seen[wantFD] != 1 || seen[wantAS] != 1 {
		t.Fatalf("entryImports = %#v, want one each of %q and %q", host.entryImports, wantFD, wantAS)
	}
}

func TestValidateInjectAppModels_Duplicate(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sess, _, _ := newTestSession(t, mod)
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

func TestDecideSkipsCore(t *testing.T) {
	mod := &meta.Module{
		Name: "core", Path: "/virtual/modules/core",
		ApplicationStr: "core", ServiceEntryPoint: "service/index.ts",
	}
	sess, _, _ := newTestSession(t, mod)
	if err := InjectAppModels(sess, nil); err != nil {
		t.Fatalf("inject: %v", err)
	}
	for _, name := range []string{"FieldDefault", "AppSetting"} {
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
	sess, _, db := newTestSession(t, mod)
	hand := "/virtual/modules/partner/service/models/field_default.ts"
	seedDeclaration(t, db, "FieldDefault", "hand", hand, "partner")
	if err := InjectAppModels(sess, nil); err != nil {
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
	sessA, _, db := newTestSession(t, modA)
	if err := InjectAppModels(sessA, nil); err != nil {
		t.Fatalf("inject A: %v", err)
	}
	if !sessA.Plan("FieldDefault").NeedInject {
		t.Fatal("A should inject")
	}

	modB := &meta.Module{
		Name: "partner_bank", Path: "/virtual/modules/partner_bank",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	hostB := &fakeHost{
		mod:          modB,
		db:           db,
		modulesPath:  "/virtual/modules",
		virtualPaths: map[string]string{},
	}
	sessB := NewSession(hostB, sessA.Registry())
	if err := InjectAppModels(sessB, nil); err != nil {
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
	sess, host, db := newTestSession(t, mod)
	seedDeclaration(t, db, "FieldDefault", "hand", "/virtual/modules/partner/service/models/field_default.ts", "partner")
	base := &meta.Module{
		Name: "base", Path: "/virtual/modules/base",
		ApplicationStr: "base", ServiceEntryPoint: "service/index.ts",
	}
	if err := BundleInjectAppModels(sess, []*meta.Module{mod, base}); err != nil {
		t.Fatalf("bundle: %v", err)
	}
	if paths := sess.InjectPaths("FieldDefault"); len(paths) != 1 {
		t.Fatalf("expected only base C2 path, got %#v", paths)
	}
	want := generatedPath(specByNameOrPanic("FieldDefault"), base.Path)
	if paths := sess.InjectPaths("FieldDefault"); paths[0] != want {
		t.Fatalf("path = %q want %q", paths[0], want)
	}
	if _, ok := host.virtualPaths[want]; !ok {
		t.Fatalf("expected virtual source for base at %q", want)
	}
	partnerGen := generatedPath(specByNameOrPanic("FieldDefault"), mod.Path)
	if _, ok := host.virtualPaths[partnerGen]; ok {
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
	sess, host, db := newTestSession(t, mod)
	canon := "/virtual/modules/partner/service/models/__generated__/field_default.ts"
	seedDeclaration(t, db, "FieldDefault", "virt", canon, "partner")
	if err := BundleInjectAppModels(sess, []*meta.Module{mod}); err != nil {
		t.Fatalf("bundle: %v", err)
	}
	if sess.LastInjectPath("FieldDefault") != canon {
		t.Fatalf("path = %q want canonical %q", sess.LastInjectPath("FieldDefault"), canon)
	}
	if _, ok := host.virtualPaths[canon]; !ok {
		t.Fatal("expected virtual source at canonical meta path")
	}
}
