// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

// Package sqliteretry retries transient SQLite lock/busy errors on short writes.
package sqliteretry

import (
	"strings"
	"time"
)

// WithLockRetry retries op when SQLite returns a transient lock/busy error.
func WithLockRetry(op func() error) error {
	const maxAttempts = 8
	var err error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		err = op()
		if err == nil {
			return nil
		}
		if !IsTransientLock(err) {
			return err
		}
		if attempt == maxAttempts-1 {
			break
		}
		time.Sleep(time.Duration(attempt+1) * 50 * time.Millisecond)
	}
	return err
}

// IsTransientLock reports whether err looks like a retryable SQLite lock/busy failure.
func IsTransientLock(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") ||
		strings.Contains(msg, "database table is locked") ||
		strings.Contains(msg, "database is busy") ||
		strings.Contains(msg, "database schema is locked")
}
