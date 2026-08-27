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

	exportcli "github.com/choysum-dev/choysum/internal/cli/export"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func TestCLI_ExportUnsupportedProfile(t *testing.T) {
	runtimeScope := newExportCLITestScope(t)
	cmd := newExportCmd(func() scope.Scope { return runtimeScope })
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{"--profile", "record", "out.po"})

	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "unsupported profile") {
		t.Fatalf("Execute() err = %v", err)
	}
}

func TestCLI_ExportReportOnly(t *testing.T) {
	runtimeScope := newExportCLITestScope(t)
	prev := runExportTerminology
	runExportTerminology = func(_ context.Context, _ scope.Scope, _ exportcli.TerminologyOptions) (importpkg.Report, []byte, error) {
		return importpkg.Report{Profile: importpkg.ProfileTerminology, Stats: importpkg.Stats{Ok: 1, Total: 1}}, []byte("po"), nil
	}
	t.Cleanup(func() { runExportTerminology = prev })

	cmd := newExportCmd(func() scope.Scope { return runtimeScope })
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{
		"--profile", "terminology",
		"--application", "auth",
		"--module", "base",
		"--lang", "zh_CN",
		"--report-only",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var report importpkg.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
}

func TestCLI_ExportStdoutMode(t *testing.T) {
	runtimeScope := newExportCLITestScope(t)
	prev := runExportTerminology
	runExportTerminology = func(_ context.Context, _ scope.Scope, _ exportcli.TerminologyOptions) (importpkg.Report, []byte, error) {
		return importpkg.Report{Profile: importpkg.ProfileTerminology, Stats: importpkg.Stats{Ok: 1, Total: 1}}, []byte("msgid \"x\"\n"), nil
	}
	t.Cleanup(func() { runExportTerminology = prev })

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
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if stdout.String() != "msgid \"x\"\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), `"ok"`) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestCLI_ExportPreservesExistingFileOnFailure(t *testing.T) {
	runtimeScope := newExportCLITestScope(t)
	prev := runExportTerminology
	runExportTerminology = func(_ context.Context, _ scope.Scope, _ exportcli.TerminologyOptions) (importpkg.Report, []byte, error) {
		return importpkg.Report{}, nil, errors.New("export failed")
	}
	t.Cleanup(func() { runExportTerminology = prev })

	outPath := filepath.Join(t.TempDir(), "base-zh_CN.po")
	original := "msgid \"keep me\"\n"
	if err := os.WriteFile(outPath, []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cmd := newExportCmd(func() scope.Scope { return runtimeScope })
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{
		"--profile", "terminology",
		"--application", "auth",
		"--module", "base",
		"--lang", "zh_CN",
		outPath,
	})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected export failure")
	}
	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != original {
		t.Fatalf("file = %q, want preserved %q", got, original)
	}
}

func TestCLI_ExportRunErrorBeforeWrite(t *testing.T) {
	runtimeScope := newExportCLITestScope(t)
	prev := runExportTerminology
	runExportTerminology = func(_ context.Context, _ scope.Scope, _ exportcli.TerminologyOptions) (importpkg.Report, []byte, error) {
		return importpkg.Report{Stats: importpkg.Stats{Error: 1}}, nil, nil
	}
	t.Cleanup(func() { runExportTerminology = prev })

	cmd := newExportCmd(func() scope.Scope { return runtimeScope })
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{
		"--profile", "terminology",
		"--application", "auth",
		"--module", "base",
		"--lang", "zh_CN",
		filepath.Join(t.TempDir(), "out.po"),
	})

	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "export finished with 1 error") {
		t.Fatalf("Execute() err = %v", err)
	}
}

func TestCLI_ExportMarshalReportError(t *testing.T) {
	runtimeScope := newExportCLITestScope(t)
	prevRun := runExportTerminology
	prevMarshal := marshalExportReport
	runExportTerminology = func(_ context.Context, _ scope.Scope, _ exportcli.TerminologyOptions) (importpkg.Report, []byte, error) {
		return importpkg.Report{Stats: importpkg.Stats{Ok: 1, Total: 1}}, []byte("po"), nil
	}
	marshalExportReport = func(importpkg.Report) ([]byte, error) {
		return nil, errors.New("marshal failed")
	}
	t.Cleanup(func() {
		runExportTerminology = prevRun
		marshalExportReport = prevMarshal
	})

	cmd := newExportCmd(func() scope.Scope { return runtimeScope })
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{
		"--profile", "terminology",
		"--application", "auth",
		"--module", "base",
		"--lang", "zh_CN",
		filepath.Join(t.TempDir(), "out.po"),
	})

	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "marshal export report") {
		t.Fatalf("Execute() err = %v", err)
	}
}

func TestCLI_ExportStdoutWriteError(t *testing.T) {
	runtimeScope := newExportCLITestScope(t)
	prev := runExportTerminology
	runExportTerminology = func(_ context.Context, _ scope.Scope, _ exportcli.TerminologyOptions) (importpkg.Report, []byte, error) {
		return importpkg.Report{Stats: importpkg.Stats{Ok: 1, Total: 1}}, []byte("po"), nil
	}
	t.Cleanup(func() { runExportTerminology = prev })

	cmd := newExportCmd(func() scope.Scope { return runtimeScope })
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(failWriter{})
	cmd.SetErr(io.Discard)
	cmd.SetContext(nil)
	cmd.SetArgs([]string{
		"--profile", "terminology",
		"--application", "auth",
		"--module", "base",
		"--lang", "zh_CN",
	})

	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "stdout write failed") {
		t.Fatalf("Execute() err = %v", err)
	}
}

func TestCLI_ExportStderrWriteError(t *testing.T) {
	runtimeScope := newExportCLITestScope(t)
	prev := runExportTerminology
	runExportTerminology = func(_ context.Context, _ scope.Scope, _ exportcli.TerminologyOptions) (importpkg.Report, []byte, error) {
		return importpkg.Report{Stats: importpkg.Stats{Ok: 1, Total: 1}}, []byte("po"), nil
	}
	t.Cleanup(func() { runExportTerminology = prev })

	cmd := newExportCmd(func() scope.Scope { return runtimeScope })
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(failWriter{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{
		"--profile", "terminology",
		"--application", "auth",
		"--module", "base",
		"--lang", "zh_CN",
	})

	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "stdout write failed") {
		t.Fatalf("Execute() err = %v", err)
	}
}

func TestCLI_ExportRequiresScope(t *testing.T) {
	cmd := newExportCmd(func() scope.Scope { return nil })
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{
		"--profile", "terminology",
		"--application", "auth",
		"--module", "base",
		"--lang", "zh_CN",
		"out.po",
	})

	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "scope") {
		t.Fatalf("Execute() err = %v", err)
	}
}

func TestWriteExportPOValidation(t *testing.T) {
	if err := writeExportPO(" ", []byte("x")); err == nil || !strings.Contains(err.Error(), "output path is required") {
		t.Fatalf("writeExportPO() err = %v", err)
	}
}

func TestWriteExportPOCreatesNestedDir(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "nested", "out.po")
	if err := writeExportPO(outPath, []byte("msgid \"x\"\n")); err != nil {
		t.Fatalf("writeExportPO: %v", err)
	}
	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "msgid \"x\"\n" {
		t.Fatalf("file = %q", got)
	}
}

func TestCLI_ExportReportOnlyStdoutWriteError(t *testing.T) {
	runtimeScope := newExportCLITestScope(t)
	prev := runExportTerminology
	runExportTerminology = func(_ context.Context, _ scope.Scope, _ exportcli.TerminologyOptions) (importpkg.Report, []byte, error) {
		return importpkg.Report{Stats: importpkg.Stats{Ok: 1, Total: 1}}, nil, nil
	}
	t.Cleanup(func() { runExportTerminology = prev })

	cmd := newExportCmd(func() scope.Scope { return runtimeScope })
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(failWriter{})
	cmd.SetErr(io.Discard)
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{
		"--profile", "terminology",
		"--application", "auth",
		"--module", "base",
		"--lang", "zh_CN",
		"--report-only",
	})

	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "stdout write failed") {
		t.Fatalf("Execute() err = %v", err)
	}
}

func TestCLI_ExportFileStdoutWriteError(t *testing.T) {
	runtimeScope := newExportCLITestScope(t)
	prev := runExportTerminology
	runExportTerminology = func(_ context.Context, _ scope.Scope, _ exportcli.TerminologyOptions) (importpkg.Report, []byte, error) {
		return importpkg.Report{Stats: importpkg.Stats{Ok: 1, Total: 1}}, []byte("po"), nil
	}
	t.Cleanup(func() { runExportTerminology = prev })

	cmd := newExportCmd(func() scope.Scope { return runtimeScope })
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(failWriter{})
	cmd.SetErr(io.Discard)
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{
		"--profile", "terminology",
		"--application", "auth",
		"--module", "base",
		"--lang", "zh_CN",
		filepath.Join(t.TempDir(), "out.po"),
	})

	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "stdout write failed") {
		t.Fatalf("Execute() err = %v", err)
	}
}

func TestCLI_ExportStdoutModeEmptyPO(t *testing.T) {
	runtimeScope := newExportCLITestScope(t)
	prev := runExportTerminology
	runExportTerminology = func(_ context.Context, _ scope.Scope, _ exportcli.TerminologyOptions) (importpkg.Report, []byte, error) {
		return importpkg.Report{Stats: importpkg.Stats{Ok: 1, Total: 1}}, nil, nil
	}
	t.Cleanup(func() { runExportTerminology = prev })

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
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}

func TestWriteExportPOMkdirError(t *testing.T) {
	base := t.TempDir()
	blocker := filepath.Join(base, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := writeExportPO(filepath.Join(blocker, "nested", "out.po"), []byte("x")); err == nil {
		t.Fatal("expected mkdir error")
	}
}

func TestWriteExportPOCreateTempError(t *testing.T) {
	prev := exportPOCreateTemp
	exportPOCreateTemp = func(string, string) (*os.File, error) {
		return nil, errors.New("create temp failed")
	}
	t.Cleanup(func() { exportPOCreateTemp = prev })

	err := writeExportPO(filepath.Join(t.TempDir(), "out.po"), []byte("x"))
	if err == nil || !strings.Contains(err.Error(), "write PO file") {
		t.Fatalf("writeExportPO() err = %v", err)
	}
}

func TestWriteExportPOTempWriteError(t *testing.T) {
	prev := exportPOCreateTemp
	exportPOCreateTemp = func(dir, pattern string) (*os.File, error) {
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
	t.Cleanup(func() { exportPOCreateTemp = prev })

	err := writeExportPO(filepath.Join(t.TempDir(), "out.po"), []byte("x"))
	if err == nil || !strings.Contains(err.Error(), "write PO file") {
		t.Fatalf("writeExportPO() err = %v", err)
	}
}

func TestWriteExportPOTempCloseError(t *testing.T) {
	prev := exportPOCloseTemp
	exportPOCloseTemp = func(*os.File) error { return errors.New("close failed") }
	t.Cleanup(func() { exportPOCloseTemp = prev })

	err := writeExportPO(filepath.Join(t.TempDir(), "out.po"), []byte("x"))
	if err == nil || !strings.Contains(err.Error(), "write PO file") {
		t.Fatalf("writeExportPO() err = %v", err)
	}
}

func TestWriteExportPOWriteError(t *testing.T) {
	dir := t.TempDir()
	if err := writeExportPO(dir, []byte("x")); err == nil {
		t.Fatal("expected write error when output path is a directory")
	}
}

func TestCLI_ExportWritePOErrorOnFilePath(t *testing.T) {
	runtimeScope := newExportCLITestScope(t)
	prev := runExportTerminology
	runExportTerminology = func(_ context.Context, _ scope.Scope, _ exportcli.TerminologyOptions) (importpkg.Report, []byte, error) {
		return importpkg.Report{Stats: importpkg.Stats{Ok: 1, Total: 1}}, []byte("po"), nil
	}
	t.Cleanup(func() { runExportTerminology = prev })

	cmd := newExportCmd(func() scope.Scope { return runtimeScope })
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{
		"--profile", "terminology",
		"--application", "auth",
		"--module", "base",
		"--lang", "zh_CN",
		t.TempDir(),
	})

	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "write PO file") {
		t.Fatalf("Execute() err = %v", err)
	}
}

func TestCLI_ExportWritesPOInCurrentDirectory(t *testing.T) {
	runtimeScope := newExportCLITestScope(t)
	prev := runExportTerminology
	runExportTerminology = func(_ context.Context, _ scope.Scope, _ exportcli.TerminologyOptions) (importpkg.Report, []byte, error) {
		return importpkg.Report{Stats: importpkg.Stats{Ok: 1, Total: 1}}, []byte("po"), nil
	}
	t.Cleanup(func() { runExportTerminology = prev })

	workDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	cmd := newExportCmd(func() scope.Scope { return runtimeScope })
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{
		"--profile", "terminology",
		"--application", "auth",
		"--module", "base",
		"--lang", "zh_CN",
		"local.po",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(workDir, "local.po"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "po" {
		t.Fatalf("file = %q", got)
	}
}

func TestCLI_ExportRunTerminologyError(t *testing.T) {
	runtimeScope := newExportCLITestScope(t)
	prev := runExportTerminology
	runExportTerminology = func(_ context.Context, _ scope.Scope, _ exportcli.TerminologyOptions) (importpkg.Report, []byte, error) {
		return importpkg.Report{}, nil, errors.New("boom")
	}
	t.Cleanup(func() { runExportTerminology = prev })

	cmd := newExportCmd(func() scope.Scope { return runtimeScope })
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{
		"--profile", "terminology",
		"--application", "auth",
		"--module", "base",
		"--lang", "zh_CN",
	})

	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("Execute() err = %v", err)
	}
}
