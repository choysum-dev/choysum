// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	importcli "github.com/choysum-dev/choysum/internal/cli/import"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) {
	return 0, errors.New("stdout write failed")
}

func TestCLI_ImportStopKeep_EmitsReportBeforeError(t *testing.T) {
	runtimeScope := newImportCLITestScope(t)
	sourcePath := filepath.Join(t.TempDir(), "base_Country.csv")
	if err := os.WriteFile(sourcePath, []byte("unused\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cmd := newImportCmd(func() scope.Scope { return runtimeScope })
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{
		sourcePath,
		"--format", "stub",
		"--policy", "stop_keep",
		"--stub-unit-count", "3",
		"--stub-fail-unit-index", "2",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected stop_keep error")
	}
	if stdout.Len() == 0 {
		t.Fatal("expected JSON report on stdout before error")
	}
	var report importpkg.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal report: %v\nstdout=%q", err, stdout.String())
	}
	if report.Stats.Ok != 1 {
		t.Fatalf("stats ok = %d, want 1", report.Stats.Ok)
	}
}

func TestCLI_ImportRejectsDryRunWithNonAtomicPolicy(t *testing.T) {
	runtimeScope := newImportCLITestScope(t)
	sourcePath := filepath.Join(t.TempDir(), "base_Country.csv")
	if err := os.WriteFile(sourcePath, []byte("unused\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cmd := newImportCmd(func() scope.Scope { return runtimeScope })
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{
		sourcePath,
		"--format", "stub",
		"--policy", "best_effort",
		"--dry-run",
	})

	err := cmd.Execute()
	if !errors.Is(err, importpkg.ErrDryRunRequiresAtomic) {
		t.Fatalf("Execute() error = %v, want ErrDryRunRequiresAtomic", err)
	}
}

func TestCLI_ImportRejectsNonCSVWithModelOverride(t *testing.T) {
	runtimeScope := newImportCLITestScope(t)
	sourcePath := filepath.Join(t.TempDir(), "input.txt")
	if err := os.WriteFile(sourcePath, []byte("Name\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cmd := newImportCmd(func() scope.Scope { return runtimeScope })
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{sourcePath, "--model", "base.Country"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), ".csv") {
		t.Fatalf("Execute() error = %v, want csv validation error", err)
	}
}

func TestCLI_ImportDryRunAtomic(t *testing.T) {
	runtimeScope := newImportCLITestScope(t)
	sourcePath := filepath.Join(t.TempDir(), "base_Country.csv")
	if err := os.WriteFile(sourcePath, []byte("unused\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cmd := newImportCmd(func() scope.Scope { return runtimeScope })
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{
		sourcePath,
		"--format", "stub",
		"--dry-run",
		"--stub-unit-count", "2",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var report importpkg.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal report: %v", err)
	}
	if report.Stats.Ok != 2 {
		t.Fatalf("stats ok = %d, want 2", report.Stats.Ok)
	}
}

func TestCLI_ImportModelOverride(t *testing.T) {
	runtimeScope := newImportCLITestScope(t)
	sourcePath := filepath.Join(t.TempDir(), "country_import_ok.csv")
	if err := os.WriteFile(sourcePath, []byte("unused\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cmd := newImportCmd(func() scope.Scope { return runtimeScope })
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{
		sourcePath,
		"--model", "base.Country",
		"--format", "stub",
		"--stub-unit-count", "1",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestCLI_ImportAtomicErrorsExitNonZero(t *testing.T) {
	runtimeScope := newImportCLITestScope(t)
	sourcePath := filepath.Join(t.TempDir(), "base_Country.csv")
	if err := os.WriteFile(sourcePath, []byte("unused\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cmd := newImportCmd(func() scope.Scope { return runtimeScope })
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{
		sourcePath,
		"--format", "stub",
		"--policy", "atomic",
		"--stub-unit-count", "2",
		"--stub-fail-unit-index", "1",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected atomic import error")
	}
	if stdout.Len() == 0 {
		t.Fatal("expected report on stdout")
	}
}

func TestCLI_ImportRequiresScope(t *testing.T) {
	sourcePath := filepath.Join(t.TempDir(), "base_Country.csv")
	if err := os.WriteFile(sourcePath, []byte("unused\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cmd := newImportCmd(func() scope.Scope { return nil })
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{sourcePath})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "scope") {
		t.Fatalf("Execute() error = %v, want scope error", err)
	}
}

func TestCLI_ImportStdoutWriteError(t *testing.T) {
	runtimeScope := newImportCLITestScope(t)
	sourcePath := filepath.Join(t.TempDir(), "base_Country.csv")
	if err := os.WriteFile(sourcePath, []byte("unused\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cmd := newImportCmd(func() scope.Scope { return runtimeScope })
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(failWriter{})
	cmd.SetErr(io.Discard)
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{
		sourcePath,
		"--format", "stub",
		"--stub-unit-count", "1",
	})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "stdout write failed") {
		t.Fatalf("Execute() error = %v, want stdout write error", err)
	}
}

func TestCLI_ImportStatsErrorExit(t *testing.T) {
	runtimeScope := newImportCLITestScope(t)
	sourcePath := filepath.Join(t.TempDir(), "base_Country.csv")
	if err := os.WriteFile(sourcePath, []byte("unused\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	orig := runImportRecord
	t.Cleanup(func() { runImportRecord = orig })
	runImportRecord = func(context.Context, scope.Scope, importcli.RecordOptions) (importpkg.Report, error) {
		return importpkg.Report{
			Policy: importpkg.PolicyAtomic,
			Stats:  importpkg.Stats{Total: 2, Error: 1},
		}, nil
	}

	cmd := newImportCmd(func() scope.Scope { return runtimeScope })
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{sourcePath, "--format", "stub"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "import finished with 1 error(s)") {
		t.Fatalf("Execute() error = %v, want stats error exit", err)
	}
}

func TestCLI_ImportMarshalReportError(t *testing.T) {
	runtimeScope := newImportCLITestScope(t)
	sourcePath := filepath.Join(t.TempDir(), "base_Country.csv")
	if err := os.WriteFile(sourcePath, []byte("unused\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	orig := marshalImportReport
	t.Cleanup(func() { marshalImportReport = orig })
	marshalImportReport = func(importpkg.Report) ([]byte, error) {
		return nil, errors.New("marshal failed")
	}

	cmd := newImportCmd(func() scope.Scope { return runtimeScope })
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{sourcePath, "--format", "stub", "--stub-unit-count", "1"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "marshal import report") {
		t.Fatalf("Execute() error = %v, want marshal error", err)
	}
}

func TestCLI_ImportRejectsInvalidPolicy(t *testing.T) {
	runtimeScope := newImportCLITestScope(t)
	sourcePath := filepath.Join(t.TempDir(), "base_Country.csv")
	if err := os.WriteFile(sourcePath, []byte("unused\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cmd := newImportCmd(func() scope.Scope { return runtimeScope })
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{sourcePath, "--policy", "bogus"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "unsupported import policy") {
		t.Fatalf("Execute() error = %v, want invalid policy error", err)
	}
}

func TestCLI_ImportRunE_NilContext(t *testing.T) {
	runtimeScope := newImportCLITestScope(t)
	sourcePath := filepath.Join(t.TempDir(), "base_Country.csv")
	if err := os.WriteFile(sourcePath, []byte("unused\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cmd := newImportCmd(func() scope.Scope { return runtimeScope })
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		sourcePath,
		"--format", "stub",
		"--stub-unit-count", "1",
	})

	if err := cmd.RunE(cmd, []string{
		sourcePath,
		"--format", "stub",
		"--stub-unit-count", "1",
	}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
}
