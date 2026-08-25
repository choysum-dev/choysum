// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package importcli

import (
	"context"
	"fmt"
	"strings"

	"github.com/choysum-dev/choysum/internal/import/runner"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// RecordOptions configures a CLI record import run.
type RecordOptions struct {
	Model             string
	SourcePath        string
	Format            string
	Policy            importpkg.Policy
	DryRun            bool
	ColumnMapping     map[string]string
	StubUnitCount     int
	StubFailUnitIndex int
}

// RunRecord executes a record import from CLI flags and returns the report.
func RunRecord(ctx context.Context, runtimeScope scope.Scope, opts RecordOptions) (importpkg.Report, error) {
	spec, err := recordSpecFromOptions(opts)
	if err != nil {
		return importpkg.Report{}, err
	}
	return runner.Run(ctx, runtimeScope, spec)
}

func recordSpecFromOptions(opts RecordOptions) (importpkg.Spec, error) {
	model := strings.TrimSpace(opts.Model)
	sourcePath := strings.TrimSpace(opts.SourcePath)
	if model == "" {
		return importpkg.Spec{}, fmt.Errorf("model is required")
	}
	if sourcePath == "" {
		return importpkg.Spec{}, fmt.Errorf("source path is required")
	}

	format := strings.TrimSpace(opts.Format)
	if format == "" {
		format = "csv"
	}

	policy := opts.Policy
	if policy == importpkg.PolicyUnspecified {
		policy = importpkg.PolicyAtomic
	}

	columnMapping := opts.ColumnMapping
	if columnMapping == nil {
		columnMapping = map[string]string{}
	}

	return importpkg.Spec{
		Profile: importpkg.ProfileRecord,
		Caller:  importpkg.CallerCLI,
		Policy:  policy,
		DryRun:  opts.DryRun,
		Model:   model,
		Source: importpkg.Source{
			Format: format,
			Path:   sourcePath,
		},
		Options: importpkg.Options{
			ColumnMapping:     columnMapping,
			StubUnitCount:     opts.StubUnitCount,
			StubFailUnitIndex: opts.StubFailUnitIndex,
		},
	}, nil
}
