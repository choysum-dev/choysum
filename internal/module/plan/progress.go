// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package plan

import "context"

// BuildPlanProgress carries incremental progress information during
// dependency resolution and build planning.
type BuildPlanProgress struct {
	Step                 string
	CurrentModule        string
	ResolvedModules      int
	ResolvedDependencies int
}

// BuildPlanProgressReporter is a callback that receives incremental
// build-plan progress events.
type BuildPlanProgressReporter func(progress BuildPlanProgress)

type buildPlanProgressReporterContextKey struct{}

// WithBuildPlanProgressReporter stores a BuildPlanProgressReporter in ctx
// so downstream planning operations can report progress.
func WithBuildPlanProgressReporter(ctx context.Context, reporter BuildPlanProgressReporter) context.Context {
	if reporter == nil {
		return ctx
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, buildPlanProgressReporterContextKey{}, reporter)
}

// BuildPlanProgressReporterFromContext returns a BuildPlanProgressReporter
// from ctx when one has been stored via WithBuildPlanProgressReporter.
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
