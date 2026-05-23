// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

type LogConfig struct {
	Format string `mapstructure:"format"`
	Level  string `mapstructure:"level"`
}

func NewDefaultLogConfig() *LogConfig {
	return &LogConfig{
		Level: "info",
	}
}
