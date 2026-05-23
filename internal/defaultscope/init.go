// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package defaultscope

import (
	"context"
	"log/slog"

	"github.com/choysum-dev/choysum/pkg/scope"
)

func init() {
	// Register the default scope factory.
	scope.Register("default", func(ctx context.Context, input scope.FactoryInput, logger *slog.Logger) scope.Scope {
		runtimeScope, reason := newDefaultScopeFromInputWithReason(ctx, input, logger)
		if runtimeScope == nil {
			if logger != nil {
				logger.Error("default scope unavailable", "reason", reason)
			}
			return nil
		}
		return runtimeScope
	})
}
