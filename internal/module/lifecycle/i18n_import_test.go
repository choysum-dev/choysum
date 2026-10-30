// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/import/runner"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type nilContextScope struct {
	scope.Scope
}

func (nilContextScope) Context() context.Context { return nil }

func TestImportModuleTerminology_nilModule(t *testing.T) {
	t.Parallel()
	if err := importModuleTerminology(nil, nil, ""); err != nil {
		t.Fatalf("importModuleTerminology(nil): %v", err)
	}
}

func TestImportModuleTerminology_emptyModuleName(t *testing.T) {
	t.Parallel()
	rs := newI18nTestScope(t)
	if err := importModuleTerminology(rs, &meta.Module{ApplicationStr: "auth"}, ""); err != nil {
		t.Fatalf("empty name: %v", err)
	}
}

func TestImportModuleTerminology_nonCoreEmptyApplication(t *testing.T) {
	t.Parallel()
	rs := newI18nTestScope(t)
	if err := importModuleTerminology(rs, &meta.Module{Name: "demo", ApplicationStr: ""}, ""); err != nil {
		t.Fatalf("empty application: %v", err)
	}
}

func TestImportModuleTerminology_resolvesModuleRootFromModulesPath(t *testing.T) {
	t.Parallel()
	rs := newI18nTestScope(t)
	modulesPath := t.TempDir()
	moduleRoot := filepath.Join(modulesPath, "demo")
	i18nDir := filepath.Join(moduleRoot, "i18n")
	if err := os.MkdirAll(i18nDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(i18nDir, "zh_CN.po"), []byte(`
msgctxt "web/a@title"
msgid "Hello"
msgstr "你好"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	mod := &meta.Module{Name: "demo", ApplicationStr: "auth"}
	if err := importModuleTerminology(rs, mod, modulesPath); err != nil {
		t.Fatalf("importModuleTerminology: %v", err)
	}
}

func TestImportModuleTerminology_importError(t *testing.T) {
	t.Cleanup(func() { importpkg.SetRun(runner.Run) })
	importpkg.SetRun(func(_ context.Context, _ scope.Scope, _ importpkg.Spec) (importpkg.Report, error) {
		return importpkg.Report{}, errors.New("forced terminology failure")
	})

	rs := newI18nTestScope(t)
	mod := &meta.Module{Name: "demo", ApplicationStr: "auth", Path: t.TempDir()}
	err := importModuleTerminology(rs, mod, "")
	if err == nil || !strings.Contains(err.Error(), "import terminology for module demo") {
		t.Fatalf("error = %v, want wrapped import failure", err)
	}
}

func TestImportModuleTerminology_coreImportError(t *testing.T) {
	t.Cleanup(func() { importpkg.SetRun(runner.Run) })
	importpkg.SetRun(func(_ context.Context, _ scope.Scope, _ importpkg.Spec) (importpkg.Report, error) {
		return importpkg.Report{}, errors.New("forced core terminology failure")
	})

	rs := newI18nTestScope(t)
	mod := &meta.Module{Name: "core", ApplicationStr: "core", Path: t.TempDir()}
	err := importModuleTerminology(rs, mod, "")
	if err == nil || !strings.Contains(err.Error(), "import terminology for module core") {
		t.Fatalf("error = %v, want wrapped core import failure", err)
	}
}

func TestRunTerminologyImport_nilScopeContext(t *testing.T) {
	t.Cleanup(func() { importpkg.SetRun(runner.Run) })
	var gotCtx context.Context
	importpkg.SetRun(func(ctx context.Context, _ scope.Scope, _ importpkg.Spec) (importpkg.Report, error) {
		gotCtx = ctx
		return importpkg.Report{Profile: importpkg.ProfileTerminology}, nil
	})

	base := newI18nTestScope(t)
	rs := nilContextScope{Scope: base}
	if err := runTerminologyImport(rs, "auth", "demo", t.TempDir()); err != nil {
		t.Fatalf("runTerminologyImport: %v", err)
	}
	if gotCtx != context.Background() {
		t.Fatalf("ctx = %v, want context.Background()", gotCtx)
	}
}

func TestRunTerminologyImport_usesScopeContext(t *testing.T) {
	t.Cleanup(func() { importpkg.SetRun(runner.Run) })
	wantCtx := context.WithValue(context.Background(), struct{ k string }{"k"}, "v")
	var gotCtx context.Context
	importpkg.SetRun(func(ctx context.Context, _ scope.Scope, _ importpkg.Spec) (importpkg.Report, error) {
		gotCtx = ctx
		return importpkg.Report{Profile: importpkg.ProfileTerminology}, nil
	})

	base := newI18nTestScope(t)
	rs := base.WithContext(wantCtx)
	if err := runTerminologyImport(rs, "auth", "demo", t.TempDir()); err != nil {
		t.Fatalf("runTerminologyImport: %v", err)
	}
	if gotCtx != wantCtx {
		t.Fatalf("ctx = %v, want scope context %v", gotCtx, wantCtx)
	}
}

func TestDeleteModuleTerminology_nilModule(t *testing.T) {
	t.Parallel()
	if err := deleteModuleTerminology(nil, nil); err != nil {
		t.Fatalf("deleteModuleTerminology(nil): %v", err)
	}
}

func TestResolveModuleRoot_helpers(t *testing.T) {
	t.Parallel()
	if got := resolveModuleRoot(&meta.Module{Path: "/custom"}, "/modules", "demo"); got != "/custom" {
		t.Fatalf("mod path = %q", got)
	}
	if got := resolveModuleRoot(nil, "/modules", "demo"); got != filepath.Join("/modules", "demo") {
		t.Fatalf("joined = %q", got)
	}
	if got := resolveModuleRoot(nil, "", "demo"); got != "" {
		t.Fatalf("empty = %q", got)
	}
}
