// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"log/slog"
	"strings"
	"time"
)

// LogModuleCommitTxHold records how long a per-module Commit Required
// transaction was held (install / upgrade / uninstall).
func LogModuleCommitTxHold(logger *slog.Logger, op string, caller string, started time.Time, err error) {
	if logger == nil || started.IsZero() {
		return
	}
	op = strings.TrimSpace(op)
	if op == "" {
		op = "module"
	}
	caller = strings.TrimSpace(caller)
	if caller == "" {
		caller = "unknown"
	}
	attrs := []any{
		"op", op,
		"caller", caller,
		"duration_ms", time.Since(started).Milliseconds(),
	}
	msg := op + " commit transaction hold completed"
	if err != nil {
		attrs = append(attrs, "error", err.Error())
	}
	logger.Info(msg, attrs...)
}

// LogInstallOuterTxHold records install Commit / legacy outer Required hold time.
// Prefer LogModuleCommitTxHold for new call sites.
func LogInstallOuterTxHold(logger *slog.Logger, caller string, started time.Time, err error) {
	LogModuleCommitTxHold(logger, "install", caller, started, err)
}
