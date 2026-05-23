// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package defaultscope

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	slogGorm "github.com/orandin/slog-gorm"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
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
		gormDB, err = gorm.Open(postgres.Open(dsn), gormConfig)
	case "sqlite":
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
	// SQLite is very sensitive to concurrent writers across multiple pooled
	// connections and can easily end up in `database is locked` during migrations
	// / module install bursts.
	//
	// However, the module manager lease/migration logic may use short-lived
	// transactions via nested scope.Run calls while a caller is already inside
	// another scope.Run.
	// Using a *single* connection can deadlock. Keep the pool small but >1.
	if dialect == "sqlite" {
		sqlDB.SetMaxIdleConns(2)
		sqlDB.SetMaxOpenConns(2)
		sqlDB.SetConnMaxLifetime(0)
	} else {
		defaults := config.NewDefaultDbConfig()
		maxIdleConns := dbOpts.MaxIdleConns
		maxOpenConns := dbOpts.MaxOpenConns
		connMaxLifetimeSeconds := dbOpts.ConnMaxLifetimeSeconds

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

	return gormDB
}
