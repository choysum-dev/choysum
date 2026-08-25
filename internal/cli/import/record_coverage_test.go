// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package importcli

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/defaultscope"
	_ "github.com/choysum-dev/choysum/internal/import/adapter/stub"
	"github.com/choysum-dev/choysum/internal/import/registry"
	"github.com/choysum-dev/choysum/internal/import/writer/stub"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func init() {
	registry.RegisterWriter(importpkg.ProfileRecord, stub.Writer{})
}

func TestRecordSpecFromOptions(t *testing.T) {
	t.Run("requires model", func(t *testing.T) {
		_, err := recordSpecFromOptions(RecordOptions{SourcePath: "x.csv"})
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("requires source path", func(t *testing.T) {
		_, err := recordSpecFromOptions(RecordOptions{Model: "base.Country"})
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("defaults format and policy", func(t *testing.T) {
		spec, err := recordSpecFromOptions(RecordOptions{
			Model:      "base.Country",
			SourcePath: "base_Country.csv",
		})
		if err != nil {
			t.Fatalf("recordSpecFromOptions: %v", err)
		}
		if spec.Source.Format != "csv" {
			t.Fatalf("format = %q, want csv", spec.Source.Format)
		}
		if spec.Policy != importpkg.PolicyAtomic {
			t.Fatalf("policy = %q, want atomic", spec.Policy)
		}
		if spec.Options.ColumnMapping == nil {
			t.Fatal("expected non-nil column mapping map")
		}
	})
	t.Run("preserves explicit options", func(t *testing.T) {
		spec, err := recordSpecFromOptions(RecordOptions{
			Model:             "partner.Partner",
			SourcePath:        "partner-Partner.csv",
			Format:            "stub",
			Policy:            importpkg.PolicyBestEffort,
			DryRun:            true,
			ColumnMapping:     map[string]string{"a": "b"},
			StubUnitCount:     3,
			StubFailUnitIndex: 2,
		})
		if err != nil {
			t.Fatalf("recordSpecFromOptions: %v", err)
		}
		if spec.Profile != importpkg.ProfileRecord || spec.Caller != importpkg.CallerCLI {
			t.Fatalf("unexpected spec identity: %+v", spec)
		}
		if spec.Policy != importpkg.PolicyBestEffort || !spec.DryRun {
			t.Fatalf("unexpected policy/dry-run: %+v", spec)
		}
		if spec.Options.StubUnitCount != 3 || spec.Options.StubFailUnitIndex != 2 {
			t.Fatalf("unexpected stub options: %+v", spec.Options)
		}
		if spec.Options.ColumnMapping["a"] != "b" {
			t.Fatalf("column mapping = %#v", spec.Options.ColumnMapping)
		}
	})
}

func TestRunRecord_StubImport(t *testing.T) {
	runtimeScope := newRecordTestScope(t)
	report, err := RunRecord(context.Background(), runtimeScope, RecordOptions{
		Model:         "base.Country",
		SourcePath:    filepath.Join(t.TempDir(), "base_Country.csv"),
		Format:        "stub",
		StubUnitCount: 1,
	})
	if err != nil {
		t.Fatalf("RunRecord: %v", err)
	}
	if report.Stats.Ok != 1 {
		t.Fatalf("stats ok = %d, want 1", report.Stats.Ok)
	}
}

func newRecordTestScope(t *testing.T) scope.Scope {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "record-cli.db")
	cfg := &config.Config{
		Db: &config.DbConfig{
			Dialect: "sqlite",
			DSN:     dbPath,
		},
	}
	runtimeScope := defaultscope.NewDefaultScope(
		context.Background(),
		scopetest.FactoryInputFromConfig(cfg),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if err := runtimeScope.Session().AutoMigrate(&stub.Row{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return runtimeScope
}
