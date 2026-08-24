// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package bridge_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/defaultscope"
	importbridge "github.com/choysum-dev/choysum/internal/import/bridge"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
)

func TestWithImportProvider_ExposesRun(t *testing.T) {
	engineIface, err := quickjsengine.NewFactory(importbridge.WithImportProvider(jsengine.StaticScopeProvider(nil)))()
	if err != nil {
		t.Fatalf("NewFactory: %v", err)
	}
	engine := engineIface.(*quickjsengine.QuickjsEngine)
	t.Cleanup(func() { _ = engine.Close() })

	got := engine.Ctx.Eval(`typeof $choysum.import.run === "function" && typeof $choysum.orm.call === "function"`)
	defer got.Free()
	if got.IsException() {
		t.Fatalf("Eval: %v", engine.Ctx.Exception())
	}
	if !got.ToBool() {
		t.Fatal("expected $choysum.import.run and $choysum.orm.call to be functions")
	}

	promise := engine.Ctx.Eval(`$choysum.import.run({profile:"record",caller:"user",policy:"atomic",model:"base.Country",source:{format:"csv",path:"missing.csv"}})`)
	defer promise.Free()
	if promise.IsException() {
		t.Fatalf("run call: %v", engine.Ctx.Exception())
	}
}

func TestRun_RejectsPathTraversal(t *testing.T) {
	root := t.TempDir()
	modulesPath := filepath.Join(root, "modules")
	cfg := &config.Config{
		Db:          &config.DbConfig{Dialect: "sqlite", DSN: filepath.Join(t.TempDir(), "path.db")},
		ModulesPath: modulesPath,
	}
	runtimeScope := defaultscope.NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(cfg), nil)

	_, err := importbridge.Run(context.Background(), runtimeScope, importpkg.Spec{
		Profile: importpkg.ProfileRecord,
		Caller:  importpkg.CallerUser,
		Policy:  importpkg.PolicyAtomic,
		Model:   "base.Country",
		Source:  importpkg.Source{Format: "csv", Path: "../secret.csv"},
	})
	if err == nil {
		t.Fatal("expected path traversal error")
	}
	if !strings.Contains(err.Error(), "escapes modules root") {
		t.Fatalf("error = %v, want escapes modules root", err)
	}
}

func TestRun_RejectsAbsoluteSourcePath(t *testing.T) {
	root := t.TempDir()
	cfg := &config.Config{
		Db:          &config.DbConfig{Dialect: "sqlite", DSN: filepath.Join(t.TempDir(), "abs.db")},
		ModulesPath: filepath.Join(root, "modules"),
	}
	runtimeScope := defaultscope.NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(cfg), nil)

	_, err := importbridge.Run(context.Background(), runtimeScope, importpkg.Spec{
		Profile: importpkg.ProfileRecord,
		Caller:  importpkg.CallerUser,
		Policy:  importpkg.PolicyAtomic,
		Model:   "base.Country",
		Source:  importpkg.Source{Format: "csv", Path: filepath.Join(root, "modules", "x.csv")},
	})
	if err == nil {
		t.Fatal("expected absolute path rejection")
	}
	if !strings.Contains(err.Error(), "must be relative") {
		t.Fatalf("error = %v, want must be relative", err)
	}
}
