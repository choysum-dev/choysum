// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cdp

import (
	"context"
	"testing"
	"time"
)

func requireChromium(t *testing.T) string {
	t.Helper()
	path, err := ResolveChromiumPath()
	if err != nil {
		t.Skipf("chromium unavailable: %v", err)
	}
	return path
}

func startTestSession(t *testing.T) *Session {
	t.Helper()
	execPath := requireChromium(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	t.Cleanup(cancel)
	headless := true
	session, err := Start(ctx, StartOptions{ExecPath: execPath, Headless: &headless})
	if err != nil {
		t.Fatalf("cdp.Start: %v", err)
	}
	t.Cleanup(session.Close)
	return session
}
