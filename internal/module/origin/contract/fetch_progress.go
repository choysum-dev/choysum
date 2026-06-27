// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package contract

import (
	"context"
	"strings"
)

type FetchProgressStage string

const (
	FetchProgressStageDownload FetchProgressStage = "download"
	FetchProgressStageVerify   FetchProgressStage = "verify"
	FetchProgressStageExtract  FetchProgressStage = "extract"
)

type FetchProgressReporter func(stage FetchProgressStage, moduleName string)

type fetchProgressReporterContextKey struct{}

func WithFetchProgressReporter(ctx context.Context, reporter FetchProgressReporter) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if reporter == nil {
		return ctx
	}
	return context.WithValue(ctx, fetchProgressReporterContextKey{}, reporter)
}

func FetchProgressReporterFromContext(ctx context.Context) FetchProgressReporter {
	if ctx == nil {
		return nil
	}
	reporter, _ := ctx.Value(fetchProgressReporterContextKey{}).(FetchProgressReporter)
	return reporter
}

func ReportFetchProgress(ctx context.Context, stage FetchProgressStage, moduleName string) {
	reporter := FetchProgressReporterFromContext(ctx)
	if reporter == nil {
		return
	}
	reporter(stage, strings.TrimSpace(moduleName))
}
