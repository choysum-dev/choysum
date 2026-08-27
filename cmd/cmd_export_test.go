// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	exportcli "github.com/choysum-dev/choysum/internal/cli/export"
	"github.com/choysum-dev/choysum/internal/defaultscope"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func TestCLI_ExportTerminologyWritesPOFile(t *testing.T) {
	runtimeScope := newExportCLITestScope(t)
	prev := runExportTerminology
	runExportTerminology = func(_ context.Context, _ scope.Scope, opts exportcli.TerminologyOptions) (importpkg.Report, []byte, error) {
		if opts.Application != "auth" || opts.Module != "base" || opts.Lang != "zh_CN" {
			t.Fatalf("opts = %+v", opts)
		}
		return importpkg.Report{
			Profile: importpkg.ProfileTerminology,
			Stats:   importpkg.Stats{Ok: 1, Total: 1},
		}, []byte("msgid \"Hello\"\n"), nil
	}
	t.Cleanup(func() { runExportTerminology = prev })

	outPath := filepath.Join(t.TempDir(), "base-zh_CN.po")
	cmd := newExportCmd(func() scope.Scope { return runtimeScope })
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{
		"--profile", "terminology",
		"--application", "auth",
		"--module", "base",
		"--lang", "zh_CN",
		outPath,
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "msgid \"Hello\"\n" {
		t.Fatalf("file = %q", got)
	}

	var report importpkg.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal report: %v\nstdout=%q", err, stdout.String())
	}
	if report.Stats.Ok != 1 {
		t.Fatalf("report stats = %+v", report.Stats)
	}
}

func TestCLI_ExportRequiresProfile(t *testing.T) {
	runtimeScope := newExportCLITestScope(t)
	cmd := newExportCmd(func() scope.Scope { return runtimeScope })
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{"out.po"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected profile error")
	}
}

func newExportCLITestScope(t *testing.T) scope.Scope {
	t.Helper()
	cfg := &config.Config{
		Db: &config.DbConfig{
			Dialect: "sqlite",
			DSN:     filepath.Join(t.TempDir(), "export-cli.db"),
		},
	}
	return defaultscope.NewDefaultScope(
		context.Background(),
		scopetest.FactoryInputFromConfig(cfg),
		nil,
	)
}
