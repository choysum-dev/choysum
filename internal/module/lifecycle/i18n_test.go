// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/defaultscope"
	i18nimport "github.com/choysum-dev/choysum/internal/i18n/import"
	i18nmodels "github.com/choysum-dev/choysum/internal/i18n/models"
	"github.com/choysum-dev/choysum/internal/i18n/store"
	"github.com/choysum-dev/choysum/internal/import/runner"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newI18nTestScope(t *testing.T) scope.Scope {
	t.Helper()
	return newI18nTestScopeWithModulesPath(t, "")
}

func newI18nTestScopeWithModulesPath(t *testing.T, modulesPath string) scope.Scope {
	t.Helper()
	store.ResetSharedRegistryForTests()
	cfg := &config.Config{
		Db: &config.DbConfig{
			Dialect: "sqlite",
			DSN:     filepath.Join(t.TempDir(), "lifecycle_i18n.db"),
		},
		ModulesPath: modulesPath,
	}
	runtimeScope := defaultscope.NewDefaultScope(
		context.Background(),
		scopetest.FactoryInputFromConfig(cfg),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if sqlDB, err := runtimeScope.Session().DB.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	return runtimeScope
}

func TestImportAndDeleteModuleTerminology(t *testing.T) {
	t.Parallel()
	rs := newI18nTestScope(t)
	moduleRoot := t.TempDir()
	i18nDir := filepath.Join(moduleRoot, "i18n")
	if err := os.MkdirAll(i18nDir, 0o755); err != nil {
		t.Fatal(err)
	}
	poBody := `
msgid ""
msgstr "Language: zh_CN\n"

msgctxt "web/a@title"
msgid "Hello"
msgstr "你好"
`
	if err := os.WriteFile(filepath.Join(i18nDir, "zh_CN.po"), []byte(poBody), 0o644); err != nil {
		t.Fatal(err)
	}

	mod := &meta.Module{
		Name:           "demo",
		ApplicationStr: "auth",
		Path:           moduleRoot,
	}
	if err := importModuleTerminology(rs, mod, ""); err != nil {
		t.Fatalf("importModuleTerminology: %v", err)
	}

	var row i18nmodels.TranslationTerm
	if err := rs.Session().Table("auth_translation_term").Where("src = ?", "Hello").Take(&row).Error; err != nil {
		t.Fatalf("expected term row: %v", err)
	}
	if row.Value != "你好" || row.Source != i18nmodels.SourcePackaged {
		t.Fatalf("unexpected row: %+v", row)
	}

	// Override must survive re-import.
	row.Source = i18nmodels.SourceOverride
	row.Value = "覆盖"
	if err := rs.Session().Table("auth_translation_term").Save(&row).Error; err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(i18nDir, "zh_CN.po"), []byte(`
msgctxt "web/a@title"
msgid "Hello"
msgstr "新包"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := importModuleTerminology(rs, mod, ""); err != nil {
		t.Fatal(err)
	}
	var after i18nmodels.TranslationTerm
	if err := rs.Session().Table("auth_translation_term").Where("src = ?", "Hello").Take(&after).Error; err != nil {
		t.Fatal(err)
	}
	if after.Value != "覆盖" || after.Source != i18nmodels.SourceOverride {
		t.Fatalf("override not preserved: %+v", after)
	}

	if err := deleteModuleTerminology(rs, mod); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := rs.Session().Table("auth_translation_term").Where("module = ?", "demo").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected deleted terms, got %d", count)
	}
}

func TestImportModuleTerminologyCoreModuleWritesCoreTable(t *testing.T) {
	t.Parallel()
	rs := newI18nTestScope(t)
	modulesPath := t.TempDir()
	coreRoot := filepath.Join(modulesPath, "core")
	i18nDir := filepath.Join(coreRoot, "i18n")
	if err := os.MkdirAll(i18nDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(i18nDir, "zh_CN.po"), []byte(`
msgctxt "service/orm@err"
msgid "Record not found"
msgstr "记录不存在"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	coreMod := &meta.Module{Name: "core", ApplicationStr: "core", Path: coreRoot}
	if err := importModuleTerminology(rs, coreMod, modulesPath); err != nil {
		t.Fatalf("import core: %v", err)
	}

	var row i18nmodels.TranslationTerm
	if err := rs.Session().Table("core_translation_term").Where("module = ? AND src = ?", "core", "Record not found").Take(&row).Error; err != nil {
		t.Fatalf("core term: %v", err)
	}
	if row.Application != "core" || row.Value != "记录不存在" {
		t.Fatalf("unexpected row: %+v", row)
	}
}

func TestImportModuleTerminologySkipsNonCoreOnCoreApp(t *testing.T) {
	t.Parallel()
	rs := newI18nTestScope(t)
	if err := importModuleTerminology(rs, &meta.Module{Name: "other", ApplicationStr: "core"}, ""); err != nil {
		t.Fatal(err)
	}
	if err := importModuleTerminology(rs, &meta.Module{Name: "demo", ApplicationStr: "auth", Path: filepath.Join(t.TempDir(), "nope")}, ""); err != nil {
		t.Fatal(err)
	}
}

func TestImportModuleTerminologyDoesNotFanOutFrameworkPO(t *testing.T) {
	t.Parallel()
	rs := newI18nTestScope(t)
	modulesPath := t.TempDir()

	coreI18n := filepath.Join(modulesPath, "core", "i18n")
	if err := os.MkdirAll(coreI18n, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(coreI18n, "zh_CN.po"), []byte(`
msgctxt "service/orm@err"
msgid "Record not found"
msgstr "记录不存在"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	demoRoot := filepath.Join(modulesPath, "demo")
	demoI18n := filepath.Join(demoRoot, "i18n")
	if err := os.MkdirAll(demoI18n, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(demoI18n, "zh_CN.po"), []byte(`
msgctxt "web/a@title"
msgid "Hello"
msgstr "你好"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	mod := &meta.Module{
		Name:           "demo",
		ApplicationStr: "auth",
		Path:           demoRoot,
	}
	if err := importModuleTerminology(rs, mod, modulesPath); err != nil {
		t.Fatalf("importModuleTerminology: %v", err)
	}

	var demoRow i18nmodels.TranslationTerm
	if err := rs.Session().Table("auth_translation_term").Where("module = ? AND src = ?", "demo", "Hello").Take(&demoRow).Error; err != nil {
		t.Fatalf("demo term: %v", err)
	}
	if demoRow.Value != "你好" {
		t.Fatalf("demo value = %q", demoRow.Value)
	}
	var count int64
	if err := rs.Session().Table("auth_translation_term").Where("module = ?", "core").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected no Module=core rows in auth table, got %d", count)
	}
}

func TestImportModuleTerminologyPurgesStaleSchemeACoreRows(t *testing.T) {
	t.Parallel()
	rs := newI18nTestScope(t)
	if err := i18nmodels.MigrateTranslationTermTable(rs, "auth"); err != nil {
		t.Fatal(err)
	}
	if err := rs.Session().Table("auth_translation_term").Create(&i18nmodels.TranslationTerm{
		Application: "auth", Module: "core", Lang: "zh_CN",
		Scope: "service/orm@err", Src: "Record not found", Value: "stale",
		Kind: i18nmodels.KindLiteral, Source: i18nmodels.SourcePackaged,
	}).Error; err != nil {
		t.Fatal(err)
	}

	demoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(demoRoot, "i18n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(demoRoot, "i18n", "zh_CN.po"), []byte(`
msgctxt "web/a@title"
msgid "Hello"
msgstr "你好"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	mod := &meta.Module{Name: "demo", ApplicationStr: "auth", Path: demoRoot}
	if err := importModuleTerminology(rs, mod, ""); err != nil {
		t.Fatalf("importModuleTerminology: %v", err)
	}
	var count int64
	if err := rs.Session().Table("auth_translation_term").Where("module = ?", "core").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected Scheme A Module=core rows purged, got %d", count)
	}
	// Second import is idempotent: no Module=core rows to delete, still succeeds.
	if err := importModuleTerminology(rs, mod, ""); err != nil {
		t.Fatalf("re-import: %v", err)
	}
}

func TestPurgeStaleFrameworkTermsFromHost(t *testing.T) {
	t.Parallel()
	if err := purgeStaleFrameworkTermsFromHost(nil, ""); err != nil {
		t.Fatalf("empty host: %v", err)
	}
	if err := purgeStaleFrameworkTermsFromHost(nil, "core"); err != nil {
		t.Fatalf("framework host: %v", err)
	}
	if err := purgeStaleFrameworkTermsFromHost(nil, "auth"); err == nil {
		t.Fatal("expected error when runtime session is missing")
	}
}

func TestDeleteModuleTerminologyMissingSession(t *testing.T) {
	t.Parallel()
	mod := &meta.Module{Name: "demo", ApplicationStr: "auth"}
	if err := deleteModuleTerminology(nil, mod); err == nil {
		t.Fatal("expected error when runtime session is missing")
	}
}

func TestDeleteModuleTerminologyCoreModule(t *testing.T) {
	t.Parallel()
	rs := newI18nTestScope(t)
	if err := i18nmodels.MigrateTranslationTermTable(rs, "core"); err != nil {
		t.Fatal(err)
	}
	if err := rs.Session().Table("core_translation_term").Create(&i18nmodels.TranslationTerm{
		Application: "core", Module: "core", Lang: "zh_CN",
		Scope: "a@b", Src: "x", Value: "y",
		Kind: i18nmodels.KindLiteral, Source: i18nmodels.SourcePackaged,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := deleteModuleTerminology(rs, &meta.Module{Name: "core", ApplicationStr: "core"}); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := rs.Session().Table("core_translation_term").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected core terms deleted, got %d", count)
	}
}

func TestDeleteModuleTerminologySkipsCoreAppNonCoreModule(t *testing.T) {
	t.Parallel()
	rs := newI18nTestScope(t)
	if err := deleteModuleTerminology(rs, &meta.Module{Name: "auth", ApplicationStr: "core"}); err != nil {
		t.Fatalf("expected no-op, got %v", err)
	}
}

func TestDeleteModuleTerminologySkipsHostAppCoreModule(t *testing.T) {
	t.Parallel()
	rs := newI18nTestScope(t)
	if err := deleteModuleTerminology(rs, &meta.Module{Name: "core", ApplicationStr: "auth"}); err != nil {
		t.Fatalf("expected no-op, got %v", err)
	}
}

func TestReloadModuleTerminology_importsPoAndReportsStats(t *testing.T) {
	t.Parallel()
	modulesPath := t.TempDir()
	moduleRoot := filepath.Join(modulesPath, "demo")
	i18nDir := filepath.Join(moduleRoot, "i18n")
	if err := os.MkdirAll(i18nDir, 0o755); err != nil {
		t.Fatal(err)
	}
	pkgJSON := `{
  "name": "@choysum-dev/demo",
  "choysum": { "moduleName": "demo", "application": "auth" }
}`
	if err := os.WriteFile(filepath.Join(moduleRoot, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(i18nDir, "zh_CN.po"), []byte(`
msgctxt "auth.model.User.fields"
msgid "Light"
msgstr "浅色"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	rs := newI18nTestScopeWithModulesPath(t, modulesPath)
	stats, err := ReloadModuleTerminology(rs, "demo")
	if err != nil {
		t.Fatalf("ReloadModuleTerminology: %v", err)
	}
	if stats.Upserted < 1 {
		t.Fatalf("Upserted = %d, want >= 1", stats.Upserted)
	}

	var row i18nmodels.TranslationTerm
	if err := rs.Session().Table("auth_translation_term").Where("src = ?", "Light").Take(&row).Error; err != nil {
		t.Fatalf("expected Light term: %v", err)
	}
	if row.Value != "浅色" {
		t.Fatalf("Value = %q, want 浅色", row.Value)
	}

	if err := os.WriteFile(filepath.Join(i18nDir, "zh_CN.po"), []byte(`
msgctxt "auth.model.User.fields"
msgid "Light"
msgstr "亮色"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	stats2, err := ReloadModuleTerminology(rs, "demo")
	if err != nil {
		t.Fatalf("ReloadModuleTerminology second: %v", err)
	}
	if stats2.Upserted < 1 {
		t.Fatalf("second Upserted = %d, want >= 1", stats2.Upserted)
	}
	var after i18nmodels.TranslationTerm
	if err := rs.Session().Table("auth_translation_term").Where("src = ?", "Light").Take(&after).Error; err != nil {
		t.Fatal(err)
	}
	if after.Value != "亮色" {
		t.Fatalf("Value after reload = %q, want 亮色", after.Value)
	}
}

func TestReloadModuleTerminology_requiresModuleName(t *testing.T) {
	t.Parallel()
	rs := newI18nTestScope(t)
	if _, err := ReloadModuleTerminology(rs, "  "); err == nil {
		t.Fatal("expected error for empty module name")
	}
	if _, err := ReloadModuleTerminology(nil, "demo"); err == nil {
		t.Fatal("expected error for nil scope")
	}
	modulesPath := t.TempDir()
	rsWithPath := newI18nTestScopeWithModulesPath(t, modulesPath)
	if _, err := ReloadModuleTerminology(rsWithPath, "../other"); err == nil {
		t.Fatal("expected error for path-traversal module name")
	}
	if _, err := ReloadModuleTerminology(rs, filepath.Join(string(filepath.Separator), "tmp", "escape")); err == nil {
		t.Fatal("expected error for absolute module name")
	}
}

func TestEnsureModuleRootWithinModulesPath(t *testing.T) {
	t.Parallel()
	modulesPath := filepath.Join(string(filepath.Separator), "workspace", "modules")
	if err := ensureModuleRootWithinModulesPath(modulesPath, filepath.Join(modulesPath, "demo")); err != nil {
		t.Fatalf("in-tree root: %v", err)
	}
	if err := ensureModuleRootWithinModulesPath(modulesPath, filepath.Join(string(filepath.Separator), "other")); err == nil {
		t.Fatal("expected escape error for sibling root")
	}
}

func TestReloadModuleTerminology_rejectsSymlinkEscapingModulesPath(t *testing.T) {
	t.Parallel()
	modulesPath := t.TempDir()
	outside := t.TempDir()
	moduleRoot := filepath.Join(modulesPath, "demo")
	if err := os.Symlink(outside, moduleRoot); err != nil {
		t.Skipf("symlink not available: %v", err)
	}
	rs := newI18nTestScopeWithModulesPath(t, modulesPath)
	if _, err := ReloadModuleTerminology(rs, "demo"); err == nil {
		t.Fatal("expected error for symlink escaping modules path")
	}
}

func TestReloadModuleTerminology_requiresModulesPath(t *testing.T) {
	t.Parallel()
	rs := newI18nTestScopeWithModulesPath(t, "")
	if _, err := ReloadModuleTerminology(rs, "demo"); err == nil {
		t.Fatal("expected error when modules path is empty")
	}
}

func TestReloadModuleTerminology_resolvePathErrors(t *testing.T) {
	t.Cleanup(func() { evalSymlinks = filepath.EvalSymlinks })

	modulesPath := t.TempDir()
	rs := newI18nTestScopeWithModulesPath(t, modulesPath)

	evalSymlinks = func(path string) (string, error) {
		return "", errors.New("forced modules path resolve failure")
	}
	if _, err := ReloadModuleTerminology(rs, "demo"); err == nil || !strings.Contains(err.Error(), "resolve modules path") {
		t.Fatalf("modules path resolve error = %v", err)
	}

	var calls int
	evalSymlinks = func(path string) (string, error) {
		calls++
		if calls == 1 {
			return path, nil
		}
		return "", errors.New("forced module root resolve failure")
	}
	if _, err := ReloadModuleTerminology(rs, "demo"); err == nil || !strings.Contains(err.Error(), "resolve module path") {
		t.Fatalf("module root resolve error = %v", err)
	}

	calls = 0
	evalSymlinks = func(path string) (string, error) {
		calls++
		if calls == 1 {
			return path, nil
		}
		return "", os.ErrNotExist
	}
	if _, err := ReloadModuleTerminology(rs, "demo"); err == nil || !strings.Contains(err.Error(), "not found under modules path") {
		t.Fatalf("missing module error = %v", err)
	}
}

func TestReloadModuleTerminology_nilContextFallsBackToBackground(t *testing.T) {
	t.Parallel()
	modulesPath := t.TempDir()
	moduleRoot := filepath.Join(modulesPath, "demo")
	i18nDir := filepath.Join(moduleRoot, "i18n")
	if err := os.MkdirAll(i18nDir, 0o755); err != nil {
		t.Fatal(err)
	}
	pkgJSON := `{
  "name": "@choysum-dev/demo",
  "choysum": { "moduleName": "demo", "application": "auth" }
}`
	if err := os.WriteFile(filepath.Join(moduleRoot, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(i18nDir, "zh_CN.po"), []byte(`
msgctxt "auth.model.User.fields"
msgid "Night"
msgstr "夜间"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	base := newI18nTestScopeWithModulesPath(t, modulesPath)
	rs := nilContextScope{Scope: base}
	if _, err := ReloadModuleTerminology(rs, "demo"); err != nil {
		t.Fatalf("ReloadModuleTerminology: %v", err)
	}
	// Stats may still accumulate via the WithContext child; the goal here is the
	// nil Context() → background fallback path, not the upsert counter.
}

func TestReloadModuleTerminology_rejectsInvalidPackageJSON(t *testing.T) {
	t.Parallel()
	modulesPath := t.TempDir()
	moduleRoot := filepath.Join(modulesPath, "demo")
	if err := os.MkdirAll(moduleRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(moduleRoot, "package.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	rs := newI18nTestScopeWithModulesPath(t, modulesPath)
	if _, err := ReloadModuleTerminology(rs, "demo"); err == nil {
		t.Fatal("expected error for invalid package.json")
	}
}

func TestReloadModuleTerminology_importError(t *testing.T) {
	t.Cleanup(func() { importpkg.SetRun(runner.Run) })
	importpkg.SetRun(func(_ context.Context, _ scope.Scope, _ importpkg.Spec) (importpkg.Report, error) {
		return importpkg.Report{}, errors.New("forced terminology failure")
	})

	modulesPath := t.TempDir()
	moduleRoot := filepath.Join(modulesPath, "demo")
	if err := os.MkdirAll(moduleRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	pkgJSON := `{
  "name": "@choysum-dev/demo",
  "choysum": { "moduleName": "demo", "application": "auth" }
}`
	if err := os.WriteFile(filepath.Join(moduleRoot, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	rs := newI18nTestScopeWithModulesPath(t, modulesPath)
	if _, err := ReloadModuleTerminology(rs, "demo"); err == nil {
		t.Fatal("expected import error from forced Run failure")
	}
}

func TestReloadTerminologyForLang_onlyTouchesMatchingPo(t *testing.T) {
	// Not parallel: stubs package-level listInstalledModulesForTerminology.
	modulesPath := t.TempDir()

	modA := filepath.Join(modulesPath, "mod_a")
	modB := filepath.Join(modulesPath, "mod_b")
	for _, root := range []string{modA, modB} {
		if err := os.MkdirAll(filepath.Join(root, "i18n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(modA, "i18n", "zh_CN.po"), []byte(`
msgctxt "a@title"
msgid "Alpha"
msgstr "甲"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	// mod_b has a different language only — must not be imported for zh_CN.
	if err := os.WriteFile(filepath.Join(modB, "i18n", "en_US.po"), []byte(`
msgctxt "b@title"
msgid "Beta"
msgstr "Beta"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	rs := newI18nTestScopeWithModulesPath(t, modulesPath)
	prevList := listInstalledModulesForTerminology
	t.Cleanup(func() { listInstalledModulesForTerminology = prevList })
	listInstalledModulesForTerminology = func(scope.Scope) ([]meta.Module, error) {
		return []meta.Module{
			{Name: "mod_a", ApplicationStr: "auth", Path: modA, Status: meta.Installed},
			{Name: "mod_b", ApplicationStr: "auth", Path: modB, Status: meta.Installed},
		}, nil
	}

	stats, err := ReloadTerminologyForLang(rs, "zh_CN")
	if err != nil {
		t.Fatalf("ReloadTerminologyForLang: %v", err)
	}
	if stats.ModulesScanned != 2 {
		t.Fatalf("ModulesScanned=%d, want 2", stats.ModulesScanned)
	}
	if stats.ModulesWithPo != 1 {
		t.Fatalf("ModulesWithPo=%d, want 1", stats.ModulesWithPo)
	}
	if stats.Upserted < 1 {
		t.Fatalf("Upserted=%d, want >= 1", stats.Upserted)
	}
	if stats.Lang != "zh_CN" {
		t.Fatalf("Lang=%q, want zh_CN", stats.Lang)
	}

	var alpha i18nmodels.TranslationTerm
	if err := rs.Session().Table("auth_translation_term").Where("src = ?", "Alpha").Take(&alpha).Error; err != nil {
		t.Fatalf("expected Alpha term: %v", err)
	}
	if alpha.Value != "甲" {
		t.Fatalf("Alpha value=%q, want 甲", alpha.Value)
	}
	var betaCount int64
	if err := rs.Session().Table("auth_translation_term").Where("src = ?", "Beta").Count(&betaCount).Error; err != nil {
		t.Fatal(err)
	}
	if betaCount != 0 {
		t.Fatalf("Beta rows=%d, want 0 (en_US.po must not be imported)", betaCount)
	}
}

func TestReloadTerminologyForLang_rejectsEmptyAndPathLang(t *testing.T) {
	t.Parallel()
	rs := newI18nTestScope(t)
	if _, err := ReloadTerminologyForLang(rs, "  "); err == nil {
		t.Fatal("expected error for empty lang")
	}
	if _, err := ReloadTerminologyForLang(nil, "zh_CN"); err == nil {
		t.Fatal("expected error for nil scope")
	}
	if _, err := ReloadTerminologyForLang(rs, "../zh_CN"); err == nil {
		t.Fatal("expected error for path-like lang")
	}
	if _, err := ReloadTerminologyForLang(rs, "C:foo"); err == nil {
		t.Fatal("expected error for drive-like lang")
	}
	if _, err := ReloadTerminologyForLang(rs, "zh-CN"); err == nil {
		t.Fatal("expected error for hyphenated lang")
	}
	rsNoPath := newI18nTestScopeWithModulesPath(t, "")
	if _, err := ReloadTerminologyForLang(rsNoPath, "zh_CN"); err == nil {
		t.Fatal("expected error when modules path is unset")
	}
}

func TestReloadTerminologyForLang_emptyInstalledList(t *testing.T) {
	// Not parallel: stubs package-level listInstalledModulesForTerminology.
	modulesPath := t.TempDir()
	rs := newI18nTestScopeWithModulesPath(t, modulesPath)
	prevList := listInstalledModulesForTerminology
	t.Cleanup(func() { listInstalledModulesForTerminology = prevList })
	listInstalledModulesForTerminology = func(scope.Scope) ([]meta.Module, error) {
		return nil, nil
	}
	stats, err := ReloadTerminologyForLang(rs, "zh_CN")
	if err != nil {
		t.Fatalf("ReloadTerminologyForLang: %v", err)
	}
	if stats.ModulesScanned != 0 || stats.ModulesWithPo != 0 || stats.Upserted != 0 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestReloadTerminologyForLang_listAndSymlinkErrors(t *testing.T) {
	modulesPath := t.TempDir()
	rs := newI18nTestScopeWithModulesPath(t, modulesPath)

	prevList := listInstalledModulesForTerminology
	t.Cleanup(func() { listInstalledModulesForTerminology = prevList })
	listInstalledModulesForTerminology = func(scope.Scope) ([]meta.Module, error) {
		return nil, errors.New("list failed")
	}
	if _, err := ReloadTerminologyForLang(rs, "zh_CN"); err == nil || !strings.Contains(err.Error(), "list failed") {
		t.Fatalf("expected list error, got %v", err)
	}

	listInstalledModulesForTerminology = prevList
	prevEval := evalSymlinks
	t.Cleanup(func() { evalSymlinks = prevEval })
	evalSymlinks = func(string) (string, error) { return "", errors.New("symlink boom") }
	if _, err := ReloadTerminologyForLang(rs, "zh_CN"); err == nil || !strings.Contains(err.Error(), "symlink boom") {
		t.Fatalf("expected symlink error, got %v", err)
	}
	evalSymlinks = prevEval

	// Import error aborts the whole reload (fail-fast).
	badRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(badRoot, "i18n", "zh_CN.po"), 0o755); err != nil {
		t.Fatal(err)
	}
	listInstalledModulesForTerminology = func(scope.Scope) ([]meta.Module, error) {
		return []meta.Module{{Name: "bad", ApplicationStr: "auth", Path: badRoot, Status: meta.Installed}}, nil
	}
	if _, err := ReloadTerminologyForLang(rs, "zh_CN"); err == nil {
		t.Fatal("expected import read error to abort reload")
	}
}

func TestListInstalledModulesForTerminology_default(t *testing.T) {
	t.Parallel()
	if _, err := listInstalledModulesForTerminology(nil); err == nil {
		t.Fatal("expected nil scope error")
	}
	rs := newI18nTestScope(t)
	if err := rs.Session().AutoMigrate(&meta.Module{}); err != nil {
		t.Fatal(err)
	}
	mod := meta.Module{Name: "listed_demo", ApplicationStr: "auth", Status: meta.Installed, Path: t.TempDir()}
	if err := rs.Session().Create(&mod).Error; err != nil {
		t.Fatal(err)
	}
	mods, err := listInstalledModulesForTerminology(rs)
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) < 1 {
		t.Fatalf("expected installed modules, got %d", len(mods))
	}
}

func TestImportModuleLanguagePo_edges(t *testing.T) {
	// Not parallel: stubs package-level upsertPackagedTerms / deleteModuleTerms.
	rs := newI18nTestScope(t)
	reg := store.RegistryFor(rs)

	hit, _, err := importModuleLanguagePo(rs, reg, nil, "", "zh_CN")
	if err != nil || hit {
		t.Fatalf("nil mod: hit=%v err=%v", hit, err)
	}
	hit, _, err = importModuleLanguagePo(rs, reg, &meta.Module{Name: "x", ApplicationStr: ""}, "", "zh_CN")
	if err != nil || hit {
		t.Fatalf("empty application: hit=%v err=%v", hit, err)
	}
	hit, _, err = importModuleLanguagePo(rs, reg, &meta.Module{Name: "web", ApplicationStr: "core"}, "/tmp", "zh_CN")
	if err != nil || hit {
		t.Fatalf("core non-core module skip: hit=%v err=%v", hit, err)
	}
	hit, _, err = importModuleLanguagePo(rs, reg, &meta.Module{Name: "web", ApplicationStr: "auth"}, "", "zh_CN")
	if err != nil || hit {
		t.Fatalf("empty root: hit=%v err=%v", hit, err)
	}

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "i18n", "zh_CN.po"), 0o755); err != nil {
		t.Fatal(err)
	}
	hit, _, err = importModuleLanguagePo(rs, reg, &meta.Module{Name: "demo", ApplicationStr: "auth", Path: root}, "", "zh_CN")
	if err == nil || hit {
		t.Fatalf("expected read error with hit=false, hit=%v err=%v", hit, err)
	}

	poRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(poRoot, "i18n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(poRoot, "i18n", "zh_CN.po"), []byte(`
msgctxt "a@t"
msgid "Hi"
msgstr "你好"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	prevUpsert := upsertPackagedTerms
	t.Cleanup(func() { upsertPackagedTerms = prevUpsert })
	upsertPackagedTerms = func(scope.Scope, *store.Registry, string, string, string, []byte) (*i18nimport.ImportStats, error) {
		return nil, errors.New("forced upsert failure")
	}
	hit, _, err = importModuleLanguagePo(rs, reg, &meta.Module{Name: "demo", ApplicationStr: "auth", Path: poRoot}, "", "zh_CN")
	if err == nil || !hit {
		t.Fatalf("expected upsert error, hit=%v err=%v", hit, err)
	}

	upsertPackagedTerms = func(scope.Scope, *store.Registry, string, string, string, []byte) (*i18nimport.ImportStats, error) {
		return &i18nimport.ImportStats{Upserted: 1}, nil
	}
	prevDelete := deleteModuleTerms
	t.Cleanup(func() { deleteModuleTerms = prevDelete })
	deleteModuleTerms = func(scope.Scope, *store.Registry, string, string) error {
		return errors.New("forced purge failure")
	}
	hit, _, err = importModuleLanguagePo(rs, reg, &meta.Module{Name: "demo", ApplicationStr: "auth", Path: poRoot}, "", "zh_CN")
	if err == nil || !hit || !strings.Contains(err.Error(), "forced purge failure") {
		t.Fatalf("expected purge error, hit=%v err=%v", hit, err)
	}
}

func TestListInstalledModulesForTerminology_findError(t *testing.T) {
	t.Parallel()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "find-err.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	rs := &coverageScopeNoSessionWithDB{db: db}
	if _, err := listInstalledModulesForTerminology(rs); err == nil {
		t.Fatal("expected Find error when meta_module is missing")
	}
}

type coverageScopeNoSessionWithDB struct {
	db *gorm.DB
}

func (s *coverageScopeNoSessionWithDB) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *coverageScopeNoSessionWithDB) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(s)
}
func (s *coverageScopeNoSessionWithDB) Session() *scope.Session {
	return &scope.Session{DB: s.db}
}
func (s *coverageScopeNoSessionWithDB) WithContext(context.Context) scope.Scope { return s }
func (s *coverageScopeNoSessionWithDB) Context() context.Context                { return context.Background() }
func (s *coverageScopeNoSessionWithDB) Logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
