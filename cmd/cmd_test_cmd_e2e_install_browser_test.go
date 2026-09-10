// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/scope"
)

func TestInstallChromiumBrowserMissingScript(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)

	err := installChromiumBrowser()
	if err == nil {
		t.Fatal("expected error when install_chromium.py is missing")
	}
	msg := err.Error()
	if !strings.Contains(msg, "install-browser") || !strings.Contains(msg, "install_chromium.py") {
		t.Fatalf("unexpected error: %q", msg)
	}
}

func TestInstallChromiumBrowserScriptSuccess(t *testing.T) {
	tmp := t.TempDir()
	scriptDir := filepath.Join(tmp, "scripts", "ci")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(scriptDir, "install_chromium.py")
	if err := os.WriteFile(script, []byte("#!/usr/bin/env python3\nimport sys\nsys.exit(0)\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(tmp)

	if err := installChromiumBrowser(); err != nil {
		t.Fatalf("installChromiumBrowser: %v", err)
	}
}

func TestInstallChromiumBrowserScriptFailure(t *testing.T) {
	tmp := t.TempDir()
	scriptDir := filepath.Join(tmp, "scripts", "ci")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(scriptDir, "install_chromium.py")
	if err := os.WriteFile(script, []byte("#!/usr/bin/env python3\nimport sys\nsys.exit(2)\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(tmp)

	err := installChromiumBrowser()
	if err == nil || !strings.Contains(err.Error(), "install-browser") {
		t.Fatalf("expected install-browser failure, got %v", err)
	}
}

func TestE2ECmdInstallBrowserFlagSkipsModuleArg(t *testing.T) {
	tmp := t.TempDir()
	scriptDir := filepath.Join(tmp, "scripts", "ci")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(scriptDir, "install_chromium.py")
	if err := os.WriteFile(script, []byte("#!/usr/bin/env python3\nimport sys\nsys.exit(0)\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(tmp)

	cmd := newE2ECmd(func() scope.Scope { return nil }, nil)
	if err := cmd.Flags().Set("install-browser", "true"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Args(cmd, nil); err != nil {
		t.Fatalf("Args with --install-browser: %v", err)
	}
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("RunE --install-browser: %v", err)
	}
}
