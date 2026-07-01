// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runtime

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/choysum-dev/choysum/internal/logger"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func NewRunDBOptions(cfg *config.Config) RunDBOptions {
	if cfg == nil || cfg.Db == nil {
		return RunDBOptions{}
	}
	dialect := strings.ToLower(strings.TrimSpace(cfg.Db.Dialect))
	dsn := cfg.Db.DSN
	return RunDBOptions{
		Dialect:     dialect,
		DSN:         dsn,
		AllowCreate: dialect == "sqlite" && IsDefaultRunSQLitePath(dsn, config.DefaultSQLitePath(cfg.DefaultChoysumPath)),
	}
}

func NewScopeForRun(scopeInput RunScopeInput, logConfig *config.LogConfig) (scope.Scope, error) {
	if err := scopeInput.CLIOptions().Validate(); err != nil {
		return nil, fmt.Errorf("invalid cli runtime options: %w", err)
	}
	if err := scopeInput.ServerOptions().Validate(); err != nil {
		return nil, fmt.Errorf("invalid run server options: %w", err)
	}
	if err := scopeInput.DBOptions().Validate(); err != nil {
		return nil, fmt.Errorf("invalid run db options: %w", err)
	}
	if scopeInput.ConfigOptions() == nil {
		return nil, fmt.Errorf("config is required")
	}

	var runtimeScope scope.Scope
	var panicErr any

	func() {
		defer func() {
			if r := recover(); r != nil {
				panicErr = r
			}
		}()
		l := logger.NewLoggerWithWriter(CloneLogConfig(logConfig), os.Stderr)
		runtimeScope = scope.NewScope(context.Background(), scopeInput, l)
	}()

	if panicErr != nil {
		return nil, fmt.Errorf("failed to initialize scope: %v", panicErr)
	}
	if runtimeScope == nil {
		return nil, fmt.Errorf("failed to initialize scope")
	}
	return runtimeScope, nil
}
