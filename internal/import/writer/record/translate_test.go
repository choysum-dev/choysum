// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package record

import (
	"encoding/json"
	"testing"
)

func TestEncodeTranslatedScalar(t *testing.T) {
	t.Parallel()
	got, err := encodeTranslatedScalar("Import Alpha")
	if err != nil {
		t.Fatalf("encodeTranslatedScalar: %v", err)
	}
	var parsed map[string]string
	if err := json.Unmarshal([]byte(got), &parsed); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if parsed[translatedBaseLang] != "Import Alpha" {
		t.Fatalf("parsed = %#v", parsed)
	}
}
