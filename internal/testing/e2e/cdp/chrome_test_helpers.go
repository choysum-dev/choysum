// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cdp

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
)

// Overridable in tests to force empty/duplicate candidate paths.
var (
	chromiumResolvePath = ResolveChromiumPath
	chromiumSystemPaths = systemChromeCandidates
)

var (
	sharedChromeOnce   sync.Once
	sharedChromeMu     sync.Mutex
	sharedChrome       *Session
	sharedChromeCancel context.CancelFunc
	sharedChromeErr    error
)

func chromiumCandidates() []string {
	seen := map[string]bool{}
	var out []string
	add := func(p string) {
		if p == "" || seen[p] {
			return
		}
		if st, err := os.Stat(p); err != nil || st.IsDir() {
			return
		}
		seen[p] = true
		out = append(out, p)
	}
	// Prefer system Chrome for local unit tests: pinned CfT caches can crash on
	// some macOS hosts and surface "unexpected quit" dialogs during go test.
	for _, p := range chromiumSystemPaths() {
		add(p)
	}
	if p, err := chromiumResolvePath(); err == nil {
		add(p)
	}
	return out
}

// requireChromium returns a Chromium binary path, skipping if none exist.
func requireChromium(t *testing.T) string {
	t.Helper()
	cands := chromiumCandidates()
	if len(cands) == 0 {
		t.Skip("chromium unavailable")
	}
	return cands[0]
}

// chromeSharedEnabled reports whether package tests should reuse one Chromium.
// Set CHOYSUM_TEST_CHROME_SHARED=0|false|no|off to force a fresh browser per session.
func chromeSharedEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("CHOYSUM_TEST_CHROME_SHARED"))) {
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

// startTestSession returns a Chromium session for package tests.
// By default one shared headless browser is reused for the process
// (CHOYSUM_TEST_CHROME_SHARED=0 disables sharing). Callers that Close the
// browser mid-test must use startPrivateTestSession instead.
func startTestSession(t *testing.T) *Session {
	t.Helper()
	if !chromeSharedEnabled() {
		return startPrivateTestSession(t)
	}
	return startSharedTestSession(t)
}

// StartTestSession is the exported form of startTestSession for pagehost tests.
func StartTestSession(t *testing.T) *Session {
	t.Helper()
	return startTestSession(t)
}

// startPrivateTestSession launches a dedicated Chromium that Close() tears down.
func startPrivateTestSession(t *testing.T) *Session {
	t.Helper()
	session := launchTestSession(t)
	t.Cleanup(session.Close)
	return session
}

// StartPrivateTestSession is the exported form of startPrivateTestSession.
func StartPrivateTestSession(t *testing.T) *Session {
	t.Helper()
	return startPrivateTestSession(t)
}

func startSharedTestSession(t *testing.T) *Session {
	t.Helper()
	sharedChromeOnce.Do(func() {
		cands := chromiumCandidates()
		if len(cands) == 0 {
			sharedChromeMu.Lock()
			sharedChromeErr = errChromiumUnavailable
			sharedChromeMu.Unlock()
			return
		}
		headless := true
		var lastErr error
		for _, execPath := range cands {
			ctx, cancel := context.WithCancel(context.Background())
			session, err := Start(ctx, StartOptions{ExecPath: execPath, Headless: &headless})
			if err == nil {
				session.shared = true
				sharedChromeMu.Lock()
				sharedChrome = session
				sharedChromeCancel = cancel
				sharedChromeMu.Unlock()
				return
			}
			cancel()
			lastErr = err
		}
		sharedChromeMu.Lock()
		sharedChromeErr = lastErr
		sharedChromeMu.Unlock()
	})
	sharedChromeMu.Lock()
	session := sharedChrome
	err := sharedChromeErr
	sharedChromeMu.Unlock()
	if session != nil {
		if ctx := session.Context(); ctx == nil || ctx.Err() != nil {
			// Stale published browser: drop it and start a fresh shared session.
			retireSharedTestSession(session)
			return startSharedTestSession(t)
		}
	}
	if session == nil {
		if err == nil || errors.Is(err, errChromiumUnavailable) {
			t.Skip("chromium unavailable")
		}
		t.Skipf("chromium start failed: %v", err)
	}
	t.Cleanup(func() {
		resetSharedTestTab(t, session)
	})
	return session
}

// CloseSharedTestSession tears down the process-wide shared Chromium, if any.
// Package TestMain should call this after m.Run(). Safe to call more than once;
// a later StartTestSession may start a new shared browser.
func CloseSharedTestSession() {
	sharedChromeMu.Lock()
	session := sharedChrome
	cancel := sharedChromeCancel
	sharedChrome = nil
	sharedChromeCancel = nil
	sharedChromeErr = nil
	sharedChromeOnce = sync.Once{}
	sharedChromeMu.Unlock()
	if session != nil {
		session.shared = false
		session.Close()
	}
	if cancel != nil {
		cancel()
	}
}

// retireSharedTestSession drops session from the process-wide slot when it is
// still the published shared browser, then closes it. No-op for nil or stale
// pointers that are no longer published.
func retireSharedTestSession(session *Session) {
	if session == nil {
		return
	}
	sharedChromeMu.Lock()
	if sharedChrome != session {
		sharedChromeMu.Unlock()
		return
	}
	cancel := sharedChromeCancel
	sharedChrome = nil
	sharedChromeCancel = nil
	sharedChromeErr = nil
	sharedChromeOnce = sync.Once{}
	sharedChromeMu.Unlock()
	session.shared = false
	session.Close()
	if cancel != nil {
		cancel()
	}
}

func resetSharedTestTab(t errorReporter, session *Session) {
	if session == nil {
		return
	}
	if session.Context() == nil || session.Context().Err() != nil {
		retireSharedTestSession(session)
		return
	}
	// Fetch is target-scoped. A prior Page may have left interception on after
	// Close skipped DisableFetch (fresh Page objects only disable when they
	// enabled Fetch). Always clear it before the next shared-session test.
	_ = runFetchDisable(session.Context())
	page, err := session.NewPage()
	if err != nil {
		if t != nil {
			t.Errorf("shared chrome reset: NewPage failed: %v", err)
		}
		retireSharedTestSession(session)
		return
	}
	page.Close()
}

// errorReporter is satisfied by *testing.T; tests may pass a recorder.
type errorReporter interface {
	Errorf(format string, args ...any)
}

func launchTestSession(t *testing.T) *Session {
	t.Helper()
	cands := chromiumCandidates()
	if len(cands) == 0 {
		t.Skip("chromium unavailable")
	}
	headless := true
	var lastErr error
	for _, execPath := range cands {
		// Bind session lifetime to Background, not a short start deadline: Start
		// watches the parent ctx and would tear down the browser when it ends.
		// Private sessions register Close via t.Cleanup in the caller.
		ctx, cancel := context.WithCancel(context.Background())
		session, err := Start(ctx, StartOptions{ExecPath: execPath, Headless: &headless})
		if err == nil {
			t.Cleanup(cancel)
			return session
		}
		cancel()
		lastErr = err
	}
	t.Skipf("chromium start failed: %v", lastErr)
	return nil
}

// startChromiumOrSkip is used by tests outside package helpers that need an ExecPath.
func startChromiumOrSkip(t *testing.T) (execPath string, session *Session) {
	t.Helper()
	session = startTestSession(t)
	return session.ExecPath(), session
}

var errChromiumUnavailable = errors.New("chromium unavailable")
