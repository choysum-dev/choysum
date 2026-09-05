// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/buke/typescript-go-internal/v7/pkg/locale"
)

func TestProgram_ServiceOK(t *testing.T) {
	t.Parallel()
	repo, modules := fixtureRoots(t, "service_ok")
	opts, err := BuildCompilerOptions(modules, repo)
	if err != nil {
		t.Fatal(err)
	}
	files, err := CollectRootFiles(modules, "demo", ScopeService)
	if err != nil {
		t.Fatal(err)
	}
	host := newHost(modules, newTypecheckFS(nil))
	program := buildProgram(host, files, opts)
	diags := collectDiagnostics(t.Context(), program)
	for _, d := range diags {
		if d.Category().Name() == "error" {
			t.Fatalf("unexpected error: %s", d.Localize(locale.Default))
		}
	}
}

func TestProgram_ServiceTypeError(t *testing.T) {
	t.Parallel()
	repo, modules := fixtureRoots(t, "service_err")
	opts, err := BuildCompilerOptions(modules, repo)
	if err != nil {
		t.Fatal(err)
	}
	files, err := CollectRootFiles(modules, "demo", ScopeService)
	if err != nil {
		t.Fatal(err)
	}
	host := newHost(modules, newTypecheckFS(nil))
	program := buildProgram(host, files, opts)
	diags := collectDiagnostics(t.Context(), program)
	found := false
	for _, d := range diags {
		if d.Category().Name() != "error" {
			continue
		}
		if f := d.File(); f != nil && strings.Contains(filepath.ToSlash(f.FileName()), "bad.ts") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected error on bad.ts, got %d diagnostics", len(diags))
	}
}
