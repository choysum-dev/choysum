// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/module/artifact/staging"
	moduleplan "github.com/choysum-dev/choysum/internal/module/plan"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
)

type testLogScope struct {
	ctx    context.Context
	logger *slog.Logger
}

func (s *testLogScope) Run(fn func(scope.Scope) error) error {
	if fn == nil {
		return nil
	}
	return fn(s)
}

func (s *testLogScope) Session() *scope.Session {
	return nil
}

func (s *testLogScope) Transactor() scope.Transactor {
	return nil
}

func (s *testLogScope) WithContext(ctx context.Context) scope.Scope {
	clone := *s
	clone.ctx = ctx
	return &clone
}

func (s *testLogScope) Context() context.Context {
	return s.ctx
}

func (s *testLogScope) Logger() *slog.Logger {
	return s.logger
}

func attrsToMap(t *testing.T, attrs []any) map[string]any {
	t.Helper()
	if len(attrs)%2 != 0 {
		t.Fatalf("expected even attr list, got %#v", attrs)
	}
	result := make(map[string]any, len(attrs)/2)
	for i := 0; i < len(attrs); i += 2 {
		key, ok := attrs[i].(string)
		if !ok {
			t.Fatalf("expected string key at index %d, got %#v", i, attrs[i])
		}
		result[key] = attrs[i+1]
	}
	return result
}

func TestModuleOperationPlanInfoAttrsIncludesSmallNameLists(t *testing.T) {
	attrs := attrsToMap(t, moduleOperationPlanInfoAttrs(moduleplan.Plan{
		ModuleOrder:         []string{"core"},
		AffectedApps:        []string{"auth", "base", "task"},
		NeedsGlobalWebBuild: true,
	}))

	if got := attrs["modules_count"]; got != 1 {
		t.Fatalf("modules_count = %#v, want 1", got)
	}
	if got := attrs["apps_count"]; got != 3 {
		t.Fatalf("apps_count = %#v, want 3", got)
	}
	if got := attrs["needs_global_web_build"]; got != true {
		t.Fatalf("needs_global_web_build = %#v, want true", got)
	}
	modules, ok := attrs["modules"].([]string)
	if !ok || len(modules) != 1 || modules[0] != "core" {
		t.Fatalf("modules = %#v, want [core]", attrs["modules"])
	}
	apps, ok := attrs["apps"].([]string)
	if !ok || len(apps) != 3 || apps[0] != "auth" || apps[1] != "base" || apps[2] != "task" {
		t.Fatalf("apps = %#v, want [auth base task]", attrs["apps"])
	}
}

func TestModuleOperationPlanInfoAttrsOmitsLargeOrAmbiguousNameLists(t *testing.T) {
	attrs := attrsToMap(t, moduleOperationPlanInfoAttrs(moduleplan.Plan{
		ModuleOrder:  []string{"m1", "m2", "m3", "m4", "m5", "m6", "m7", "m8", "m9"},
		AffectedApps: []string{"auth", "", "auth"},
	}))

	if _, ok := attrs["modules"]; ok {
		t.Fatalf("expected large module list to stay out of info attrs, got %#v", attrs["modules"])
	}
	if _, ok := attrs["apps"]; ok {
		t.Fatalf("expected ambiguous app list to stay out of info attrs, got %#v", attrs["apps"])
	}
	if got := attrs["modules_count"]; got != 9 {
		t.Fatalf("modules_count = %#v, want 9", got)
	}
	if got := attrs["apps_count"]; got != 3 {
		t.Fatalf("apps_count = %#v, want 3", got)
	}
}

func TestModuleOperationCompletedInfoAttrsIncludesSmallNameLists(t *testing.T) {
	attrs := attrsToMap(t, moduleOperationCompletedInfoAttrs(moduleplan.Plan{
		ModuleOrder:  []string{"core"},
		AffectedApps: []string{"auth", "base", "task"},
	}, 24591*time.Millisecond))

	if got := attrs["duration_ms"]; got != int64(24591) {
		t.Fatalf("duration_ms = %#v, want 24591", got)
	}
	if got := attrs["modules_count"]; got != 1 {
		t.Fatalf("modules_count = %#v, want 1", got)
	}
	if got := attrs["apps_count"]; got != 3 {
		t.Fatalf("apps_count = %#v, want 3", got)
	}
	modules, ok := attrs["modules"].([]string)
	if !ok || len(modules) != 1 || modules[0] != "core" {
		t.Fatalf("modules = %#v, want [core]", attrs["modules"])
	}
	apps, ok := attrs["apps"].([]string)
	if !ok || len(apps) != 3 || apps[0] != "auth" || apps[1] != "base" || apps[2] != "task" {
		t.Fatalf("apps = %#v, want [auth base task]", attrs["apps"])
	}
}

func TestModuleOperationCompletedInfoAttrsOmitsLargeOrAmbiguousNameLists(t *testing.T) {
	attrs := attrsToMap(t, moduleOperationCompletedInfoAttrs(moduleplan.Plan{
		ModuleOrder:  []string{"m1", "m2", "m3", "m4", "m5", "m6", "m7", "m8", "m9"},
		AffectedApps: []string{"auth", "", "auth"},
	}, 2*time.Second))

	if got := attrs["duration_ms"]; got != int64(2000) {
		t.Fatalf("duration_ms = %#v, want 2000", got)
	}
	if _, ok := attrs["modules"]; ok {
		t.Fatalf("expected large module list to stay out of completed info attrs, got %#v", attrs["modules"])
	}
	if _, ok := attrs["apps"]; ok {
		t.Fatalf("expected ambiguous app list to stay out of completed info attrs, got %#v", attrs["apps"])
	}
	if got := attrs["modules_count"]; got != 9 {
		t.Fatalf("modules_count = %#v, want 9", got)
	}
	if got := attrs["apps_count"]; got != 3 {
		t.Fatalf("apps_count = %#v, want 3", got)
	}
}

func TestModuleOperationStepInfoAttrsIncludesStepDurationAndVersions(t *testing.T) {
	attrs := attrsToMap(t, moduleOperationStepInfoAttrs(" build ", 24591*time.Millisecond, "from_version", "v0.1.0", "to_version", "v0.2.0"))

	if got := attrs["duration_ms"]; got != int64(24591) {
		t.Fatalf("duration_ms = %#v, want 24591", got)
	}
	if got := attrs["step"]; got != "build" {
		t.Fatalf("step = %#v, want build", got)
	}
	if got := attrs["from_version"]; got != "v0.1.0" {
		t.Fatalf("from_version = %#v, want v0.1.0", got)
	}
	if got := attrs["to_version"]; got != "v0.2.0" {
		t.Fatalf("to_version = %#v, want v0.2.0", got)
	}
}

func TestModuleOperationStepInfoAttrsOmitsBlankStep(t *testing.T) {
	attrs := attrsToMap(t, moduleOperationStepInfoAttrs("   ", 2*time.Second))

	if got := attrs["duration_ms"]; got != int64(2000) {
		t.Fatalf("duration_ms = %#v, want 2000", got)
	}
	if _, ok := attrs["step"]; ok {
		t.Fatalf("expected blank step to be omitted, got %#v", attrs["step"])
	}
}

func TestModuleOperationStepMessageByOperation(t *testing.T) {
	tests := []struct {
		name string
		op   moduleplan.OpType
		want string
	}{
		{name: "install", op: moduleplan.OpInstall, want: "module install step completed"},
		{name: "uninstall", op: moduleplan.OpUninstall, want: "module uninstall step completed"},
		{name: "upgrade", op: moduleplan.OpUpgrade, want: "module upgrade step completed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := moduleOperationStepMessage(tt.op); got != tt.want {
				t.Fatalf("moduleOperationStepMessage(%q) = %q, want %q", tt.op, got, tt.want)
			}
		})
	}
}

func TestModuleUpgraderLogUpgradeStepIncludesOperationContext(t *testing.T) {
	var logBuf bytes.Buffer
	testScope := &testLogScope{
		ctx:    staging.WithOpID(context.Background(), "op-test-123"),
		logger: slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo})),
	}

	upgrader := &moduleUpgrader{runtimeScope: testScope}
	upgrader.logUpgradeStep("core", moduleStepData, time.Now().Add(-25*time.Millisecond), "from_version", "v0.1.0", "to_version", "v0.2.0")

	logs := logBuf.String()
	for _, want := range []string{
		`"msg":"module upgrade step completed"`,
		`"opid":"op-test-123"`,
		`"op":"upgrade"`,
		`"module":"core"`,
		`"step":"data"`,
		`"from_version":"v0.1.0"`,
		`"to_version":"v0.2.0"`,
		`"duration_ms":`,
	} {
		if !strings.Contains(logs, want) {
			t.Fatalf("expected logs to contain %q, got %q", want, logs)
		}
	}
}

func TestModuleUpgraderLogUpgradeStepPrefersOperationContextOpID(t *testing.T) {
	var logBuf bytes.Buffer
	testScope := &testLogScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo})),
	}

	upgrader := &moduleUpgrader{runtimeScope: testScope, ctx: &opContext{opid: "op-shared-456"}}
	upgrader.logUpgradeStep("core", moduleStepFinalize, time.Now().Add(-25*time.Millisecond), "from_version", "v0.1.0", "to_version", "v0.2.0")

	logs := logBuf.String()
	if !strings.Contains(logs, `"opid":"op-shared-456"`) {
		t.Fatalf("expected logs to reuse shared opid, got %q", logs)
	}
	if strings.Contains(logs, `"opid":"op-test-123"`) {
		t.Fatalf("did not expect unrelated opid in logs, got %q", logs)
	}
}

func TestLogModuleOperationStepUsesInstallAndUninstallMessages(t *testing.T) {
	tests := []struct {
		name    string
		op      moduleplan.OpType
		module  string
		step    string
		wantMsg string
	}{
		{name: "install", op: moduleplan.OpInstall, module: "base", step: moduleStepPrepare, wantMsg: `"msg":"module install step completed"`},
		{name: "uninstall", op: moduleplan.OpUninstall, module: "task", step: moduleStepCleanup, wantMsg: `"msg":"module uninstall step completed"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logBuf bytes.Buffer
			testScope := &testLogScope{
				ctx:    context.Background(),
				logger: slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo})),
			}

			logModuleOperationStep(testScope, &opContext{opid: "op-shared-789"}, tt.op, tt.module, tt.step, time.Now().Add(-25*time.Millisecond))

			logs := logBuf.String()
			for _, want := range []string{
				tt.wantMsg,
				`"opid":"op-shared-789"`,
				`"module":"` + tt.module + `"`,
				`"step":"` + tt.step + `"`,
				`"duration_ms":`,
			} {
				if !strings.Contains(logs, want) {
					t.Fatalf("expected logs to contain %q, got %q", want, logs)
				}
			}
		})
	}
}

func TestSyncModuleIndexAfterInstall_MetaIncludedTriggersSyncErrorIgnored(t *testing.T) {
	called := false
	m := &ModuleManager{
		runtimeScope: &testLogScope{ctx: context.Background(), logger: slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))},
		moduleIndexSyncLocal: func(context.Context, scope.Scope, statepkg.LockerFactory) (ModuleIndexSyncStats, error) {
			called = true
			return ModuleIndexSyncStats{}, errors.New("sync failed")
		},
	}

	if err := m.syncModuleIndexAfterInstall(context.Background(), m.runtimeScope.Logger(), []string{"core", "meta"}); err != nil {
		t.Fatalf("syncModuleIndexAfterInstall() error = %v, want nil", err)
	}
	if !called {
		t.Fatal("expected sync function to be called when module plan includes meta")
	}
}

func TestSyncModuleIndexAfterInstall_MetaNotIncludedSkipsSync(t *testing.T) {
	called := false
	m := &ModuleManager{
		runtimeScope: &testLogScope{ctx: context.Background(), logger: slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))},
		moduleIndexSyncLocal: func(context.Context, scope.Scope, statepkg.LockerFactory) (ModuleIndexSyncStats, error) {
			called = true
			return ModuleIndexSyncStats{}, nil
		},
	}

	if err := m.syncModuleIndexAfterInstall(context.Background(), m.runtimeScope.Logger(), []string{"core", "base"}); err != nil {
		t.Fatalf("syncModuleIndexAfterInstall() error = %v, want nil", err)
	}
	if called {
		t.Fatal("expected sync function to be skipped when module plan does not include meta")
	}
}

func TestSyncModuleIndexAfterInstall_UsesDefaultSyncFunctionWhenNil(t *testing.T) {
	m := &ModuleManager{
		runtimeScope: &testLogScope{ctx: context.Background(), logger: slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))},
		lockerFactory: func(scope.Scope) statepkg.Locker {
			return &moduleIndexSyncTestLocker{}
		},
	}

	if err := m.syncModuleIndexAfterInstall(context.Background(), m.runtimeScope.Logger(), []string{"meta"}); err != nil {
		t.Fatalf("syncModuleIndexAfterInstall(default sync) error = %v, want nil", err)
	}
}

func TestContainsModuleName_CaseInsensitiveAndTrimmed(t *testing.T) {
	if !containsModuleName([]string{" core ", " Meta "}, "meta") {
		t.Fatal("expected containsModuleName() to match case-insensitive trimmed target")
	}
	if containsModuleName([]string{"core", "base"}, "meta") {
		t.Fatal("did not expect containsModuleName() to match absent target")
	}
	if containsModuleName([]string{"meta"}, "   ") {
		t.Fatal("did not expect blank target to match")
	}
}
