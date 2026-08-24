// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package term_test

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/defaultscope"
	i18nimport "github.com/choysum-dev/choysum/internal/i18n/import"
	i18nmodels "github.com/choysum-dev/choysum/internal/i18n/models"
	"github.com/choysum-dev/choysum/internal/i18n/store"
	_ "github.com/choysum-dev/choysum/internal/import/runner"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func newTermEqScope(t *testing.T) scope.Scope {
	t.Helper()
	store.ResetSharedRegistryForTests()
	cfg := &config.Config{
		Db: &config.DbConfig{
			Dialect: "sqlite",
			DSN:     filepath.Join(t.TempDir(), "term_eq.db"),
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

func TestWriterEquivalence_UpsertPackagedTermsVsImportRun(t *testing.T) {
	poBody := []byte(`
msgid ""
msgstr "Language: zh_CN\n"

msgctxt "web/a@title"
msgid "Hello"
msgstr "你好"
`)

	directScope := newTermEqScope(t)
	runScope := newTermEqScope(t)

	moduleRoot := t.TempDir()
	i18nDir := filepath.Join(moduleRoot, "i18n")
	if err := os.MkdirAll(i18nDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(i18nDir, "zh_CN.po"), poBody, 0o644); err != nil {
		t.Fatal(err)
	}

	regDirect := store.RegistryFor(directScope)
	if _, err := i18nimport.UpsertPackagedTerms(directScope, regDirect, "auth", "demo", "zh_CN", poBody); err != nil {
		t.Fatalf("UpsertPackagedTerms: %v", err)
	}

	_, err := importpkg.Run(context.Background(), runScope, importpkg.Spec{
		Profile:     importpkg.ProfileTerminology,
		Caller:      importpkg.CallerLifecycle,
		Policy:      importpkg.PolicyAtomic,
		Module:      "demo",
		Application: "auth",
		Source:      importpkg.Source{Format: "po", Path: moduleRoot},
	})
	if err != nil {
		t.Fatalf("import.Run: %v", err)
	}

	var directRow, runRow i18nmodels.TranslationTerm
	if err := directScope.Session().Table("auth_translation_term").Where("src = ?", "Hello").Take(&directRow).Error; err != nil {
		t.Fatalf("direct row: %v", err)
	}
	if err := runScope.Session().Table("auth_translation_term").Where("src = ?", "Hello").Take(&runRow).Error; err != nil {
		t.Fatalf("run row: %v", err)
	}
	if directRow.Value != runRow.Value || directRow.Source != runRow.Source || directRow.Kind != runRow.Kind {
		t.Fatalf("direct=%+v run=%+v", directRow, runRow)
	}
}
