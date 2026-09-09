// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cdp

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestResolveChromiumPathMissingBinaryMessage(t *testing.T) {
	t.Setenv("CHOYSUM_CHROMIUM_PATH", "")
	t.Setenv("CHOYSUM_HOME", t.TempDir())
	t.Setenv("CHOYSUM_CHROMIUM_PATH", t.TempDir()+"/no-such-chromium-binary")

	_, err := ResolveChromiumPath()
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	for _, want := range []string{"CHOYSUM_CHROMIUM_PATH", "choysum test e2e --install-browser", "scripts/ci/install_chromium.py"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q missing %q", msg, want)
		}
	}
}

func TestMissingBinaryErrorMentionsInstall(t *testing.T) {
	err := missingBinaryError("chromium binary not found")
	msg := err.Error()
	if !strings.Contains(msg, "CHOYSUM_CHROMIUM_PATH") || !strings.Contains(msg, "install_chromium.py") {
		t.Fatalf("unexpected: %q", msg)
	}
}

func TestResolveChromiumPathSystemOrEnv(t *testing.T) {
	path, err := ResolveChromiumPath()
	if err != nil {
		t.Skipf("chromium unavailable: %v", err)
	}
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		t.Fatalf("resolved path not a file: %q err=%v", path, err)
	}
}

func TestResolveChromiumPathEnvOverride(t *testing.T) {
	real, err := ResolveChromiumPath()
	if err != nil {
		t.Skipf("chromium unavailable: %v", err)
	}
	t.Setenv("CHOYSUM_CHROMIUM_PATH", real)
	got, err := ResolveChromiumPath()
	if err != nil {
		t.Fatalf("ResolveChromiumPath: %v", err)
	}
	if got != real {
		t.Fatalf("got %q want %q", got, real)
	}
}

func TestResolveChromiumPathCachedHome(t *testing.T) {
	real, err := ResolveChromiumPath()
	if err != nil {
		t.Skipf("chromium unavailable: %v", err)
	}
	home := t.TempDir()
	t.Setenv("CHOYSUM_HOME", home)
	t.Setenv("CHOYSUM_CHROMIUM_PATH", "")

	base := filepath.Join(home, "browsers", "chromium-"+PinnedCfTRevision)
	cands := cachedBinaryCandidates(base)
	if len(cands) == 0 {
		t.Fatal("no cached candidates")
	}
	target := cands[0]
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, target); err != nil {
		t.Fatalf("seed cached binary symlink: %v", err)
	}

	got, err := ResolveChromiumPath()
	if err != nil {
		t.Fatalf("ResolveChromiumPath: %v", err)
	}
	if got != target {
		t.Fatalf("got %q want cached %q", got, target)
	}
}

func TestStartCloseContextExecPath(t *testing.T) {
	session := startTestSession(t)
	if session.Context() == nil {
		t.Fatal("nil browser context")
	}
	if session.ExecPath() == "" {
		t.Fatal("empty exec path")
	}
	if err := session.Context().Err(); err != nil {
		t.Fatalf("context already dead: %v", err)
	}
}

func TestWantHeadless(t *testing.T) {
	t.Setenv("CHOYSUM_E2E_HEADED", "")
	if !WantHeadless(StartOptions{}) {
		t.Fatal("default should be headless")
	}
	for _, v := range []string{"0", "false", "no", "FALSE", " off "} {
		t.Setenv("CHOYSUM_E2E_HEADED", v)
		if !WantHeadless(StartOptions{}) {
			t.Fatalf("CHOYSUM_E2E_HEADED=%q should stay headless", v)
		}
	}
	for _, v := range []string{"1", "true", "yes", "TRUE"} {
		t.Setenv("CHOYSUM_E2E_HEADED", v)
		if WantHeadless(StartOptions{}) {
			t.Fatalf("CHOYSUM_E2E_HEADED=%q should be headed", v)
		}
	}
	// Explicit opts win over ambient env (unit tests / helpers force headless).
	t.Setenv("CHOYSUM_E2E_HEADED", "1")
	h := true
	if !WantHeadless(StartOptions{Headless: &h}) {
		t.Fatal("explicit Headless=true must win over CHOYSUM_E2E_HEADED=1")
	}
	h = false
	t.Setenv("CHOYSUM_E2E_HEADED", "0")
	if WantHeadless(StartOptions{Headless: &h}) {
		t.Fatal("explicit Headless=false must win over CHOYSUM_E2E_HEADED=0")
	}
}

func TestStartNilContextHeadless(t *testing.T) {
	execPath := requireChromium(t)
	t.Setenv("CHOYSUM_E2E_HEADED", "0")
	session, err := Start(nil, StartOptions{ExecPath: execPath})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer session.Close()
	if !session.Headless() {
		t.Fatal("expected headless session")
	}
}

func TestStartBadExecPath(t *testing.T) {
	_, err := Start(context.Background(), StartOptions{ExecPath: filepath.Join(t.TempDir(), "missing-chrome")})
	if err == nil {
		t.Fatal("expected start error")
	}
	if !strings.Contains(err.Error(), "cdp: start browser") {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestNilSessionMethods(t *testing.T) {
	var s *Session
	s.Close()
	if s.Context() != nil {
		t.Fatal("expected nil context")
	}
	if s.ExecPath() != "" {
		t.Fatal("expected empty exec path")
	}
}

func TestWithTimeout(t *testing.T) {
	ctx := context.Background()
	c1, cancel1 := WithTimeout(ctx, 0)
	cancel1()
	if c1 != ctx {
		t.Fatal("expected same ctx for d<=0")
	}
	c2, cancel2 := WithTimeout(ctx, time.Millisecond)
	defer cancel2()
	if _, ok := c2.Deadline(); !ok {
		t.Fatal("expected deadline")
	}
}

func TestCachedAndSystemCandidatesNonEmpty(t *testing.T) {
	cands := cachedBinaryCandidates(filepath.Join(t.TempDir(), "base"))
	if len(cands) == 0 {
		t.Fatal("cachedBinaryCandidates empty")
	}
	sys := systemChromeCandidates()
	switch runtime.GOOS {
	case "darwin", "linux":
		if len(sys) == 0 {
			t.Fatal("systemChromeCandidates empty")
		}
	}
}

func TestResolveChromiumPathMissingUsesHomeFallback(t *testing.T) {
	t.Setenv("CHOYSUM_CHROMIUM_PATH", "")
	t.Setenv("CHOYSUM_HOME", "")
	// May succeed via system Chrome; just ensure no panic.
	_, _ = ResolveChromiumPath()
}
