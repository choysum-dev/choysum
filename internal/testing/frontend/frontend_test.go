// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

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
	// Empty discover (no modules/<app>/web) → success without exec'ing Node.
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
		t.Fatal("empty discover should not fail")
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

func TestWarnIllegalFrontendMarks_NoPanic(t *testing.T) {
	repo := t.TempDir()
	web := filepath.Join(repo, "modules", "demo", "web")
	if err := os.MkdirAll(web, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(web, "ok.test.ts"), []byte("it('ok', () => {})\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	warnIllegalFrontendMarks(repo, "demo")
}
