// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package sqliteretry

import (
	"errors"
	"strings"
	"testing"
)

func TestIsTransientLock(t *testing.T) {
	if IsTransientLock(nil) {
		t.Fatal("nil")
	}
	if !IsTransientLock(errors.New("database is locked")) {
		t.Fatal("locked")
	}
	if !IsTransientLock(errors.New("database table is locked")) {
		t.Fatal("table locked")
	}
	if !IsTransientLock(errors.New("Database Is Busy")) {
		t.Fatal("busy")
	}
	if !IsTransientLock(errors.New("database schema is locked")) {
		t.Fatal("schema locked")
	}
	if !IsTransientLock(errors.New("locking protocol")) {
		t.Fatal("protocol")
	}
	if IsTransientLock(errors.New("unique constraint")) {
		t.Fatal("non-lock")
	}
}

func TestWithLockRetry(t *testing.T) {
	if err := WithLockRetry(func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	calls := 0
	err := WithLockRetry(func() error {
		calls++
		if calls < 3 {
			return errors.New("database table is locked")
		}
		return nil
	})
	if err != nil || calls != 3 {
		t.Fatalf("retry until success: calls=%d err=%v", calls, err)
	}
	calls = 0
	err = WithLockRetry(func() error {
		calls++
		return errors.New("unique constraint failed")
	})
	if err == nil || calls != 1 || !strings.Contains(err.Error(), "unique") {
		t.Fatalf("non-lock must not retry: calls=%d err=%v", calls, err)
	}
	calls = 0
	err = WithLockRetry(func() error {
		calls++
		return errors.New("database is locked")
	})
	if err == nil || calls != 8 || !strings.Contains(err.Error(), "locked") {
		t.Fatalf("exhaust retries: calls=%d err=%v", calls, err)
	}
}
