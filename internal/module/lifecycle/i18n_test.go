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

	i18nmodels "github.com/choysum-dev/choysum/internal/i18n/models"
	"github.com/choysum-dev/choysum/internal/i18n/store"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type i18nTestScope struct {
	ctx     context.Context
	logger  *slog.Logger
	session *scope.Session
}

func (s *i18nTestScope) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *i18nTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(s)
}
func (s *i18nTestScope) Session() *scope.Session { return s.session }
func (s *i18nTestScope) WithContext(ctx context.Context) scope.Scope {
	if ctx == nil {
		ctx = s.ctx
	}
	return &i18nTestScope{ctx: ctx, logger: s.logger, session: s.session}
}
func (s *i18nTestScope) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}
func (s *i18nTestScope) Logger() *slog.Logger { return s.logger }

func newI18nTestScope(t *testing.T) *i18nTestScope {
	t.Helper()
	store.ResetSharedRegistryForTests()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "lifecycle_i18n.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return &i18nTestScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
}

func TestImportAndDeleteModuleTerminology(t *testing.T) {
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

	mod := &meta.IrModule{
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

func TestImportModuleTerminologySkipsCoreAndMissingDir(t *testing.T) {
	rs := newI18nTestScope(t)
	if err := importModuleTerminology(rs, &meta.IrModule{Name: "core", ApplicationStr: "core"}, ""); err != nil {
		t.Fatal(err)
	}
	if err := importModuleTerminology(rs, &meta.IrModule{Name: "demo", ApplicationStr: "auth", Path: filepath.Join(t.TempDir(), "nope")}, ""); err != nil {
		t.Fatal(err)
	}
}
