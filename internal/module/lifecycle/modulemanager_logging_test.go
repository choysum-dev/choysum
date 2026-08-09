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

	"github.com/choysum-dev/choysum/internal/module/artifact/pipeline"
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
		logger: slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug})),
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
		logger: slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug})),
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
				logger: slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug})),
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

type releaseSequenceLocker struct {
	releaseErrs     []error
	releaseCalls    int
	releaseTimeouts []time.Duration
}

func (l *releaseSequenceLocker) Acquire(context.Context, string, string, time.Duration) error {
	return nil
}

func (l *releaseSequenceLocker) Renew(context.Context, string, string, time.Duration) error {
	return nil
}

func (l *releaseSequenceLocker) Release(ctx context.Context, _, _ string) error {
	l.releaseCalls++
	if deadline, ok := ctx.Deadline(); ok {
		l.releaseTimeouts = append(l.releaseTimeouts, time.Until(deadline))
	} else {
		l.releaseTimeouts = append(l.releaseTimeouts, 0)
	}

	if len(l.releaseErrs) == 0 {
		return nil
	}
	idx := l.releaseCalls - 1
	if idx >= len(l.releaseErrs) {
		idx = len(l.releaseErrs) - 1
	}
	return l.releaseErrs[idx]
}

func newDebugTestLogScope(buf *bytes.Buffer) *testLogScope {
	return &testLogScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})),
	}
}

func TestReleaseLeaseWithContextFallback_NilLockerNoop(t *testing.T) {
	releaseLeaseWithContextFallback(newDebugTestLogScope(&bytes.Buffer{}), nil, context.Background(), "lease-resource", "owner-1", "module manager")
}

func TestReleaseLeaseWithContextFallback_PrimarySuccessNoFallback(t *testing.T) {
	locker := &releaseSequenceLocker{releaseErrs: []error{nil}}
	releaseLeaseWithContextFallback(newDebugTestLogScope(&bytes.Buffer{}), locker, context.Background(), "lease-resource", "owner-1", "module manager")

	if locker.releaseCalls != 1 {
		t.Fatalf("release call count = %d, want 1", locker.releaseCalls)
	}
	if len(locker.releaseTimeouts) != 1 || locker.releaseTimeouts[0] <= 0 || locker.releaseTimeouts[0] > 5*time.Second {
		t.Fatalf("primary release timeout = %#v, want within (0, 5s]", locker.releaseTimeouts)
	}
}

func TestReleaseLeaseWithContextFallback_PrimaryNotOwnerSkipsFallback(t *testing.T) {
	var logBuf bytes.Buffer
	locker := &releaseSequenceLocker{releaseErrs: []error{statepkg.ErrLeaseNotOwner}}
	releaseLeaseWithContextFallback(newDebugTestLogScope(&logBuf), locker, context.Background(), "lease-resource", "owner-1", "module manager")

	if locker.releaseCalls != 1 {
		t.Fatalf("release call count = %d, want 1", locker.releaseCalls)
	}
	if len(locker.releaseTimeouts) != 1 || locker.releaseTimeouts[0] <= 0 || locker.releaseTimeouts[0] > 5*time.Second {
		t.Fatalf("primary release timeout = %#v, want within (0, 5s]", locker.releaseTimeouts)
	}

	logs := logBuf.String()
	if !strings.Contains(logs, `"msg":"module manager lease release failed"`) {
		t.Fatalf("expected warn log for non-owner release failure, got %q", logs)
	}
	if strings.Contains(logs, `"msg":"module manager lease release retry with background context"`) {
		t.Fatalf("did not expect retry debug log for non-owner error, got %q", logs)
	}
}

func TestReleaseLeaseWithContextFallback_CanceledOperationContextSkipsPrimary(t *testing.T) {
	opCtx, cancel := context.WithCancel(context.Background())
	cancel()

	locker := &releaseSequenceLocker{releaseErrs: []error{nil}}
	releaseLeaseWithContextFallback(newDebugTestLogScope(&bytes.Buffer{}), locker, opCtx, "lease-resource", "owner-1", "module manager")

	if locker.releaseCalls != 1 {
		t.Fatalf("release call count = %d, want 1", locker.releaseCalls)
	}
	if len(locker.releaseTimeouts) != 1 || locker.releaseTimeouts[0] < 20*time.Second {
		t.Fatalf("fallback release timeout = %#v, want >= 20s", locker.releaseTimeouts)
	}
}

func TestReleaseLeaseWithContextFallback_PrimaryErrorFallsBackAndWarns(t *testing.T) {
	var logBuf bytes.Buffer
	locker := &releaseSequenceLocker{releaseErrs: []error{errors.New("primary release failed"), errors.New("background release failed")}}
	releaseLeaseWithContextFallback(newDebugTestLogScope(&logBuf), locker, context.Background(), "lease-resource", "owner-1", "module manager")

	if locker.releaseCalls != 2 {
		t.Fatalf("release call count = %d, want 2", locker.releaseCalls)
	}
	if len(locker.releaseTimeouts) != 2 {
		t.Fatalf("release timeout count = %d, want 2", len(locker.releaseTimeouts))
	}
	if locker.releaseTimeouts[0] <= 0 || locker.releaseTimeouts[0] > 5*time.Second {
		t.Fatalf("primary release timeout = %v, want within (0, 5s]", locker.releaseTimeouts[0])
	}
	if locker.releaseTimeouts[1] < 20*time.Second {
		t.Fatalf("fallback release timeout = %v, want >= 20s", locker.releaseTimeouts[1])
	}

	logs := logBuf.String()
	if !strings.Contains(logs, `"msg":"module manager lease release retry with background context"`) {
		t.Fatalf("expected retry debug log, got %q", logs)
	}
	if !strings.Contains(logs, `"msg":"module manager lease release failed"`) {
		t.Fatalf("expected fallback warn log, got %q", logs)
	}
	if !strings.Contains(logs, "background release failed") {
		t.Fatalf("expected fallback error detail in logs, got %q", logs)
	}
}

func TestReleaseLeaseWithContextFallback_FallbackExpectedErrorsSkipWarn(t *testing.T) {
	tests := []struct {
		name        string
		fallbackErr error
	}{
		{name: "not-owner", fallbackErr: statepkg.ErrLeaseNotOwner},
		{name: "not-held", fallbackErr: statepkg.ErrLeaseNotHeld},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logBuf bytes.Buffer
			locker := &releaseSequenceLocker{releaseErrs: []error{errors.New("primary release failed"), tt.fallbackErr}}
			releaseLeaseWithContextFallback(newDebugTestLogScope(&logBuf), locker, context.Background(), "lease-resource", "owner-1", "module manager")

			if locker.releaseCalls != 2 {
				t.Fatalf("release call count = %d, want 2", locker.releaseCalls)
			}

			logs := logBuf.String()
			if !strings.Contains(logs, `"msg":"module manager lease release retry with background context"`) {
				t.Fatalf("expected retry debug log, got %q", logs)
			}
			if strings.Contains(logs, `"msg":"module manager lease release failed"`) {
				t.Fatalf("did not expect warn log for expected fallback errors, got %q", logs)
			}
		})
	}
}

func TestHandleUpgradeEnsureProgress(t *testing.T) {
	stages := make(map[string]string)
	setSpinnerStage := func(stage, message string) {
		stages[stage] = message
	}
	clear := func() {
		for k := range stages {
			delete(stages, k)
		}
	}

	t.Run("ensure started with progress", func(t *testing.T) {
		clear()
		ok := handleUpgradeEnsureProgress(pipeline.ProgressEvent{
			Stage:   pipeline.ProgressStageModuleInstallStarted,
			Module:  "web",
			Current: 1,
			Total:   2,
		}, setSpinnerStage)
		if !ok {
			t.Fatal("expected handled")
		}
		if msg := stages["upgrading.ensure"]; msg != "web: ensuring modules (1/2)" {
			t.Fatalf("message=%q", msg)
		}
	})

	t.Run("ensure started without progress uses unknown module", func(t *testing.T) {
		clear()
		ok := handleUpgradeEnsureProgress(pipeline.ProgressEvent{
			Stage: pipeline.ProgressStageModuleInstallStarted,
		}, setSpinnerStage)
		if !ok {
			t.Fatal("expected handled")
		}
		if msg := stages["upgrading.ensure"]; msg != "unknown: ensuring module" {
			t.Fatalf("message=%q", msg)
		}
	})

	t.Run("ensure failed with and without progress", func(t *testing.T) {
		clear()
		ok := handleUpgradeEnsureProgress(pipeline.ProgressEvent{
			Stage:   pipeline.ProgressStageModuleInstallFailed,
			Module:  "auth",
			Current: 2,
			Total:   3,
		}, setSpinnerStage)
		if !ok || stages["upgrading.ensure"] != "auth: failed ensuring module (2/3)" {
			t.Fatalf("ok=%v msg=%q", ok, stages["upgrading.ensure"])
		}
		clear()
		ok = handleUpgradeEnsureProgress(pipeline.ProgressEvent{
			Stage:  pipeline.ProgressStageModuleInstallFailed,
			Module: "auth",
		}, setSpinnerStage)
		if !ok || stages["upgrading.ensure"] != "auth: failed ensuring module" {
			t.Fatalf("ok=%v msg=%q", ok, stages["upgrading.ensure"])
		}
	})

	t.Run("upgrade started/failed branches", func(t *testing.T) {
		clear()
		_ = handleUpgradeEnsureProgress(pipeline.ProgressEvent{
			Stage:   pipeline.ProgressStageModuleUpgradeStarted,
			Module:  "partner",
			Current: 1,
			Total:   1,
		}, setSpinnerStage)
		if stages["upgrading.modules"] != "partner: upgrading modules (1/1)" {
			t.Fatalf("message=%q", stages["upgrading.modules"])
		}
		clear()
		_ = handleUpgradeEnsureProgress(pipeline.ProgressEvent{
			Stage:  pipeline.ProgressStageModuleUpgradeStarted,
			Module: "partner",
		}, setSpinnerStage)
		if stages["upgrading.modules"] != "partner: upgrading module" {
			t.Fatalf("message=%q", stages["upgrading.modules"])
		}
		clear()
		_ = handleUpgradeEnsureProgress(pipeline.ProgressEvent{
			Stage:   pipeline.ProgressStageModuleUpgradeFailed,
			Module:  "partner",
			Current: 1,
			Total:   2,
		}, setSpinnerStage)
		if stages["upgrading.modules"] != "partner: failed upgrading module (1/2)" {
			t.Fatalf("message=%q", stages["upgrading.modules"])
		}
		clear()
		_ = handleUpgradeEnsureProgress(pipeline.ProgressEvent{
			Stage:  pipeline.ProgressStageModuleUpgradeFailed,
			Module: "partner",
		}, setSpinnerStage)
		if stages["upgrading.modules"] != "partner: failed upgrading module" {
			t.Fatalf("message=%q", stages["upgrading.modules"])
		}
	})

	t.Run("unknown stage returns false", func(t *testing.T) {
		clear()
		ok := handleUpgradeEnsureProgress(pipeline.ProgressEvent{
			Stage: pipeline.ProgressStageWebBuildStarted,
		}, setSpinnerStage)
		if ok || len(stages) != 0 {
			t.Fatalf("ok=%v stages=%v", ok, stages)
		}
	})
}

func TestHandlePipelineSharedProgress(t *testing.T) {
	stages := make(map[string]string)
	setSpinnerStage := func(stage, message string) {
		stages[stage] = message
	}

	t.Run("app stage started with event total", func(t *testing.T) {
		for k := range stages {
			delete(stages, k)
		}
		ok := handlePipelineSharedProgress(pipeline.ProgressEvent{
			Stage: pipeline.ProgressStageAppStageStarted,
			Total: 3,
		}, "base", 5, setSpinnerStage)
		if !ok {
			t.Fatal("expected handled")
		}
		if msg := stages["pipeline.app_stage"]; !strings.Contains(msg, "base: building application artifacts (3 apps)") {
			t.Fatalf("unexpected message: %q", msg)
		}
	})

	t.Run("app stage started fallback to affectedAppsCount", func(t *testing.T) {
		for k := range stages {
			delete(stages, k)
		}
		ok := handlePipelineSharedProgress(pipeline.ProgressEvent{
			Stage: pipeline.ProgressStageAppStageStarted,
			Total: 0,
		}, "base", 4, setSpinnerStage)
		if !ok {
			t.Fatal("expected handled")
		}
		if msg := stages["pipeline.app_stage"]; !strings.Contains(msg, "4 apps") {
			t.Fatalf("unexpected message: %q", msg)
		}
	})

	t.Run("app build started with progress", func(t *testing.T) {
		for k := range stages {
			delete(stages, k)
		}
		ok := handlePipelineSharedProgress(pipeline.ProgressEvent{
			Stage:   pipeline.ProgressStageAppBuildStarted,
			App:     "crm",
			Current: 2,
			Total:   5,
		}, "base", 3, setSpinnerStage)
		if !ok {
			t.Fatal("expected handled")
		}
		if msg := stages["pipeline.app_build"]; !strings.Contains(msg, "crm: building backend app artifacts (2/5)") {
			t.Fatalf("unexpected message: %q", msg)
		}
	})

	t.Run("app build started without progress", func(t *testing.T) {
		for k := range stages {
			delete(stages, k)
		}
		ok := handlePipelineSharedProgress(pipeline.ProgressEvent{
			Stage: pipeline.ProgressStageAppBuildStarted,
			App:   "crm",
		}, "base", 3, setSpinnerStage)
		if !ok {
			t.Fatal("expected handled")
		}
		if msg := stages["pipeline.app_build"]; msg != "crm: building backend app artifacts" {
			t.Fatalf("unexpected message: %q", msg)
		}
	})

	t.Run("app build started with empty app falls back to unknown", func(t *testing.T) {
		for k := range stages {
			delete(stages, k)
		}
		ok := handlePipelineSharedProgress(pipeline.ProgressEvent{
			Stage: pipeline.ProgressStageAppBuildStarted,
		}, "base", 3, setSpinnerStage)
		if !ok {
			t.Fatal("expected handled")
		}
		if msg := stages["pipeline.app_build"]; !strings.Contains(msg, "unknown: building backend app artifacts") {
			t.Fatalf("unexpected message: %q", msg)
		}
	})

	t.Run("app generate started with progress", func(t *testing.T) {
		for k := range stages {
			delete(stages, k)
		}
		ok := handlePipelineSharedProgress(pipeline.ProgressEvent{
			Stage:   pipeline.ProgressStageAppGenerateStarted,
			App:     "crm",
			Current: 1,
			Total:   3,
		}, "base", 3, setSpinnerStage)
		if !ok {
			t.Fatal("expected handled")
		}
		if msg := stages["pipeline.app_generate"]; !strings.Contains(msg, "crm: generating app modules (1/3)") {
			t.Fatalf("unexpected message: %q", msg)
		}
	})

	t.Run("app generate started without progress", func(t *testing.T) {
		for k := range stages {
			delete(stages, k)
		}
		ok := handlePipelineSharedProgress(pipeline.ProgressEvent{
			Stage: pipeline.ProgressStageAppGenerateStarted,
			App:   "crm",
		}, "base", 3, setSpinnerStage)
		if !ok {
			t.Fatal("expected handled")
		}
		if msg := stages["pipeline.app_generate"]; msg != "crm: generating app modules" {
			t.Fatalf("unexpected message: %q", msg)
		}
	})

	t.Run("bundles build started", func(t *testing.T) {
		for k := range stages {
			delete(stages, k)
		}
		ok := handlePipelineSharedProgress(pipeline.ProgressEvent{
			Stage: pipeline.ProgressStageBundlesBuildStarted,
		}, "base", 7, setSpinnerStage)
		if !ok {
			t.Fatal("expected handled")
		}
		if msg := stages["pipeline.bundles_build"]; !strings.Contains(msg, "apps=7") {
			t.Fatalf("unexpected message: %q", msg)
		}
	})

	t.Run("web build started", func(t *testing.T) {
		for k := range stages {
			delete(stages, k)
		}
		ok := handlePipelineSharedProgress(pipeline.ProgressEvent{
			Stage: pipeline.ProgressStageWebBuildStarted,
		}, "base", 2, setSpinnerStage)
		if !ok {
			t.Fatal("expected handled")
		}
		if msg := stages["pipeline.web_build"]; !strings.Contains(msg, "apps=2") {
			t.Fatalf("unexpected message: %q", msg)
		}
	})

	t.Run("unknown stage returns false", func(t *testing.T) {
		for k := range stages {
			delete(stages, k)
		}
		ok := handlePipelineSharedProgress(pipeline.ProgressEvent{
			Stage: pipeline.ProgressStageAppStageCompleted,
		}, "base", 3, setSpinnerStage)
		if ok {
			t.Fatal("expected unhandled for non-shared stage")
		}
		if len(stages) != 0 {
			t.Fatalf("expected no stage updates, got %#v", stages)
		}
	})
}
