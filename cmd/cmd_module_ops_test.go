// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/module/ops"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func TestFormatPlanBlockers(t *testing.T) {
	err := formatPlanBlockers(nil)
	if err == nil || !strings.Contains(err.Error(), "has blockers") {
		t.Fatalf("empty blockers = %v", err)
	}
	err = formatPlanBlockers([]ops.PlanIssue{
		{Code: "A", Message: "first"},
		{Code: "B", Message: ""},
		{Message: "  "},
	})
	if err == nil || !strings.Contains(err.Error(), "first") || !strings.Contains(err.Error(), "B") {
		t.Fatalf("blockers = %v", err)
	}
}

func TestRunCLIModuleOpValidation(t *testing.T) {
	if err := runCLIModuleOp(context.Background(), nil, cliModuleOpOptions{Action: ops.ActionInstall, ModuleName: "core"}); err == nil || !strings.Contains(err.Error(), "scope") {
		t.Fatalf("nil env = %v", err)
	}
	env := &commandTestScope{ctx: context.Background()}
	if err := runCLIModuleOp(context.Background(), env, cliModuleOpOptions{ModuleName: "core"}); err == nil || !strings.Contains(err.Error(), "action") {
		t.Fatalf("empty action = %v", err)
	}
	if err := runCLIModuleOp(context.Background(), env, cliModuleOpOptions{Action: ops.ActionInstall}); err == nil || !strings.Contains(err.Error(), "module name") {
		t.Fatalf("empty module = %v", err)
	}
}

func TestRunCLIModuleOpExecutorCreateError(t *testing.T) {
	prev := newCLICompilerExecutor
	t.Cleanup(func() { newCLICompilerExecutor = prev })
	newCLICompilerExecutor = func(_ scope.Scope, _ ...jsexecutor.Option) (jsexecutor.JsExecutor, error) {
		return nil, errors.New("no engine")
	}
	env := &commandTestScope{ctx: context.Background()}
	err := runCLIModuleOp(context.Background(), env, cliModuleOpOptions{
		Action:     ops.ActionUninstall,
		ModuleName: "core",
	})
	if err == nil || !strings.Contains(err.Error(), "Error creating compiler executor") {
		t.Fatalf("create err = %v", err)
	}
}
