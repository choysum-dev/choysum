// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package scripts

import (
	"context"
	"strings"
	"testing"
)

func TestHistoryStore_PrepareCutoverError(t *testing.T) {
	runtimeScope := &testScope{db: nil}
	store := NewHistoryStore(runtimeScope)
	_, _, err := store.Prepare(context.Background(), Script{
		ModuleName: "meta",
		Version:    "0.1.0",
		Phase:      PhasePre,
		Name:       "init",
		Checksum:   "abc",
	})
	if err == nil || !strings.Contains(err.Error(), "db is nil") {
		t.Fatalf("Prepare() error = %v, want db is nil", err)
	}
}
