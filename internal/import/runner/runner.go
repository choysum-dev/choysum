// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runner

import (
	"context"
	"strings"

	"github.com/choysum-dev/choysum/internal/import/adapter"
	"github.com/choysum-dev/choysum/internal/import/artifact"
	"github.com/choysum-dev/choysum/internal/import/registry"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// Run is the default import runner implementation.
func Run(ctx context.Context, runtimeScope scope.Scope, spec importpkg.Spec) (importpkg.Report, error) {
	builder, err := adapter.PlanBuilderFor(spec.Source.Format)
	if err != nil {
		return importpkg.Report{}, err
	}
	p, err := builder.Build(ctx, spec)
	if err != nil {
		return importpkg.Report{}, err
	}

	writer, err := registry.WriterFor(spec.Profile)
	if err != nil {
		return importpkg.Report{}, err
	}

	report, err := executePlan(ctx, runtimeScope, spec, p, writer)
	report = attachErrorArtifact(ctx, runtimeScope, spec, report)
	return report, err
}

func attachErrorArtifact(ctx context.Context, runtimeScope scope.Scope, spec importpkg.Spec, report importpkg.Report) importpkg.Report {
	if len(report.Messages) == 0 {
		return report
	}
	companyID := strings.TrimSpace(spec.Options.CompanyID)
	if companyID == "" {
		return report
	}
	if err := artifact.WriteErrorArtifact(ctx, runtimeScope, companyID, &report); err != nil {
		return report
	}
	return report
}
