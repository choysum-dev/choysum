// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendbuilder

import (
	"strings"
	"time"
)

// withSQLiteLockRetry retries op when SQLite returns a transient lock/busy error.
// Module Persist can race with lease/index writers during dense install graphs.
func withSQLiteLockRetry(op func() error) error {
	const maxAttempts = 8
	var err error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		err = op()
		if err == nil {
			return nil
		}
		if !isTransientSQLiteLock(err) {
			return err
		}
		if attempt == maxAttempts-1 {
			break
		}
		time.Sleep(time.Duration(attempt+1) * 50 * time.Millisecond)
	}
	return err
}

func isTransientSQLiteLock(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") ||
		strings.Contains(msg, "database is busy") ||
		strings.Contains(msg, "database schema is locked") ||
		strings.Contains(msg, "locking protocol")
}
