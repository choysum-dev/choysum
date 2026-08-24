// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package initdata

import (
	"errors"
	"strings"
	"testing"

	dataloader "github.com/choysum-dev/choysum/internal/module/evolution/data"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

func TestLoadErrorToMessage_Golden(t *testing.T) {
	err := &dataloader.LoadError{
		Kind:        dataloader.LoadErrorKindRef,
		Code:        dataloader.LoadErrorCodeRefNotFound,
		FilePath:    "/tmp/auth/data/bootstrap.json",
		RecordIndex: 1,
		Module:      "auth",
		Name:        "role_demo",
		Application: "auth",
		Model:       "Role",
		FieldPath:   "values.ParentId",
		Ref:         "auth.role_missing",
		Message:     "reference not found",
	}

	msg := LoadErrorToMessage(err)
	if msg.Type != importpkg.MessageError {
		t.Fatalf("type = %q, want error", msg.Type)
	}
	if msg.Row != 2 {
		t.Fatalf("row = %d, want RecordIndex+1", msg.Row)
	}
	if msg.Field != "values.ParentId" {
		t.Fatalf("field = %q", msg.Field)
	}
	if msg.Code != dataloader.LoadErrorCodeRefNotFound {
		t.Fatalf("code = %q", msg.Code)
	}
	if msg.RecordRef != "auth.role_missing" {
		t.Fatalf("record_ref = %q", msg.RecordRef)
	}
	if msg.Text == "" {
		t.Fatal("text should not be empty")
	}
	for _, want := range []string{"reference not found", "file=/tmp/auth/data/bootstrap.json", "module=auth", "name=role_demo", "model=auth.Role"} {
		if !strings.Contains(msg.Text, want) {
			t.Fatalf("text = %q, want substring %q", msg.Text, want)
		}
	}
}

func TestLoadErrorToMessage_nil(t *testing.T) {
	if (LoadErrorToMessage(nil) != importpkg.Message{}) {
		t.Fatal("nil LoadError should return zero Message")
	}
}

func TestLoadErrorToMessage_causeOnly(t *testing.T) {
	err := &dataloader.LoadError{
		Code:  dataloader.LoadErrorCodeDBError,
		Cause: errors.New("db failed"),
	}
	msg := LoadErrorToMessage(err)
	if !strings.Contains(msg.Text, "db failed") {
		t.Fatalf("text = %q, want cause text", msg.Text)
	}
}

func TestLoadErrorToMessage_suffixOnly(t *testing.T) {
	err := &dataloader.LoadError{
		Code:  dataloader.LoadErrorCodeMissingName,
		Model: "Role",
	}
	msg := LoadErrorToMessage(err)
	if !strings.Contains(msg.Text, "model=Role") {
		t.Fatalf("text = %q, want model suffix", msg.Text)
	}
}

func TestLoadErrorToMessage_negativeRecordIndex(t *testing.T) {
	msg := LoadErrorToMessage(&dataloader.LoadError{RecordIndex: -1, Message: "bad"})
	if msg.Row != 0 {
		t.Fatalf("row = %d, want 0 for negative RecordIndex", msg.Row)
	}
}
