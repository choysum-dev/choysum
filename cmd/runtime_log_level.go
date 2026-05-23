// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"os"
	"strings"

	"github.com/choysum-dev/choysum/internal/logger"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

func cloneCommandLogConfig(cfg *config.LogConfig) *config.LogConfig {
	if cfg == nil {
		return config.NewDefaultLogConfig()
	}
	cloned := *cfg
	return &cloned
}

func normalizeRuntimeLogLevelFlag(level string, commandName string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(level))
	if normalized == "" {
		return "warn", nil
	}
	switch normalized {
	case "debug", "info", "warn", "error":
		return normalized, nil
	default:
		return "", xfmt.Errorf("%s: invalid --runtime-log-level %q (expected debug|info|warn|error)", commandName, level)
	}
}

func rebuildScopeWithRuntimeLogLevel(runtimeScope scope.Scope, level string, commandName string) (scope.Scope, error) {
	if runtimeScope == nil {
		return nil, xfmt.Errorf("scope is not initialized")
	}
	normalizedLevel, err := normalizeRuntimeLogLevelFlag(level, commandName)
	if err != nil {
		return nil, err
	}
	factoryInput := scope.FactoryInputFromScope(runtimeScope)
	if factoryInput == nil {
		return nil, xfmt.Errorf("config is not initialized")
	}
	logCfg := cloneCommandLogConfig(scope.LogConfigFromScope(runtimeScope))
	logCfg.Level = normalizedLevel
	rebuiltScope := scope.NewScope(runtimeScope.Context(), factoryInput, logger.NewLoggerWithWriter(logCfg, os.Stderr))
	if rebuiltScope == nil {
		return nil, xfmt.Errorf("failed to initialize scope for runtime log level")
	}
	return rebuiltScope, nil
}
