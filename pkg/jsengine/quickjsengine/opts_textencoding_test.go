// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsengine

import (
	"testing"
)

func TestDecodeUTF8Bytes_NonFatalReplacesInvalidBytes(t *testing.T) {
	got, err := decodeUTF8Bytes([]byte{0x66, 0x6f, 0x80, 0x6f}, false)
	if err != nil {
		t.Fatalf("decodeUTF8Bytes returned error: %v", err)
	}
	if got != "fo\ufffdo" {
		t.Fatalf("decodeUTF8Bytes = %q, want %q", got, "fo\ufffdo")
	}
}

func TestDecodeUTF8Bytes_FatalRejectsInvalidBytes(t *testing.T) {
	_, err := decodeUTF8Bytes([]byte{0x80}, true)
	if err == nil {
		t.Fatal("expected error for invalid UTF-8 in fatal mode")
	}
}

func TestDecodeUTF8Bytes_PreservesValidUTF8(t *testing.T) {
	got, err := decodeUTF8Bytes([]byte("hello"), false)
	if err != nil {
		t.Fatalf("decodeUTF8Bytes returned error: %v", err)
	}
	if got != "hello" {
		t.Fatalf("decodeUTF8Bytes = %q, want %q", got, "hello")
	}
}
