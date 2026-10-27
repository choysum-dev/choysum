// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"log/slog"
	"strings"
	"time"
)

// moduleCommitTxHoldInfoMinDuration is the floor for successful TX-hold logs at Info.
// Kept separate from moduleOperationStepInfoMinDuration so step vs hold thresholds can diverge.
const moduleCommitTxHoldInfoMinDuration = 100 * time.Millisecond

// LogModuleCommitTxHold records how long a per-module Commit Required
// transaction was held (install / upgrade / uninstall).
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
	msg := op + " commit transaction hold completed"
	if err != nil {
		attrs = append(attrs, "error", err.Error())
		logger.Info(msg, attrs...)
		return
	}
	if duration >= moduleCommitTxHoldInfoMinDuration {
		logger.Info(msg, attrs...)
		return
	}
	logger.Debug(msg, attrs...)
}

// LogInstallOuterTxHold records install Commit / legacy outer Required hold time.
// Prefer LogModuleCommitTxHold for new call sites.
func LogInstallOuterTxHold(logger *slog.Logger, caller string, started time.Time, err error) {
	LogModuleCommitTxHold(logger, "install", caller, started, err)
}
