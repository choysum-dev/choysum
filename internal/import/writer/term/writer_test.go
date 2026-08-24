// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package term

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/defaultscope"
	"github.com/choysum-dev/choysum/internal/i18n/store"
	"github.com/choysum-dev/choysum/internal/import/plan"
	initdataplan "github.com/choysum-dev/choysum/internal/import/plan/initdata"
	planstub "github.com/choysum-dev/choysum/internal/import/plan/stub"
	termplan "github.com/choysum-dev/choysum/internal/import/plan/term"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func newWriterTestScope(t *testing.T) scope.Scope {
	t.Helper()
	store.ResetSharedRegistryForTests()
	cfg := &config.Config{
		Db: &config.DbConfig{
			Dialect: "sqlite",
			DSN:     filepath.Join(t.TempDir(), "term-writer.db"),
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

func TestWriter_WriteSuccess(t *testing.T) {
	runtimeScope := newWriterTestScope(t)
	dir := t.TempDir()
	poPath := filepath.Join(dir, "zh_CN.po")
	if err := os.WriteFile(poPath, []byte(`
msgid ""
msgstr "Language: zh_CN\n"

msgctxt "web/a@title"
msgid "Hello"
msgstr "你好"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	unit := termplan.Unit{
		Application: "auth",
		ModuleName:  "demo",
		Lang:        "zh_CN",
		PoPath:      poPath,
	}
	if err := (Writer{}).Write(context.Background(), runtimeScope, []plan.Unit{unit}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	var count int64
	if err := runtimeScope.Session().Table("auth_translation_term").Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("row count = %d, want 1", count)
	}
}

func TestWriter_WriteUnexpectedUnitType(t *testing.T) {
	err := (Writer{}).Write(context.Background(), nil, []plan.Unit{planstub.Unit{Index: 1}})
	var impErr *importpkg.Error
	if !errors.As(err, &impErr) || impErr.Code != importpkg.CodeInvalidFormat {
		t.Fatalf("Write() error = %v, want CodeInvalidFormat", err)
	}
}

func TestWriter_WriteWrongPlanUnitType(t *testing.T) {
	err := (Writer{}).Write(context.Background(), nil, []plan.Unit{initdataplan.Unit{Index: 1}})
	var impErr *importpkg.Error
	if !errors.As(err, &impErr) || impErr.Code != importpkg.CodeInvalidFormat {
		t.Fatalf("Write() error = %v, want CodeInvalidFormat", err)
	}
}

func TestWriter_WriteMissingPOFile(t *testing.T) {
	runtimeScope := newWriterTestScope(t)
	unit := termplan.Unit{
		Application: "auth",
		ModuleName:  "demo",
		Lang:        "zh_CN",
		PoPath:      filepath.Join(t.TempDir(), "missing.po"),
	}
	err := (Writer{}).Write(context.Background(), runtimeScope, []plan.Unit{unit})
	var impErr *importpkg.Error
	if !errors.As(err, &impErr) || impErr.Code != importpkg.CodeInvalidFormat {
		t.Fatalf("Write() error = %v, want read PO error", err)
	}
}

func TestWriter_WriteUpsertError(t *testing.T) {
	unit := termplan.Unit{
		Application: "auth",
		ModuleName:  "demo",
		Lang:        "zh_CN",
		PoPath:      filepath.Join(t.TempDir(), "empty.po"),
	}
	if err := os.WriteFile(unit.PoPath, []byte("msgid \"\"\nmsgstr \"\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := (Writer{}).Write(context.Background(), nil, []plan.Unit{unit})
	if err == nil || err.Error() != "import po: missing runtime session" {
		t.Fatalf("Write() error = %v, want missing session", err)
	}
}

func TestWriter_WriteNilUnits(t *testing.T) {
	if err := (Writer{}).Write(context.Background(), nil, nil); err != nil {
		t.Fatalf("Write(nil): %v", err)
	}
}
