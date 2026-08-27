// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package exportcli

import (
	"context"
	"fmt"
	"strings"

	"github.com/choysum-dev/choysum/internal/export/runner"
	"github.com/choysum-dev/choysum/internal/i18n/langcode"
	"github.com/choysum-dev/choysum/internal/i18n/terms"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// TerminologyOptions configures a CLI terminology export run.
type TerminologyOptions struct {
	Application string
	Module      string
	Lang        string
}

// RunTerminology executes a terminology export and returns the report and PO bytes.
func RunTerminology(ctx context.Context, runtimeScope scope.Scope, opts TerminologyOptions) (importpkg.Report, []byte, error) {
	spec, err := terminologySpecFromOptions(opts)
	if err != nil {
		return importpkg.Report{}, nil, err
	}
	runCtx := terms.OutgoingContextForInternalRPC(ctx, runtimeScope)
	report, result, err := runner.RunWithResult(runCtx, runtimeScope, spec)
	return report, result.POBytes, err
}

func terminologySpecFromOptions(opts TerminologyOptions) (exportpkg.Spec, error) {
	application := strings.TrimSpace(opts.Application)
	module := strings.TrimSpace(opts.Module)
	lang := strings.TrimSpace(opts.Lang)
	if application == "" {
		return exportpkg.Spec{}, fmt.Errorf("application is required")
	}
	if module == "" {
		return exportpkg.Spec{}, fmt.Errorf("module is required")
	}
	if lang == "" {
		return exportpkg.Spec{}, fmt.Errorf("lang is required")
	}
	if !langcode.Valid(lang) {
		return exportpkg.Spec{}, fmt.Errorf("invalid lang format")
	}
	return exportpkg.Spec{
		Profile:     exportpkg.ProfileTerminology,
		Caller:      exportpkg.CallerCLI,
		Format:      "po",
		Application: application,
		Module:      module,
		Lang:        lang,
	}, nil
}
