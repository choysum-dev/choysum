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
	peek func(ctx context.Context, name string) (*meta.IrModule, error)
	load func(name string) (*meta.IrModule, error)
}

func (r fakeResolver) Peek(ctx context.Context, name string) (*meta.IrModule, error) {
	if r.peek == nil {
		return nil, nil
	}
	return r.peek(ctx, name)
}
func (r fakeResolver) Load(name string) (*meta.IrModule, error) {
	if r.load == nil {
		return nil, nil
	}
	return r.load(name)
}

func TestBuildPlanValidationAndGuardErrors(t *testing.T) {
	root := &meta.IrModule{Name: "auth", ApplicationStr: "auth"}

	if _, err := BuildPlan(context.Background(), OpInstall, nil, fakeResolver{}); err == nil || err.Error() != "root module is nil" {
		t.Fatalf("unexpected nil root error: %v", err)
	}
	if _, err := BuildPlan(context.Background(), OpInstall, root, nil); err == nil || err.Error() != "resolver is nil" {
		t.Fatalf("unexpected nil resolver error: %v", err)
	}
	if _, err := BuildPlan(context.Background(), OpType("broken"), root, fakeResolver{}); err == nil || err.Error() != `unknown op: "broken"` {
		t.Fatalf("unexpected unknown op error: %v", err)
	}

	plan, err := BuildPlan(nil, OpUpgrade, root, fakeResolver{})
	if err != nil {
		t.Fatalf("BuildPlan(nil ctx) error: %v", err)
	}
	if len(plan.ModuleOrder) != 1 || plan.ModuleOrder[0] != "auth" {
		t.Fatalf("unexpected module order with nil ctx: %v", plan.ModuleOrder)
	}
}

func TestBuildPlanInstallErrorsAndAppCollection(t *testing.T) {
	root := &meta.IrModule{
		Name:           "base",
		ApplicationStr: "crm",
		DependsStr:     []byte(` ["dep", "dep", " ", "webmod"] `),
	}
	peekCalls := 0
	loadCalls := 0
	r := fakeResolver{
		peek: func(ctx context.Context, name string) (*meta.IrModule, error) {
			peekCalls++
			switch name {
			case "dep":
				return &meta.IrModule{Name: "dep", ApplicationStr: "crm"}, nil
			case "webmod":
				return &meta.IrModule{Name: "webmod", ApplicationStr: "web", WebEntryPoint: "web/index.ts"}, nil
			default:
				return nil, nil
			}
		},
		load: func(name string) (*meta.IrModule, error) {
			loadCalls++
			return nil, nil
		},
	}

	plan, err := BuildPlan(context.Background(), OpInstall, root, r)
	if err != nil {
		t.Fatalf("BuildPlan() error: %v", err)
	}
	if len(plan.ModuleOrder) != 3 || plan.ModuleOrder[0] != "dep" || plan.ModuleOrder[1] != "webmod" || plan.ModuleOrder[2] != "base" {
		t.Fatalf("unexpected module order: %v", plan.ModuleOrder)
	}
	if len(plan.AffectedApps) != 1 || plan.AffectedApps[0] != "crm" {
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

func TestBuildPlanInstallDependencyErrors(t *testing.T) {
	tests := []struct {
		name string
		root *meta.IrModule
		res  fakeResolver
		want string
	}{
		{
			name: "load web module error",
			root: &meta.IrModule{Name: "auth", ApplicationStr: "auth"},
			res: fakeResolver{
				load: func(name string) (*meta.IrModule, error) {
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
			root: &meta.IrModule{Name: "auth", DependsStr: []byte(`{broken`)},
			res:  fakeResolver{},
			want: "unmarshal depends for auth",
		},
		{
			name: "load dependency error",
			root: &meta.IrModule{Name: "auth", DependsStr: []byte(` ["dep"] `)},
			res: fakeResolver{
				load: func(name string) (*meta.IrModule, error) { return nil, errors.New("load dep failed") },
			},
			want: "load dependency dep: load dep failed",
		},
		{
			name: "peek dependency error",
			root: &meta.IrModule{Name: "auth", DependsStr: []byte(` ["dep"] `)},
			res: fakeResolver{
				peek: func(ctx context.Context, name string) (*meta.IrModule, error) { return nil, errors.New("peek dep failed") },
			},
			want: "peek dependency dep: peek dep failed",
		},
		{
			name: "dependency cycle error",
			root: &meta.IrModule{Name: "auth", DependsStr: []byte(` ["dep"] `)},
			res: fakeResolver{
				peek: func(ctx context.Context, name string) (*meta.IrModule, error) {
					switch name {
					case "dep":
						return &meta.IrModule{Name: "dep", DependsStr: []byte(` ["auth"] `)}, nil
					case "auth":
						return &meta.IrModule{Name: "auth", DependsStr: []byte(` ["dep"] `)}, nil
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
	root := &meta.IrModule{Name: "auth", DependsStr: []byte(` ["dep"] `)}
	resolver := fakeResolver{
		peek: func(ctx context.Context, name string) (*meta.IrModule, error) {
			return &meta.IrModule{Name: name}, nil
		},
	}
	cancel()

	_, err := BuildPlan(ctx, OpInstall, root, resolver)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}

func TestBuildPlanUninstallLoadError(t *testing.T) {
	root := &meta.IrModule{Name: "auth", ApplicationStr: "auth"}
	_, err := BuildPlan(context.Background(), OpUninstall, root, fakeResolver{
		load: func(name string) (*meta.IrModule, error) {
			return nil, errors.New("load failed")
		},
	})
	if err == nil || err.Error() != "load module auth: load failed" {
		t.Fatalf("unexpected uninstall load error: %v", err)
	}
}

func TestBuildPlan_NeedsGlobalWebBuildFalseWithoutWebModule(t *testing.T) {
	root := &meta.IrModule{Name: "auth", ApplicationStr: "auth"}
	r := fakeResolver{
		peek: func(ctx context.Context, name string) (*meta.IrModule, error) {
			return &meta.IrModule{Name: name, ApplicationStr: name}, nil
		},
		load: func(name string) (*meta.IrModule, error) { return nil, nil },
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
	root := &meta.IrModule{Name: "web", ApplicationStr: "web", WebEntryPoint: "web/index.ts"}
	r := fakeResolver{
		peek: func(ctx context.Context, name string) (*meta.IrModule, error) {
			return &meta.IrModule{Name: name, ApplicationStr: name}, nil
		},
		load: func(name string) (*meta.IrModule, error) { return nil, nil },
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
	root := &meta.IrModule{Name: "auth", ApplicationStr: "auth"}
	r := fakeResolver{
		peek: func(ctx context.Context, name string) (*meta.IrModule, error) {
			return &meta.IrModule{Name: name, ApplicationStr: name}, nil
		},
		load: func(name string) (*meta.IrModule, error) {
			if name == "web" {
				return &meta.IrModule{Name: "web", Status: meta.Installed, WebEntryPoint: "web/index.ts"}, nil
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
	root := &meta.IrModule{Name: "auth", ApplicationStr: "auth"}
	r := fakeResolver{
		peek: func(ctx context.Context, name string) (*meta.IrModule, error) {
			return &meta.IrModule{Name: name, ApplicationStr: name}, nil
		},
		load: func(name string) (*meta.IrModule, error) { return nil, nil },
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
	root := &meta.IrModule{Name: "base", ApplicationStr: "base"}
	modules := map[string]*meta.IrModule{
		"base":     {Name: "base", ApplicationStr: "base", Dependents: []*meta.IrModule{{Name: "auth"}}},
		"auth":     {Name: "auth", ApplicationStr: "auth", Dependents: []*meta.IrModule{{Name: "auth_ext"}}},
		"auth_ext": {Name: "auth_ext", ApplicationStr: "auth"},
	}
	r := fakeResolver{
		peek: func(ctx context.Context, name string) (*meta.IrModule, error) { return nil, nil },
		load: func(name string) (*meta.IrModule, error) { return modules[name], nil },
	}

	plan, err := BuildPlan(context.Background(), OpUninstall, root, r)
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}
	if len(plan.ModuleOrder) != 3 || plan.ModuleOrder[0] != "auth_ext" || plan.ModuleOrder[1] != "auth" || plan.ModuleOrder[2] != "base" {
		t.Fatalf("unexpected uninstall order: %v", plan.ModuleOrder)
	}
	if len(plan.AffectedApps) != 2 {
		t.Fatalf("expected two affected apps, got %v", plan.AffectedApps)
	}
}

func TestBuildPlan_UninstallTreatsMissingModuleAsNoOp(t *testing.T) {
	root := &meta.IrModule{Name: "missing", ApplicationStr: "auth"}
	r := fakeResolver{
		peek: func(ctx context.Context, name string) (*meta.IrModule, error) { return nil, nil },
		load: func(name string) (*meta.IrModule, error) { return nil, nil },
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
	root := &meta.IrModule{Name: "base", ApplicationStr: "base"}
	modules := map[string]*meta.IrModule{
		"base": {Name: "base", ApplicationStr: "base", Dependents: []*meta.IrModule{{Name: "auth"}}},
		"auth": {Name: "auth", ApplicationStr: "auth", Dependents: []*meta.IrModule{{Name: "base"}}},
	}
	r := fakeResolver{
		peek: func(ctx context.Context, name string) (*meta.IrModule, error) { return nil, nil },
		load: func(name string) (*meta.IrModule, error) { return modules[name], nil },
	}

	_, err := BuildPlan(context.Background(), OpUninstall, root, r)
	if err == nil || err.Error() != "dependent cycle detected: base -> auth -> base" {
		t.Fatalf("unexpected uninstall cycle error: %v", err)
	}
}
