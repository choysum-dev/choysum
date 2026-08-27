// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package exportcli

import (
	"context"
	"errors"
	"testing"

	"github.com/choysum-dev/choysum/internal/export/registry"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/scope"
)

var realCLIExportExecutor = newCLIExportExecutor

func TestNewCLIExportExecutorDefault(t *testing.T) {
	runtimeScope := newRecordExportTestScope(t)
	if _, err := realCLIExportExecutor(runtimeScope); err == nil {
		t.Fatal("expected error without registered executor factory")
	}
}

func TestRunRecord_RecordSpecError(t *testing.T) {
	_, _, err := RunRecord(context.Background(), nil, RecordOptions{
		OutputPath: "base.Country.csv",
	})
	if err == nil {
		t.Fatal("expected record spec error")
	}
}

func TestCliExportEngineAdapter(t *testing.T) {
	stub := stubCLIExportExecutor{}
	adapter := cliExportEngineAdapter{inner: stub}
	if err := adapter.Load(nil); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, err := adapter.Execute(context.Background(), &jsengine.JsRequest{Service: "base.Country.Search"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if err := adapter.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestRunRecord_ValidateSpecError(t *testing.T) {
	prev := validateExportSpec
	validateExportSpec = func(exportpkg.Spec) error { return errors.New("validate boom") }
	t.Cleanup(func() { validateExportSpec = prev })

	_, _, err := RunRecord(context.Background(), nil, RecordOptions{
		Model:      "base.Country",
		OutputPath: "base.Country.csv",
	})
	if err == nil || err.Error() != "validate boom" {
		t.Fatalf("RunRecord() err = %v", err)
	}
}

func TestRunRecord_ExecutorCreateError(t *testing.T) {
	prev := newCLIExportExecutor
	newCLIExportExecutor = func(scope.Scope) (jsexecutor.JsExecutor, error) {
		return nil, errors.New("executor create boom")
	}
	t.Cleanup(func() { newCLIExportExecutor = prev })

	_, _, err := RunRecord(context.Background(), nil, RecordOptions{
		Model:      "base.Country",
		OutputPath: "base.Country.csv",
	})
	if err == nil || err.Error() != "executor create boom" {
		t.Fatalf("RunRecord() err = %v", err)
	}
}

func TestRunRecord_ExecutorStartError(t *testing.T) {
	prev := newCLIExportExecutor
	newCLIExportExecutor = func(scope.Scope) (jsexecutor.JsExecutor, error) {
		return failingStartExecutor{}, nil
	}
	t.Cleanup(func() { newCLIExportExecutor = prev })

	_, _, err := RunRecord(context.Background(), nil, RecordOptions{
		Model:      "base.Country",
		OutputPath: "base.Country.csv",
	})
	if err == nil || err.Error() != "start boom" {
		t.Fatalf("RunRecord() err = %v", err)
	}
}

func TestRunRecord_RunExportError(t *testing.T) {
	prevRun := runRecordExport
	prevExec := newCLIExportExecutor
	runRecordExport = func(context.Context, scope.Scope, exportpkg.Spec) (importpkg.Report, registry.Result, error) {
		return importpkg.Report{}, registry.Result{}, errors.New("run boom")
	}
	newCLIExportExecutor = func(scope.Scope) (jsexecutor.JsExecutor, error) {
		return stubCLIExportExecutor{}, nil
	}
	t.Cleanup(func() {
		runRecordExport = prevRun
		newCLIExportExecutor = prevExec
	})

	_, _, err := RunRecord(context.Background(), nil, RecordOptions{
		Model:      "base.Country",
		OutputPath: "base.Country.csv",
	})
	if err == nil || err.Error() != "run boom" {
		t.Fatalf("RunRecord() err = %v", err)
	}
}

func TestRecordSpecFromOptions_InvalidMode(t *testing.T) {
	_, err := recordSpecFromOptions(RecordOptions{
		Model:      "base.Country",
		OutputPath: "base.Country.csv",
		Mode:       exportpkg.Mode("bogus"),
	})
	if err == nil {
		t.Fatal("expected mode error")
	}
}

type failingStartExecutor struct{ stubCLIExportExecutor }

func (failingStartExecutor) Start() error { return errors.New("start boom") }
