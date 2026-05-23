// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"reflect"
	"strings"
	"testing"
)

func TestExecutionErrorFields(t *testing.T) {
	typeOf := reflect.TypeOf(Execution{})

	field, ok := typeOf.FieldByName("ErrorHash")
	if !ok {
		t.Fatalf("missing ErrorHash field")
	}
	if tag := field.Tag.Get("gorm"); tag == "" || !containsTag(tag, "column:error_hash") {
		t.Fatalf("ErrorHash gorm tag mismatch: %q", tag)
	}

	field, ok = typeOf.FieldByName("ErrorTruncated")
	if !ok {
		t.Fatalf("missing ErrorTruncated field")
	}
	if tag := field.Tag.Get("gorm"); tag == "" || !containsTag(tag, "column:error_truncated") {
		t.Fatalf("ErrorTruncated gorm tag mismatch: %q", tag)
	}
}

func containsTag(tag string, fragment string) bool {
	return strings.Contains(tag, fragment)
}
