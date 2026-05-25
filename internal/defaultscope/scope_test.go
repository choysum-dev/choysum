// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package defaultscope

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/gorm"
)

type runRecord struct {
	ID   int `gorm:"primaryKey"`
	Name string
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func testBufferedLogger(buffer *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buffer, nil))
}

func testLoggerWithLevel(level slog.Level) *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: level}))
}

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	return &config.Config{
		Db: &config.DbConfig{
			Dialect:         "sqlite",
			DSN:             filepath.Join(t.TempDir(), "choysum.db"),
			MaxIdleConns:    7,
			MaxOpenConns:    9,
			ConnMaxLifetime: 60,
		},
	}
}

func TestNewDefaultScopeRegistersFactory(t *testing.T) {
	if !scope.Exists("default") {
		t.Fatal("expected default scope to be registered in factory")
	}
}

func TestDefaultScopeFactoryLogsOnMissingDatabaseConfig(t *testing.T) {
	var buffer bytes.Buffer
	logger := testBufferedLogger(&buffer)

	created := scope.NewScopeByName("default", context.Background(), scopetest.FactoryInputFromConfig(&config.Config{}), logger)
	if created != nil {
		t.Fatalf("expected nil scope for missing database config, got %#v", created)
	}

	output := buffer.String()
	if !strings.Contains(output, "default scope unavailable") || !strings.Contains(output, "reason=\"missing database dialect or dsn\"") {
		t.Fatalf("expected structured default scope failure log, got %q", output)
	}
}

func TestSessionAndNewWithoutDatabase(t *testing.T) {
	ctx := context.WithValue(context.Background(), "key", "value")
	logger := testLogger()
	runtimeScope := &defaultScope{ctx: ctx, logger: logger}

	if runtimeScope.Session() != nil {
		t.Fatal("expected Session() to return nil without db")
	}

	derivedCtx := context.WithValue(context.Background(), "key", "child")
	child := runtimeScope.WithContext(derivedCtx).(*defaultScope)
	if child.Context() != derivedCtx {
		t.Fatal("expected derived context to be preserved")
	}
	if child.Logger() != logger {
		t.Fatal("expected logger to be preserved")
	}
	if child.Session() != nil {
		t.Fatal("expected child Session() to return nil without db")
	}
}

func TestWithContextRebindsFromTransactionContext(t *testing.T) {
	runtimeScope := NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(testConfig(t)), testLogger()).(*defaultScope)
	err := runtimeScope.Transactor().Required(context.Background(), func(txScope scope.Scope, tx scope.Transaction) error {
		child := runtimeScope.WithContext(tx.Context()).(*defaultScope)
		if child.tx == nil {
			t.Fatal("expected transaction to be rebound from context")
		}
		if child.session == nil {
			t.Fatal("expected transaction session to be rebound from context")
		}
		if child.Session() != tx.Session() {
			t.Fatal("expected child Session() to return transaction-bound session")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Required with transaction context: %v", err)
	}
}

func TestRunCommitsAndRollsBack(t *testing.T) {
	cfg := testConfig(t)
	runtimeScope := NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(cfg), testLogger()).(*defaultScope)

	if err := runtimeScope.Session().AutoMigrate(&runRecord{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	if err := runtimeScope.Run(func(execScope scope.Scope) error {
		tx := execScope.Session()
		if tx == nil {
			return errors.New("missing transaction session")
		}
		return tx.Create(&runRecord{Name: "committed"}).Error
	}); err != nil {
		t.Fatalf("Run commit path: %v", err)
	}
	if runtimeScope.session != nil {
		t.Fatal("expected session to be cleared after successful Run")
	}

	var count int64
	if err := runtimeScope.Session().Model(&runRecord{}).Where("name = ?", "committed").Count(&count).Error; err != nil {
		t.Fatalf("Count committed rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("committed row count = %d, want 1", count)
	}

	wantErr := errors.New("force rollback")
	err := runtimeScope.Run(func(execScope scope.Scope) error {
		if err := execScope.Session().Create(&runRecord{Name: "rolled-back"}).Error; err != nil {
			return err
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run rollback path error = %v, want %v", err, wantErr)
	}
	if runtimeScope.session != nil {
		t.Fatal("expected session to be cleared after rollback")
	}

	if err := runtimeScope.Session().Model(&runRecord{}).Where("name = ?", "rolled-back").Count(&count).Error; err != nil {
		t.Fatalf("Count rolled-back rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("rolled-back row count = %d, want 0", count)
	}
}

func TestNewDefaultScopeAndNewDB(t *testing.T) {
	ctx := context.WithValue(context.Background(), "trace", "root")
	cfg := testConfig(t)
	logger := testLogger()
	runtimeScope := NewDefaultScope(ctx, scopetest.FactoryInputFromConfig(cfg), logger).(*defaultScope)

	if runtimeScope.Context() != ctx {
		t.Fatal("expected context to be preserved")
	}
	if runtimeScope.Logger() != logger {
		t.Fatal("expected logger to be preserved")
	}
	dbOpts, ok := scope.DatabaseRuntimeOptionsFromScope(runtimeScope)
	if !ok {
		t.Fatal("expected db runtime options to be preserved")
	}
	if dbOpts.Dialect != cfg.Db.Dialect || dbOpts.DSN != cfg.Db.DSN {
		t.Fatalf("expected db runtime options to match input, got %#v", dbOpts)
	}
	if runtimeScope.Session() == nil {
		t.Fatal("expected session backed by db")
	}

	sqlDB, err := runtimeScope.db.DB()
	if err != nil {
		t.Fatalf("db.DB: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if sqlDB.Stats().MaxOpenConnections != 2 {
		t.Fatalf("sqlite MaxOpenConnections = %d, want 2", sqlDB.Stats().MaxOpenConnections)
	}

	childCtx := context.WithValue(context.Background(), "trace", "child")
	child := runtimeScope.WithContext(childCtx).(*defaultScope)
	if child.Context() != childCtx {
		t.Fatal("expected child context to be used")
	}
	if child.db != runtimeScope.db {
		t.Fatal("expected db handle to be shared")
	}
	if child.session != nil {
		t.Fatal("expected child scope to start without active transaction")
	}
}

func TestNewDefaultScopeCreatesMissingSQLiteParentDir(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
	}{
		{
			name: "plain path",
			dsn:  filepath.Join(t.TempDir(), "nested", "db", "choysum.db"),
		},
		{
			name: "file uri",
			dsn: func() string {
				path := filepath.Join(t.TempDir(), "uri", "db", "choysum.db")
				return "file:" + filepath.ToSlash(path) + "?mode=rwc&_fk=1"
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{Db: &config.DbConfig{Dialect: "sqlite", DSN: tt.dsn}}
			runtimeScope := NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(cfg), testLogger()).(*defaultScope)

			sqlDB, err := runtimeScope.db.DB()
			if err != nil {
				t.Fatalf("db.DB: %v", err)
			}
			t.Cleanup(func() { _ = sqlDB.Close() })

			path, ok := sqliteFilePathFromDSN(tt.dsn)
			if !ok {
				t.Fatalf("expected sqlite path from dsn %q", tt.dsn)
			}
			st, err := os.Stat(filepath.Dir(path))
			if err != nil {
				t.Fatalf("Stat(parent dir): %v", err)
			}
			if !st.IsDir() {
				t.Fatalf("expected parent dir for %q to exist", path)
			}
		})
	}
}

func TestNewDbPanicsOnInvalidDialect(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for invalid database dialect")
		}
	}()

	_ = newDb(context.Background(), scope.DatabaseRuntimeOptions{Dialect: "invalid", DSN: "ignored"}, testLogger())
}

func TestSQLTraceEnabled(t *testing.T) {
	var nilCtx context.Context

	if sqlTraceEnabled(context.Background(), nil) {
		t.Fatal("expected nil logger to disable sql trace")
	}
	if sqlTraceEnabled(context.Background(), testLogger()) {
		t.Fatal("expected info logger to disable sql trace")
	}
	if !sqlTraceEnabled(context.Background(), testLoggerWithLevel(slog.LevelDebug)) {
		t.Fatal("expected debug logger to enable sql trace")
	}
	if !sqlTraceEnabled(nilCtx, testLoggerWithLevel(slog.LevelDebug)) {
		t.Fatal("expected nil context to fall back to background")
	}
}

func TestSessionReturnsNonTransactionalDatabaseHandle(t *testing.T) {
	runtimeScope := NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(testConfig(t)), testLogger()).(*defaultScope)
	session := runtimeScope.Session()
	if session == nil {
		t.Fatal("expected non-nil session")
	}
	if errors.Is(session.Error, gorm.ErrInvalidTransaction) {
		t.Fatal("expected Session() outside Run to return a usable db handle")
	}
}
