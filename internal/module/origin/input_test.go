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

func TestParseInputRejectsInvalidForms(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		wantErrPart string
	}{
		{name: "empty", input: "   ", wantErrPart: "empty module input"},
		{name: "invalid local name", input: "bad.name", wantErrPart: "invalid local module name"},
		{name: "invalid registry module name", input: "bad!@v1.0.0", wantErrPart: "invalid registry module name"},
		{name: "missing registry version", input: "auth@", wantErrPart: "expected <module>@<version>"},
		{name: "invalid registry version", input: "auth@v1~0", wantErrPart: "invalid registry version"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseInput(tt.input)
			if err == nil || !strings.Contains(err.Error(), tt.wantErrPart) {
				t.Fatalf("ParseInput(%q) error = %v, want substring %q", tt.input, err, tt.wantErrPart)
			}
		})
	}
}

func TestParseInputLocalReferenceAndCanonicalRef(t *testing.T) {
	t.Parallel()

	parsed, err := ParseInput("auth")
	if err != nil {
		t.Fatalf("ParseInput(local) error = %v", err)
	}
	if parsed.Kind != InputKindLocal {
		t.Fatalf("kind = %q, want %q", parsed.Kind, InputKindLocal)
	}
	if parsed.LocalName != "auth" {
		t.Fatalf("local name = %q, want %q", parsed.LocalName, "auth")
	}
	if parsed.CanonicalRef() != "auth" {
		t.Fatalf("canonical local ref = %q, want %q", parsed.CanonicalRef(), "auth")
	}
}

func TestParsedInputCanonicalRefRegistryDefaultsLatest(t *testing.T) {
	t.Parallel()

	parsed := ParsedInput{Kind: InputKindRegistry, ModuleName: "auth", Version: "   "}
	if parsed.CanonicalRef() != "auth@latest" {
		t.Fatalf("canonical registry ref = %q, want %q", parsed.CanonicalRef(), "auth@latest")
	}
}
