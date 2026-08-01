// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTermReferenceKeyStabilityAndJSONRoundTrip(t *testing.T) {
	identity := []string{"模块", "scope.with.dots", "用户 🍀", "literal"}
	const stableKey = "__terms.363ae6a8a1e59d9731353a73636f70652e776974682e646f747331313ae794a8e688b720f09f8d80373a6c69746572616c"
	key := TermReferenceKey(identity[0], identity[1], identity[2], identity[3])
	if key != stableKey {
		t.Fatalf("key changed: got %q want %q", key, stableKey)
	}
	if key != TermReferenceKey(identity[0], identity[1], identity[2], identity[3]) {
		t.Fatal("key is not deterministic")
	}
	if !strings.HasPrefix(key, TermReferenceNamespace+".") ||
		strings.Contains(strings.TrimPrefix(key, TermReferenceNamespace+"."), ".") {
		t.Fatalf("key is not vue-i18n path safe: %q", key)
	}
	for index := range identity {
		changed := append([]string(nil), identity...)
		changed[index] += "!"
		if TermReferenceKey(changed[0], changed[1], changed[2], changed[3]) == key {
			t.Fatalf("identity component %d did not affect key", index)
		}
	}

	reference := NewTermReference(identity[0], identity[1], identity[2], identity[3])
	raw, err := json.Marshal(reference)
	if err != nil {
		t.Fatal(err)
	}
	const stableJSON = `{"key":"__terms.363ae6a8a1e59d9731353a73636f70652e776974682e646f747331313ae794a8e688b720f09f8d80373a6c69746572616c","module":"模块","scope":"scope.with.dots","src":"用户 🍀","kind":"literal"}`
	if string(raw) != stableJSON {
		t.Fatalf("JSON shape changed: got %s want %s", raw, stableJSON)
	}
	var roundtrip TermReference
	if err := json.Unmarshal(raw, &roundtrip); err != nil {
		t.Fatal(err)
	}
	if roundtrip != reference {
		t.Fatalf("JSON round-trip mismatch: got %+v want %+v", roundtrip, reference)
	}

	itemRaw, err := json.Marshal(FieldSelectionItem{
		Value: "active", Label: "Active", LabelText: &reference,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(itemRaw), `"labelText":`) || strings.Contains(string(itemRaw), `"termReference":`) {
		t.Fatalf("labelText wire property changed: %s", itemRaw)
	}
	var itemRoundtrip FieldSelectionItem
	if err := json.Unmarshal(itemRaw, &itemRoundtrip); err != nil {
		t.Fatal(err)
	}
	if itemRoundtrip.LabelText == nil || *itemRoundtrip.LabelText != reference {
		t.Fatalf("selection JSON round-trip mismatch: got %+v", itemRoundtrip)
	}
}

func TestField_SetResolvedSpec(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var field *Field
		if err := field.SetResolvedSpec(&FieldResolvedSpec{FieldName: "x"}); err != nil {
			t.Fatalf("nil receiver: unexpected error: %v", err)
		}
	})

	t.Run("nil spec clears ResolvedSpec", func(t *testing.T) {
		field := &Field{ResolvedSpec: `{"fieldName":"old"}`}
		if err := field.SetResolvedSpec(nil); err != nil {
			t.Fatalf("nil spec: unexpected error: %v", err)
		}
		if field.ResolvedSpec != "" {
			t.Fatalf("expected empty, got %q", field.ResolvedSpec)
		}
	})

	t.Run("valid spec serialises to JSON", func(t *testing.T) {
		field := &Field{}
		spec := &FieldResolvedSpec{
			FieldName: "amount",
			Structural: FieldStructuralSpec{
				Name:      "amount",
				FieldType: "float",
				StorageHints: &FieldStructuralStorageHints{
					Required: ptr(true),
					Indexed:  ptr(true),
				},
			},
			Behavior: FieldBehaviorSpec{
				Compute: &FieldBehaviorComputeSpec{
					Method: "computeAmount",
					Deps:   []string{"price", "qty"},
					Store:  true,
				},
			},
			Migration: FieldMigrationDecision{
				StorageKind:        "column",
				ShouldCreateColumn: true,
				ResolvedColumnType: "DOUBLE PRECISION",
				ReasonCode:         "OK",
			},
			Diagnostics: []FieldDiagnostic{
				{Code: "W001", Severity: "warning", Message: "test diagnostic"},
			},
		}
		if err := field.SetResolvedSpec(spec); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if field.ResolvedSpec == "" {
			t.Fatal("expected non-empty ResolvedSpec")
		}

		var roundtrip FieldResolvedSpec
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
		field := &Field{}
		searchable := ptr(true)
		spec := &FieldResolvedSpec{
			FieldName: "total",
			Structural: FieldStructuralSpec{
				Name:      "total",
				FieldType: "float",
				Related: &FieldRelatedSpec{
					Path:  "order_id",
					Store: false,
					Deps:  []string{"order"},
				},
			},
			Resolved: struct {
				Store      ResolvedValue[bool]  `json:"store"`
				Searchable ResolvedValue[*bool] `json:"searchable"`
			}{
				Store:      ResolvedValue[bool]{Value: false, Source: "related"},
				Searchable: ResolvedValue[*bool]{Value: searchable, Source: "explicit"},
			},
		}
		if err := field.SetResolvedSpec(spec); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var roundtrip FieldResolvedSpec
		if err := json.Unmarshal([]byte(field.ResolvedSpec), &roundtrip); err != nil {
			t.Fatalf("unmarshal roundtrip: %v", err)
		}
		if roundtrip.Resolved.Store.Source != "related" {
			t.Fatalf("Store.Source = %q, want related", roundtrip.Resolved.Store.Source)
		}
		if roundtrip.Resolved.Searchable.Value == nil || *roundtrip.Resolved.Searchable.Value != true {
			t.Fatal("expected Searchable.Value = true")
		}
		if strings.Contains(field.ResolvedSpec, `"runAs"`) {
			t.Fatalf("resolved metadata must omit removed runAs contract, got %s", field.ResolvedSpec)
		}
	})
}

func TestField_GetResolvedSpec(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var field *Field
		spec, err := field.GetResolvedSpec()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if spec != nil {
			t.Fatal("expected nil spec for nil receiver")
		}
	})

	t.Run("empty ResolvedSpec returns nil", func(t *testing.T) {
		field := &Field{ResolvedSpec: ""}
		spec, err := field.GetResolvedSpec()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if spec != nil {
			t.Fatal("expected nil spec for empty ResolvedSpec")
		}
	})

	t.Run("whitespace-only ResolvedSpec returns nil", func(t *testing.T) {
		field := &Field{ResolvedSpec: "   "}
		spec, err := field.GetResolvedSpec()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if spec != nil {
			t.Fatal("expected nil spec for whitespace-only ResolvedSpec")
		}
	})

	t.Run("valid JSON roundtrip", func(t *testing.T) {
		original := &FieldResolvedSpec{
			FieldName: "active",
			Behavior: FieldBehaviorSpec{
				Compute: &FieldBehaviorComputeSpec{
					Method: "computeActive",
					Deps:   []string{"state"},
					Store:  true,
				},
			},
			Migration: FieldMigrationDecision{
				StorageKind:        "virtual",
				ShouldCreateColumn: false,
				ReasonCode:         "COMPUTE_VIRTUAL",
			},
		}

		field := &Field{}
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
		field := &Field{ResolvedSpec: "not-valid-json"}
		_, err := field.GetResolvedSpec()
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
	})
}

func ptr[T any](v T) *T {
	return &v
}
