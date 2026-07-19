// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"log/slog"
	"strings"
	"time"
)

// LogInstallOuterTxHold records how long an outer Required install transaction
// was held (bootstrap / CLI wrappers). Used for Prepare–Commit TX boundary work.
func LogInstallOuterTxHold(logger *slog.Logger, caller string, started time.Time, err error) {
	if logger == nil || started.IsZero() {
		return
	}
	caller = strings.TrimSpace(caller)
	if caller == "" {
		caller = "unknown"
	}
	attrs := []any{
		"caller", caller,
		"duration_ms", time.Since(started).Milliseconds(),
	}
	if err != nil {
		attrs = append(attrs, "error", err.Error())
		logger.Info("install outer transaction hold completed", attrs...)
		return
	}
	logger.Info("install outer transaction hold completed", attrs...)
}
