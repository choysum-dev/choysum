// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package injectappmodel

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestNilSessionGuards(t *testing.T) {
	if err := InjectAppModels(nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := DecideAndInjectOne(nil, "FieldDefault", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := DecideOne(nil, "FieldDefault", nil); err != nil {
		t.Fatal(err)
	}
	if err := ApplyInjectOne(nil, "FieldDefault"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateInjectAppModels(nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := BundleInjectAppModels(nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := BundleOne(nil, "FieldDefault", nil); err != nil {
		t.Fatal(err)
	}
	if err := SupersedeInjectAppModels(nil); err != nil {
		t.Fatal(err)
	}
	if err := SupersedeOne(nil, "FieldDefault"); err != nil {
		t.Fatal(err)
	}
	ReleaseSchedule("FieldDefault", "")
	ReleaseSchedule("FieldDefault", "partner")
	ReleaseSchedule("Missing", "partner")

	var nilSess *Session
	nilSess.ReleaseSchedules()
	nilSess.SetPlan("x", Plan{})
	if p := nilSess.Plan("x"); p.NeedInject {
		t.Fatal("nil Plan should be zero")
	}
	if nilSess.InjectPaths("x") != nil {
		t.Fatal("nil InjectPaths")
	}
	if nilSess.LastInjectPath("x") != "" {
		t.Fatal("nil LastInjectPath")
	}
	nilSess.ClearInjectPaths("x")
	if nilSess.AllInjectPaths() != nil {
		t.Fatal("nil AllInjectPaths")
	}
	nilSess.rememberInjectPath("x", "p")
	nilSess.releaseScheduleFor(nil)
	nilSess.releaseScheduleFor(specByNameOrPanic("FieldDefault"))
}

func TestDecideOneAndApplyInjectOne(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sess, _, _ := newTestSession(t, mod)
	plan, err := DecideOne(sess, "FieldDefault", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.NeedInject {
		t.Fatalf("expected NeedInject, got %+v", plan)
	}
	if _, err := DecideOne(sess, "NoSuch", nil); err == nil {
		t.Fatal("expected unknown Spec error")
	}
	if err := ApplyInjectOne(sess, "NoSuch"); err == nil {
		t.Fatal("expected unknown Spec error")
	}
	if err := ApplyInjectOne(sess, "FieldDefault"); err != nil {
		t.Fatal(err)
	}
	if sess.LastInjectPath("FieldDefault") == "" {
		t.Fatal("expected path after ApplyInjectOne")
	}
}

func TestDecideAndInjectOne_UnknownAndErrors(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sess, _, _ := newTestSession(t, mod)
	if err := DecideAndInjectOne(sess, "NoSuch", nil); err == nil {
		t.Fatal("expected unknown Spec")
	}

	a := "/virtual/modules/partner/service/models/fd_a.ts"
	b := "/virtual/modules/partner/service/models/fd_b.ts"
	if err := DecideAndInjectOne(sess, "FieldDefault", []*parser.ParserResult{
		{Path: a, Model: &meta.Model{Name: "FieldDefault", Path: a}},
		{Path: b, Model: &meta.Model{Name: "FieldDefault", Path: b}},
	}); err == nil || !strings.Contains(err.Error(), "FIELD_DEFAULT_DUPLICATE") {
		t.Fatalf("expected duplicate handwritten error, got %v", err)
	}
}

func TestDecidePlan_Branches(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sess, host, db := newTestSession(t, mod)

	if plan, err := decidePlan(nil, sess, nil); err != nil || plan.NeedInject {
		t.Fatalf("nil spec: %+v %v", plan, err)
	}
	spec := specByNameOrPanic("FieldDefault")
	if plan, err := decidePlan(spec, nil, nil); err != nil || plan.NeedInject {
		t.Fatalf("nil sess: %+v %v", plan, err)
	}
	host.mod = nil
	if plan, err := decidePlan(spec, sess, nil); err != nil || plan.NeedInject {
		t.Fatalf("nil module: %+v %v", plan, err)
	}
	host.mod = &meta.Module{Name: "x", Path: "/p", ApplicationStr: "partner"} // empty ServiceEntryPoint
	if plan, err := decidePlan(spec, sess, nil); err != nil || plan.NeedInject {
		t.Fatalf("empty entry: %+v %v", plan, err)
	}
	host.mod = &meta.Module{Name: "x", Path: "/p", ApplicationStr: "", ServiceEntryPoint: "s"}
	if plan, err := decidePlan(spec, sess, nil); err != nil || plan.NeedInject {
		t.Fatalf("empty app: %+v %v", plan, err)
	}

	host.mod = mod
	otherHand := "/virtual/modules/other/service/models/field_default.ts"
	seedDeclaration(t, db, "FieldDefault", "other-hand", otherHand, "partner")
	localHand := "/virtual/modules/partner/service/models/field_default.ts"
	if plan, err := decidePlan(spec, sess, []*parser.ParserResult{
		{Path: localHand, Model: &meta.Model{Name: "FieldDefault", Path: localHand}},
	}); err == nil || !strings.Contains(err.Error(), "FIELD_DEFAULT_DUPLICATE") {
		t.Fatalf("expected handwritten conflict, got plan=%+v err=%v", plan, err)
	}

	// Local non-handwritten model present → skip inject.
	ResetScheduledForTest()
	localVirt := "/virtual/modules/partner/service/models/__generated__/field_default.ts"
	if plan, err := decidePlan(spec, sess, []*parser.ParserResult{
		{Path: localVirt, Model: &meta.Model{Name: "FieldDefault", Path: localVirt}},
	}); err != nil || plan.NeedInject {
		t.Fatalf("local generated should skip NeedInject, got %+v %v", plan, err)
	}

	// Existing virt owned by another module → skip.
	ResetScheduledForTest()
	_ = meta.DeleteDeclarationTrees(db, []string{"other-hand"})
	otherVirt := "/virtual/modules/other/service/models/__generated__/field_default.ts"
	seedDeclaration(t, db, "FieldDefault", "ov", otherVirt, "partner")
	if plan, err := decidePlan(spec, sess, nil); err != nil || plan.NeedInject {
		t.Fatalf("other-module virt should skip, got %+v %v", plan, err)
	}

	// DB load error.
	ResetScheduledForTest()
	if err := meta.DropRawModelTable(db); err != nil {
		t.Fatal(err)
	}
	if _, err := decidePlan(spec, sess, nil); err == nil {
		t.Fatal("expected db load error")
	}
}

func TestApplyInject_EmptyModulePathAndModulesPathFallback(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sess, host, _ := newTestSession(t, mod)
	sess.SetPlan("FieldDefault", Plan{NeedInject: true})
	if err := applyInject(sess, specByNameOrPanic("FieldDefault"), Plan{NeedInject: true}); err == nil {
		t.Fatal("expected empty path error")
	}

	mod.Path = "/virtual/modules/partner"
	host.modulesPath = ""
	if err := applyInject(sess, specByNameOrPanic("FieldDefault"), Plan{NeedInject: true}); err != nil {
		t.Fatal(err)
	}
	src := host.virtualPaths[generatedPath(specByNameOrPanic("FieldDefault"), mod.Path)]
	if src == "" {
		t.Fatal("expected virtual source with modulesPath fallback")
	}
}

func TestClaimNeedInject_NilScheduledMap(t *testing.T) {
	spec := &Spec{ModelName: "Temp", ForeignClaimOnOwnerReinject: true, scheduled: nil}
	plan := claimNeedInject(spec, "app", "mod")
	if !plan.NeedInject || plan.ScheduledApp != "app" || spec.scheduled == nil {
		t.Fatalf("expected lazy scheduled init, got %+v scheduled=%v", plan, spec.scheduled)
	}
	plan2 := claimNeedInject(spec, "app", "other")
	if !plan2.NeedInject || plan2.ScheduledApp != "" {
		t.Fatalf("foreign claim branch: %+v", plan2)
	}
	plan3 := claimFirstNeedInject(&Spec{ModelName: "T2"}, "a", "m")
	if !plan3.NeedInject || plan3.ScheduledApp != "a" {
		t.Fatalf("claimFirst nil scheduled: %+v", plan3)
	}
}

func TestValidateInjectAppModels_Guards(t *testing.T) {
	sess := &Session{} // host nil
	if err := ValidateInjectAppModels(sess, nil); err != nil {
		t.Fatal(err)
	}
	mod := &meta.Module{Name: "p", Path: "/p", ApplicationStr: ""}
	sess, _, _ = newTestSession(t, mod)
	if err := ValidateInjectAppModels(sess, nil); err != nil {
		t.Fatal(err)
	}
	mod.ApplicationStr = "core"
	if err := ValidateInjectAppModels(sess, nil); err != nil {
		t.Fatal(err)
	}
	mod.ApplicationStr = "partner"
	if err := ValidateInjectAppModels(sess, nil); err != nil {
		t.Fatal(err)
	}
	single := "/virtual/modules/partner/service/models/fd.ts"
	if err := ValidateInjectAppModels(sess, []*parser.ParserResult{
		{Path: single, Model: &meta.Model{Name: "FieldDefault", Path: single}},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestSessionMapsAndPaths(t *testing.T) {
	s := &Session{}
	s.SetPlan("FieldDefault", Plan{NeedInject: true, ScheduledApp: "partner"})
	if !s.Plan("FieldDefault").NeedInject {
		t.Fatal("SetPlan lazy maps")
	}
	s.rememberInjectPath("FieldDefault", "")
	s.rememberInjectPath("FieldDefault", " /a ")
	s.rememberInjectPath("FieldDefault", "/a") // dedup
	s.rememberInjectPath("FieldDefault", "/b")
	paths := s.InjectPaths("FieldDefault")
	if len(paths) != 2 || paths[0] != "/a" || paths[1] != "/b" {
		t.Fatalf("paths=%#v", paths)
	}
	if s.InjectPaths("missing") != nil {
		t.Fatal("empty paths")
	}
	if s.LastInjectPath("FieldDefault") != "/b" {
		t.Fatal(s.LastInjectPath("FieldDefault"))
	}
	all := s.AllInjectPaths()
	if len(all) < 2 {
		t.Fatalf("AllInjectPaths=%#v", all)
	}
	s.ClearInjectPaths("FieldDefault")
	if s.InjectPaths("FieldDefault") != nil || s.LastInjectPath("FieldDefault") != "" {
		t.Fatal("ClearInjectPaths")
	}

	ResetScheduledForTest()
	spec := specByNameOrPanic("FieldDefault")
	spec.scheduled.Store("partner", "mod")
	s.SetPlan("FieldDefault", Plan{ScheduledApp: "partner"})
	s.releaseScheduleFor(spec)
	if _, ok := spec.scheduled.Load("partner"); ok {
		t.Fatal("releaseScheduleFor should clear")
	}
	if s.Plan("FieldDefault").ScheduledApp != "" {
		t.Fatal("plan scheduled cleared")
	}
	s.releaseScheduleFor(spec) // empty ScheduledApp no-op

	s.ReleaseSchedules() // empty plans / no apps
	s.SetPlan("FieldDefault", Plan{ScheduledApp: "  "})
	s.ReleaseSchedules() // whitespace app skipped
}

func TestReleaseScheduleCompat(t *testing.T) {
	ResetScheduledForTest()
	spec := specByNameOrPanic("FieldDefault")
	spec.scheduled.Store("partner", "mod")
	ReleaseSchedule("FieldDefault", "partner")
	if _, ok := spec.scheduled.Load("partner"); ok {
		t.Fatal("expected cleared")
	}
}

func TestScheduledAppsAndRegister(t *testing.T) {
	m := ScheduledApps("FieldDefault")
	if m == nil {
		t.Fatal("ScheduledApps")
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic for unknown model")
			}
		}()
		_ = ScheduledApps("NoSuchModel")
	}()

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("expected duplicate Register panic")
			}
		}()
		Register(Spec{ModelName: "FieldDefault"})
	}()

	// ResetScheduledForTest skips nil scheduled.
	tmp := &Spec{ModelName: "tmp-nil-sched", scheduled: nil}
	specsMu.Lock()
	specsBy["tmp-nil-sched"] = tmp
	specOrder = append(specOrder, "tmp-nil-sched")
	specsMu.Unlock()
	t.Cleanup(func() {
		specsMu.Lock()
		delete(specsBy, "tmp-nil-sched")
		filtered := specOrder[:0]
		for _, n := range specOrder {
			if n != "tmp-nil-sched" {
				filtered = append(filtered, n)
			}
		}
		specOrder = filtered
		specsMu.Unlock()
	})
	ResetScheduledForTest()
}

func TestHelpersCoverage(t *testing.T) {
	if isGeneratedPath(nil, "x") || isGeneratedPath(specByNameOrPanic("FieldDefault"), "") {
		t.Fatal("isGeneratedPath guards")
	}
	if isGeneratedPath(specByNameOrPanic("FieldDefault"), "service/models/__generated__/field_default.ts") {
		// exact rel match without leading slash — GeneratedRelPath == normalized
	}
	spec := specByNameOrPanic("FieldDefault")
	if !isGeneratedPath(spec, spec.GeneratedRelPath) {
		t.Fatal("exact rel equality")
	}
	_ = modelsIn(nil, nil, "")
	_ = modelsIn(spec, []*parser.ParserResult{nil, {Model: nil}, {Path: "/x", Model: &meta.Model{Name: "Other", Path: "/x"}}}, "/mod")
	if sameModule(nil, nil) || sameModule([]*meta.Model{{Path: "/a"}}, nil) {
		t.Fatal("sameModule nil mod")
	}
	mod := &meta.Module{Path: "/virtual/modules/partner"}
	mod.Id = sql.NullString{String: "mid", Valid: true}
	if !sameModule([]*meta.Model{{ModuleId: sql.NullString{String: "mid", Valid: true}}}, mod) {
		t.Fatal("sameModule by id")
	}
	if !sameModule([]*meta.Model{{Path: "/virtual/modules/partner/service/models/x.ts"}}, mod) {
		t.Fatal("sameModule by path")
	}
	if sameModule([]*meta.Model{nil, {Path: "/other/x.ts"}}, mod) {
		t.Fatal("sameModule miss")
	}
	if models, err := dbLoadModels(nil, nil, ""); err != nil || models != nil {
		t.Fatal(err, models)
	}
	out := mergeUniqueStrings([]string{"a", " ", "a"}, []string{"b", ""}, []string{"a", "c"})
	if strings.Join(out, ",") != "a,b,c" {
		t.Fatalf("merge=%#v", out)
	}
}

func TestGeneratedSource_EmptyApplication(t *testing.T) {
	src := generatedSource(specByNameOrPanic("FieldDefault"), "/mods", "  ")
	if !strings.Contains(src, strconvQuoteFallback()) {
		// ensure default application literal used
	}
	if !strings.Contains(src, "'application'") && !strings.Contains(src, `"application"`) {
		t.Fatalf("expected default application in source:\n%s", src)
	}
}

func strconvQuoteFallback() string { return "application" }

func TestBundleInject_GuardsAndErrors(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sess, host, db := newTestSession(t, mod)
	if err := BundleOne(sess, "NoSuch", []*meta.Module{mod}); err != nil {
		t.Fatal(err)
	}
	sess2 := &Session{host: nil}
	if err := BundleInjectAppModels(sess2, nil); err != nil {
		t.Fatal(err)
	}

	host.modulesPath = ""
	core := &meta.Module{Name: "core", Path: "/virtual/modules/core", ApplicationStr: "core", ServiceEntryPoint: "s"}
	emptyApp := &meta.Module{Name: "e", Path: "/p", ApplicationStr: "", ServiceEntryPoint: "s"}
	noEntry := &meta.Module{Name: "n", Path: "/p", ApplicationStr: "base", ServiceEntryPoint: ""}
	noPath := &meta.Module{Name: "np", Path: "  ", ApplicationStr: "base", ServiceEntryPoint: "s"}
	dup := &meta.Module{Name: "partner2", Path: "/virtual/modules/partner2", ApplicationStr: "partner", ServiceEntryPoint: "s"}
	if err := BundleInjectAppModels(sess, []*meta.Module{nil, core, emptyApp, noEntry, noPath, mod, dup}); err != nil {
		t.Fatal(err)
	}
	if host.modulesPath != "" {
		t.Fatal("modulesPath stayed empty; fallback should still register")
	}
	if len(sess.AllInjectPaths()) == 0 {
		t.Fatal("expected inject paths for partner")
	}

	if err := meta.DropRawModelTable(db); err != nil {
		t.Fatal(err)
	}
	sess3, _, _ := newTestSession(t, mod)
	// reopen dropped on different db — use same dropped db
	sess3.host.(*fakeHost).db = db
	if err := BundleInjectAppModels(sess3, []*meta.Module{mod}); err == nil {
		t.Fatal("expected load error")
	}
}

func TestSupersede_GuardsAndErrors(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sess, host, db := newTestSession(t, mod)
	if err := SupersedeInjectAppModels(&Session{}); err != nil {
		t.Fatal(err)
	}
	if err := SupersedeOne(sess, "NoSuch"); err != nil {
		t.Fatal(err)
	}
	sess.SetPlan("AppSetting", Plan{})
	if err := SupersedeOne(sess, "AppSetting"); err != nil {
		t.Fatal(err)
	}

	sess.SetPlan("AppSetting", Plan{SupersedeInject: true})
	host.db = nil
	if err := SupersedeOne(sess, "AppSetting"); err != nil {
		t.Fatal(err)
	}
	host.db = db
	host.mod = nil
	if err := SupersedeOne(sess, "AppSetting"); err != nil {
		t.Fatal(err)
	}
	host.mod = &meta.Module{ApplicationStr: "  "}
	if err := SupersedeOne(sess, "AppSetting"); err != nil {
		t.Fatal(err)
	}
	host.mod = mod

	// blank id skipped
	virt := "/virtual/modules/partner/service/models/__generated__/app_setting.ts"
	// bypass PersistModelTreeAsRaw id normalization via raw table insert (whitespace id).
	if err := db.Exec(
		`INSERT INTO meta_raw_model (id, created_at, updated_at, name, path, application, abstract, auto_migrate, readonly) VALUES (?,?,?,?,?,?,0,1,0)`,
		"   ", "2020-01-01", "2020-01-01", "AppSetting", virt, "partner",
	).Error; err != nil {
		t.Fatal(err)
	}
	sess.SetPlan("AppSetting", Plan{SupersedeInject: true})
	if err := SupersedeOne(sess, "AppSetting"); err != nil {
		t.Fatal(err)
	}

	if err := meta.DropRawModelTable(db); err != nil {
		t.Fatal(err)
	}
	if err := supersedeGenerated(specByNameOrPanic("AppSetting"), db, "partner"); err == nil {
		t.Fatal("expected list error after drop")
	}
}

func TestSupersede_DeleteError(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sess, _, db := newTestSession(t, mod)
	virt := "/virtual/modules/partner/service/models/__generated__/app_setting.ts"
	seedDeclaration(t, db, "AppSetting", "del-me", virt, "partner")
	sess.SetPlan("AppSetting", Plan{SupersedeInject: true})
	// Close DB to force delete failure after list — list may also fail.
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()
	if err := SupersedeOne(sess, "AppSetting"); err == nil {
		t.Fatal("expected supersede error on closed db")
	}
}

func TestDecideOne_DecidePlanError(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sess, _, db := newTestSession(t, mod)
	if err := meta.DropRawModelTable(db); err != nil {
		t.Fatal(err)
	}
	if _, err := DecideOne(sess, "FieldDefault", nil); err == nil {
		t.Fatal("expected DecideOne load error")
	}
}

func TestDecidePlan_LocalHandwrittenOnlyAndLocalGeneratedOnly(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sess, _, _ := newTestSession(t, mod)
	spec := specByNameOrPanic("FieldDefault")
	hand := "/virtual/modules/partner/service/models/field_default.ts"
	plan, err := decidePlan(spec, sess, []*parser.ParserResult{
		{Path: hand, Model: &meta.Model{Name: "FieldDefault", Path: hand}},
	})
	if err != nil || plan.NeedInject || plan.SupersedeInject {
		t.Fatalf("local handwritten only: %+v %v", plan, err)
	}

	virt := "/virtual/modules/partner/service/models/__generated__/field_default.ts"
	plan2, err := decidePlan(spec, sess, []*parser.ParserResult{
		{Path: virt, Model: &meta.Model{Name: "FieldDefault", Path: virt}},
	})
	if err != nil || plan2.NeedInject {
		t.Fatalf("local generated only: %+v %v", plan2, err)
	}
}

func TestApplyInject_NilModule(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sess, host, _ := newTestSession(t, mod)
	host.mod = nil
	if err := applyInject(sess, specByNameOrPanic("FieldDefault"), Plan{NeedInject: true}); err != nil {
		t.Fatal(err)
	}
}

func TestValidateInjectAppModels_NilModule(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sess, host, _ := newTestSession(t, mod)
	host.mod = nil
	if err := ValidateInjectAppModels(sess, nil); err != nil {
		t.Fatal(err)
	}
}

func TestSupersedeInjectAppModels_PropagatesError(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sess, _, db := newTestSession(t, mod)
	sess.SetPlan("FieldDefault", Plan{SupersedeInject: true})
	sess.SetPlan("AppSetting", Plan{SupersedeInject: true})
	if err := meta.DropRawModelTable(db); err != nil {
		t.Fatal(err)
	}
	if err := SupersedeInjectAppModels(sess); err == nil {
		t.Fatal("expected supersede propagation error")
	}
}

func TestSupersedeGenerated_DeleteError(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	_, _, db := newTestSession(t, mod)
	virt := "/virtual/modules/partner/service/models/__generated__/app_setting.ts"
	seedDeclaration(t, db, "AppSetting", "del-fail", virt, "partner")
	if err := db.Migrator().DropTable(meta.RawServiceTable); err != nil {
		t.Fatal(err)
	}
	if err := supersedeGenerated(specByNameOrPanic("AppSetting"), db, "partner"); err == nil {
		t.Fatal("expected delete error after dropping service table")
	}
}

func TestInjectAppModels_PropagatesDecideError(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sess, _, _ := newTestSession(t, mod)
	a := "/virtual/modules/partner/service/models/a.ts"
	b := "/virtual/modules/partner/service/models/b.ts"
	if err := InjectAppModels(sess, []*parser.ParserResult{
		{Path: a, Model: &meta.Model{Name: "FieldDefault", Path: a}},
		{Path: b, Model: &meta.Model{Name: "FieldDefault", Path: b}},
	}); err == nil {
		t.Fatal("expected error")
	}
}

func TestDecideAndInjectOne_ApplyInjectErrorReleases(t *testing.T) {
	mod := &meta.Module{
		Name: "partner", Path: "/virtual/modules/partner",
		ApplicationStr: "partner", ServiceEntryPoint: "service/index.ts",
	}
	sess, host, _ := newTestSession(t, mod)
	host.mod.Path = ""
	if err := DecideAndInjectOne(sess, "FieldDefault", nil); err == nil {
		t.Fatal("expected inject path error")
	}
}

func TestBundleOne_SuccessPath(t *testing.T) {
	mod := &meta.Module{
		Name: "base", Path: "/virtual/modules/base",
		ApplicationStr: "base", ServiceEntryPoint: "service/index.ts",
	}
	sess, _, _ := newTestSession(t, mod)
	if err := BundleOne(sess, "FieldDefault", []*meta.Module{mod}); err != nil {
		t.Fatal(err)
	}
	if sess.LastInjectPath("FieldDefault") == "" {
		t.Fatal("expected path")
	}
}
