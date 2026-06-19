// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	testsemantics "github.com/choysum-dev/choysum/internal/testing/semantics"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func TestCLITestSemanticsContract_RuntimeOptionsPrefix(t *testing.T) {
	t.Run("unit", func(t *testing.T) {
		cmd := newTestUnitCmdFromScope(func() scope.Scope { return &commandTestScope{} })
		err := cmd.RunE(cmd, []string{"auth"})
		if err == nil || !strings.Contains(err.Error(), testsemantics.InvalidRuntimeOptionsMessage("test unit")) {
			t.Fatalf("expected runtime options prefix %q, got %v", testsemantics.InvalidRuntimeOptionsMessage("test unit"), err)
		}
	})

	t.Run("typecheck", func(t *testing.T) {
		cmd := newTypecheckCmd(func() scope.Scope { return &commandTestScope{} }, commandRuntimeOptionsFromScope(func() scope.Scope { return &commandTestScope{} }))
		err := cmd.RunE(cmd, []string{"auth"})
		if err == nil || !strings.Contains(err.Error(), testsemantics.InvalidRuntimeOptionsMessage("typecheck")) {
			t.Fatalf("expected runtime options prefix %q, got %v", testsemantics.InvalidRuntimeOptionsMessage("typecheck"), err)
		}
	})

	t.Run("e2e", func(t *testing.T) {
		cmd := newE2ECmd(func() scope.Scope { return &commandTestScope{} }, commandRuntimeOptionsFromScope(func() scope.Scope { return &commandTestScope{} }))
		err := cmd.RunE(cmd, []string{"auth"})
		if err == nil || !strings.Contains(err.Error(), testsemantics.InvalidRuntimeOptionsMessage("e2e")) {
			t.Fatalf("expected runtime options prefix %q, got %v", testsemantics.InvalidRuntimeOptionsMessage("e2e"), err)
		}
	})
}

func TestCLITestSemanticsContract_UnknownTarget(t *testing.T) {
	t.Run("unit unknown app", func(t *testing.T) {
		modulesPath := t.TempDir()
		cfg := newCommandTestConfig(modulesPath)
		scopeGetter := func() scope.Scope { return &commandTestScope{cfg: cfg} }
		cmd := newTestUnitCmdFromScope(scopeGetter)
		err := cmd.RunE(cmd, []string{"auth"})
		if err == nil || !strings.Contains(err.Error(), testsemantics.UnknownAppMessage("auth")) {
			t.Fatalf("expected unknown app error %q, got %v", testsemantics.UnknownAppMessage("auth"), err)
		}
	})

	t.Run("typecheck unknown app", func(t *testing.T) {
		modulesPath := t.TempDir()
		cfg := newCommandTestConfig(modulesPath)
		scopeGetter := func() scope.Scope { return &commandTestScope{cfg: cfg} }
		cmd := newTypecheckCmd(scopeGetter, commandRuntimeOptionsFromScope(scopeGetter))
		err := cmd.RunE(cmd, []string{"auth"})
		want := testsemantics.PrefixForCommand("typecheck", testsemantics.UnknownAppMessage("auth"))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("expected unknown app error %q, got %v", want, err)
		}
	})

	t.Run("e2e unknown module", func(t *testing.T) {
		modulesPath := t.TempDir()
		cfg := newCommandTestConfig(modulesPath)
		scopeGetter := func() scope.Scope { return &commandTestScope{cfg: cfg} }
		cmd := newE2ECmd(scopeGetter, commandRuntimeOptionsFromScope(scopeGetter))
		err := cmd.RunE(cmd, []string{"auth"})
		want := testsemantics.UnknownModuleMessage("auth", modulesPath)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("expected unknown module error %q, got %v", want, err)
		}
	})
}

func TestCLITestSemanticsContract_NoTestsFound(t *testing.T) {
	t.Run("unit existing app without tests", func(t *testing.T) {
		modulesPath := t.TempDir()
		if err := os.MkdirAll(filepath.Join(modulesPath, "auth", "service"), 0o755); err != nil {
			t.Fatalf("mkdir service dir: %v", err)
		}
		cfg := newCommandTestConfig(modulesPath)
		scopeGetter := func() scope.Scope { return &commandTestScope{cfg: cfg} }
		cmd := newTestUnitCmdFromScope(scopeGetter)
		stdout, err := captureStdoutForContract(t, func() error {
			return cmd.RunE(cmd, []string{"auth"})
		})
		if err != nil {
			t.Fatalf("expected no-tests success, got %v", err)
		}
		if !strings.Contains(stdout, testsemantics.NoTestsFoundMessage) {
			t.Fatalf("expected stdout to contain %q, got %q", testsemantics.NoTestsFoundMessage, stdout)
		}
	})

	t.Run("typecheck existing app without checkable inputs", func(t *testing.T) {
		modulesPath := t.TempDir()
		if err := os.MkdirAll(filepath.Join(modulesPath, "auth", "service"), 0o755); err != nil {
			t.Fatalf("mkdir service dir: %v", err)
		}
		cfg := newCommandTestConfig(modulesPath)
		scopeGetter := func() scope.Scope { return &commandTestScope{cfg: cfg} }
		cmd := newTypecheckCmd(scopeGetter, commandRuntimeOptionsFromScope(scopeGetter))
		stdout, err := captureStdoutForContract(t, func() error {
			return cmd.RunE(cmd, []string{"auth"})
		})
		if err != nil {
			t.Fatalf("expected no-tests success, got %v", err)
		}
		if !strings.Contains(stdout, testsemantics.NoTestsFoundMessage) {
			t.Fatalf("expected stdout to contain %q, got %q", testsemantics.NoTestsFoundMessage, stdout)
		}
	})

	t.Run("e2e module without specs", func(t *testing.T) {
		modulesPath := t.TempDir()
		writeCommandPackage(t, modulesPath, "auth", `{"name":"@choysum-dev/auth","version":"0.0.0","choysum":{"moduleName":"auth","application":"auth"}}`)
		cfg := newCommandTestConfig(modulesPath)
		scopeGetter := func() scope.Scope { return &commandTestScope{cfg: cfg} }
		cmd := newE2ECmd(scopeGetter, commandRuntimeOptionsFromScope(scopeGetter))
		stdout, err := captureStdoutForContract(t, func() error {
			return cmd.RunE(cmd, []string{"auth"})
		})
		if err != nil {
			t.Fatalf("expected no-tests success, got %v", err)
		}
		if !strings.Contains(stdout, testsemantics.NoTestsFoundMessage) {
			t.Fatalf("expected stdout to contain %q, got %q", testsemantics.NoTestsFoundMessage, stdout)
		}
	})

	t.Run("e2e all without runnable modules returns matrix error", func(t *testing.T) {
		modulesPath := t.TempDir()
		cfg := newCommandTestConfig(modulesPath)
		scopeGetter := func() scope.Scope { return &commandTestScope{cfg: cfg} }
		cmd := newE2ECmd(scopeGetter, commandRuntimeOptionsFromScope(scopeGetter))
		if err := cmd.Flags().Set("all", "true"); err != nil {
			t.Fatalf("set --all=true: %v", err)
		}
		err := cmd.RunE(cmd, nil)
		want := testsemantics.NoRunnableE2EModulesMessage(modulesPath)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("expected no-runnable-modules error %q, got %v", want, err)
		}
	})
}

func captureStdoutForContract(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	oldStdout := os.Stdout
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	runErr := fn()
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("close reader: %v", err)
	}
	return string(data), runErr
}
