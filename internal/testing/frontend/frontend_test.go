// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func seedMathFEApp(t *testing.T, repoRoot, app string) {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	fixture := filepath.Join(filepath.Dir(thisFile), "testdata", "fixtures", "runner")
	web := filepath.Join(repoRoot, "modules", app, "web")
	if err := os.MkdirAll(web, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"math.ts", "math.test.ts"} {
		raw, err := os.ReadFile(filepath.Join(fixture, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(web, name), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestValidateFrontendTestDependenciesNoOp(t *testing.T) {
	if err := ValidateFrontendTestDependencies("", "", false); err != nil {
		t.Fatalf("expected no-op: %v", err)
	}
	if err := ValidateFrontendTestDependencies(t.TempDir(), "auth", true); err != nil {
		t.Fatalf("expected no-op with coverage: %v", err)
	}
}

func TestRunOneAppFrontendTests_NoNpx(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // no node/npx/vitest on PATH
	repoRoot := t.TempDir()
	seedMathFEApp(t, repoRoot, "no_fe_app")
	failed, err := RunOneAppFrontendTests(
		context.Background(),
		repoRoot,
		"no_fe_app",
		"",
		"",
		false,
		false,
		false,
		false,
		"",
		0, 0, 0, 0,
		t.TempDir(),
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if failed {
		t.Fatal("expected QJS FE run without Node on PATH")
	}
}

func TestRunOneAppFrontendTests_CanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	failed, err := RunOneAppFrontendTests(ctx, t.TempDir(), "x", "", "", false, false, false, false, "", 0, 0, 0, 0, t.TempDir(), false)
	if err == nil || !failed {
		t.Fatalf("expected canceled context failure, got failed=%v err=%v", failed, err)
	}
}

func TestSanitizeFrontendAppToken(t *testing.T) {
	if got := sanitizeFrontendAppToken("a/b c"); got != "a_b_c" {
		t.Fatalf("got %q", got)
	}
	if got := sanitizeFrontendAppToken(""); got != "app" {
		t.Fatalf("empty -> app, got %q", got)
	}
}

func TestRunOneAppFrontendTests_IllegalPreflight(t *testing.T) {
	repo := t.TempDir()
	web := filepath.Join(repo, "modules", "demo", "web")
	if err := os.MkdirAll(web, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(web, "bad.test.ts"), []byte("import { it } from 'vitest'\nit('x', () => {})\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	failed, err := RunOneAppFrontendTests(
		context.Background(),
		repo,
		"demo",
		"",
		"",
		false,
		false,
		false,
		false,
		"",
		0, 0, 0, 0,
		t.TempDir(),
		false,
	)
	if err == nil || !failed {
		t.Fatalf("expected failed preflight, failed=%v err=%v", failed, err)
	}
	if !strings.Contains(err.Error(), "illegal mark") {
		t.Fatalf("err = %v", err)
	}
}
