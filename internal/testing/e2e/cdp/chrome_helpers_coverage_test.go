// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cdp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChromeSharedEnabledEnv(t *testing.T) {
	t.Setenv("CHOYSUM_TEST_CHROME_SHARED", "")
	if !chromeSharedEnabled() {
		t.Fatal("empty env should enable sharing")
	}
	for _, v := range []string{"0", "false", "no", "FALSE", " No "} {
		t.Setenv("CHOYSUM_TEST_CHROME_SHARED", v)
		if chromeSharedEnabled() {
			t.Fatalf("CHOYSUM_TEST_CHROME_SHARED=%q should disable sharing", v)
		}
	}
	t.Setenv("CHOYSUM_TEST_CHROME_SHARED", "1")
	if !chromeSharedEnabled() {
		t.Fatal("CHOYSUM_TEST_CHROME_SHARED=1 should enable sharing")
	}
}

func TestExportedStartTestSessionWrappers(t *testing.T) {
	private := StartPrivateTestSession(t)
	if private == nil || private.Context() == nil {
		t.Fatal("StartPrivateTestSession returned nil session")
	}
	if private.shared {
		t.Fatal("private session must not be marked shared")
	}
	shared := StartTestSession(t)
	if shared == nil || shared.Context() == nil {
		t.Fatal("StartTestSession returned nil session")
	}
}

func TestStartTestSessionRespectsSharedDisabled(t *testing.T) {
	t.Setenv("CHOYSUM_TEST_CHROME_SHARED", "0")
	session := startTestSession(t)
	if session.shared {
		t.Fatal("SHARED=0 must return a private session")
	}
}

func TestResetSharedTestTabGuards(t *testing.T) {
	resetSharedTestTab(t, nil)
	resetSharedTestTab(t, &Session{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	resetSharedTestTab(t, &Session{browserCtx: ctx})
}

func TestResetSharedTestTabNewPageError(t *testing.T) {
	session := startTestSession(t)
	old := enableNetworkForPage
	enableNetworkForPage = func(p *Page) error { return errors.New("net enable boom") }
	t.Cleanup(func() { enableNetworkForPage = old })

	rec := &errorRecorder{}
	resetSharedTestTab(rec, session)
	if len(rec.msgs) != 1 || !strings.Contains(rec.msgs[0], "net enable boom") {
		t.Fatalf("expected NewPage error report, got %#v", rec.msgs)
	}
	resetSharedTestTab(nil, session) // t == nil still returns on NewPage error
}

type errorRecorder struct{ msgs []string }

func (e *errorRecorder) Errorf(format string, args ...any) {
	e.msgs = append(e.msgs, fmt.Sprintf(format, args...))
}

// TestZzSharedChromeHelpersCoverage runs last so it can tear down the process
// shared browser and exercise skip/error paths without breaking earlier tests.
func TestZzSharedChromeHelpersCoverage(t *testing.T) {
	// Ensure a live shared session exists, then close it to cover teardown of
	// a real browser + cancel func (not only the nil idempotent path).
	if chromeSharedEnabled() {
		_ = startSharedTestSession(t)
	}
	CloseSharedTestSession()
	CloseSharedTestSession() // idempotent nil path

	t.Run("sharedUnavailable", func(t *testing.T) {
		oldResolve, oldSystem := chromiumResolvePath, chromiumSystemPaths
		t.Cleanup(func() {
			chromiumResolvePath = oldResolve
			chromiumSystemPaths = oldSystem
			CloseSharedTestSession()
		})
		t.Setenv("CHOYSUM_CHROMIUM_PATH", "")
		chromiumResolvePath = func() (string, error) { return "", errors.New("missing") }
		chromiumSystemPaths = func() []string { return nil }
		startSharedTestSession(t)
		t.Fatal("expected skip when chromium unavailable")
	})

	t.Run("sharedStartFail", func(t *testing.T) {
		fake := filepath.Join(t.TempDir(), "not-chrome")
		if err := os.WriteFile(fake, []byte("not a browser\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		oldResolve, oldSystem := chromiumResolvePath, chromiumSystemPaths
		t.Cleanup(func() {
			chromiumResolvePath = oldResolve
			chromiumSystemPaths = oldSystem
			CloseSharedTestSession()
		})
		chromiumResolvePath = func() (string, error) { return fake, nil }
		chromiumSystemPaths = func() []string { return nil }
		startSharedTestSession(t)
		t.Fatal("expected skip after shared start failures")
	})

	t.Run("sharedStartFailThenOK", func(t *testing.T) {
		fake := filepath.Join(t.TempDir(), "not-chrome")
		if err := os.WriteFile(fake, []byte("not a browser\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		good := requireChromium(t)
		oldResolve, oldSystem := chromiumResolvePath, chromiumSystemPaths
		t.Cleanup(func() {
			chromiumResolvePath = oldResolve
			chromiumSystemPaths = oldSystem
			CloseSharedTestSession()
		})
		chromiumResolvePath = func() (string, error) { return "", errors.New("missing") }
		chromiumSystemPaths = func() []string { return []string{fake, good} }
		session := startSharedTestSession(t)
		if session == nil || session.Context() == nil {
			t.Fatal("expected shared session after first candidate failed")
		}
	})
}
