// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package defaultscope

import (
	"database/sql"
	"errors"
	"strings"
	"testing"
)

func TestCheckMySQLTimezoneProbe(t *testing.T) {
	t.Parallel()

	t.Run("ready", func(t *testing.T) {
		t.Parallel()
		if err := checkMySQLTimezoneProbe(sql.NullBool{Valid: true, Bool: true}, nil); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})

	t.Run("missing tables", func(t *testing.T) {
		t.Parallel()
		err := checkMySQLTimezoneProbe(sql.NullBool{Valid: true, Bool: false}, nil)
		if err == nil {
			t.Fatal("expected error when CONVERT_TZ probe is false")
		}
		if !strings.Contains(err.Error(), "timezone tables are missing") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("null result", func(t *testing.T) {
		t.Parallel()
		err := checkMySQLTimezoneProbe(sql.NullBool{}, nil)
		if err == nil {
			t.Fatal("expected error when probe scan is null")
		}
	})

	t.Run("query error", func(t *testing.T) {
		t.Parallel()
		err := checkMySQLTimezoneProbe(sql.NullBool{}, errors.New("connection reset"))
		if err == nil {
			t.Fatal("expected wrapped query error")
		}
		if !strings.Contains(err.Error(), "timezone tables probe failed") {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(err.Error(), "connection reset") {
			t.Fatalf("expected cause in error: %v", err)
		}
	})
}
