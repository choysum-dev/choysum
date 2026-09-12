// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendbuilder

import (
	"errors"
	"strings"
	"testing"
)

func TestIsTransientSQLiteLock(t *testing.T) {
	if isTransientSQLiteLock(nil) {
		t.Fatal("nil")
	}
	if !isTransientSQLiteLock(errors.New("database is locked")) {
		t.Fatal("locked")
	}
	if !isTransientSQLiteLock(errors.New("Database Is Busy")) {
		t.Fatal("busy")
	}
	if !isTransientSQLiteLock(errors.New("database schema is locked")) {
		t.Fatal("schema locked")
	}
	if !isTransientSQLiteLock(errors.New("locking protocol")) {
		t.Fatal("protocol")
	}
	if isTransientSQLiteLock(errors.New("unique constraint")) {
		t.Fatal("non-lock")
	}
}

func TestWithSQLiteLockRetry(t *testing.T) {
	if err := withSQLiteLockRetry(func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	calls := 0
	err := withSQLiteLockRetry(func() error {
		calls++
		if calls < 3 {
			return errors.New("database is locked")
		}
		return nil
	})
	if err != nil || calls != 3 {
		t.Fatalf("retry until success: calls=%d err=%v", calls, err)
	}
	calls = 0
	err = withSQLiteLockRetry(func() error {
		calls++
		return errors.New("unique constraint failed")
	})
	if err == nil || calls != 1 || !strings.Contains(err.Error(), "unique") {
		t.Fatalf("non-lock must not retry: calls=%d err=%v", calls, err)
	}
	calls = 0
	err = withSQLiteLockRetry(func() error {
		calls++
		return errors.New("database is locked")
	})
	if err == nil || calls != 8 || !strings.Contains(err.Error(), "locked") {
		t.Fatalf("exhaust retries: calls=%d err=%v", calls, err)
	}
}
