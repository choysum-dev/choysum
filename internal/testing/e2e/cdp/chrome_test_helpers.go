// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cdp

import (
	"context"
	"os"
	"testing"
)

// Overridable in tests to force empty/duplicate candidate paths.
var (
	chromiumResolvePath = ResolveChromiumPath
	chromiumSystemPaths = systemChromeCandidates
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
	if p, err := chromiumResolvePath(); err == nil {
		add(p)
	}
	for _, p := range chromiumSystemPaths() {
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

// startTestSession launches headless Chrome, trying ResolveChromiumPath then system candidates.
// Skips when no binary can start (broken CfT caches, sandboxes, etc.).
func startTestSession(t *testing.T) *Session {
	t.Helper()
	cands := chromiumCandidates()
	if len(cands) == 0 {
		t.Skip("chromium unavailable")
	}
	headless := true
	var lastErr error
	for _, execPath := range cands {
		// Bind session lifetime to the test, not a short start deadline: Start
		// watches the parent ctx and would tear down the browser when it ends.
		ctx, cancel := context.WithCancel(context.Background())
		session, err := Start(ctx, StartOptions{ExecPath: execPath, Headless: &headless})
		if err == nil {
			t.Cleanup(func() {
				session.Close()
				cancel()
			})
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
