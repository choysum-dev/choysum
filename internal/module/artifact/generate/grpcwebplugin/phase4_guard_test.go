// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package grpcwebplugin

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPhase4Guard_NoV8GoDependency(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	goModPath := filepath.Join(repoRoot, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	if strings.Contains(string(content), "github.com/tommie/v8go") {
		t.Fatalf("unexpected legacy dependency found in go.mod: github.com/tommie/v8go")
	}
}

func TestPhase4Guard_NoV8GoResidueInGoSum(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	goSumPath := filepath.Join(repoRoot, "go.sum")
	content, err := os.ReadFile(goSumPath)
	if err != nil {
		t.Fatalf("read go.sum: %v", err)
	}
	if strings.Contains(string(content), "github.com/tommie/v8go") {
		t.Fatalf("unexpected legacy dependency residue found in go.sum: github.com/tommie/v8go")
	}
}

func TestPhase4Guard_LegacyGrpcJsEngineRemoved(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	legacyPath := filepath.Join(repoRoot, "internal", "module", "generator", "grpcwebplugin", "jsengine")
	_, err := os.Stat(legacyPath)
	if err == nil {
		t.Fatalf("legacy path should be removed: %s", legacyPath)
	}
	if !os.IsNotExist(err) {
		t.Fatalf("stat legacy path %s: %v", legacyPath, err)
	}
}

func TestPhase4Guard_LegacyProtocgenesPackageRemoved(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	legacyPath := filepath.Join(repoRoot, "pkg", "jsengine", "scripts", "protocgenes")
	_, err := os.Stat(legacyPath)
	if err == nil {
		t.Fatalf("legacy path should be removed: %s", legacyPath)
	}
	if !os.IsNotExist(err) {
		t.Fatalf("stat legacy path %s: %v", legacyPath, err)
	}
}

func TestPhase4Guard_ParityScriptsRemoved(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	legacyPath := filepath.Join(repoRoot, "internal", "module", "generator", "grpcwebplugin", "parityscripts")
	_, err := os.Stat(legacyPath)
	if err == nil {
		t.Fatalf("legacy path should be removed: %s", legacyPath)
	}
	if !os.IsNotExist(err) {
		t.Fatalf("stat legacy path %s: %v", legacyPath, err)
	}
}

type phase4GuardViolationError struct {
	msg  string
	path string
}

func (e *phase4GuardViolationError) Error() string {
	return e.msg + ": " + e.path
}

func repoRootFromThisFile(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(thisFile)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("resolve repo root from %s: go.mod not found", thisFile)
		}
		dir = parent
	}
}
