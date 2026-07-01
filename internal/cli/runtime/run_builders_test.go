// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runtime

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func TestNewRunDBOptions(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		if got := NewRunDBOptions(nil); got != (RunDBOptions{}) {
			t.Fatalf("NewRunDBOptions(nil) = %#v, want zero value", got)
		}
	})

	t.Run("sqlite default path enables create", func(t *testing.T) {
		root := t.TempDir()
		cfg := &config.Config{
			DefaultChoysumPath: root,
			Db: &config.DbConfig{
				Dialect: "sqlite",
				DSN:     config.DefaultSQLiteDSN(root),
			},
		}
		got := NewRunDBOptions(cfg)
		if got.Dialect != "sqlite" {
			t.Fatalf("NewRunDBOptions().Dialect = %q, want sqlite", got.Dialect)
		}
		if !got.AllowCreate {
			t.Fatal("NewRunDBOptions().AllowCreate = false, want true for default sqlite path")
		}
	})

	t.Run("sqlite non-default path keeps create disabled", func(t *testing.T) {
		cfg := &config.Config{
			DefaultChoysumPath: t.TempDir(),
			Db: &config.DbConfig{
				Dialect: "sqlite",
				DSN:     "/tmp/custom.sqlite?mode=rwc",
			},
		}
		if got := NewRunDBOptions(cfg); got.AllowCreate {
			t.Fatal("NewRunDBOptions().AllowCreate = true, want false for non-default sqlite path")
		}
	})

	t.Run("non-sqlite never enables create", func(t *testing.T) {
		cfg := &config.Config{
			Db: &config.DbConfig{
				Dialect: "POSTGRES",
				DSN:     "postgres://127.0.0.1/choysum",
			},
		}
		got := NewRunDBOptions(cfg)
		if got.Dialect != "postgres" {
			t.Fatalf("NewRunDBOptions().Dialect = %q, want postgres", got.Dialect)
		}
		if got.AllowCreate {
			t.Fatal("NewRunDBOptions().AllowCreate = true, want false for non-sqlite")
		}
	})
}

func TestNewScopeForRun(t *testing.T) {
	newValidInput := func(environment string) RunScopeInput {
		cfgOptions := &ScopeInputConfigOptions{
			ModulesPath:           "/tmp/modules",
			TmpPath:               "/tmp/tmp",
			DefaultChoysumPath:    "/tmp/choysum",
			ModuleCatalogIndexURL: config.DefaultModuleCatalogIndexURL,
			Server:                &config.ServerConfig{Environment: environment},
		}
		cliOptions := Options{
			ModulesPath:           "/tmp/modules",
			TmpPath:               "/tmp/tmp",
			DefaultChoysumPath:    "/tmp/choysum",
			ModuleCatalogIndexURL: config.DefaultModuleCatalogIndexURL,
		}
		serverOptions := RunServerOptions{BindAddress: "127.0.0.1", Port: 8080}
		dbOptions := RunDBOptions{Dialect: "sqlite", DSN: "/tmp/choysum/choysum.sqlite"}
		return NewRunScopeInput(cfgOptions, cliOptions, serverOptions, dbOptions)
	}

	t.Run("invalid cli options", func(t *testing.T) {
		input := newValidInput("runtime-builders-invalid-cli")
		input = NewRunScopeInput(input.ConfigOptions(), Options{}, input.ServerOptions(), input.DBOptions())
		_, err := NewScopeForRun(input, nil)
		if err == nil || !strings.Contains(err.Error(), "invalid cli runtime options") {
			t.Fatalf("NewScopeForRun() error = %v, want invalid cli runtime options", err)
		}
	})

	t.Run("invalid server options", func(t *testing.T) {
		input := newValidInput("runtime-builders-invalid-server")
		input = NewRunScopeInput(input.ConfigOptions(), input.CLIOptions(), RunServerOptions{}, input.DBOptions())
		_, err := NewScopeForRun(input, nil)
		if err == nil || !strings.Contains(err.Error(), "invalid run server options") {
			t.Fatalf("NewScopeForRun() error = %v, want invalid run server options", err)
		}
	})

	t.Run("invalid db options", func(t *testing.T) {
		input := newValidInput("runtime-builders-invalid-db")
		input = NewRunScopeInput(input.ConfigOptions(), input.CLIOptions(), input.ServerOptions(), RunDBOptions{})
		_, err := NewScopeForRun(input, nil)
		if err == nil || !strings.Contains(err.Error(), "invalid run db options") {
			t.Fatalf("NewScopeForRun() error = %v, want invalid run db options", err)
		}
	})

	t.Run("missing config options", func(t *testing.T) {
		input := newValidInput("runtime-builders-missing-config")
		input = NewRunScopeInput(nil, input.CLIOptions(), input.ServerOptions(), input.DBOptions())
		_, err := NewScopeForRun(input, nil)
		if err == nil || !strings.Contains(err.Error(), "config is required") {
			t.Fatalf("NewScopeForRun() error = %v, want config required", err)
		}
	})

	t.Run("missing registered factory", func(t *testing.T) {
		input := newValidInput("runtime-builders-missing-factory")
		_, err := NewScopeForRun(input, nil)
		if err == nil || !strings.Contains(err.Error(), "failed to initialize scope") {
			t.Fatalf("NewScopeForRun() error = %v, want scope initialization failure", err)
		}
	})

	t.Run("factory panic is recovered", func(t *testing.T) {
		envName := "runtime-builders-panic"
		scope.Register(envName, func(context.Context, scope.FactoryInput, *slog.Logger) scope.Scope {
			panic("boom")
		})
		input := newValidInput(envName)
		_, err := NewScopeForRun(input, nil)
		if err == nil || !strings.Contains(err.Error(), "failed to initialize scope") {
			t.Fatalf("NewScopeForRun() error = %v, want recovered panic error", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		envName := "runtime-builders-success"
		scope.Register(envName, func(ctx context.Context, input scope.FactoryInput, logger *slog.Logger) scope.Scope {
			return NewScopeWithoutDB(ctx, input, logger)
		})
		input := newValidInput(envName)
		runtimeScope, err := NewScopeForRun(input, &config.LogConfig{Level: "info"})
		if err != nil {
			t.Fatalf("NewScopeForRun() error = %v", err)
		}
		if runtimeScope == nil {
			t.Fatal("NewScopeForRun() returned nil scope")
		}
		if runtimeScope.Context() == nil || runtimeScope.Logger() == nil {
			t.Fatal("NewScopeForRun() returned scope with nil context/logger")
		}
	})

	t.Run("success with nil logger config", func(t *testing.T) {
		envName := "runtime-builders-success-nil-log"
		scope.Register(envName, func(ctx context.Context, input scope.FactoryInput, logger *slog.Logger) scope.Scope {
			if logger == nil {
				logger = slog.New(slog.NewTextHandler(io.Discard, nil))
			}
			return NewScopeWithoutDB(ctx, input, logger)
		})
		input := newValidInput(envName)
		runtimeScope, err := NewScopeForRun(input, nil)
		if err != nil {
			t.Fatalf("NewScopeForRun(nil log config) error = %v", err)
		}
		if runtimeScope == nil {
			t.Fatal("NewScopeForRun(nil log config) returned nil scope")
		}
	})
}
