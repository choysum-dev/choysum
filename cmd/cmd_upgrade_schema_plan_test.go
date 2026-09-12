// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/module/evolution/schema"
	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func TestSchemaPlanModuleName(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"auth", "auth"},
		{"auth@1.2.3", "auth"},
		{"  sales@latest  ", "sales"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := schemaPlanModuleName(tt.in); got != tt.want {
			t.Fatalf("schemaPlanModuleName(%q)=%q want %q", tt.in, got, tt.want)
		}
	}
}

func TestPrintSchemaPlan(t *testing.T) {
	printSchemaPlan(nil, "m", schema.SchemaPlan{}, nil)
	env := &schemaPlanTestScope{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	printSchemaPlan(env, "m", schema.SchemaPlan{
		Ops:      []schema.PlanOp{{Kind: schema.OpCreateTable, Safety: schema.SafetyAuto, Table: "t", Detail: "create"}},
		Leftover: []schema.Leftover{{Kind: schema.LeftoverColumn, Table: "t", Name: "old"}},
	}, errors.New("guarded"))
	printSchemaPlan(&schemaPlanTestScope{}, "m", schema.SchemaPlan{}, nil)
}

func TestExecuteSchemaPlanOnly(t *testing.T) {
	env := &schemaPlanTestScope{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	ok := &fakeLifecycleService{plan: schema.SchemaPlan{Ops: []schema.PlanOp{{Kind: schema.OpCreateTable}}}}
	if code := executeSchemaPlanOnly(context.Background(), env, ok, []upgradePlanItem{
		{requestedInput: "auth", resolvedInput: "auth@1.0.0"},
	}); code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	fail := &fakeLifecycleService{err: errors.New("guarded ops")}
	if code := executeSchemaPlanOnly(context.Background(), env, fail, []upgradePlanItem{
		{requestedInput: "auth", resolvedInput: "auth"},
	}); code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
}

func TestUpgradeCommandRegistersSchemaPlanFlag(t *testing.T) {
	cmd := newUpgradeCmd(func() scope.Scope { return nil })
	flag := cmd.Flags().Lookup("schema-plan")
	if flag == nil {
		t.Fatal("expected --schema-plan flag")
	}
	if !strings.Contains(flag.Usage, "without applying DDL") {
		t.Fatalf("unexpected usage: %q", flag.Usage)
	}
}

type fakeLifecycleService struct {
	plan schema.SchemaPlan
	err  error
}

func (f *fakeLifecycleService) Install(context.Context, lifecycle.InstallRequest) error {
	return nil
}
func (f *fakeLifecycleService) Upgrade(context.Context, lifecycle.UpgradeRequest) error {
	return nil
}
func (f *fakeLifecycleService) Uninstall(context.Context, lifecycle.UninstallRequest) error {
	return nil
}
func (f *fakeLifecycleService) SchemaPlan(context.Context, string) (schema.SchemaPlan, error) {
	return f.plan, f.err
}

type schemaPlanTestScope struct {
	logger *slog.Logger
}

func (e *schemaPlanTestScope) Run(fn func(scope.Scope) error) error { return fn(e) }
func (e *schemaPlanTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}
func (e *schemaPlanTestScope) Session() *scope.Session                 { return nil }
func (e *schemaPlanTestScope) WithContext(context.Context) scope.Scope { return e }
func (e *schemaPlanTestScope) Context() context.Context                { return context.Background() }
func (e *schemaPlanTestScope) Logger() *slog.Logger                    { return e.logger }
func (e *schemaPlanTestScope) Config() *config.Config {
	return &config.Config{Server: config.NewDefaultServerConfig()}
}
func (e *schemaPlanTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}
