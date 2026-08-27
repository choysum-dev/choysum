// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	exportcli "github.com/choysum-dev/choysum/internal/cli/export"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/spf13/cobra"
)

func TestRunRecordExportCommandDefaultEmptyMode(t *testing.T) {
	runtimeScope := newExportCLITestScope(t)
	prev := runExportRecord
	runExportRecord = func(_ context.Context, _ scope.Scope, opts exportcli.RecordOptions) (importpkg.Report, []byte, error) {
		if opts.Mode != exportpkg.ModeData {
			t.Fatalf("mode = %q", opts.Mode)
		}
		return importpkg.Report{Stats: importpkg.Stats{Ok: 1, Total: 1}}, []byte("x"), nil
	}
	t.Cleanup(func() { runExportRecord = prev })

	outPath := filepath.Join(t.TempDir(), "base.Country.csv")
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	if err := runRecordExportCommand(cmd, func() scope.Scope { return runtimeScope }, []string{outPath}, "", ""); err != nil {
		t.Fatalf("runRecordExportCommand: %v", err)
	}
}

func TestRunRecordExportCommandNilContext(t *testing.T) {
	runtimeScope := newExportCLITestScope(t)
	prev := runExportRecord
	var gotCtx context.Context
	runExportRecord = func(ctx context.Context, _ scope.Scope, _ exportcli.RecordOptions) (importpkg.Report, []byte, error) {
		gotCtx = ctx
		return importpkg.Report{Stats: importpkg.Stats{Ok: 1, Total: 1}}, []byte("x"), nil
	}
	t.Cleanup(func() { runExportRecord = prev })

	outPath := filepath.Join(t.TempDir(), "base.Country.csv")
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	if err := runRecordExportCommand(cmd, func() scope.Scope { return runtimeScope }, []string{outPath}, "base.Country", "data"); err != nil {
		t.Fatalf("runRecordExportCommand: %v", err)
	}
	if gotCtx == nil {
		t.Fatal("expected background context")
	}
}

func TestCLI_ExportRecordModelOverride(t *testing.T) {
	runtimeScope := newExportCLITestScope(t)
	prev := runExportRecord
	runExportRecord = func(_ context.Context, _ scope.Scope, opts exportcli.RecordOptions) (importpkg.Report, []byte, error) {
		if opts.Model != "partner.Partner" {
			t.Fatalf("opts = %+v", opts)
		}
		return importpkg.Report{Profile: importpkg.ProfileRecord, Stats: importpkg.Stats{Ok: 1, Total: 1}}, []byte("x"), nil
	}
	t.Cleanup(func() { runExportRecord = prev })

	outPath := filepath.Join(t.TempDir(), "anything.csv")
	cmd := newExportCmd(func() scope.Scope { return runtimeScope })
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{"--model", "partner.Partner", outPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestCLI_ExportRecordInvalidOutputExtension(t *testing.T) {
	runtimeScope := newExportCLITestScope(t)
	cmd := newExportCmd(func() scope.Scope { return runtimeScope })
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{filepath.Join(t.TempDir(), "base.Country.po")})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected csv validation error")
	}
}

func TestCLI_ExportRecordUninferableModel(t *testing.T) {
	runtimeScope := newExportCLITestScope(t)
	cmd := newExportCmd(func() scope.Scope { return runtimeScope })
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{filepath.Join(t.TempDir(), "countries.csv")})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected model inference error")
	}
}

func TestCLI_ExportRecordUnsupportedMode(t *testing.T) {
	runtimeScope := newExportCLITestScope(t)
	cmd := newExportCmd(func() scope.Scope { return runtimeScope })
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{"--mode", "bogus", filepath.Join(t.TempDir(), "base.Country.csv")})

	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "unsupported mode") {
		t.Fatalf("Execute() err = %v", err)
	}
}

func TestCLI_ExportRecordRequiresScope(t *testing.T) {
	cmd := newExportCmd(func() scope.Scope { return nil })
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{filepath.Join(t.TempDir(), "base.Country.csv")})

	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "scope") {
		t.Fatalf("Execute() err = %v", err)
	}
}

func TestCLI_ExportRecordNilContext(t *testing.T) {
	runtimeScope := newExportCLITestScope(t)
	prev := runExportRecord
	runExportRecord = func(ctx context.Context, _ scope.Scope, _ exportcli.RecordOptions) (importpkg.Report, []byte, error) {
		if ctx == nil {
			t.Fatal("expected non-nil context")
		}
		return importpkg.Report{Stats: importpkg.Stats{Ok: 1, Total: 1}}, []byte("x"), nil
	}
	t.Cleanup(func() { runExportRecord = prev })

	cmd := newExportCmd(func() scope.Scope { return runtimeScope })
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(nil)
	cmd.SetArgs([]string{filepath.Join(t.TempDir(), "base.Country.csv")})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestCLI_ExportRecordRunError(t *testing.T) {
	runtimeScope := newExportCLITestScope(t)
	prev := runExportRecord
	runExportRecord = func(context.Context, scope.Scope, exportcli.RecordOptions) (importpkg.Report, []byte, error) {
		return importpkg.Report{}, nil, errors.New("record run boom")
	}
	t.Cleanup(func() { runExportRecord = prev })

	cmd := newExportCmd(func() scope.Scope { return runtimeScope })
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{filepath.Join(t.TempDir(), "base.Country.csv")})

	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "record run boom") {
		t.Fatalf("Execute() err = %v", err)
	}
	if !strings.Contains(stdout.String(), `"profile"`) {
		t.Fatalf("stdout = %q, want report printed before error", stdout.String())
	}
}

func TestCLI_ExportRecordStatsError(t *testing.T) {
	runtimeScope := newExportCLITestScope(t)
	prev := runExportRecord
	runExportRecord = func(context.Context, scope.Scope, exportcli.RecordOptions) (importpkg.Report, []byte, error) {
		return importpkg.Report{Stats: importpkg.Stats{Error: 2}}, []byte("x"), nil
	}
	t.Cleanup(func() { runExportRecord = prev })

	cmd := newExportCmd(func() scope.Scope { return runtimeScope })
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{filepath.Join(t.TempDir(), "base.Country.csv")})

	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "export finished with 2 error") {
		t.Fatalf("Execute() err = %v", err)
	}
}

func TestCLI_ExportRecordMarshalReportError(t *testing.T) {
	runtimeScope := newExportCLITestScope(t)
	prevRun := runExportRecord
	prevMarshal := marshalExportReport
	runExportRecord = func(context.Context, scope.Scope, exportcli.RecordOptions) (importpkg.Report, []byte, error) {
		return importpkg.Report{Stats: importpkg.Stats{Ok: 1, Total: 1}}, []byte("x"), nil
	}
	marshalExportReport = func(importpkg.Report) ([]byte, error) {
		return nil, errors.New("marshal failed")
	}
	t.Cleanup(func() {
		runExportRecord = prevRun
		marshalExportReport = prevMarshal
	})

	cmd := newExportCmd(func() scope.Scope { return runtimeScope })
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{filepath.Join(t.TempDir(), "base.Country.csv")})

	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "marshal export report") {
		t.Fatalf("Execute() err = %v", err)
	}
}

func TestCLI_ExportRecordStdoutWriteError(t *testing.T) {
	runtimeScope := newExportCLITestScope(t)
	prev := runExportRecord
	runExportRecord = func(context.Context, scope.Scope, exportcli.RecordOptions) (importpkg.Report, []byte, error) {
		return importpkg.Report{Stats: importpkg.Stats{Ok: 1, Total: 1}}, []byte("x"), nil
	}
	t.Cleanup(func() { runExportRecord = prev })

	cmd := newExportCmd(func() scope.Scope { return runtimeScope })
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(failWriter{})
	cmd.SetErr(io.Discard)
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{filepath.Join(t.TempDir(), "base.Country.csv")})

	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "stdout write failed") {
		t.Fatalf("Execute() err = %v", err)
	}
}

func TestCLI_ExportRecordWriteCSVError(t *testing.T) {
	runtimeScope := newExportCLITestScope(t)
	prev := runExportRecord
	runExportRecord = func(context.Context, scope.Scope, exportcli.RecordOptions) (importpkg.Report, []byte, error) {
		return importpkg.Report{Stats: importpkg.Stats{Ok: 1, Total: 1}}, []byte("x"), nil
	}
	t.Cleanup(func() { runExportRecord = prev })

	cmd := newExportCmd(func() scope.Scope { return runtimeScope })
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "base.Country.csv")
	if err := os.Mkdir(csvPath, 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	cmd.SetArgs([]string{csvPath})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected write CSV error")
	}
}

func TestCLI_ExportRecordExplicitProfile(t *testing.T) {
	runtimeScope := newExportCLITestScope(t)
	prev := runExportRecord
	runExportRecord = func(_ context.Context, _ scope.Scope, opts exportcli.RecordOptions) (importpkg.Report, []byte, error) {
		if opts.Mode != exportpkg.ModeData {
			t.Fatalf("mode = %q", opts.Mode)
		}
		return importpkg.Report{Stats: importpkg.Stats{Ok: 1, Total: 1}}, []byte("csv"), nil
	}
	t.Cleanup(func() { runExportRecord = prev })

	outPath := filepath.Join(t.TempDir(), "base.Country.csv")
	cmd := newExportCmd(func() scope.Scope { return runtimeScope })
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{"--profile", "record", outPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestWriteExportCSVValidation(t *testing.T) {
	if err := writeExportCSV(" ", []byte("x")); err == nil || !strings.Contains(err.Error(), "output path is required") {
		t.Fatalf("writeExportCSV() err = %v", err)
	}
}

func TestWriteExportCSVCreatesNestedDir(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "nested", "base.Country.csv")
	if err := writeExportCSV(outPath, []byte("Name\nA\n")); err != nil {
		t.Fatalf("writeExportCSV: %v", err)
	}
	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "Name\nA\n" {
		t.Fatalf("file = %q", got)
	}
}

func TestWriteExportCSVMkdirError(t *testing.T) {
	base := t.TempDir()
	blocker := filepath.Join(base, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := writeExportCSV(filepath.Join(blocker, "nested", "out.csv"), []byte("x")); err == nil {
		t.Fatal("expected mkdir error")
	}
}

func TestWriteExportCSVCreateTempError(t *testing.T) {
	prev := exportCSVCreateTemp
	exportCSVCreateTemp = func(string, string) (*os.File, error) {
		return nil, errors.New("create temp failed")
	}
	t.Cleanup(func() { exportCSVCreateTemp = prev })

	err := writeExportCSV(filepath.Join(t.TempDir(), "out.csv"), []byte("x"))
	if err == nil || !strings.Contains(err.Error(), "write CSV file") {
		t.Fatalf("writeExportCSV() err = %v", err)
	}
}

func TestWriteExportCSVTempWriteError(t *testing.T) {
	prev := exportCSVCreateTemp
	exportCSVCreateTemp = func(dir, pattern string) (*os.File, error) {
		f, err := os.CreateTemp(dir, pattern)
		if err != nil {
			return nil, err
		}
		name := f.Name()
		if err := f.Close(); err != nil {
			return nil, err
		}
		return os.OpenFile(name, os.O_RDONLY, 0)
	}
	t.Cleanup(func() { exportCSVCreateTemp = prev })

	err := writeExportCSV(filepath.Join(t.TempDir(), "out.csv"), []byte("x"))
	if err == nil || !strings.Contains(err.Error(), "write CSV file") {
		t.Fatalf("writeExportCSV() err = %v", err)
	}
}

func TestWriteExportCSVCloseError(t *testing.T) {
	prev := exportCSVCloseTemp
	exportCSVCloseTemp = func(*os.File) error { return errors.New("close failed") }
	t.Cleanup(func() { exportCSVCloseTemp = prev })

	err := writeExportCSV(filepath.Join(t.TempDir(), "out.csv"), []byte("x"))
	if err == nil || !strings.Contains(err.Error(), "write CSV file") {
		t.Fatalf("writeExportCSV() err = %v", err)
	}
}

func TestWriteExportCSVWriteError(t *testing.T) {
	dir := t.TempDir()
	if err := writeExportCSV(dir, []byte("x")); err == nil {
		t.Fatal("expected write error when output path is a directory")
	}
}
