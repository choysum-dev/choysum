// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTextDescriptorKey(t *testing.T) {
	identity := []string{"模块", "scope.with.dots", "用户 🍀", "literal"}
	key := TextDescriptorKey(identity[0], identity[1], identity[2], identity[3])
	if key != TextDescriptorKey(identity[0], identity[1], identity[2], identity[3]) {
		t.Fatal("key is not deterministic")
	}
	if !strings.HasPrefix(key, TextDescriptorNamespace+".") ||
		strings.Contains(strings.TrimPrefix(key, TextDescriptorNamespace+"."), ".") {
		t.Fatalf("key is not vue-i18n path safe: %q", key)
	}
	for index := range identity {
		changed := append([]string(nil), identity...)
		changed[index] += "!"
		if TextDescriptorKey(changed[0], changed[1], changed[2], changed[3]) == key {
			t.Fatalf("identity component %d did not affect key", index)
		}
	}

	descriptor := NewTextDescriptor(identity[0], identity[1], identity[2], identity[3])
	raw, err := json.Marshal(descriptor)
	if err != nil {
		t.Fatal(err)
	}
	var roundtrip TextDescriptor
	if err := json.Unmarshal(raw, &roundtrip); err != nil {
		t.Fatal(err)
	}
	if roundtrip != descriptor {
		t.Fatalf("JSON round-trip mismatch: got %+v want %+v", roundtrip, descriptor)
	}
}

func TestIrField_SetResolvedSpec(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var field *IrField
		if err := field.SetResolvedSpec(&IrFieldResolvedSpec{FieldName: "x"}); err != nil {
			t.Fatalf("nil receiver: unexpected error: %v", err)
		}
	})

	t.Run("nil spec clears ResolvedSpec", func(t *testing.T) {
		field := &IrField{ResolvedSpec: `{"fieldName":"old"}`}
		if err := field.SetResolvedSpec(nil); err != nil {
			t.Fatalf("nil spec: unexpected error: %v", err)
		}
		if field.ResolvedSpec != "" {
			t.Fatalf("expected empty, got %q", field.ResolvedSpec)
		}
	})

	t.Run("valid spec serialises to JSON", func(t *testing.T) {
		field := &IrField{}
		spec := &IrFieldResolvedSpec{
			FieldName: "amount",
			Structural: IrFieldStructuralSpec{
				Name:      "amount",
				FieldType: "float",
				StorageHints: &IrFieldStructuralStorageHints{
					Required: ptr(true),
					Indexed:  ptr(true),
				},
			},
			Behavior: IrFieldBehaviorSpec{
				Compute: &IrFieldBehaviorComputeSpec{
					Method: "computeAmount",
					Deps:   []string{"price", "qty"},
					Store:  true,
				},
			},
			Migration: IrFieldMigrationDecision{
				StorageKind:        "column",
				ShouldCreateColumn: true,
				ResolvedColumnType: "DOUBLE PRECISION",
				ReasonCode:         "OK",
			},
			Diagnostics: []IrFieldDiagnostic{
				{Code: "W001", Severity: "warning", Message: "test diagnostic"},
			},
		}
		if err := field.SetResolvedSpec(spec); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if field.ResolvedSpec == "" {
			t.Fatal("expected non-empty ResolvedSpec")
		}

		var roundtrip IrFieldResolvedSpec
		if err := json.Unmarshal([]byte(field.ResolvedSpec), &roundtrip); err != nil {
			t.Fatalf("unmarshal roundtrip: %v", err)
		}
		if roundtrip.FieldName != "amount" {
			t.Fatalf("FieldName = %q, want amount", roundtrip.FieldName)
		}
		if roundtrip.Migration.ReasonCode != "OK" {
			t.Fatalf("ReasonCode = %q, want OK", roundtrip.Migration.ReasonCode)
		}
	})

	t.Run("spec with related and resolved values", func(t *testing.T) {
		field := &IrField{}
		searchable := ptr(true)
		runAs := ptr("admin")
		spec := &IrFieldResolvedSpec{
			FieldName: "total",
			Structural: IrFieldStructuralSpec{
				Name:      "total",
				FieldType: "float",
				Related: &IrFieldRelatedSpec{
					Path:  "order_id",
					Store: false,
					Deps:  []string{"order"},
				},
			},
			Resolved: struct {
				Store      IrResolvedValue[bool]    `json:"store"`
				Searchable IrResolvedValue[*bool]   `json:"searchable"`
				RunAs      IrResolvedValue[*string] `json:"runAs"`
			}{
				Store:      IrResolvedValue[bool]{Value: false, Source: "related"},
				Searchable: IrResolvedValue[*bool]{Value: searchable, Source: "explicit"},
				RunAs:      IrResolvedValue[*string]{Value: runAs, Source: "explicit"},
			},
		}
		if err := field.SetResolvedSpec(spec); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var roundtrip IrFieldResolvedSpec
		if err := json.Unmarshal([]byte(field.ResolvedSpec), &roundtrip); err != nil {
			t.Fatalf("unmarshal roundtrip: %v", err)
		}
		if roundtrip.Resolved.Store.Source != "related" {
			t.Fatalf("Store.Source = %q, want related", roundtrip.Resolved.Store.Source)
		}
		if roundtrip.Resolved.Searchable.Value == nil || *roundtrip.Resolved.Searchable.Value != true {
			t.Fatal("expected Searchable.Value = true")
		}
		if roundtrip.Resolved.RunAs.Value == nil || *roundtrip.Resolved.RunAs.Value != "admin" {
			t.Fatalf("expected RunAs.Value = admin")
		}
	})
}

func TestIrField_GetResolvedSpec(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var field *IrField
		spec, err := field.GetResolvedSpec()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if spec != nil {
			t.Fatal("expected nil spec for nil receiver")
		}
	})

	t.Run("empty ResolvedSpec returns nil", func(t *testing.T) {
		field := &IrField{ResolvedSpec: ""}
		spec, err := field.GetResolvedSpec()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if spec != nil {
			t.Fatal("expected nil spec for empty ResolvedSpec")
		}
	})

	t.Run("whitespace-only ResolvedSpec returns nil", func(t *testing.T) {
		field := &IrField{ResolvedSpec: "   "}
		spec, err := field.GetResolvedSpec()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if spec != nil {
			t.Fatal("expected nil spec for whitespace-only ResolvedSpec")
		}
	})

	t.Run("valid JSON roundtrip", func(t *testing.T) {
		original := &IrFieldResolvedSpec{
			FieldName: "active",
			Behavior: IrFieldBehaviorSpec{
				Compute: &IrFieldBehaviorComputeSpec{
					Method: "computeActive",
					Deps:   []string{"state"},
					Store:  true,
				},
			},
			Migration: IrFieldMigrationDecision{
				StorageKind:        "virtual",
				ShouldCreateColumn: false,
				ReasonCode:         "COMPUTE_VIRTUAL",
			},
		}

		field := &IrField{}
		if err := field.SetResolvedSpec(original); err != nil {
			t.Fatalf("set: %v", err)
		}

		got, err := field.GetResolvedSpec()
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if got == nil {
			t.Fatal("expected non-nil spec")
		}
		if got.FieldName != "active" {
			t.Fatalf("FieldName = %q, want active", got.FieldName)
		}
		if got.Migration.ReasonCode != "COMPUTE_VIRTUAL" {
			t.Fatalf("ReasonCode = %q, want COMPUTE_VIRTUAL", got.Migration.ReasonCode)
		}
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		field := &IrField{ResolvedSpec: "not-valid-json"}
		_, err := field.GetResolvedSpec()
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
	})
}

func ptr[T any](v T) *T {
	return &v
}
