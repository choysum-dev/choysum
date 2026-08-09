// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package plan

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
)

type fakeResolver struct {
	peek func(ctx context.Context, name string) (*meta.Module, error)
	load func(name string) (*meta.Module, error)
}

func (r fakeResolver) Peek(ctx context.Context, name string) (*meta.Module, error) {
	if r.peek == nil {
		return nil, nil
	}
	return r.peek(ctx, name)
}
func (r fakeResolver) Load(name string) (*meta.Module, error) {
	if r.load == nil {
		return nil, nil
	}
	return r.load(name)
}

func TestBuildPlanValidationAndGuardErrors(t *testing.T) {
	root := &meta.Module{Name: "auth", ApplicationStr: "auth"}

	if _, err := BuildPlan(context.Background(), OpInstall, nil, fakeResolver{}); err == nil || err.Error() != "root module is nil" {
		t.Fatalf("unexpected nil root error: %v", err)
	}
	if _, err := BuildPlan(context.Background(), OpInstall, root, nil); err == nil || err.Error() != "resolver is nil" {
		t.Fatalf("unexpected nil resolver error: %v", err)
	}
	if _, err := BuildPlan(context.Background(), OpType("broken"), root, fakeResolver{}); err == nil || err.Error() != `unknown op: "broken"` {
		t.Fatalf("unexpected unknown op error: %v", err)
	}

	plan, err := BuildPlan(context.TODO(), OpUpgrade, root, fakeResolver{})
	if err != nil {
		t.Fatalf("BuildPlan(context.TODO()) error: %v", err)
	}
	if len(plan.ModuleOrder) != 1 || plan.ModuleOrder[0] != "auth" {
		t.Fatalf("unexpected module order with context.TODO(): %v", plan.ModuleOrder)
	}
}

func TestBuildPlanInstallErrorsAndAppCollection(t *testing.T) {
	root := &meta.Module{
		Name:           "base",
		ApplicationStr: "crm",
		DependsStr:     []byte(` ["dep", "dep", " ", "webmod"] `),
	}
	peekCalls := 0
	loadCalls := 0
	r := fakeResolver{
		peek: func(ctx context.Context, name string) (*meta.Module, error) {
			peekCalls++
			switch name {
			case "dep":
				return &meta.Module{Name: "dep", ApplicationStr: "crm"}, nil
			case "webmod":
				return &meta.Module{Name: "webmod", ApplicationStr: "web", WebEntryPoint: "web/index.ts"}, nil
			default:
				return nil, nil
			}
		},
		load: func(name string) (*meta.Module, error) {
			loadCalls++
			return nil, nil
		},
	}

	plan, err := BuildPlan(context.Background(), OpInstall, root, r, WithSkipWebShell(true))
	if err != nil {
		t.Fatalf("BuildPlan() error: %v", err)
	}
	if len(plan.ModuleOrder) != 3 || plan.ModuleOrder[0] != "dep" || plan.ModuleOrder[1] != "webmod" || plan.ModuleOrder[2] != "base" {
		t.Fatalf("unexpected module order: %v", plan.ModuleOrder)
	}
	if len(plan.AffectedApps) != 2 || plan.AffectedApps[0] != "crm" || plan.AffectedApps[1] != "web" {
		t.Fatalf("unexpected affected apps: %v", plan.AffectedApps)
	}
	if !plan.NeedsGlobalWebBuild {
		t.Fatal("expected web entry point dependency to require global web build")
	}
	if peekCalls != 2 {
		t.Fatalf("expected peek to dedupe dependencies, got %d calls", peekCalls)
	}
	if loadCalls != 2 {
		t.Fatalf("expected load to be called for dep and webmod, got %d", loadCalls)
	}
}

func TestBuildPlanInstallAutoIncludesWebShell(t *testing.T) {
	root := &meta.Module{
		Name:           "partner",
		ApplicationStr: "partner",
		WebEntryPoint:  "web/index.ts",
		DependsStr:     []byte(`["core"]`),
	}
	r := fakeResolver{
		peek: func(ctx context.Context, name string) (*meta.Module, error) {
			switch name {
			case "core":
				return &meta.Module{Name: "core", ApplicationStr: "core"}, nil
			case "auth":
				return &meta.Module{Name: "auth", ApplicationStr: "auth", DependsStr: []byte(`["core"]`)}, nil
			case "web":
				return &meta.Module{Name: "web", ApplicationStr: "web", WebEntryPoint: "web/index.ts", DependsStr: []byte(`["core","auth"]`)}, nil
			default:
				return nil, nil
			}
		},
		load: func(name string) (*meta.Module, error) { return nil, nil },
	}

	plan, err := BuildPlan(context.Background(), OpInstall, root, r)
	if err != nil {
		t.Fatalf("BuildPlan() error: %v", err)
	}
	if !plan.NeedsGlobalWebBuild {
		t.Fatal("expected NeedsGlobalWebBuild")
	}
	want := []string{"core", "auth", "web", "partner"}
	if len(plan.ModuleOrder) != len(want) {
		t.Fatalf("module order=%v, want %v", plan.ModuleOrder, want)
	}
	for i, name := range want {
		if plan.ModuleOrder[i] != name {
			t.Fatalf("module order=%v, want %v", plan.ModuleOrder, want)
		}
	}
}

func TestBuildPlanUpgradeEnsureOrderInstallsMissingWebShell(t *testing.T) {
	root := &meta.Module{
		Name:           "partner",
		ApplicationStr: "partner",
		WebEntryPoint:  "web/index.ts",
	}
	r := fakeResolver{
		peek: func(ctx context.Context, name string) (*meta.Module, error) {
			switch name {
			case "core":
				return &meta.Module{Name: "core", ApplicationStr: "core"}, nil
			case "auth":
				return &meta.Module{Name: "auth", ApplicationStr: "auth", DependsStr: []byte(`["core"]`)}, nil
			case "web":
				return &meta.Module{Name: "web", ApplicationStr: "web", WebEntryPoint: "web/index.ts", DependsStr: []byte(`["core","auth"]`)}, nil
			default:
				return nil, nil
			}
		},
		load: func(name string) (*meta.Module, error) {
			if name == "core" {
				return &meta.Module{Name: "core", Status: meta.Installed, ApplicationStr: "core"}, nil
			}
			return nil, nil
		},
	}

	plan, err := BuildPlan(context.Background(), OpUpgrade, root, r)
	if err != nil {
		t.Fatalf("BuildPlan() error: %v", err)
	}
	if len(plan.ModuleOrder) != 1 || plan.ModuleOrder[0] != "partner" {
		t.Fatalf("ModuleOrder=%v, want [partner]", plan.ModuleOrder)
	}
	wantEnsure := []string{"auth", "web"}
	if len(plan.EnsureOrder) != len(wantEnsure) {
		t.Fatalf("EnsureOrder=%v, want %v", plan.EnsureOrder, wantEnsure)
	}
	for i, name := range wantEnsure {
		if plan.EnsureOrder[i] != name {
			t.Fatalf("EnsureOrder=%v, want %v", plan.EnsureOrder, wantEnsure)
		}
	}
	if !plan.NeedsGlobalWebBuild {
		t.Fatal("expected NeedsGlobalWebBuild")
	}
}

func TestBuildPlanSkipWebShell(t *testing.T) {
	root := &meta.Module{
		Name:           "partner",
		ApplicationStr: "partner",
		WebEntryPoint:  "web/index.ts",
	}
	r := fakeResolver{
		load: func(name string) (*meta.Module, error) { return nil, nil },
	}
	plan, err := BuildPlan(context.Background(), OpInstall, root, r, WithSkipWebShell(true))
	if err != nil {
		t.Fatalf("BuildPlan() error: %v", err)
	}
	if moduleOrderContains(plan.ModuleOrder, "web") {
		t.Fatalf("expected no web in ModuleOrder with SkipWebShell, got %v", plan.ModuleOrder)
	}
	if len(plan.EnsureOrder) != 0 {
		t.Fatalf("expected empty EnsureOrder, got %v", plan.EnsureOrder)
	}
}

func TestBuildPlanInstallDependencyErrors(t *testing.T) {
	tests := []struct {
		name string
		root *meta.Module
		res  fakeResolver
		want string
	}{
		{
			name: "load web module error",
			root: &meta.Module{Name: "auth", ApplicationStr: "auth"},
			res: fakeResolver{
				load: func(name string) (*meta.Module, error) {
					if name == "web" {
						return nil, errors.New("load web failed")
					}
					return nil, nil
				},
			},
			want: "load web module for plan: load web failed",
		},
		{
			name: "depends unmarshal error",
			root: &meta.Module{Name: "auth", DependsStr: []byte(`{broken`)},
			res:  fakeResolver{},
			want: "unmarshal depends for auth",
		},
		{
			name: "load dependency error",
			root: &meta.Module{Name: "auth", DependsStr: []byte(` ["dep"] `)},
			res: fakeResolver{
				load: func(name string) (*meta.Module, error) { return nil, errors.New("load dep failed") },
			},
			want: "load dependency dep: load dep failed",
		},
		{
			name: "peek dependency error",
			root: &meta.Module{Name: "auth", DependsStr: []byte(` ["dep"] `)},
			res: fakeResolver{
				peek: func(ctx context.Context, name string) (*meta.Module, error) {
					return nil, errors.New("peek dep failed")
				},
			},
			want: "peek dependency dep: peek dep failed",
		},
		{
			name: "dependency cycle error",
			root: &meta.Module{Name: "auth", DependsStr: []byte(` ["dep"] `)},
			res: fakeResolver{
				peek: func(ctx context.Context, name string) (*meta.Module, error) {
					switch name {
					case "dep":
						return &meta.Module{Name: "dep", DependsStr: []byte(` ["auth"] `)}, nil
					case "auth":
						return &meta.Module{Name: "auth", DependsStr: []byte(` ["dep"] `)}, nil
					}
					return nil, nil
				},
			},
			want: "dependency cycle detected: auth -> dep -> auth",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := BuildPlan(context.Background(), OpInstall, tc.root, tc.res)
			if err == nil || err.Error() != tc.want && !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestBuildPlanInstallContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	root := &meta.Module{Name: "auth", DependsStr: []byte(` ["dep"] `)}
	resolver := fakeResolver{
		peek: func(ctx context.Context, name string) (*meta.Module, error) {
			return &meta.Module{Name: name}, nil
		},
	}
	cancel()

	_, err := BuildPlan(ctx, OpInstall, root, resolver)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}

func TestBuildPlanUninstallLoadError(t *testing.T) {
	root := &meta.Module{Name: "auth", ApplicationStr: "auth"}
	_, err := BuildPlan(context.Background(), OpUninstall, root, fakeResolver{
		load: func(name string) (*meta.Module, error) {
			return nil, errors.New("load failed")
		},
	})
	if err == nil || err.Error() != "load module auth: load failed" {
		t.Fatalf("unexpected uninstall load error: %v", err)
	}
}

func TestBuildPlan_NeedsGlobalWebBuildFalseWithoutWebModule(t *testing.T) {
	root := &meta.Module{Name: "auth", ApplicationStr: "auth"}
	r := fakeResolver{
		peek: func(ctx context.Context, name string) (*meta.Module, error) {
			return &meta.Module{Name: name, ApplicationStr: name}, nil
		},
		load: func(name string) (*meta.Module, error) { return nil, nil },
	}

	plan, err := BuildPlan(context.Background(), OpInstall, root, r)
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}
	if plan.NeedsGlobalWebBuild {
		t.Fatalf("expected NeedsGlobalWebBuild=false, got true")
	}
}

func TestBuildPlan_NeedsGlobalWebBuildTrueWhenRootIsWeb(t *testing.T) {
	root := &meta.Module{Name: "web", ApplicationStr: "web", WebEntryPoint: "web/index.ts"}
	r := fakeResolver{
		peek: func(ctx context.Context, name string) (*meta.Module, error) {
			return &meta.Module{Name: name, ApplicationStr: name}, nil
		},
		load: func(name string) (*meta.Module, error) { return nil, nil },
	}

	plan, err := BuildPlan(context.Background(), OpInstall, root, r)
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}
	if !plan.NeedsGlobalWebBuild {
		t.Fatalf("expected NeedsGlobalWebBuild=true, got false")
	}
}

func TestBuildPlan_NeedsGlobalWebBuildTrueWhenWebInstalled(t *testing.T) {
	root := &meta.Module{Name: "auth", ApplicationStr: "auth"}
	r := fakeResolver{
		peek: func(ctx context.Context, name string) (*meta.Module, error) {
			return &meta.Module{Name: name, ApplicationStr: name}, nil
		},
		load: func(name string) (*meta.Module, error) {
			if name == "web" {
				return &meta.Module{Name: "web", Status: meta.Installed, WebEntryPoint: "web/index.ts"}, nil
			}
			return nil, nil
		},
	}

	plan, err := BuildPlan(context.Background(), OpInstall, root, r)
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}
	if !plan.NeedsGlobalWebBuild {
		t.Fatalf("expected NeedsGlobalWebBuild=true, got false")
	}
}

func TestBuildPlan_UpgradeUsesRootOnly(t *testing.T) {
	root := &meta.Module{Name: "auth", ApplicationStr: "auth"}
	r := fakeResolver{
		peek: func(ctx context.Context, name string) (*meta.Module, error) {
			return &meta.Module{Name: name, ApplicationStr: name}, nil
		},
		load: func(name string) (*meta.Module, error) { return nil, nil },
	}

	plan, err := BuildPlan(context.Background(), OpUpgrade, root, r)
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}
	if len(plan.ModuleOrder) != 1 || plan.ModuleOrder[0] != "auth" {
		t.Fatalf("expected ModuleOrder=[auth], got %v", plan.ModuleOrder)
	}
	if len(plan.AffectedApps) != 1 || plan.AffectedApps[0] != "auth" {
		t.Fatalf("expected AffectedApps=[auth], got %v", plan.AffectedApps)
	}
}

func TestBuildPlan_UninstallOrdersDependentsFirst(t *testing.T) {
	root := &meta.Module{Name: "base", ApplicationStr: "base"}
	modules := map[string]*meta.Module{
		"base":       {Name: "base", ApplicationStr: "base", Dependents: []*meta.Module{{Name: "auth"}}},
		"auth":       {Name: "auth", ApplicationStr: "auth", Dependents: []*meta.Module{{Name: "auth_addon"}}},
		"auth_addon": {Name: "auth_addon", ApplicationStr: "auth"},
	}
	r := fakeResolver{
		peek: func(ctx context.Context, name string) (*meta.Module, error) { return nil, nil },
		load: func(name string) (*meta.Module, error) { return modules[name], nil },
	}

	plan, err := BuildPlan(context.Background(), OpUninstall, root, r)
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}
	if len(plan.ModuleOrder) != 3 || plan.ModuleOrder[0] != "auth_addon" || plan.ModuleOrder[1] != "auth" || plan.ModuleOrder[2] != "base" {
		t.Fatalf("unexpected uninstall order: %v", plan.ModuleOrder)
	}
	if len(plan.AffectedApps) != 2 {
		t.Fatalf("expected two affected apps, got %v", plan.AffectedApps)
	}
}

func TestBuildPlan_UninstallTreatsMissingModuleAsNoOp(t *testing.T) {
	root := &meta.Module{Name: "missing", ApplicationStr: "auth"}
	r := fakeResolver{
		peek: func(ctx context.Context, name string) (*meta.Module, error) { return nil, nil },
		load: func(name string) (*meta.Module, error) { return nil, nil },
	}

	plan, err := BuildPlan(context.Background(), OpUninstall, root, r)
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}
	if len(plan.ModuleOrder) != 0 {
		t.Fatalf("expected empty uninstall order for missing module, got %v", plan.ModuleOrder)
	}
	if len(plan.AffectedApps) != 0 {
		t.Fatalf("expected no affected apps, got %v", plan.AffectedApps)
	}
}

func TestBuildPlan_UninstallDetectsDependentCycle(t *testing.T) {
	root := &meta.Module{Name: "base", ApplicationStr: "base"}
	modules := map[string]*meta.Module{
		"base": {Name: "base", ApplicationStr: "base", Dependents: []*meta.Module{{Name: "auth"}}},
		"auth": {Name: "auth", ApplicationStr: "auth", Dependents: []*meta.Module{{Name: "base"}}},
	}
	r := fakeResolver{
		peek: func(ctx context.Context, name string) (*meta.Module, error) { return nil, nil },
		load: func(name string) (*meta.Module, error) { return modules[name], nil },
	}

	_, err := BuildPlan(context.Background(), OpUninstall, root, r)
	if err == nil || err.Error() != "dependent cycle detected: base -> auth -> base" {
		t.Fatalf("unexpected uninstall cycle error: %v", err)
	}
}

func TestBuildPlan_AffectedAppsSortedForStableLogs(t *testing.T) {
	root := &meta.Module{
		Name:           "root",
		ApplicationStr: "zeta",
		DependsStr:     []byte(` ["dep_b", "dep_a", "webmod"] `),
	}
	r := fakeResolver{
		peek: func(ctx context.Context, name string) (*meta.Module, error) {
			switch name {
			case "dep_a":
				return &meta.Module{Name: "dep_a", ApplicationStr: "alpha"}, nil
			case "dep_b":
				return &meta.Module{Name: "dep_b", ApplicationStr: "beta"}, nil
			case "webmod":
				return &meta.Module{Name: "webmod", ApplicationStr: "web", WebEntryPoint: "web/index.ts"}, nil
			default:
				return nil, nil
			}
		},
		load: func(name string) (*meta.Module, error) { return nil, nil },
	}

	plan, err := BuildPlan(context.Background(), OpInstall, root, r, WithSkipWebShell(true))
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}

	if len(plan.AffectedApps) != 4 {
		t.Fatalf("expected 4 affected apps, got %v", plan.AffectedApps)
	}
	want := []string{"alpha", "beta", "web", "zeta"}
	for i := range want {
		if plan.AffectedApps[i] != want[i] {
			t.Fatalf("expected sorted affected apps %v, got %v", want, plan.AffectedApps)
		}
	}
}

func TestWithBuildPlanProgressReporter_NilReporterReturnsSameCtx(t *testing.T) {
	ctx := context.Background()
	result := WithBuildPlanProgressReporter(ctx, nil)
	if result != ctx {
		t.Fatal("expected same ctx when reporter is nil")
	}
}

func TestWithBuildPlanProgressReporter_MissingContextUsesBackground(t *testing.T) {
	result := WithBuildPlanProgressReporter(context.TODO(), func(progress BuildPlanProgress) {})
	if result == nil {
		t.Fatal("expected non-nil ctx when input ctx is nil")
	}
	reporter := BuildPlanProgressReporterFromContext(result)
	if reporter == nil {
		t.Fatal("expected reporter stored in ctx")
	}
}

func TestBuildPlanProgressReporterFromContext_MissingContext(t *testing.T) {
	reporter := BuildPlanProgressReporterFromContext(context.TODO())
	if reporter != nil {
		t.Fatal("expected nil reporter from nil context")
	}
}

func TestBuildPlanProgressReporterFromContext_MissingReporter(t *testing.T) {
	reporter := BuildPlanProgressReporterFromContext(context.Background())
	if reporter != nil {
		t.Fatal("expected nil reporter when none stored")
	}
}

func TestBuildPlanProgressReporterFromContext_StoredReporter(t *testing.T) {
	var received BuildPlanProgress
	ctx := WithBuildPlanProgressReporter(context.Background(), func(p BuildPlanProgress) {
		received = p
	})
	reporter := BuildPlanProgressReporterFromContext(ctx)
	if reporter == nil {
		t.Fatal("expected stored reporter")
	}
	reporter(BuildPlanProgress{Step: "test_step", CurrentModule: "test_module"})
	if received.Step != "test_step" || received.CurrentModule != "test_module" {
		t.Fatalf("reporter received = %+v, want step=test_step module=test_module", received)
	}
}

func TestReportBuildPlanProgress_NoReporterStored(t *testing.T) {
	// Should not panic when no reporter is in context.
	reportBuildPlanProgress(context.Background(), BuildPlanProgress{Step: "nop"})
}

func TestReportBuildPlanProgress_WithReporter(t *testing.T) {
	var steps []string
	ctx := WithBuildPlanProgressReporter(context.Background(), func(p BuildPlanProgress) {
		steps = append(steps, p.Step)
	})
	reportBuildPlanProgress(ctx, BuildPlanProgress{Step: "resolve_dependencies"})
	reportBuildPlanProgress(ctx, BuildPlanProgress{Step: "topological_sort"})
	if len(steps) != 2 || steps[0] != "resolve_dependencies" || steps[1] != "topological_sort" {
		t.Fatalf("steps = %v, want [resolve_dependencies, topological_sort]", steps)
	}
}
