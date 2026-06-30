// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import "testing"

func attrsMapFromList(t *testing.T, attrs []any) map[string]any {
	t.Helper()
	if len(attrs)%2 != 0 {
		t.Fatalf("expected even attr list, got %#v", attrs)
	}
	result := make(map[string]any, len(attrs)/2)
	for index := 0; index < len(attrs); index += 2 {
		key, ok := attrs[index].(string)
		if !ok {
			t.Fatalf("expected string key at index %d, got %#v", index, attrs[index])
		}
		result[key] = attrs[index+1]
	}
	return result
}

func TestCurrentOrRequestedAttrUsesCurrentValue(t *testing.T) {
	attrs := attrsMapFromList(t, currentOrRequestedAttr("input", "inputs", " core ", []string{"base", "task"}))
	if got := attrs["input"]; got != "core" {
		t.Fatalf("input = %#v, want core", got)
	}
	if _, ok := attrs["inputs"]; ok {
		t.Fatalf("expected current value to suppress plural list, got %#v", attrs["inputs"])
	}
}

func TestCurrentOrRequestedAttrFallsBackToRequestedList(t *testing.T) {
	attrs := attrsMapFromList(t, currentOrRequestedAttr("module", "modules", "", []string{" base ", "", "task"}))
	modules, ok := attrs["modules"].([]string)
	if !ok || len(modules) != 2 || modules[0] != "base" || modules[1] != "task" {
		t.Fatalf("modules = %#v, want [base task]", attrs["modules"])
	}
	if _, ok := attrs["module"]; ok {
		t.Fatalf("expected plural list for multiple requested values, got %#v", attrs["module"])
	}
}

func TestModuleInstallFailureAttrsAddsResolvedModuleWhenDifferent(t *testing.T) {
	attrs := attrsMapFromList(t, moduleInstallFailureAttrs("registry/base@1.0.0", "base"))
	if got := attrs["input"]; got != "registry/base@1.0.0" {
		t.Fatalf("input = %#v, want registry/base@1.0.0", got)
	}
	if got := attrs["module"]; got != "base" {
		t.Fatalf("module = %#v, want base", got)
	}
}

func TestModuleInstallFailureAttrsAvoidsDuplicateModuleField(t *testing.T) {
	attrs := attrsMapFromList(t, moduleInstallFailureAttrs("core", "core"))
	if got := attrs["input"]; got != "core" {
		t.Fatalf("input = %#v, want core", got)
	}
	if _, ok := attrs["module"]; ok {
		t.Fatalf("expected duplicate module field to be omitted, got %#v", attrs["module"])
	}
}

func TestModuleCommandFailureAttrsAddsOperation(t *testing.T) {
	attrs := attrsMapFromList(t, moduleCommandFailureAttrs(" upgrade "))
	if got := attrs["operation"]; got != "upgrade" {
		t.Fatalf("operation = %#v, want upgrade", got)
	}
}

func TestModuleCommandFailureAttrsOmitsBlankOperation(t *testing.T) {
	if attrs := moduleCommandFailureAttrs("   "); attrs != nil {
		t.Fatalf("expected blank operation to be omitted, got %#v", attrs)
	}
}
