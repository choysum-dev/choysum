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
	internalorigin "github.com/choysum-dev/choysum/internal/module/origin"
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
	mixed := &fakeLifecycleService{err: errors.New("guarded ops")}
	if code := executeSchemaPlanOnly(context.Background(), env, mixed, []upgradePlanItem{
		{requestedInput: "auth", resolvedInput: "auth"},
		{requestedInput: "base", resolvedInput: "base"},
	}); code != 1 {
		t.Fatalf("mixed exit code = %d, want 1", code)
	}
	if mixed.installs != 0 || mixed.upgrades != 0 || mixed.uninstalls != 0 {
		t.Fatal("schema-plan must not mutate modules")
	}
	if mixed.schemaPlans != 2 {
		t.Fatalf("schemaPlans = %d, want 2", mixed.schemaPlans)
	}
}

func TestSchemaPlanItemsFromArgs(t *testing.T) {
	plans, err := schemaPlanItemsFromArgs([]string{"auth@latest", " base "})
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 2 || plans[0].resolvedInput != "auth" || plans[1].resolvedInput != "base" {
		t.Fatalf("%#v", plans)
	}
	if _, err := schemaPlanItemsFromArgs([]string{"  "}); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("empty: %v", err)
	}
	if _, err := schemaPlanItemsFromArgs([]string{"bad/name"}); err == nil {
		t.Fatal("expected parse error")
	}

	origParse := parseModuleInput
	t.Cleanup(func() { parseModuleInput = origParse })
	parseModuleInput = func(string) (internalorigin.ParsedInput, error) {
		return internalorigin.ParsedInput{LocalName: "from-local"}, nil
	}
	plans, err = schemaPlanItemsFromArgs([]string{"x"})
	if err != nil || len(plans) != 1 || plans[0].resolvedInput != "from-local" {
		t.Fatalf("local name fallback: %#v %v", plans, err)
	}
	parseModuleInput = func(string) (internalorigin.ParsedInput, error) {
		return internalorigin.ParsedInput{}, nil
	}
	if _, err := schemaPlanItemsFromArgs([]string{"x"}); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("both names empty: %v", err)
	}
}

func TestRunUpgradeSchemaPlan(t *testing.T) {
	env := &schemaPlanTestScope{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if code := runUpgradeSchemaPlan(context.Background(), env, []string{"  "}); code != 1 {
		t.Fatalf("empty args exit = %d", code)
	}
	// Nil logger error path.
	if code := runUpgradeSchemaPlan(context.Background(), &schemaPlanTestScope{}, []string{"bad/name"}); code != 1 {
		t.Fatalf("parse error exit = %d", code)
	}
	// Valid name reaches SchemaPlan (fails without DB session) and returns non-zero.
	if code := runUpgradeSchemaPlan(context.Background(), env, []string{"auth@latest"}); code != 1 {
		t.Fatalf("schema plan without DB exit = %d", code)
	}
}

func TestUpgradeCommandSchemaPlanDispatchesEarly(t *testing.T) {
	var gotCode int
	origExit := upgradeExit
	upgradeExit = func(code int) {
		gotCode = code
		// Return normally so the explicit `return` after upgradeExit is covered.
	}
	t.Cleanup(func() { upgradeExit = origExit })

	env := &schemaPlanTestScope{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	cmd := newUpgradeCmd(func() scope.Scope { return env })
	cmd.SetArgs([]string{"--schema-plan", "auth"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if gotCode != 1 {
		t.Fatalf("exit code = %d, want 1", gotCode)
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
	plan        schema.SchemaPlan
	err         error
	installs    int
	upgrades    int
	uninstalls  int
	schemaPlans int
}

func (f *fakeLifecycleService) Install(context.Context, lifecycle.InstallRequest) error {
	f.installs++
	return nil
}
func (f *fakeLifecycleService) Upgrade(context.Context, lifecycle.UpgradeRequest) error {
	f.upgrades++
	return nil
}
func (f *fakeLifecycleService) Uninstall(context.Context, lifecycle.UninstallRequest) error {
	f.uninstalls++
	return nil
}
func (f *fakeLifecycleService) SchemaPlan(context.Context, string) (schema.SchemaPlan, error) {
	f.schemaPlans++
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
