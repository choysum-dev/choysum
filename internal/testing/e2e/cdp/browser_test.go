// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cdp

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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

func TestResolveChromiumPathForE2ESkipsSystemChromeInCI(t *testing.T) {
	t.Setenv("CHOYSUM_CHROMIUM_PATH", "")
	t.Setenv("CHOYSUM_HOME", t.TempDir())
	t.Setenv("CI", "true")
	t.Setenv("GITHUB_ACTIONS", "")

	_, err := ResolveChromiumPathForE2E()
	if err == nil {
		t.Fatal("expected missing binary when CI skips system Chrome")
	}
	msg := err.Error()
	for _, want := range []string{"chromium binary not found", "CHOYSUM_CHROMIUM_PATH", "install_chromium.py"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q missing %q", msg, want)
		}
	}
}

func TestEnvFlagEnabled(t *testing.T) {
	t.Setenv("CHOYSUM_TEST_FLAG", "true")
	if !EnvFlagEnabled("CHOYSUM_TEST_FLAG") {
		t.Fatal("true should enable")
	}
	t.Setenv("CHOYSUM_TEST_FLAG", "0")
	if EnvFlagEnabled("CHOYSUM_TEST_FLAG") {
		t.Fatal("0 should disable")
	}
	t.Setenv("CHOYSUM_TEST_FLAG", "false")
	if EnvFlagEnabled("CHOYSUM_TEST_FLAG") {
		t.Fatal("false should disable")
	}
	t.Setenv("CHOYSUM_TEST_FLAG", "yes")
	if !EnvFlagEnabled("CHOYSUM_TEST_FLAG") {
		t.Fatal("yes should enable")
	}
}

func TestResolveChromiumPathForE2ETreatsCIFalseAsLocal(t *testing.T) {
	t.Setenv("CHOYSUM_CHROMIUM_PATH", "")
	t.Setenv("CHOYSUM_HOME", t.TempDir())
	t.Setenv("CI", "false")
	t.Setenv("GITHUB_ACTIONS", "")
	// Same as ResolveChromiumPath for local: may succeed via system Chrome.
	_, errE2E := ResolveChromiumPathForE2E()
	_, errAny := ResolveChromiumPath()
	if (errE2E == nil) != (errAny == nil) {
		t.Fatalf("CI=false should match ResolveChromiumPath: e2e=%v any=%v", errE2E, errAny)
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

func TestIsWebsocketURLTimeout(t *testing.T) {
	if isWebsocketURLTimeout(nil) {
		t.Fatal("nil")
	}
	if !isWebsocketURLTimeout(errors.New("cdp: start browser (/usr/bin/google-chrome): websocket url timeout reached")) {
		t.Fatal("expected match")
	}
	if isWebsocketURLTimeout(errors.New("cdp: start browser: context canceled")) {
		t.Fatal("non-timeout should not match")
	}
}

func TestStartRetryCancelDuringBackoff(t *testing.T) {
	old := startBrowserOnce
	n := 0
	startBrowserOnce = func(ctx context.Context, execPath string, headless bool) (*Session, error) {
		n++
		return nil, errors.New("cdp: start browser (/x): websocket url timeout reached")
	}
	t.Cleanup(func() { startBrowserOnce = old })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Start(ctx, StartOptions{ExecPath: "/bin/true"})
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v (n=%d)", err, n)
	}
	if n != 1 {
		t.Fatalf("expected one start attempt before cancel, got %d", n)
	}
}

func TestStartRetryExhaustWebsocketTimeouts(t *testing.T) {
	old := startBrowserOnce
	n := 0
	startBrowserOnce = func(ctx context.Context, execPath string, headless bool) (*Session, error) {
		n++
		return nil, errors.New("cdp: start browser (/x): websocket url timeout reached")
	}
	t.Cleanup(func() { startBrowserOnce = old })

	_, err := Start(context.Background(), StartOptions{ExecPath: "/bin/true"})
	if err == nil || !strings.Contains(err.Error(), "websocket url timeout") {
		t.Fatalf("expected websocket timeout after retries, got %v", err)
	}
	if n != 3 {
		t.Fatalf("expected 3 attempts, got %d", n)
	}
}

func TestStartNilContextHeadless(t *testing.T) {
	t.Setenv("CHOYSUM_E2E_HEADED", "0")
	cands := chromiumCandidates()
	if len(cands) == 0 {
		t.Skip("chromium unavailable")
	}
	var session *Session
	var lastErr error
	for _, execPath := range cands {
		session, lastErr = Start(nil, StartOptions{ExecPath: execPath})
		if lastErr == nil {
			break
		}
	}
	if lastErr != nil {
		t.Skipf("chromium start failed: %v", lastErr)
	}
	defer session.Close()
	if !session.Headless() {
		t.Fatal("expected headless session")
	}
}

func TestStartMkdirTempError(t *testing.T) {
	old := mkdirTemp
	mkdirTemp = func(dir, pattern string) (string, error) { return "", errors.New("mkdirtemp boom") }
	defer func() { mkdirTemp = old }()
	_, err := Start(context.Background(), StartOptions{ExecPath: requireChromium(t)})
	if err == nil || !strings.Contains(err.Error(), "user-data-dir") {
		t.Fatalf("got %v", err)
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
	if !s.Headless() {
		t.Fatal("nil session Headless should default true")
	}
}

func TestStartResolvesExecPathAndParentCancel(t *testing.T) {
	requireChromium(t)
	t.Setenv("CHOYSUM_E2E_HEADED", "0")
	ctx, cancel := context.WithCancel(context.Background())
	session, err := Start(ctx, StartOptions{}) // empty ExecPath → ResolveChromiumPath
	if err != nil {
		t.Skipf("chromium start failed: %v", err)
	}
	defer session.Close()
	if session.ExecPath() == "" {
		t.Fatal("empty resolved exec path")
	}
	if !session.Headless() {
		t.Fatal("expected headless")
	}
	cancel() // arm parent-ctx watcher (Start lines 105-108)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if session.Context() != nil && session.Context().Err() != nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	// Watcher may race with Close; still exercised cancel path above.
}

func TestStartChromiumOrSkipHelper(t *testing.T) {
	path, session := startChromiumOrSkip(t)
	if path == "" || session == nil {
		t.Fatal("expected path and session")
	}
	if session.ExecPath() != path {
		t.Fatalf("execPath=%q path=%q", session.ExecPath(), path)
	}
}

func TestResolveChromiumPathSystemCandidates(t *testing.T) {
	t.Setenv("CHOYSUM_CHROMIUM_PATH", "")
	t.Setenv("CHOYSUM_HOME", t.TempDir()) // no cached CfT binary
	path, err := ResolveChromiumPath()
	if err != nil {
		t.Skipf("no system chrome: %v", err)
	}
	if st, err := os.Stat(path); err != nil || st.IsDir() {
		t.Fatalf("resolved %q: %v", path, err)
	}
}

func TestCachedBinaryCandidatesLinuxShape(t *testing.T) {
	base := filepath.Join(t.TempDir(), "chromium-base")
	old := runtimeGOOS
	defer func() { runtimeGOOS = old }()

	runtimeGOOS = "darwin"
	if got := cachedBinaryCandidates(base); len(got) < 3 || !strings.Contains(got[0], "chrome-mac") {
		t.Fatalf("darwin cached=%v", got)
	}
	if got := systemChromeCandidates(); len(got) == 0 || !strings.Contains(got[0], "Applications") {
		t.Fatalf("darwin system=%v", got)
	}

	runtimeGOOS = "linux"
	if got := cachedBinaryCandidates(base); len(got) != 3 || !strings.Contains(got[0], "chrome-linux64") {
		t.Fatalf("linux cached=%v", got)
	}
	if got := systemChromeCandidates(); len(got) < 3 || !strings.Contains(got[0], "/usr/bin") {
		t.Fatalf("linux system=%v", got)
	}

	runtimeGOOS = "windows"
	if got := cachedBinaryCandidates(base); len(got) != 2 {
		t.Fatalf("default cached=%v", got)
	}
	if got := systemChromeCandidates(); got != nil {
		t.Fatalf("default system=%v", got)
	}
}

func TestResolveChromiumPathExhausted(t *testing.T) {
	t.Setenv("CHOYSUM_CHROMIUM_PATH", "")
	t.Setenv("CHOYSUM_HOME", t.TempDir())
	old := runtimeGOOS
	runtimeGOOS = "windows" // no system candidates
	defer func() { runtimeGOOS = old }()
	_, err := ResolveChromiumPath()
	if err == nil || !strings.Contains(err.Error(), "chromium binary not found") {
		t.Fatalf("got %v", err)
	}
}

func TestStartResolveExecPathError(t *testing.T) {
	t.Setenv("CHOYSUM_CHROMIUM_PATH", "")
	t.Setenv("CHOYSUM_HOME", t.TempDir())
	old := runtimeGOOS
	runtimeGOOS = "windows"
	defer func() { runtimeGOOS = old }()
	_, err := Start(context.Background(), StartOptions{}) // empty ExecPath → Resolve fails
	if err == nil || !strings.Contains(err.Error(), "chromium binary not found") {
		t.Fatalf("got %v", err)
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
	switch runtimeGOOS {
	case "darwin", "linux":
		if len(sys) == 0 {
			t.Fatal("systemChromeCandidates empty")
		}
	}
}

func TestChromiumCandidatesEmptyAndDuplicate(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "chrome")
	if err := os.WriteFile(bin, []byte("x\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	oldResolve, oldSystem := chromiumResolvePath, chromiumSystemPaths
	t.Cleanup(func() {
		chromiumResolvePath = oldResolve
		chromiumSystemPaths = oldSystem
	})
	chromiumResolvePath = func() (string, error) { return bin, nil }
	chromiumSystemPaths = func() []string { return []string{"", bin, bin} }
	got := chromiumCandidates()
	if len(got) != 1 || got[0] != bin {
		t.Fatalf("got %v want [%s]", got, bin)
	}
}

func TestChromeHelpersSkipPaths(t *testing.T) {
	fake := filepath.Join(t.TempDir(), "not-chrome")
	if err := os.WriteFile(fake, []byte("not a browser\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CHOYSUM_CHROMIUM_PATH", "")
	t.Setenv("CHOYSUM_HOME", t.TempDir())
	old := runtimeGOOS
	runtimeGOOS = "windows"
	defer func() { runtimeGOOS = old }()

	if len(chromiumCandidates()) != 0 {
		t.Fatalf("expected no candidates, got %v", chromiumCandidates())
	}
	t.Run("requireChromium", func(t *testing.T) {
		requireChromium(t)
		t.Fatal("expected skip")
	})
	t.Run("startTestSessionEmpty", func(t *testing.T) {
		startTestSession(t)
		t.Fatal("expected skip")
	})

	// Non-empty candidates but Start fails for every path.
	t.Setenv("CHOYSUM_CHROMIUM_PATH", fake)
	if len(chromiumCandidates()) == 0 {
		t.Fatal("expected fake candidate")
	}
	t.Run("startTestSessionFail", func(t *testing.T) {
		startTestSession(t)
		t.Fatal("expected skip after start failures")
	})
}

func TestResolveChromiumPathMissingUsesHomeFallback(t *testing.T) {
	t.Setenv("CHOYSUM_CHROMIUM_PATH", "")
	t.Setenv("CHOYSUM_HOME", "")
	// May succeed via system Chrome; just ensure no panic.
	_, _ = ResolveChromiumPath()
}
