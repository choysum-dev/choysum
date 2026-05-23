// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"testing"

	"github.com/choysum-dev/choysum/pkg/config"
)

func TestSanitizePayload_MasksSensitive(t *testing.T) {
	input := map[string]any{
		"password": "secret",
		"token":    "abc",
		"profile": map[string]any{
			"access_token": "nested",
			"name":         "alice",
		},
		"items": []any{
			map[string]any{"refresh_token": "r1"},
			"plain",
		},
	}

	res, err := SanitizePayload(input)
	if err != nil {
		t.Fatalf("sanitize failed: %v", err)
	}
	if res.Truncated {
		t.Fatalf("expected not truncated")
	}
	if res.Hash == "" {
		t.Fatalf("expected hash")
	}
	masked, ok := res.Value.(map[string]any)
	if !ok {
		t.Fatalf("unexpected result type: %T", res.Value)
	}
	if masked["password"] != maskValue {
		t.Fatalf("password not masked")
	}
	if masked["token"] != maskValue {
		t.Fatalf("token not masked")
	}
	profile, ok := masked["profile"].(map[string]any)
	if !ok {
		t.Fatalf("profile not map")
	}
	if profile["access_token"] != maskValue {
		t.Fatalf("access_token not masked")
	}
	if profile["name"] != "alice" {
		t.Fatalf("non-sensitive field modified")
	}
	items, ok := masked["items"].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("items not array")
	}
	item0, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("items[0] not map")
	}
	if item0["refresh_token"] != maskValue {
		t.Fatalf("refresh_token not masked")
	}
}

func TestSanitizeAndTruncate_Truncates(t *testing.T) {
	input := map[string]any{
		"value": "abcdefghijklmnopqrstuvwxyz",
	}
	res, err := sanitizeAndTruncate(input, 10)
	if err != nil {
		t.Fatalf("sanitize failed: %v", err)
	}
	if !res.Truncated {
		t.Fatalf("expected truncated")
	}
	if res.EncodedPreview == "" || len(res.EncodedPreview) != 10 {
		t.Fatalf("unexpected preview length: %d", len(res.EncodedPreview))
	}
	val, ok := res.Value.(map[string]any)
	if !ok {
		t.Fatalf("unexpected value type: %T", res.Value)
	}
	if v, ok := val["_truncated"].(bool); !ok || !v {
		t.Fatalf("missing truncated flag")
	}
}

func TestSanitizeWithTaskConfig_CustomMaxBytes(t *testing.T) {
	input := map[string]any{
		"value": "abcdefghijklmnopqrstuvwxyz",
	}
	cfg := &config.TaskSanitizeConfig{
		PayloadMaxBytes: 10,
		ResultMaxBytes:  10,
		ErrorMaxBytes:   10,
	}

	res, err := SanitizePayloadWithTaskConfig(cfg, input)
	if err != nil {
		t.Fatalf("sanitize payload failed: %v", err)
	}
	if !res.Truncated {
		t.Fatalf("expected payload truncated")
	}

	res, err = SanitizeResultWithTaskConfig(cfg, input)
	if err != nil {
		t.Fatalf("sanitize result failed: %v", err)
	}
	if !res.Truncated {
		t.Fatalf("expected result truncated")
	}

	res, err = SanitizeErrorWithTaskConfig(cfg, input)
	if err != nil {
		t.Fatalf("sanitize error failed: %v", err)
	}
	if !res.Truncated {
		t.Fatalf("expected error truncated")
	}
}

func TestSanitizeDefaultWrappers(t *testing.T) {
	input := map[string]any{
		"token": "secret-token",
		"value": "ok",
	}

	resultRes, err := SanitizeResult(input)
	if err != nil {
		t.Fatalf("SanitizeResult() error = %v", err)
	}
	resultMap, ok := resultRes.Value.(map[string]any)
	if !ok {
		t.Fatalf("unexpected result value type: %T", resultRes.Value)
	}
	if resultMap["token"] != maskValue || resultMap["value"] != "ok" {
		t.Fatalf("unexpected sanitized result payload: %#v", resultMap)
	}

	errorRes, err := SanitizeError(input)
	if err != nil {
		t.Fatalf("SanitizeError() error = %v", err)
	}
	errorMap, ok := errorRes.Value.(map[string]any)
	if !ok {
		t.Fatalf("unexpected error value type: %T", errorRes.Value)
	}
	if errorMap["token"] != maskValue || errorMap["value"] != "ok" {
		t.Fatalf("unexpected sanitized error payload: %#v", errorMap)
	}
}

func TestSanitizeHelperConfigAndEncoding(t *testing.T) {
	t.Run("size helpers use defaults and overrides", func(t *testing.T) {
		if payloadMaxBytes(nil) != defaultPayloadMaxBytes {
			t.Fatalf("payloadMaxBytes(nil) = %d, want %d", payloadMaxBytes(nil), defaultPayloadMaxBytes)
		}
		if resultMaxBytes(nil) != defaultResultMaxBytes {
			t.Fatalf("resultMaxBytes(nil) = %d, want %d", resultMaxBytes(nil), defaultResultMaxBytes)
		}
		if errorMaxBytes(nil) != defaultErrorMaxBytes {
			t.Fatalf("errorMaxBytes(nil) = %d, want %d", errorMaxBytes(nil), defaultErrorMaxBytes)
		}

		cfg := &config.TaskSanitizeConfig{
			PayloadMaxBytes: 123,
			ResultMaxBytes:  456,
			ErrorMaxBytes:   789,
		}
		if payloadMaxBytes(cfg) != 123 {
			t.Fatalf("payloadMaxBytes(cfg) = %d, want 123", payloadMaxBytes(cfg))
		}
		if resultMaxBytes(cfg) != 456 {
			t.Fatalf("resultMaxBytes(cfg) = %d, want 456", resultMaxBytes(cfg))
		}
		if errorMaxBytes(cfg) != 789 {
			t.Fatalf("errorMaxBytes(cfg) = %d, want 789", errorMaxBytes(cfg))
		}
	})

	t.Run("sorted encoding is deterministic and surfaces marshal errors", func(t *testing.T) {
		encoded, err := encodeSortedJSON(map[string]any{
			"b": 1,
			"a": []any{2, map[string]any{"d": 4, "c": 3}},
		})
		if err != nil {
			t.Fatalf("encodeSortedJSON() error = %v", err)
		}
		if string(encoded) != "{\"a\":[2,{\"c\":3,\"d\":4}],\"b\":1}" {
			t.Fatalf("unexpected encoded JSON: %s", string(encoded))
		}

		if _, err := encodeSortedJSON(map[string]any{"bad": func() {}}); err == nil {
			t.Fatal("expected encodeSortedJSON to fail on unsupported values")
		}
	})
}
