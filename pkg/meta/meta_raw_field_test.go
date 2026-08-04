// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestRawField_SetResolvedSpec(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var field *RawField
		if err := field.SetResolvedSpec(&FieldResolvedSpec{FieldName: "x"}); err != nil {
			t.Fatalf("nil receiver: unexpected error: %v", err)
		}
	})

	t.Run("nil spec clears ResolvedSpec", func(t *testing.T) {
		field := &RawField{ResolvedSpec: `{"fieldName":"old"}`}
		if err := field.SetResolvedSpec(nil); err != nil {
			t.Fatalf("nil spec: unexpected error: %v", err)
		}
		if field.ResolvedSpec != "" {
			t.Fatalf("expected empty, got %q", field.ResolvedSpec)
		}
	})

	t.Run("valid spec serialises to JSON", func(t *testing.T) {
		field := &RawField{}
		spec := &FieldResolvedSpec{
			FieldName: "amount",
			Structural: FieldStructuralSpec{
				Name:      "amount",
				FieldType: "float",
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
			t.Fatalf("unmarshal: %v", err)
		}
		if roundtrip.FieldName != "amount" {
			t.Fatalf("FieldName = %q, want amount", roundtrip.FieldName)
		}
	})

	t.Run("marshal error", func(t *testing.T) {
		prev := rawFieldJSONMarshal
		t.Cleanup(func() { rawFieldJSONMarshal = prev })
		rawFieldJSONMarshal = func(any) ([]byte, error) {
			return nil, errors.New("marshal boom")
		}
		field := &RawField{}
		if err := field.SetResolvedSpec(&FieldResolvedSpec{FieldName: "x"}); err == nil || err.Error() != "marshal boom" {
			t.Fatalf("expected marshal boom, got %v", err)
		}
	})
}

func TestRawField_GetResolvedSpec(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var field *RawField
		spec, err := field.GetResolvedSpec()
		if err != nil || spec != nil {
			t.Fatalf("expected nil,nil got %#v %v", spec, err)
		}
	})

	t.Run("empty ResolvedSpec", func(t *testing.T) {
		field := &RawField{ResolvedSpec: "  "}
		spec, err := field.GetResolvedSpec()
		if err != nil || spec != nil {
			t.Fatalf("expected nil,nil got %#v %v", spec, err)
		}
	})

	t.Run("round trip", func(t *testing.T) {
		field := &RawField{}
		want := &FieldResolvedSpec{FieldName: "n"}
		if err := field.SetResolvedSpec(want); err != nil {
			t.Fatalf("set: %v", err)
		}
		got, err := field.GetResolvedSpec()
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if got == nil || got.FieldName != "n" {
			t.Fatalf("got %#v", got)
		}
	})

	t.Run("unmarshal error", func(t *testing.T) {
		field := &RawField{ResolvedSpec: "{not-json"}
		_, err := field.GetResolvedSpec()
		if err == nil {
			t.Fatal("expected unmarshal error")
		}
	})
}
