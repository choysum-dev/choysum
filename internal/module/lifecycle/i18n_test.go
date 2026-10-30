// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/defaultscope"
	i18nmodels "github.com/choysum-dev/choysum/internal/i18n/models"
	"github.com/choysum-dev/choysum/internal/i18n/store"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func newI18nTestScope(t *testing.T) scope.Scope {
	t.Helper()
	store.ResetSharedRegistryForTests()
	cfg := &config.Config{
		Db: &config.DbConfig{
			Dialect: "sqlite",
			DSN:     filepath.Join(t.TempDir(), "lifecycle_i18n.db"),
		},
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
