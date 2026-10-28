// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"log/slog"
	"strings"
	"time"
)

// LogModuleCommitTxHold records how long a per-module Commit Required
// transaction was held (install / upgrade / uninstall). Successful holds stay
// Debug (ops metric); failures log at Error.
func LogModuleCommitTxHold(logger *slog.Logger, op string, caller string, started time.Time, err error) {
	if logger == nil || started.IsZero() {
		return
	}
	logModuleCommitTxHold(logger, op, caller, time.Since(started), err)
}

func logModuleCommitTxHold(logger *slog.Logger, op string, caller string, duration time.Duration, err error) {
	if logger == nil {
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
		"duration_ms", duration.Milliseconds(),
	}
	if err != nil {
		// Pass the error value (not err.Error()) so console/devslog/json
		// ReplaceAttr can expand xfmt / QuickJS frames into a stack trace.
		attrs = append(attrs, "error", err)
		logger.Error(op+" commit transaction hold failed", attrs...)
		return
	}
	logger.Debug(op+" commit transaction hold completed", attrs...)
}

// LogInstallOuterTxHold records install Commit / legacy outer Required hold time.
// Prefer LogModuleCommitTxHold for new call sites.
func LogInstallOuterTxHold(logger *slog.Logger, caller string, started time.Time, err error) {
	LogModuleCommitTxHold(logger, "install", caller, started, err)
}
