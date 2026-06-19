// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package semantics

import (
	"errors"
	"testing"
)

func TestMessages(t *testing.T) {
	t.Run("prefix for command", func(t *testing.T) {
		if got := PrefixForCommand("typecheck", "unknown app \"auth\""); got != "typecheck: unknown app \"auth\"" {
			t.Fatalf("PrefixForCommand() = %q", got)
		}
		if got := PrefixForCommand("", "invalid runtime options"); got != "command: invalid runtime options" {
			t.Fatalf("PrefixForCommand(empty command) = %q", got)
		}
	})

	t.Run("runtime options message", func(t *testing.T) {
		if got := InvalidRuntimeOptionsMessage("test unit"); got != "test unit: invalid runtime options" {
			t.Fatalf("InvalidRuntimeOptionsMessage() = %q", got)
		}
	})

	t.Run("unknown/no-tests messages", func(t *testing.T) {
		if got := UnknownAppMessage("auth"); got != "unknown app \"auth\"" {
			t.Fatalf("UnknownAppMessage() = %q", got)
		}
		if got := UnknownModuleMessage("auth", "/workspace/modules"); got != "unknown module \"auth\" (no package.json under /workspace/modules)" {
			t.Fatalf("UnknownModuleMessage() = %q", got)
		}
		if got := ModuleNoE2ESpecsMessage("auth"); got != "module \"auth\" has no package.json choysum.e2e.specs" {
			t.Fatalf("ModuleNoE2ESpecsMessage() = %q", got)
		}
		if got := NoRunnableE2EModulesMessage("/workspace/modules"); got != "e2e: no runnable modules found under /workspace/modules" {
			t.Fatalf("NoRunnableE2EModulesMessage() = %q", got)
		}
		if NoTestsFoundMessage != "no tests found" {
			t.Fatalf("NoTestsFoundMessage = %q", NoTestsFoundMessage)
		}
	})

	t.Run("no e2e specs detection", func(t *testing.T) {
		if IsModuleNoE2ESpecsError(nil) {
			t.Fatal("expected nil error to return false")
		}
		err := errors.New("module \"auth\" has no package.json choysum.e2e.specs")
		if !IsModuleNoE2ESpecsError(err) {
			t.Fatal("expected module-no-specs error to be detected")
		}
	})
}
