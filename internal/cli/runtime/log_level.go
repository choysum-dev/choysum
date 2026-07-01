// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runtime

import (
	"strings"

	"github.com/choysum-dev/choysum/pkg/config"
	xfmt "golang.org/x/exp/errors/fmt"
)

func CloneLogConfig(cfg *config.LogConfig) *config.LogConfig {
	if cfg == nil {
		return config.NewDefaultLogConfig()
	}
	cloned := *cfg
	return &cloned
}

func NormalizeRuntimeLogLevelFlag(level string, commandName string) (string, error) {
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
