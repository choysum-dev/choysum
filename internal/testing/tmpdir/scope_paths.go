// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package tmpdir

import (
	"context"
	"path/filepath"
	"strings"
)

type cliTestTmpRootContextKey struct{}
type cliTestRunHomeContextKey struct{}

// ContextWithCLITestTmpRoot stores the CLI test temporary root for harnesses
// that resolve paths via context (makeTestScope, frontend, typecheck).
func ContextWithCLITestTmpRoot(ctx context.Context, tmpRoot string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	tmpRoot = strings.TrimSpace(tmpRoot)
	if tmpRoot == "" {
		return ctx
	}
	return context.WithValue(ctx, cliTestTmpRootContextKey{}, tmpRoot)
}

// CLITestTmpRootFromContext reads the CLI test temporary root from context.
func CLITestTmpRootFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(cliTestTmpRootContextKey{}).(string)
	return strings.TrimSpace(v)
}

// ContextWithCLITestRunHome stores the shared DefaultChoysumPath for one CLI test run.
func ContextWithCLITestRunHome(ctx context.Context, runHome string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	runHome = strings.TrimSpace(runHome)
	if runHome == "" {
		return ctx
	}
	return context.WithValue(ctx, cliTestRunHomeContextKey{}, runHome)
}

// CLITestRunHomeFromContext reads the shared CLI test DefaultChoysumPath from context.
func CLITestRunHomeFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(cliTestRunHomeContextKey{}).(string)
	return strings.TrimSpace(v)
}

// BindCLITestRuntimePaths prepares context with a CLI test tmp root and shared
// run home. It ensures a testing run-id exists on context.
func BindCLITestRuntimePaths(ctx context.Context, workspaceRoot string) (context.Context, string, string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	testTmp, err := CLITestTmpRoot()
	if err != nil {
		return ctx, "", "", err
	}
	if strings.TrimSpace(workspaceRoot) == "" {
		workspaceRoot = "."
	}
	if abs, err := filepath.Abs(workspaceRoot); err == nil {
		workspaceRoot = abs
	}
	if TestingRunIDFromContext(ctx) == "" {
		ctx = ContextWithTestingRunID(ctx, NewTestingRunID())
	}
	runHome, err := ResolveCLITestingRunHome(ctx, workspaceRoot, testTmp)
	if err != nil {
		return ctx, "", "", err
	}
	ctx = ContextWithCLITestTmpRoot(ctx, testTmp)
	ctx = ContextWithCLITestRunHome(ctx, runHome)
	return ctx, testTmp, runHome, nil
}

// EffectiveCLITestTmpRoot returns the context override when set, otherwise tmpRoot.
func EffectiveCLITestTmpRoot(ctx context.Context, tmpRoot string) string {
	if override := CLITestTmpRootFromContext(ctx); override != "" {
		return override
	}
	return strings.TrimSpace(tmpRoot)
}

// EffectiveCLITestRunHome returns the context shared home when set.
func EffectiveCLITestRunHome(ctx context.Context) string {
	return CLITestRunHomeFromContext(ctx)
}
