// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package exportcli

import (
	"context"
	"fmt"
	"strings"

	importcli "github.com/choysum-dev/choysum/internal/cli/import"
	"github.com/choysum-dev/choysum/internal/export/runner"
	importcaller "github.com/choysum-dev/choysum/internal/import/caller"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/scope"
)

var (
	runRecordExport      = runner.RunWithResult
	validateExportSpec   = exportpkg.ValidateSpec
	newCLIExportExecutor = func(runtimeScope scope.Scope) (jsexecutor.JsExecutor, error) {
		return jsexecutor.NewRuntimeExecutor(runtimeScope, nil)
	}
)

// RecordOptions configures a CLI record export run.
type RecordOptions struct {
	Model      string
	OutputPath string
	Mode       exportpkg.Mode
}

// RunRecord executes a record export from CLI flags and returns the report and CSV bytes.
func RunRecord(ctx context.Context, runtimeScope scope.Scope, opts RecordOptions) (importpkg.Report, []byte, error) {
	spec, err := recordSpecFromOptions(opts)
	if err != nil {
		return importpkg.Report{}, nil, err
	}
	if err := validateExportSpec(spec); err != nil {
		return importpkg.Report{}, nil, err
	}

	executor, err := newCLIExportExecutor(runtimeScope)
	if err != nil {
		return importpkg.Report{}, nil, err
	}
	if err := executor.Start(); err != nil {
		return importpkg.Report{}, nil, err
	}
	defer executor.Stop()

	runCtx := importcaller.ContextWithCaller(ctx, importcaller.ExecutorCaller{Engine: cliExportEngineAdapter{inner: executor}})
	report, result, err := runRecordExport(runCtx, runtimeScope, spec)
	return report, result.CSVBytes, err
}

type cliExportEngineAdapter struct {
	inner jsexecutor.JsExecutor
}

func (a cliExportEngineAdapter) Load(_ []*jsengine.JsScript) error { return nil }

func (a cliExportEngineAdapter) Execute(ctx context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	return a.inner.Execute(ctx, req)
}

func (a cliExportEngineAdapter) Close() error { return nil }

func recordSpecFromOptions(opts RecordOptions) (exportpkg.Spec, error) {
	model := strings.TrimSpace(opts.Model)
	outputPath := strings.TrimSpace(opts.OutputPath)
	if model == "" {
		return exportpkg.Spec{}, fmt.Errorf("model is required")
	}
	if outputPath == "" {
		return exportpkg.Spec{}, fmt.Errorf("output path is required")
	}
	if err := importcli.ValidateCSVSourcePath(outputPath); err != nil {
		return exportpkg.Spec{}, err
	}

	mode := exportpkg.EffectiveMode(opts.Mode)
	if !mode.Valid() {
		return exportpkg.Spec{}, fmt.Errorf("unsupported export mode %q", opts.Mode)
	}

	return exportpkg.Spec{
		Profile: exportpkg.ProfileRecord,
		Caller:  exportpkg.CallerCLI,
		Model:   model,
		Format:  "csv",
		Mode:    mode,
	}, nil
}
