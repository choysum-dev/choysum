// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package plan

import "context"

type BuildPlanProgress struct {
	Step                 string
	CurrentModule        string
	ResolvedModules      int
	ResolvedDependencies int
}

type BuildPlanProgressReporter func(progress BuildPlanProgress)

type buildPlanProgressReporterContextKey struct{}

func WithBuildPlanProgressReporter(ctx context.Context, reporter BuildPlanProgressReporter) context.Context {
	if reporter == nil {
		return ctx
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, buildPlanProgressReporterContextKey{}, reporter)
}

func BuildPlanProgressReporterFromContext(ctx context.Context) BuildPlanProgressReporter {
	if ctx == nil {
		return nil
	}
	reporter, _ := ctx.Value(buildPlanProgressReporterContextKey{}).(BuildPlanProgressReporter)
	return reporter
}

func reportBuildPlanProgress(ctx context.Context, progress BuildPlanProgress) {
	reporter := BuildPlanProgressReporterFromContext(ctx)
	if reporter == nil {
		return
	}
	reporter(progress)
}
