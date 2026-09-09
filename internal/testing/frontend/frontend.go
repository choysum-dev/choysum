// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// ValidateFrontendTestDependencies is a no-op after FE hard-cut: QuickJS host
// does not require Node/npx/Vitest packages.
func ValidateFrontendTestDependencies(repoRoot string, app string, coverage bool) error {
	_ = repoRoot
	_ = app
	_ = coverage
	return nil
}

// RunOneAppFrontendTests runs one app's FE unit tests on QuickJS + choysumtest.
func RunOneAppFrontendTests(
	ctx context.Context,
	repoRoot string,
	app string,
	junitPath string,
	pattern string,
	coverage bool,
	coverageReport bool,
	coverageCheck bool,
	feCoverageAll bool,
	coverageReportDir string,
	coverageLines int,
	coverageFunctions int,
	coverageBranches int,
	coverageStatements int,
	tmpRoot string,
	keep bool,
) (bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return true, err
	}

	return runOneAppFrontendTestsQJS(
		ctx,
		repoRoot,
		app,
		junitPath,
		pattern,
		coverage,
		coverageReport,
		coverageCheck,
		feCoverageAll,
		coverageReportDir,
		coverageLines,
		coverageFunctions,
		coverageBranches,
		coverageStatements,
		tmpRoot,
		keep,
	)
}

func sanitizeFrontendAppToken(app string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_")
	token := strings.TrimSpace(replacer.Replace(app))
	if token == "" {
		return "app"
	}
	return token
}

func warnIllegalFrontendMarks(repoRoot, app string) {
	hits, err := ScanAppIllegalFrontendMarks(repoRoot, app)
	if err != nil {
		fmt.Fprintf(os.Stderr, "choysum test: FE illegal scan failed for %s: %v\n", app, err)
		return
	}
	if msg := FormatIllegalMarksWarn(hits, repoRoot); msg != "" {
		fmt.Fprint(os.Stderr, msg)
	}
}
