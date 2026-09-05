// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheck_RequiresOptions(t *testing.T) {
	t.Parallel()
	_, err := Check(t.Context(), Options{})
	if err == nil {
		t.Fatal("expected error for empty options")
	}
}

func TestCheck_ServiceOK(t *testing.T) {
	t.Parallel()
	repo, modules := fixtureRoots(t, "service_ok")
	res, err := Check(t.Context(), Options{
		ModulesPath: modules,
		RepoRoot:    repo,
		App:         "demo",
		Scope:       ScopeService,
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if res.HasErrors() {
		t.Fatalf("unexpected errors: %+v", res.Diagnostics)
	}
}

func TestCheck_ServiceTypeError(t *testing.T) {
	t.Parallel()
	repo, modules := fixtureRoots(t, "service_err")
	res, err := Check(t.Context(), Options{
		ModulesPath: modules,
		RepoRoot:    repo,
		App:         "demo",
		Scope:       ScopeService,
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !res.HasErrors() {
		t.Fatal("expected type errors")
	}
	found := false
	for _, d := range res.Diagnostics {
		if d.Category != "error" {
			continue
		}
		if strings.Contains(d.File, "bad.ts") {
			found = true
			if d.Code == 0 {
				t.Fatalf("expected non-zero diagnostic code, got %#v", d)
			}
		}
	}
	if !found {
		t.Fatalf("expected diagnostic for bad.ts, got %#v", res.Diagnostics)
	}
}

func fixtureRoots(t *testing.T, name string) (repoRoot, modulesPath string) {
	t.Helper()
	base := filepath.Join("testdata", name)
	abs, err := filepath.Abs(base)
	if err != nil {
		t.Fatal(err)
	}
	modules := filepath.Join(abs, "modules")
	if st, err := os.Stat(modules); err != nil || !st.IsDir() {
		t.Fatalf("fixture modules missing: %s", modules)
	}
	return abs, modules
}
