// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

const defaultDbDialect = "sqlite"
const defaultSQLiteDSNParams = "mode=rwc&_fk=1&_busy_timeout=60000&_journal_mode=WAL"

type DbConfig struct {
	Dialect         string `mapstructure:"dialect"`
	DSN             string `mapstructure:"dsn"`
	MaxOpenConns    int    `mapstructure:"maxOpenConns"`
	MaxIdleConns    int    `mapstructure:"maxIdleConns"`
	ConnMaxLifetime int    `mapstructure:"connMaxLifetime"`
}

func NewDefaultDbConfig() *DbConfig {
	return &DbConfig{
		Dialect:         defaultDbDialect,
		MaxIdleConns:    2 * maxProcs,
		MaxOpenConns:    4 * maxProcs,
		ConnMaxLifetime: 60 * 60 * 1, // 1 hour,
	}
}

func DefaultSQLitePath(defaultChoysumPath string) string {
	root := strings.TrimSpace(defaultChoysumPath)
	if root == "" {
		return ""
	}
	return filepath.Clean(filepath.Join(root, "choysum.sqlite"))
}

func DefaultSQLiteDSN(defaultChoysumPath string) string {
	path := DefaultSQLitePath(defaultChoysumPath)
	if path == "" {
		return ""
	}
	return fmt.Sprintf("file:%s?%s", filepath.ToSlash(path), defaultSQLiteDSNParams)
}
