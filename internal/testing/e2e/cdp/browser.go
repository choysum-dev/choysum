// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

// Package cdp wraps chromedp for choysum e2e host (browser/page/network).
package cdp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// PinnedCfTRevision matches scripts/ci/install_chromium.py (Chrome for Testing).
const PinnedCfTRevision = "1669021"

// runtimeGOOS is overridable in tests to exercise candidate path layouts.
var runtimeGOOS = runtime.GOOS

// mkdirTemp is os.MkdirTemp; tests may override to force user-data-dir errors.
var mkdirTemp = os.MkdirTemp

// Session is a chromedp browser allocator + tab context for one e2e scenario.
type Session struct {
	allocCtx    context.Context
	allocCancel context.CancelFunc
	browserCtx  context.Context
	cancel      context.CancelFunc
	execPath    string
	userDataDir string
	stopWatch   context.CancelFunc
	headless    bool
}

// StartOptions configures browser launch.
type StartOptions struct {
	// ExecPath overrides binary discovery when non-empty.
	ExecPath string
	// Headless selects headless mode when non-nil. When nil, WantHeadless
	// falls back to CHOYSUM_E2E_HEADED (default headless).
	Headless *bool
}

// WantHeadless reports whether Chromium should run headless.
//
// Default is headless. Explicit StartOptions.Headless wins when set.
// Otherwise only CHOYSUM_E2E_HEADED=1|true|yes opts into headed mode for
// local debugging; unset, 0, false, and no leave headless.
func WantHeadless(opts StartOptions) bool {
	if opts.Headless != nil {
		return *opts.Headless
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv("CHOYSUM_E2E_HEADED"))) {
	case "1", "true", "yes":
		return false
	default:
		// unset, "0", "false", "no", or anything else → headless
		return true
	}
}

// Start launches Chromium via chromedp ExecAllocator.
// Retries a few times on DevTools websocket readiness flakes common in CI.
func Start(ctx context.Context, opts StartOptions) (*Session, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	execPath := strings.TrimSpace(opts.ExecPath)
	if execPath == "" {
		var err error
		execPath, err = ResolveChromiumPath()
		if err != nil {
			return nil, err
		}
	}
	headless := WantHeadless(opts)

	const attempts = 3
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		session, err := startBrowserOnce(ctx, execPath, headless)
		if err == nil {
			return session, nil
		}
		lastErr = err
		if !isWebsocketURLTimeout(err) || attempt == attempts {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(attempt) * 250 * time.Millisecond):
		}
	}
	return nil, lastErr
}

func isWebsocketURLTimeout(err error) bool {
	return err != nil && strings.Contains(err.Error(), "websocket url timeout")
}

// startBrowserOnce launches one Chromium attempt; tests may override to simulate flakes.
var startBrowserOnce = startOnce

func startOnce(ctx context.Context, execPath string, headless bool) (s *Session, err error) {
	udir, err := mkdirTemp("", "choysum-e2e-chrome-*")
	if err != nil {
		return nil, fmt.Errorf("cdp: user-data-dir: %w", err)
	}
	// On failure Session.Close is never called; remove the orphaned profile dir.
	defer func() {
		if err != nil {
			_ = os.RemoveAll(udir)
		}
	}()
	allocOpts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(execPath),
		chromedp.UserDataDir(udir),
		chromedp.Flag("headless", headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("remote-allow-origins", "*"),
		chromedp.Flag("disable-crash-reporter", true),
		chromedp.Flag("disable-breakpad", true),
		// Default is 20s; CI runners can take longer before DevTools prints the WS URL.
		chromedp.WSURLReadTimeout(60*time.Second),
	)
	// Allocator uses Background so a parent deadline that already fired during
	// long install/readyz does not surface as a misleading "context canceled"
	// on first browser boot. A watcher still closes the session when parent ctx ends.
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), allocOpts...)
	browserCtx, cancel := chromedp.NewContext(allocCtx)
	if err = chromedp.Run(browserCtx, chromedp.Navigate("about:blank")); err != nil {
		cancel()
		allocCancel()
		return nil, fmt.Errorf("cdp: start browser (%s): %w", execPath, err)
	}
	// Arm parent-ctx watcher only after the first navigation succeeds so an
	// already-canceled parent cannot cancel browserCtx mid-boot.
	watchCtx, stopWatch := context.WithCancel(context.Background())
	go func() {
		select {
		case <-ctx.Done():
			cancel()
			allocCancel()
		case <-watchCtx.Done():
		}
	}()
	return &Session{
		allocCtx:    allocCtx,
		allocCancel: allocCancel,
		browserCtx:  browserCtx,
		cancel:      cancel,
		execPath:    execPath,
		userDataDir: udir,
		stopWatch:   stopWatch,
		headless:    headless,
	}, nil
}

// Close shuts down the browser session.
func (s *Session) Close() {
	if s == nil {
		return
	}
	if s.stopWatch != nil {
		s.stopWatch()
	}
	if s.cancel != nil {
		s.cancel()
	}
	if s.allocCancel != nil {
		s.allocCancel()
	}
	if s.userDataDir != "" {
		_ = os.RemoveAll(s.userDataDir)
	}
}

// Context returns the root browser context (first tab).
func (s *Session) Context() context.Context {
	if s == nil {
		return nil
	}
	return s.browserCtx
}

// ExecPath returns the resolved Chromium binary path.
func (s *Session) ExecPath() string {
	if s == nil {
		return ""
	}
	return s.execPath
}

// Headless reports whether this session was started headless.
func (s *Session) Headless() bool {
	if s == nil {
		return true
	}
	return s.headless
}

// ResolveChromiumPath finds a Chromium/Chrome binary.
// Order: CHOYSUM_CHROMIUM_PATH → $CHOYSUM_HOME/browsers/chromium-<rev>/… → system Chrome.
func ResolveChromiumPath() (string, error) {
	return resolveChromiumPath(true)
}

// ResolveChromiumPathForE2E is ResolveChromiumPath for the e2e runner preflight.
// When CI or GITHUB_ACTIONS is set, system Chrome/Chromium candidates are skipped
// so CI cannot silently pick up an unrelated browser.
func ResolveChromiumPathForE2E() (string, error) {
	allowSystem := strings.TrimSpace(os.Getenv("CI")) == "" && strings.TrimSpace(os.Getenv("GITHUB_ACTIONS")) == ""
	return resolveChromiumPath(allowSystem)
}

func resolveChromiumPath(allowSystemChrome bool) (string, error) {
	if p := strings.TrimSpace(os.Getenv("CHOYSUM_CHROMIUM_PATH")); p != "" {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
		return "", missingBinaryError(fmt.Sprintf("CHOYSUM_CHROMIUM_PATH=%q is not a usable file", p))
	}

	home := strings.TrimSpace(os.Getenv("CHOYSUM_HOME"))
	if home == "" {
		userHome, err := os.UserHomeDir()
		if err == nil {
			home = filepath.Join(userHome, ".choysum")
		}
	}
	if home != "" {
		base := filepath.Join(home, "browsers", "chromium-"+PinnedCfTRevision)
		for _, cand := range cachedBinaryCandidates(base) {
			if st, err := os.Stat(cand); err == nil && !st.IsDir() {
				return cand, nil
			}
		}
	}

	if allowSystemChrome {
		for _, cand := range systemChromeCandidates() {
			if st, err := os.Stat(cand); err == nil && !st.IsDir() {
				return cand, nil
			}
		}
	}
	return "", missingBinaryError("chromium binary not found")
}

func missingBinaryError(detail string) error {
	return fmt.Errorf("%s; set CHOYSUM_CHROMIUM_PATH or run `choysum test e2e --install-browser` (scripts/ci/install_chromium.py)", detail)
}

func cachedBinaryCandidates(base string) []string {
	switch runtimeGOOS {
	case "darwin":
		return []string{
			filepath.Join(base, "chrome-mac-arm64", "Google Chrome for Testing.app", "Contents", "MacOS", "Google Chrome for Testing"),
			filepath.Join(base, "chrome-mac-x64", "Google Chrome for Testing.app", "Contents", "MacOS", "Google Chrome for Testing"),
			filepath.Join(base, "Google Chrome for Testing.app", "Contents", "MacOS", "Google Chrome for Testing"),
			filepath.Join(base, "Chromium.app", "Contents", "MacOS", "Chromium"),
			filepath.Join(base, "chrome"),
		}
	case "linux":
		return []string{
			filepath.Join(base, "chrome-linux64", "chrome"),
			filepath.Join(base, "chrome"),
			filepath.Join(base, "chromium"),
		}
	default:
		return []string{
			filepath.Join(base, "chrome"),
			filepath.Join(base, "chromium"),
		}
	}
}

func systemChromeCandidates() []string {
	switch runtimeGOOS {
	case "darwin":
		return []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Google Chrome for Testing.app/Contents/MacOS/Google Chrome for Testing",
		}
	case "linux":
		return []string{
			"/usr/bin/google-chrome",
			"/usr/bin/google-chrome-stable",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
			"/snap/bin/chromium",
		}
	default:
		return nil
	}
}

// WithTimeout returns ctx with timeout when d > 0.
func WithTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if d <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}
