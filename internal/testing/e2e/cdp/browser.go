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

// Session is a chromedp browser allocator + tab context for one e2e scenario.
type Session struct {
	allocCtx    context.Context
	allocCancel context.CancelFunc
	browserCtx  context.Context
	cancel      context.CancelFunc
	execPath    string
	userDataDir string
	stopWatch   context.CancelFunc
}

// StartOptions configures browser launch.
type StartOptions struct {
	// ExecPath overrides binary discovery when non-empty.
	ExecPath string
	// Headless defaults to true; CHOYSUM_E2E_HEADED=1 forces headed.
	Headless *bool
}

// Start launches Chromium via chromedp ExecAllocator.
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
	headless := true
	if opts.Headless != nil {
		headless = *opts.Headless
	}
	if strings.TrimSpace(os.Getenv("CHOYSUM_E2E_HEADED")) == "1" {
		headless = false
	}

	udir, err := os.MkdirTemp("", "choysum-e2e-chrome-*")
	if err != nil {
		return nil, fmt.Errorf("cdp: user-data-dir: %w", err)
	}
	allocOpts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(execPath),
		chromedp.UserDataDir(udir),
		chromedp.Flag("headless", headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("remote-allow-origins", "*"),
	)
	// Allocator uses Background so a parent deadline that already fired during
	// long install/readyz does not surface as a misleading "context canceled"
	// on first browser boot. A watcher still closes the session when parent ctx ends.
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), allocOpts...)
	browserCtx, cancel := chromedp.NewContext(allocCtx)
	watchCtx, stopWatch := context.WithCancel(context.Background())
	go func() {
		select {
		case <-ctx.Done():
			cancel()
			allocCancel()
		case <-watchCtx.Done():
		}
	}()
	if err := chromedp.Run(browserCtx, chromedp.Navigate("about:blank")); err != nil {
		stopWatch()
		cancel()
		allocCancel()
		_ = os.RemoveAll(udir)
		return nil, fmt.Errorf("cdp: start browser (%s): %w", execPath, err)
	}
	return &Session{
		allocCtx:    allocCtx,
		allocCancel: allocCancel,
		browserCtx:  browserCtx,
		cancel:      cancel,
		execPath:    execPath,
		userDataDir: udir,
		stopWatch:   stopWatch,
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

// ResolveChromiumPath finds a Chromium/Chrome binary.
// Order: CHOYSUM_CHROMIUM_PATH → $CHOYSUM_HOME/browsers/chromium-<rev>/… → system Chrome.
func ResolveChromiumPath() (string, error) {
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

	for _, cand := range systemChromeCandidates() {
		if st, err := os.Stat(cand); err == nil && !st.IsDir() {
			return cand, nil
		}
	}
	return "", missingBinaryError("chromium binary not found")
}

func missingBinaryError(detail string) error {
	return fmt.Errorf("%s; set CHOYSUM_CHROMIUM_PATH or run `choysum test e2e --install-browser` (scripts/ci/install_chromium.py)", detail)
}

func cachedBinaryCandidates(base string) []string {
	switch runtime.GOOS {
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
	switch runtime.GOOS {
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
