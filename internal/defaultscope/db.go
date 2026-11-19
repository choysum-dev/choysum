// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package defaultscope

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	slogGorm "github.com/orandin/slog-gorm"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func sqlTraceEnabled(ctx context.Context, logger *slog.Logger) bool {
	if logger == nil {
		return false
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return logger.Enabled(ctx, slog.LevelDebug)
}

func newDb(ctx context.Context, dbOpts scope.DatabaseRuntimeOptions, logger *slog.Logger) *gorm.DB {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	gormLoggerOptions := []slogGorm.Option{
		slogGorm.WithHandler(logger.Handler()),
		slogGorm.SetLogLevel(slogGorm.ErrorLogType, slog.LevelError),
		slogGorm.SetLogLevel(slogGorm.SlowQueryLogType, slog.LevelWarn),
		slogGorm.SetLogLevel(slogGorm.DefaultLogType, slog.LevelDebug),
	}
	if sqlTraceEnabled(ctx, logger) {
		gormLoggerOptions = append(gormLoggerOptions, slogGorm.WithTraceAll())
	} else {
		gormLoggerOptions = append(gormLoggerOptions, slogGorm.WithIgnoreTrace())
	}
	gormLogger := slogGorm.New(gormLoggerOptions...)

	var gormDB *gorm.DB
	var err error
	gormConfig := &gorm.Config{
		SkipDefaultTransaction: true,
		Logger:                 gormLogger,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	}

	dialect := strings.ToLower(strings.TrimSpace(dbOpts.Dialect))
	dsn := strings.TrimSpace(dbOpts.DSN)
	// here we don't support sqlserver, cause it's not supported json type
	switch dialect {
	case "mysql":
		gormDB, err = gorm.Open(mysql.Open(dsn), gormConfig)
	case "postgres":
		gormDB, err = gorm.Open(openPostgres(dsn), gormConfig)
	case "sqlite":
		if err := ensureSQLiteParentDir(dsn); err != nil {
			panic(err)
		}
		gormConfig.DisableForeignKeyConstraintWhenMigrating = true // sqlite doesn't support foreign key on alter table
		gormDB, err = gorm.Open(sqlite.Open(dsn), gormConfig)
	default:
		panic("Invalid database dialect")
	}
	if err != nil {
		panic(err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		panic(err)
	}
	defaults := config.NewDefaultDbConfig()
	maxIdleConns := dbOpts.MaxIdleConns
	maxOpenConns := dbOpts.MaxOpenConns
	connMaxLifetimeSeconds := dbOpts.ConnMaxLifetimeSeconds

	if dialect == "sqlite" {
		// SQLite writers are serialized; a large pool does not scale like PG/MySQL
		// and raises `database is locked` during install/migration bursts.
		// Nested scope.Run / lease renew still need >1 connection (single-conn
		// deadlocks). Honor db.maxOpenConns/maxIdleConns with a small clamp.
		//
		// Config loader seeds MaxOpen/Idle with NewDefaultDbConfig (4*maxProcs /
		// 2*maxProcs). Treat those loader defaults as "omitted" so documented
		// sqlite defaults (open=4, idle=2) apply unless the user set another value.
		if maxOpenConns == defaults.MaxOpenConns {
			maxOpenConns = 0
		}
		if maxIdleConns == defaults.MaxIdleConns {
			maxIdleConns = 0
		}
		maxOpenConns, maxIdleConns = sqlitePoolLimits(maxOpenConns, maxIdleConns)
		sqlDB.SetMaxOpenConns(maxOpenConns)
		sqlDB.SetMaxIdleConns(maxIdleConns)
		sqlDB.SetConnMaxLifetime(0)
	} else {
		if maxIdleConns <= 0 {
			maxIdleConns = defaults.MaxIdleConns
		}
		if maxOpenConns <= 0 {
			maxOpenConns = defaults.MaxOpenConns
		}
		if connMaxLifetimeSeconds <= 0 {
			connMaxLifetimeSeconds = defaults.ConnMaxLifetime
		}

		sqlDB.SetMaxIdleConns(maxIdleConns)
		sqlDB.SetMaxOpenConns(maxOpenConns)
		sqlDB.SetConnMaxLifetime(time.Duration(connMaxLifetimeSeconds) * time.Second)
	}
	if sqlDB.Ping() != nil {
		panic("Failed to connect to database")
	}
	if dialect == "mysql" {
		if err := ensureMySQLTimezoneTables(sqlDB); err != nil {
			panic(err)
		}
	}

	return gormDB
}

// mysqlTimezoneTablesMissingMsg is returned when CONVERT_TZ cannot resolve IANA zones.
// Missing mysql.time_zone* tables make CONVERT_TZ return NULL and silently corrupt time buckets.
const mysqlTimezoneTablesMissingMsg = "MySQL timezone tables are missing or incomplete (CONVERT_TZ returned NULL). Load IANA zones (e.g. mysql_tzinfo_to_sql /usr/share/zoneinfo | mysql -u root mysql) before using timezone-aware time buckets."

// mysqlTimezoneProbeNamedZone is a named IANA id used at connect time.
// UTC→UTC can succeed with incomplete tz tables; named zones exercise real lookups.
const mysqlTimezoneProbeNamedZone = "Asia/Shanghai"

// checkMySQLTimezoneProbe interprets the CONVERT_TZ(UTC, named-zone) IS NOT NULL probe result.
func checkMySQLTimezoneProbe(convertTzNamedZoneNotNull sql.NullBool, queryErr error) error {
	if queryErr != nil {
		return fmt.Errorf("MySQL timezone tables probe failed: %w", queryErr)
	}
	if !convertTzNamedZoneNotNull.Valid || !convertTzNamedZoneNotNull.Bool {
		return errors.New(mysqlTimezoneTablesMissingMsg)
	}
	return nil
}

func ensureMySQLTimezoneTables(sqlDB *sql.DB) error {
	var ok sql.NullBool
	// Probe a named IANA zone — UTC→UTC alone can pass when named zones are still missing.
	err := sqlDB.QueryRow(`SELECT CONVERT_TZ(UTC_TIMESTAMP(), 'UTC', '` + mysqlTimezoneProbeNamedZone + `') IS NOT NULL`).Scan(&ok)
	return checkMySQLTimezoneProbe(ok, err)
}

func ensureSQLiteParentDir(dsn string) error {
	path, ok := sqliteFilePathFromDSN(dsn)
	if !ok {
		return nil
	}
	dir := filepath.Dir(path)
	if strings.TrimSpace(dir) == "" || dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

func sqliteFilePathFromDSN(dsn string) (string, bool) {
	trimmed := strings.TrimSpace(dsn)
	if trimmed == "" || strings.EqualFold(trimmed, ":memory:") {
		return "", false
	}
	if strings.HasPrefix(strings.ToLower(trimmed), "file:") || strings.Contains(trimmed, "://") {
		parsed, err := url.Parse(trimmed)
		if err != nil {
			return "", false
		}
		scheme := strings.ToLower(parsed.Scheme)
		if scheme != "" && scheme != "file" {
			return "", false
		}
		path := parsed.Path
		if path == "" {
			path = parsed.Opaque
		}
		if strings.TrimSpace(path) == "" || strings.EqualFold(strings.TrimSpace(path), ":memory:") {
			return "", false
		}
		return path, true
	}
	return trimmed, true
}

// sqlitePoolLimits clamps SQLite connection pool size for nested leases without
// pretending SQLite scales like a server DB.
//
// Defaults when unset or non-positive: open=4, idle=2.
// Floor 2 (nested scope.Run / lease renew), ceiling 8 for MaxOpenConns.
func sqlitePoolLimits(maxOpen, maxIdle int) (open, idle int) {
	const (
		sqliteDefaultMaxOpen = 4
		sqliteDefaultMaxIdle = 2
		sqliteMinConns       = 2
		sqliteMaxOpenCap     = 8
	)
	open = maxOpen
	if open <= 0 {
		open = sqliteDefaultMaxOpen
	}
	if open < sqliteMinConns {
		open = sqliteMinConns
	}
	if open > sqliteMaxOpenCap {
		open = sqliteMaxOpenCap
	}
	idle = maxIdle
	if idle <= 0 {
		idle = sqliteDefaultMaxIdle
	}
	if idle < sqliteMinConns {
		idle = sqliteMinConns
	}
	if idle > open {
		idle = open
	}
	return open, idle
}
