// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cdp

import (
	"strings"
	"testing"
)

func TestResolveChromiumPathMissingBinaryMessage(t *testing.T) {
	t.Setenv("CHOYSUM_CHROMIUM_PATH", "")
	t.Setenv("CHOYSUM_HOME", t.TempDir())
	// Point home at empty dir so cache miss; clear PATH-like system candidates by
	// using an impossible CHOYSUM_CHROMIUM_PATH first path... we unset and rely on
	// empty cache + likely missing system chrome in CI containers is ok — if system
	// chrome exists the test still validates error shape when forcing bad env path.
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
