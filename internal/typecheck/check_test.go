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
	_, err := Check(t.Context(), Options{})
	if err == nil {
		t.Fatal("expected error for empty options")
	}
}

func TestCheck_ServiceOK(t *testing.T) {
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

func TestCheck_WebTSFixture(t *testing.T) {
	repo, modules := fixtureRoots(t, "web_ts_ok")
	res, err := Check(t.Context(), Options{
		ModulesPath: modules,
		RepoRoot:    repo,
		App:         "demo",
		Scope:       ScopeNoVue,
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if res.HasErrors() {
		t.Fatalf("unexpected errors: %+v", res.Diagnostics)
	}

	repoErr, modulesErr := fixtureRoots(t, "web_ts_err")
	resErr, err := Check(t.Context(), Options{
		ModulesPath: modulesErr,
		RepoRoot:    repoErr,
		App:         "demo",
		Scope:       ScopeNoVue,
	})
	if err != nil {
		t.Fatalf("Check err fixture: %v", err)
	}
	if !resErr.HasErrors() {
		t.Fatal("expected type errors in web_ts_err")
	}
	found := false
	for _, d := range resErr.Diagnostics {
		if d.Category == "error" && strings.Contains(d.File, "bad.ts") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected bad.ts diagnostic, got %#v", resErr.Diagnostics)
	}
}

func fixtureRoots(t *testing.T, name string) (repoRoot, modulesPath string) {
	t.Helper()
	src := filepath.Join("testdata", name)
	if st, err := os.Stat(src); err != nil || !st.IsDir() {
		t.Fatalf("fixture missing: %s", src)
	}
	dst := t.TempDir()
	if err := copyTree(src, dst); err != nil {
		t.Fatal(err)
	}
	modules := filepath.Join(dst, "modules")
	if st, err := os.Stat(modules); err != nil || !st.IsDir() {
		t.Fatalf("fixture modules missing: %s", modules)
	}
	return dst, modules
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
