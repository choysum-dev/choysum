// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package origin

import (
	"strings"
	"testing"
)

func TestParseInputRejectsLegacyAliasSyntax(t *testing.T) {
	_, err := ParseInput("corp/demo@v1.0.0")
	if err == nil {
		t.Fatal("expected legacy alias syntax to be rejected")
	}
	if !strings.Contains(err.Error(), "registry alias syntax is no longer supported") {
		t.Fatalf("expected legacy alias rejection error, got %v", err)
	}
	if !strings.Contains(err.Error(), "<module>@<version>") {
		t.Fatalf("expected migration guidance in error, got %v", err)
	}
}

func TestParseInputRegistryReference(t *testing.T) {
	parsed, err := ParseInput("auth@v1.2.3")
	if err != nil {
		t.Fatalf("ParseInput() error = %v", err)
	}
	if parsed.Kind != InputKindRegistry {
		t.Fatalf("kind = %q, want %q", parsed.Kind, InputKindRegistry)
	}
	if parsed.ModuleName != "auth" {
		t.Fatalf("module name = %q, want %q", parsed.ModuleName, "auth")
	}
	if parsed.Version != "v1.2.3" {
		t.Fatalf("version = %q, want %q", parsed.Version, "v1.2.3")
	}
	if parsed.CanonicalRef() != "auth@v1.2.3" {
		t.Fatalf("canonical ref = %q, want %q", parsed.CanonicalRef(), "auth@v1.2.3")
	}
}
