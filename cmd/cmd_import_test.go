// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/defaultscope"
	_ "github.com/choysum-dev/choysum/internal/import/adapter/stub"
	"github.com/choysum-dev/choysum/internal/import/registry"
	_ "github.com/choysum-dev/choysum/internal/import/runner"
	stubwriter "github.com/choysum-dev/choysum/internal/import/writer/stub"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func init() {
	registry.RegisterWriter(importpkg.ProfileRecord, stubwriter.Writer{})
}

func TestCLI_BestEffort_PartialOk(t *testing.T) {
	runtimeScope := newImportCLITestScope(t)
	envGetter := func() scope.Scope { return runtimeScope }

	sourcePath := filepath.Join(t.TempDir(), "base_Country.csv")
	if err := os.WriteFile(sourcePath, []byte("unused\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cmd := newImportCmd(envGetter)
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{
		sourcePath,
		"--format", "stub",
		"--policy", "best_effort",
		"--stub-unit-count", "3",
		"--stub-fail-unit-index", "2",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var report importpkg.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal report: %v\nstdout=%q", err, stdout.String())
	}
	if report.Stats.Ok != 2 || report.Stats.Error != 1 {
		t.Fatalf("unexpected stats: %+v", report.Stats)
	}
	if report.Policy != importpkg.PolicyBestEffort {
		t.Fatalf("policy = %q, want best_effort", report.Policy)
	}

	var count int64
	if err := runtimeScope.Session().Model(&stubwriter.Row{}).Count(&count).Error; err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 2 {
		t.Fatalf("row count = %d, want 2", count)
	}
}

func TestCLI_ImportRejectsUninferableModelFilename(t *testing.T) {
	runtimeScope := newImportCLITestScope(t)
	sourcePath := filepath.Join(t.TempDir(), "country_import_ok.csv")
	if err := os.WriteFile(sourcePath, []byte("Name\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cmd := newImportCmd(func() scope.Scope { return runtimeScope })
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{sourcePath})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected model inference error")
	}
}

func TestParseImportPolicy_AcceptsKnownPolicies(t *testing.T) {
	for _, policy := range []string{"atomic", "stop_keep", "best_effort"} {
		if _, err := parseImportPolicy(policy); err != nil {
			t.Fatalf("parseImportPolicy(%q): %v", policy, err)
		}
	}
}

func TestParseImportPolicy_RejectsUnknown(t *testing.T) {
	if _, err := parseImportPolicy("bogus"); err == nil {
		t.Fatal("expected error")
	}
}

func newImportCLITestScope(t *testing.T) scope.Scope {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "import-cli.db")
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
	if err := runtimeScope.Session().AutoMigrate(&stubwriter.Row{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return runtimeScope
}
