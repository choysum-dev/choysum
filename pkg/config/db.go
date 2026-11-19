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
	// Pool sizes stay 0 ("omitted") so defaultscope can choose by dialect:
	// sqlite → open=4/idle=2 (clamped), MySQL/Postgres → ServerDbPoolDefaults().
	// Explicit positive yaml/env values are preserved for every dialect.
	return &DbConfig{
		Dialect:         defaultDbDialect,
		MaxIdleConns:    0,
		MaxOpenConns:    0,
		ConnMaxLifetime: 60 * 60 * 1, // 1 hour,
	}
}

// ServerDbPoolDefaults returns GOMAXPROCS-scaled open/idle sizes for MySQL/Postgres.
func ServerDbPoolDefaults() (maxOpen, maxIdle int) {
	return 4 * maxProcs, 2 * maxProcs
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
