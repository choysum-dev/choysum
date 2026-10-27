// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	logutil "github.com/choysum-dev/choysum/internal/logger"
	"github.com/choysum-dev/choysum/internal/module/artifact/pipeline"
	"github.com/choysum-dev/choysum/internal/module/artifact/staging"
	"github.com/choysum-dev/choysum/internal/module/evolution/hooks"
	"github.com/choysum-dev/choysum/internal/module/evolution/scripts"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	moduleplan "github.com/choysum-dev/choysum/internal/module/plan"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
	"gorm.io/gorm"
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

func TestModuleOperationPlanDebugAttrsIncludesOpsDetails(t *testing.T) {
	t.Parallel()
	attrs := attrsToMap(t, moduleOperationPlanDebugAttrs(moduleplan.Plan{
		ModuleOrder:         []string{"partner"},
		EnsureOrder:         []string{"auth", "web"},
		AffectedApps:        []string{"partner"},
		NeedsGlobalWebBuild: true,
	}))
	if got := attrs["ensure_count"]; got != 2 {
		t.Fatalf("ensure_count=%#v, want 2", got)
	}
	if got := attrs["needs_global_web_build"]; got != true {
		t.Fatalf("needs_global_web_build=%#v, want true", got)
	}
}

func TestModuleOperationPlanAndCompletedMessages(t *testing.T) {
	t.Parallel()
	if got := moduleOperationPlanMessage(moduleplan.OpInstall); got != "installing" {
		t.Fatalf("install plan msg=%q", got)
	}
	if got := moduleOperationCompletedMessage(moduleplan.OpUpgrade); got != "upgraded" {
		t.Fatalf("upgrade completed msg=%q", got)
	}
	if got := moduleOperationCompletedMessage(moduleplan.OpUninstall); got != "uninstalled" {
		t.Fatalf("uninstall completed msg=%q", got)
	}
}

func TestModuleOperationCompletedInfoAttrsOnlyDuration(t *testing.T) {
	t.Parallel()
	attrs := attrsToMap(t, moduleOperationCompletedInfoAttrs(24591*time.Millisecond))

	if got := attrs["duration_ms"]; got != int64(24591) {
		t.Fatalf("duration_ms = %#v, want 24591", got)
	}
	if len(attrs) != 1 {
		t.Fatalf("completed Info attrs must only include duration_ms, got %#v", attrs)
	}
}

func TestModuleUserLoggerOmitsOpID(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	base := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	moduleUserLogger(base, "web").Info("upgrading")
	logs := buf.String()
	if !strings.Contains(logs, `"module":"web"`) {
		t.Fatalf("expected module attr, got %q", logs)
	}
	if strings.Contains(logs, `"opid"`) || strings.Contains(logs, `"op"`) {
		t.Fatalf("user logger must not carry opid/op, got %q", logs)
	}
}

func TestModuleOperationStepInfoAttrsIncludesStepDurationAndVersions(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	attrs := attrsToMap(t, moduleOperationStepInfoAttrs("   ", 2*time.Second))

	if got := attrs["duration_ms"]; got != int64(2000) {
		t.Fatalf("duration_ms = %#v, want 2000", got)
	}
	if _, ok := attrs["step"]; ok {
		t.Fatalf("expected blank step to be omitted, got %#v", attrs["step"])
	}
}

func TestModuleOperationStepMessageByOperation(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

func TestLogModuleOperationStep_HookSubstepAttrs(t *testing.T) {
	t.Parallel()
	var logBuf bytes.Buffer
	testScope := &testLogScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug})),
	}

	step := moduleStepHook(hooks.PhasePreUpgrade)
	// Attribute coverage only; level selection is covered by TestModuleOperationStepLogLevel
	// with fixed durations (wall-clock here can still cross the soft Info floor under load).
	logModuleOperationStep(testScope, &opContext{opid: "op-hook-1"}, moduleplan.OpUpgrade, "partner", step, time.Now())

	logs := logBuf.String()
	for _, want := range []string{
		`"msg":"module upgrade step completed"`,
		`"opid":"op-hook-1"`,
		`"step":"hook.pre_upgrade"`,
		`"duration_ms":`,
	} {
		if !strings.Contains(logs, want) {
			t.Fatalf("expected logs to contain %q, got %q", want, logs)
		}
	}
}

func TestModuleOperationStepLogLevel(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		step     string
		duration time.Duration
		extra    []any
		want     slog.Level
	}{
		{name: "build always info", step: moduleStepBuild, duration: 0, want: slog.LevelInfo},
		{name: "schema always info", step: moduleStepSchema, duration: time.Millisecond, want: slog.LevelInfo},
		{name: "web_build always info", step: moduleStepWebBuild, duration: 0, want: slog.LevelInfo},
		{name: "hook skipped debug", step: "hook.pre_upgrade", duration: time.Second, extra: []any{"skipped", true, "reason", "same_version"}, want: slog.LevelDebug},
		{name: "scripts skipped debug", step: "scripts.validate", duration: time.Second, extra: []any{"skipped", true}, want: slog.LevelDebug},
		{name: "hook fast debug", step: "hook.pre_init", duration: 40 * time.Millisecond, want: slog.LevelDebug},
		{name: "hook threshold info", step: "hook.pre_init", duration: moduleOperationStepInfoMinDuration, want: slog.LevelInfo},
		{name: "scripts slow info", step: "scripts.pre", duration: 150 * time.Millisecond, want: slog.LevelInfo},
		{name: "origin_resolve fast debug", step: moduleStepOriginResolve, duration: 0, want: slog.LevelDebug},
		{name: "origin_resolve slow info", step: moduleStepOriginResolve, duration: 120 * time.Millisecond, want: slog.LevelInfo},
		{name: "base_entity_migrate debug", step: moduleStepBaseEntityMigrate, duration: time.Second, want: slog.LevelDebug},
		{name: "data stays debug", step: moduleStepData, duration: time.Second, want: slog.LevelDebug},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := moduleOperationStepLogLevel(tc.step, tc.duration, tc.extra...); got != tc.want {
				t.Fatalf("level=%v want %v", got, tc.want)
			}
		})
	}
}

func TestApplyProgressChannelStepLevel(t *testing.T) {
	t.Parallel()
	base := context.Background()
	quiet := logutil.WithQuietProgress(base)

	if got := applyProgressChannelStepLevel(base, moduleStepBuild, slog.LevelInfo); got != slog.LevelInfo {
		t.Fatalf("CLI build Info = %v, want Info", got)
	}
	if got := applyProgressChannelStepLevel(quiet, moduleStepBuild, slog.LevelInfo); got != slog.LevelDebug {
		t.Fatalf("quiet build = %v, want Debug", got)
	}
	if got := applyProgressChannelStepLevel(quiet, "hook.pre_init", slog.LevelInfo); got != slog.LevelDebug {
		t.Fatalf("quiet soft Info = %v, want Debug", got)
	}
	if got := applyProgressChannelStepLevel(base, "hook.pre_init", slog.LevelInfo); got != slog.LevelInfo {
		t.Fatalf("CLI soft Info = %v, want Info", got)
	}
	if got := applyProgressChannelStepLevel(quiet, moduleStepBuild, slog.LevelDebug); got != slog.LevelDebug {
		t.Fatalf("already Debug must stay Debug, got %v", got)
	}
}

func TestLogModuleOperationStepQuietDemotesHardStep(t *testing.T) {
	t.Parallel()
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	quietCtx := logutil.WithQuietProgress(context.Background())
	testScope := &testLogScope{ctx: quietCtx, logger: logger}

	logModuleOperationStep(testScope, &opContext{opid: "op-quiet-1"}, moduleplan.OpUpgrade, "web", moduleStepBuild, time.Now())

	if logs := logBuf.String(); logs != "" {
		t.Fatalf("quiet hard step must not emit Info, got %q", logs)
	}
}

func TestLogFinalizingPhaseEndQuietDemotes(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	logFinalizingPhaseEnd(logutil.WithQuietProgress(context.Background()), logger, time.Second)
	if logs := buf.String(); logs != "" {
		t.Fatalf("quiet phase_end must not emit Info, got %q", logs)
	}
}

func TestModuleStepHookAndScriptsNames(t *testing.T) {
	t.Parallel()
	if got := moduleStepHook(hooks.PhasePreInit); got != "hook.pre_init" {
		t.Fatalf("hook=%q", got)
	}
	if got := moduleStepScripts(scripts.PhaseEnd); got != "scripts.end" {
		t.Fatalf("scripts=%q", got)
	}
}

func TestSyncModuleIndexAfterInstall_MetaIncludedTriggersSyncErrorIgnored(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	releaseLeaseWithContextFallback(newDebugTestLogScope(&bytes.Buffer{}), nil, context.Background(), "lease-resource", "owner-1", "module manager")
}

func TestWithLeaseRenewPaused(t *testing.T) {
	if err := (*ModuleManager)(nil).withLeaseRenewPaused(nil); err != nil {
		t.Fatalf("nil manager nil fn: %v", err)
	}
	if err := (*ModuleManager)(nil).withLeaseRenewPaused(func() error { return nil }); err != nil {
		t.Fatalf("nil manager: %v", err)
	}
	m := &ModuleManager{}
	if err := m.withLeaseRenewPaused(nil); err != nil {
		t.Fatalf("nil fn: %v", err)
	}
	if err := m.withLeaseRenewPaused(func() error {
		if m.pauseLeaseRenewDepth.Load() != 1 {
			t.Fatal("expected pause while fn runs")
		}
		return nil
	}); err != nil {
		t.Fatalf("paused fn: %v", err)
	}
	if m.pauseLeaseRenewDepth.Load() != 0 {
		t.Fatal("expected pause cleared after fn")
	}

	origTTL := moduleManagerLeaseTTL
	t.Cleanup(func() { moduleManagerLeaseTTL = origTTL })
	moduleManagerLeaseTTL = time.Millisecond
	// Nil runtimeScope: over-TTL pause must not panic (warn skipped).
	if err := m.withLeaseRenewPaused(func() error {
		time.Sleep(2 * time.Millisecond)
		return nil
	}); err != nil {
		t.Fatalf("over-TTL nil scope: %v", err)
	}
	var warnBuf bytes.Buffer
	m.runtimeScope = newDebugTestLogScope(&warnBuf)
	if err := m.withLeaseRenewPaused(func() error {
		time.Sleep(2 * time.Millisecond)
		return nil
	}); err != nil {
		t.Fatalf("over-TTL warn: %v", err)
	}
	if !strings.Contains(warnBuf.String(), "module manager lease renew paused longer than safe window") {
		t.Fatalf("expected over-safe-window warn, got %q", warnBuf.String())
	}
	want := errors.New("boom")
	if err := m.withLeaseRenewPaused(func() error { return want }); !errors.Is(err, want) {
		t.Fatalf("got %v want %v", err, want)
	}

	// Nested pause must keep depth until outermost returns.
	if err := m.withLeaseRenewPaused(func() error {
		return m.withLeaseRenewPaused(func() error {
			if m.pauseLeaseRenewDepth.Load() != 2 {
				t.Fatalf("nested depth=%d", m.pauseLeaseRenewDepth.Load())
			}
			return nil
		})
	}); err != nil {
		t.Fatalf("nested pause: %v", err)
	}
	if m.pauseLeaseRenewDepth.Load() != 0 {
		t.Fatal("expected nested pause cleared")
	}

	origDialect := moduleManagerDialectNameFn
	t.Cleanup(func() { moduleManagerDialectNameFn = origDialect })
	moduleManagerDialectNameFn = func(*ModuleManager) string { return "postgres" }
	if err := m.withLeaseRenewPaused(func() error {
		if m.pauseLeaseRenewDepth.Load() != 0 {
			t.Fatal("postgres must keep lease renew active")
		}
		return nil
	}); err != nil {
		t.Fatalf("postgres pause: %v", err)
	}

	if !shouldPauseLeaseRenew("") || !shouldPauseLeaseRenew("sqlite") || !shouldPauseLeaseRenew("sqlite3") {
		t.Fatal("sqlite/unknown should pause")
	}
	if shouldPauseLeaseRenew("postgres") || shouldPauseLeaseRenew("mysql") || shouldPauseLeaseRenew("sqlserver") {
		t.Fatal("non-sqlite must not pause")
	}
	if moduleManagerDialectName(nil) != "" || moduleManagerDialectName(&ModuleManager{}) != "" {
		t.Fatal("nil dialect helpers")
	}
	if moduleManagerDialectName(&ModuleManager{runtimeScope: &testLogScope{}}) != "" {
		t.Fatal("nil session dialect")
	}
	if moduleManagerDialectName(&ModuleManager{runtimeScope: &dialectNilDBScope{}}) != "" {
		t.Fatal("nil db dialect")
	}
	if moduleManagerDialectName(&ModuleManager{runtimeScope: &dialectNilDialectorScope{}}) != "" {
		t.Fatal("nil dialector dialect")
	}
	runtimeScope := newLifecycleCommitTestScope(t)
	if got := moduleManagerDialectName(&ModuleManager{runtimeScope: runtimeScope}); got != "sqlite" && got != "sqlite3" {
		t.Fatalf("sqlite dialect=%q", got)
	}
}

type dialectNilDBScope struct {
	testLogScope
}

func (s *dialectNilDBScope) Session() *scope.Session {
	return &scope.Session{}
}

type dialectNilDialectorScope struct {
	testLogScope
}

func (s *dialectNilDialectorScope) Session() *scope.Session {
	return &scope.Session{DB: &gorm.DB{}}
}

func TestRunWithLeaseRenewPaused(t *testing.T) {
	t.Parallel()
	if err := runWithLeaseRenewPaused(nil, nil); err != nil {
		t.Fatalf("nil manager nil fn: %v", err)
	}
	called := false
	if err := runWithLeaseRenewPaused(nil, func() error {
		called = true
		return nil
	}); err != nil || !called {
		t.Fatalf("nil manager fn: called=%v err=%v", called, err)
	}
	m := &ModuleManager{}
	if err := runWithLeaseRenewPaused(m, func() error {
		if m.pauseLeaseRenewDepth.Load() != 1 {
			t.Fatal("expected pause")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

type renewCountLocker struct {
	renewCalls atomic.Int32
	renewErr   error
}

func (l *renewCountLocker) Acquire(context.Context, string, string, time.Duration) error {
	return nil
}
func (l *renewCountLocker) Renew(context.Context, string, string, time.Duration) error {
	l.renewCalls.Add(1)
	return l.renewErr
}
func (l *renewCountLocker) Release(context.Context, string, string) error { return nil }

func TestRenewLeaseOnTick(t *testing.T) {
	t.Parallel()
	(*ModuleManager)(nil).renewLeaseOnTick(nil, context.Background(), "r", "o", time.Second)
	m := &ModuleManager{runtimeScope: newDebugTestLogScope(&bytes.Buffer{})}
	m.renewLeaseOnTick(nil, context.Background(), "r", "o", time.Second)

	locker := &renewCountLocker{}
	m.pauseLeaseRenewDepth.Store(1)
	m.renewLeaseOnTick(locker, context.Background(), "r", "o", time.Second)
	if locker.renewCalls.Load() != 0 {
		t.Fatalf("paused renew calls=%d", locker.renewCalls.Load())
	}
	m.pauseLeaseRenewDepth.Store(0)
	m.renewLeaseOnTick(locker, context.Background(), "r", "o", time.Second)
	if locker.renewCalls.Load() != 1 {
		t.Fatalf("renew calls=%d", locker.renewCalls.Load())
	}
	locker.renewErr = errors.New("renew boom")
	m.renewLeaseOnTick(locker, context.Background(), "r", "o", time.Second)
	if locker.renewCalls.Load() != 2 {
		t.Fatalf("renew with err calls=%d", locker.renewCalls.Load())
	}

	// Commit holds leaseRenewMu: renew ticks must TryLock-fail without calling Renew.
	m.leaseRenewMu.Lock()
	before := locker.renewCalls.Load()
	m.renewLeaseOnTick(locker, context.Background(), "r", "o", time.Second)
	m.leaseRenewMu.Unlock()
	if locker.renewCalls.Load() != before {
		t.Fatalf("renew during pause lock calls=%d want %d", locker.renewCalls.Load(), before)
	}
}

func TestWithModuleManagerLease_RespectsRenewPause(t *testing.T) {
	origTTL, origEvery := moduleManagerLeaseTTL, moduleManagerLeaseRenewEvery
	moduleManagerLeaseTTL = 2 * time.Second
	moduleManagerLeaseRenewEvery = func(time.Duration) time.Duration { return 10 * time.Millisecond }
	t.Cleanup(func() {
		moduleManagerLeaseTTL = origTTL
		moduleManagerLeaseRenewEvery = origEvery
	})

	runtimeScope := newLifecycleCommitTestScope(t)
	locker := &renewCountLocker{}
	m := &ModuleManager{
		runtimeScope:  runtimeScope,
		lockerFactory: func(scope.Scope) statepkg.Locker { return locker },
		entities:      modmeta.CatalogEntities(),
	}

	if err := m.withModuleManagerLease(context.Background(), func() error {
		if err := m.withLeaseRenewPaused(func() error {
			time.Sleep(25 * time.Millisecond)
			return nil
		}); err != nil {
			return err
		}
		deadline := time.Now().Add(2 * time.Second)
		for locker.renewCalls.Load() < 1 && time.Now().Before(deadline) {
			time.Sleep(2 * time.Millisecond)
		}
		return nil
	}); err != nil {
		t.Fatalf("lease: %v", err)
	}
	if locker.renewCalls.Load() < 1 {
		t.Fatalf("expected at least one renew after pause cleared, got %d", locker.renewCalls.Load())
	}
}

func TestModuleManagerLeaseRenewInterval(t *testing.T) {
	orig := moduleManagerLeaseRenewEvery
	t.Cleanup(func() { moduleManagerLeaseRenewEvery = orig })
	moduleManagerLeaseRenewEvery = func(time.Duration) time.Duration { return 0 }
	if got := moduleManagerLeaseRenewInterval(time.Minute); got != time.Second {
		t.Fatalf("zero interval clamp=%v", got)
	}
	moduleManagerLeaseRenewEvery = func(time.Duration) time.Duration { return -time.Millisecond }
	if got := moduleManagerLeaseRenewInterval(time.Minute); got != time.Second {
		t.Fatalf("negative interval clamp=%v", got)
	}
	moduleManagerLeaseRenewEvery = func(ttl time.Duration) time.Duration { return ttl / 2 }
	if got := moduleManagerLeaseRenewInterval(20 * time.Millisecond); got != 10*time.Millisecond {
		t.Fatalf("positive interval=%v", got)
	}
}

func TestReleaseLeaseWithContextFallback_PrimarySuccessNoFallback(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

func TestModuleStepSpinnerMessage(t *testing.T) {
	t.Parallel()
	if got := moduleStepSpinnerMessage("auth", moduleStepSchema); got != "auth: updating database schema" {
		t.Fatalf("schema=%q", got)
	}
	if got := moduleStepSpinnerMessage("auth", moduleStepHook(hooks.PhasePostUpgrade)); got != "auth: running post-upgrade hooks" {
		t.Fatalf("post_upgrade=%q", got)
	}
	if got := moduleStepSpinnerMessage("partner", moduleStepHook(hooks.PhasePreUninstall)); got != "partner: running pre-uninstall hooks" {
		t.Fatalf("pre_uninstall=%q", got)
	}
	if got := moduleStepSpinnerMessage("partner", moduleStepCleanup); got != "partner: cleaning module data" {
		t.Fatalf("cleanup=%q", got)
	}
	if got := moduleStepSpinnerMessage("web", moduleStepPrepare); got != "" {
		t.Fatalf("micro-step must be empty, got %q", got)
	}
	if got := moduleStepSpinnerMessage("  ", moduleStepSchema); got != "unknown: updating database schema" {
		t.Fatalf("empty module=%q", got)
	}
}

func TestBindOpStepSpinner(t *testing.T) {
	t.Parallel()
	var got string
	opCtx := newOpContext()
	bindOpStepSpinner(opCtx, func(stage, message string) {
		got = message
	})
	opCtx.reportStepStart("sandbox", moduleStepSchema)
	if got != "sandbox: updating database schema" {
		t.Fatalf("got %q", got)
	}
	bindOpStepSpinner(nil, func(stage, message string) {})
	bindOpStepSpinner(opCtx, nil)
}

func TestModuleUserLoggerFallbacks(t *testing.T) {
	t.Parallel()
	if got := moduleUserLogger(nil, ""); got == nil {
		t.Fatal("expected default logger")
	}
	if got := moduleUserLogger(nil, "  "); got == nil {
		t.Fatal("expected default logger for blank module")
	}
	base := slog.New(slog.NewTextHandler(io.Discard, nil))
	if got := moduleUserLogger(base, "auth"); got == nil {
		t.Fatal("expected with-module logger")
	}
}

func TestInstallHeartbeatStageLabel(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"planning":                 "Planning module operation",
		"pipeline.app_stage":       "Preparing app builds",
		"pipeline.app_build":       "Building app backend",
		"pipeline.app_generate":    "Generating app APIs",
		"pipeline.bundles_build":   "Building shared backend bundles",
		"pipeline.web_build":       "Building shared web assets",
		"finalizing.phase_end":     "Finalizing",
		"finalizing.index_refresh": "Refreshing module index",
		"finalizing.index_sync":    "Syncing module index",
		"module.step":              "Module step",
		"":                         "",
		"custom_stage.detail":      "Custom stage detail",
	}
	for in, want := range cases {
		if got := installHeartbeatStageLabel(in); got != want {
			t.Fatalf("installHeartbeatStageLabel(%q)=%q want %q", in, got, want)
		}
	}
}

func TestHandleUninstallModuleProgress(t *testing.T) {
	t.Parallel()
	stages := map[string]string{}
	set := func(stage, message string) { stages[stage] = message }

	if !handleUninstallModuleProgress(pipeline.ProgressEvent{
		Stage: pipeline.ProgressStageModuleUninstallStarted, Module: "partner", Current: 1, Total: 2,
	}, set) {
		t.Fatal("expected handled")
	}
	if stages["uninstalling.modules"] != "partner: uninstalling module (1/2)" {
		t.Fatalf("got %q", stages["uninstalling.modules"])
	}
	if !handleUninstallModuleProgress(pipeline.ProgressEvent{
		Stage: pipeline.ProgressStageModuleUninstallStarted, Module: "partner",
	}, set) {
		t.Fatal("expected handled")
	}
	if stages["uninstalling.modules"] != "partner: uninstalling module" {
		t.Fatalf("got %q", stages["uninstalling.modules"])
	}
	if !handleUninstallModuleProgress(pipeline.ProgressEvent{
		Stage: pipeline.ProgressStageModuleUninstallFailed, Module: "partner", Current: 1, Total: 2,
	}, set) {
		t.Fatal("expected handled")
	}
	if stages["uninstalling.modules"] != "partner: module uninstall failed (1/2)" {
		t.Fatalf("got %q", stages["uninstalling.modules"])
	}
	if !handleUninstallModuleProgress(pipeline.ProgressEvent{
		Stage: pipeline.ProgressStageModuleUninstallFailed, Module: "  ",
	}, set) {
		t.Fatal("expected handled")
	}
	if stages["uninstalling.modules"] != "unknown: module uninstall failed" {
		t.Fatalf("got %q", stages["uninstalling.modules"])
	}
	if handleUninstallModuleProgress(pipeline.ProgressEvent{Stage: pipeline.ProgressStageWebBuildStarted}, set) {
		t.Fatal("expected unhandled")
	}
}

func TestModuleInstallProgressSpinnerMessage(t *testing.T) {
	t.Parallel()
	got := moduleInstallProgressSpinnerMessage(pipeline.ModuleInstallProgress{
		Module:  "auth",
		Current: 2,
		Total:   5,
		Stage:   pipeline.ModuleInstallProgressStageStarted,
	}, 5)
	if got != "auth: installing module (2/5)" {
		t.Fatalf("started=%q", got)
	}
	got = moduleInstallProgressSpinnerMessage(pipeline.ModuleInstallProgress{
		Module:  "auth",
		Current: 2,
		Total:   5,
		Stage:   pipeline.ModuleInstallProgressStageFailed,
	}, 5)
	if got != "auth: module install failed (2/5)" {
		t.Fatalf("failed=%q", got)
	}
	got = moduleInstallProgressSpinnerMessage(pipeline.ModuleInstallProgress{
		Stage:   pipeline.ModuleInstallProgressStageFailed,
		Current: 1,
		Total:   1,
	}, 1)
	if got != "unknown: module install failed (1/1)" {
		t.Fatalf("empty module=%q", got)
	}
}

func TestRestoreJSExecutorScripts(t *testing.T) {
	t.Parallel()
	restoreJSExecutorScripts(nil, nil, nil)

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))
	exec := &failingReloadScriptExecutor{err: errors.New("reload boom")}
	restoreJSExecutorScripts(logger, exec, nil)
	if !strings.Contains(buf.String(), "failed to restore js executor scripts") {
		t.Fatalf("expected warn log, got %q", buf.String())
	}
	if !exec.setCalled {
		t.Fatal("expected SetJsScripts")
	}
}

type failingReloadScriptExecutor struct {
	err       error
	setCalled bool
}

func (e *failingReloadScriptExecutor) Execute(context.Context, *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	return &jsengine.JsResponse{}, nil
}
func (e *failingReloadScriptExecutor) GetJsScripts() []*jsengine.JsScript { return nil }
func (e *failingReloadScriptExecutor) SetJsScripts([]*jsengine.JsScript)  { e.setCalled = true }
func (e *failingReloadScriptExecutor) Reload(...*jsengine.JsScript) error { return e.err }

func TestHandleUpgradeEnsureProgress(t *testing.T) {
	t.Parallel()
	stages := make(map[string]string)
	setSpinnerStage := func(stage, message string) {
		stages[stage] = message
	}
	clearStages := func() {
		for k := range stages {
			delete(stages, k)
		}
	}

	t.Run("ensure started with progress", func(t *testing.T) {
		clearStages()
		ok := handleUpgradeEnsureProgress(pipeline.ProgressEvent{
			Stage:   pipeline.ProgressStageModuleInstallStarted,
			Module:  "web",
			Current: 1,
			Total:   2,
		}, setSpinnerStage)
		if !ok {
			t.Fatal("expected handled")
		}
		if msg := stages["upgrading.ensure"]; msg != "web: installing required module (1/2)" {
			t.Fatalf("message=%q", msg)
		}
	})

	t.Run("ensure started without progress uses unknown module", func(t *testing.T) {
		clearStages()
		ok := handleUpgradeEnsureProgress(pipeline.ProgressEvent{
			Stage: pipeline.ProgressStageModuleInstallStarted,
		}, setSpinnerStage)
		if !ok {
			t.Fatal("expected handled")
		}
		if msg := stages["upgrading.ensure"]; msg != "unknown: installing required module" {
			t.Fatalf("message=%q", msg)
		}
	})

	t.Run("ensure failed with and without progress", func(t *testing.T) {
		clearStages()
		ok := handleUpgradeEnsureProgress(pipeline.ProgressEvent{
			Stage:   pipeline.ProgressStageModuleInstallFailed,
			Module:  "auth",
			Current: 2,
			Total:   3,
		}, setSpinnerStage)
		if !ok || stages["upgrading.ensure"] != "auth: failed installing required module (2/3)" {
			t.Fatalf("ok=%v msg=%q", ok, stages["upgrading.ensure"])
		}
		clearStages()
		ok = handleUpgradeEnsureProgress(pipeline.ProgressEvent{
			Stage:  pipeline.ProgressStageModuleInstallFailed,
			Module: "auth",
		}, setSpinnerStage)
		if !ok || stages["upgrading.ensure"] != "auth: failed installing required module" {
			t.Fatalf("ok=%v msg=%q", ok, stages["upgrading.ensure"])
		}
	})

	t.Run("upgrade started/failed branches", func(t *testing.T) {
		clearStages()
		_ = handleUpgradeEnsureProgress(pipeline.ProgressEvent{
			Stage:   pipeline.ProgressStageModuleUpgradeStarted,
			Module:  "partner",
			Current: 1,
			Total:   1,
		}, setSpinnerStage)
		if stages["upgrading.modules"] != "partner: upgrading module (1/1)" {
			t.Fatalf("message=%q", stages["upgrading.modules"])
		}
		clearStages()
		_ = handleUpgradeEnsureProgress(pipeline.ProgressEvent{
			Stage:  pipeline.ProgressStageModuleUpgradeStarted,
			Module: "partner",
		}, setSpinnerStage)
		if stages["upgrading.modules"] != "partner: upgrading module" {
			t.Fatalf("message=%q", stages["upgrading.modules"])
		}
		clearStages()
		_ = handleUpgradeEnsureProgress(pipeline.ProgressEvent{
			Stage:   pipeline.ProgressStageModuleUpgradeFailed,
			Module:  "partner",
			Current: 1,
			Total:   2,
		}, setSpinnerStage)
		if stages["upgrading.modules"] != "partner: module upgrade failed (1/2)" {
			t.Fatalf("message=%q", stages["upgrading.modules"])
		}
		clearStages()
		_ = handleUpgradeEnsureProgress(pipeline.ProgressEvent{
			Stage:  pipeline.ProgressStageModuleUpgradeFailed,
			Module: "partner",
		}, setSpinnerStage)
		if stages["upgrading.modules"] != "partner: module upgrade failed" {
			t.Fatalf("message=%q", stages["upgrading.modules"])
		}
	})

	t.Run("unknown stage returns false", func(t *testing.T) {
		clearStages()
		ok := handleUpgradeEnsureProgress(pipeline.ProgressEvent{
			Stage: pipeline.ProgressStageWebBuildStarted,
		}, setSpinnerStage)
		if ok || len(stages) != 0 {
			t.Fatalf("ok=%v stages=%v", ok, stages)
		}
	})
}

func TestHandlePipelineSharedProgress(t *testing.T) {
	t.Parallel()
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
		}, moduleplan.OpUpgrade, "base", 5, setSpinnerStage)
		if !ok {
			t.Fatal("expected handled")
		}
		if msg := stages["pipeline.app_stage"]; msg != "base: preparing 3 app build(s)" {
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
		}, moduleplan.OpInstall, "base", 4, setSpinnerStage)
		if !ok {
			t.Fatal("expected handled")
		}
		if msg := stages["pipeline.app_stage"]; msg != "base: preparing 4 app build(s)" {
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
		}, moduleplan.OpUpgrade, "base", 3, setSpinnerStage)
		if !ok {
			t.Fatal("expected handled")
		}
		if msg := stages["pipeline.app_build"]; msg != "crm: building app backend (2/5)" {
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
		}, moduleplan.OpUpgrade, "base", 3, setSpinnerStage)
		if !ok {
			t.Fatal("expected handled")
		}
		if msg := stages["pipeline.app_build"]; msg != "crm: building app backend" {
			t.Fatalf("unexpected message: %q", msg)
		}
	})

	t.Run("app build started with empty app falls back to unknown", func(t *testing.T) {
		for k := range stages {
			delete(stages, k)
		}
		ok := handlePipelineSharedProgress(pipeline.ProgressEvent{
			Stage: pipeline.ProgressStageAppBuildStarted,
		}, moduleplan.OpUpgrade, "base", 3, setSpinnerStage)
		if !ok {
			t.Fatal("expected handled")
		}
		if msg := stages["pipeline.app_build"]; msg != "unknown: building app backend" {
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
		}, moduleplan.OpUpgrade, "base", 3, setSpinnerStage)
		if !ok {
			t.Fatal("expected handled")
		}
		if msg := stages["pipeline.app_generate"]; msg != "crm: generating app APIs (1/3)" {
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
		}, moduleplan.OpUpgrade, "base", 3, setSpinnerStage)
		if !ok {
			t.Fatal("expected handled")
		}
		if msg := stages["pipeline.app_generate"]; msg != "crm: generating app APIs" {
			t.Fatalf("unexpected message: %q", msg)
		}
	})

	t.Run("bundles build started", func(t *testing.T) {
		for k := range stages {
			delete(stages, k)
		}
		ok := handlePipelineSharedProgress(pipeline.ProgressEvent{
			Stage: pipeline.ProgressStageBundlesBuildStarted,
		}, moduleplan.OpUpgrade, "base", 7, setSpinnerStage)
		if !ok {
			t.Fatal("expected handled")
		}
		if msg := stages["pipeline.bundles_build"]; msg != "base: building shared backend bundles" {
			t.Fatalf("unexpected message: %q", msg)
		}
	})

	t.Run("web build started", func(t *testing.T) {
		for k := range stages {
			delete(stages, k)
		}
		ok := handlePipelineSharedProgress(pipeline.ProgressEvent{
			Stage: pipeline.ProgressStageWebBuildStarted,
		}, moduleplan.OpUpgrade, "base", 2, setSpinnerStage)
		if !ok {
			t.Fatal("expected handled")
		}
		if msg := stages["pipeline.web_build"]; msg != "base: building shared web assets" {
			t.Fatalf("unexpected message: %q", msg)
		}
	})

	t.Run("unknown stage returns false", func(t *testing.T) {
		for k := range stages {
			delete(stages, k)
		}
		ok := handlePipelineSharedProgress(pipeline.ProgressEvent{
			Stage: pipeline.ProgressStageAppStageCompleted,
		}, moduleplan.OpUpgrade, "base", 3, setSpinnerStage)
		if ok {
			t.Fatal("expected unhandled for non-shared stage")
		}
		if len(stages) != 0 {
			t.Fatalf("expected no stage updates, got %#v", stages)
		}
	})

	t.Run("uninstall shared stages use rebuild lexicon", func(t *testing.T) {
		cases := []struct {
			name  string
			event pipeline.ProgressEvent
			root  string
			apps  int
			stage string
			want  string
		}{
			{
				name:  "app stage",
				event: pipeline.ProgressEvent{Stage: pipeline.ProgressStageAppStageStarted, Total: 2},
				root:  "partner",
				apps:  2,
				stage: "pipeline.app_stage",
				want:  "partner: updating 2 app(s)",
			},
			{
				name: "app build with progress",
				event: pipeline.ProgressEvent{
					Stage:   pipeline.ProgressStageAppBuildStarted,
					App:     "crm",
					Current: 1,
					Total:   2,
				},
				root:  "partner",
				apps:  2,
				stage: "pipeline.app_build",
				want:  "crm: rebuilding app backend (1/2)",
			},
			{
				name:  "app build without progress",
				event: pipeline.ProgressEvent{Stage: pipeline.ProgressStageAppBuildStarted, App: "crm"},
				root:  "partner",
				apps:  2,
				stage: "pipeline.app_build",
				want:  "crm: rebuilding app backend",
			},
			{
				name: "app generate with progress",
				event: pipeline.ProgressEvent{
					Stage:   pipeline.ProgressStageAppGenerateStarted,
					App:     "crm",
					Current: 1,
					Total:   2,
				},
				root:  "partner",
				apps:  2,
				stage: "pipeline.app_generate",
				want:  "crm: regenerating app APIs (1/2)",
			},
			{
				name:  "app generate without progress",
				event: pipeline.ProgressEvent{Stage: pipeline.ProgressStageAppGenerateStarted, App: "crm"},
				root:  "partner",
				apps:  2,
				stage: "pipeline.app_generate",
				want:  "crm: regenerating app APIs",
			},
			{
				name:  "bundles",
				event: pipeline.ProgressEvent{Stage: pipeline.ProgressStageBundlesBuildStarted},
				root:  "partner",
				apps:  2,
				stage: "pipeline.bundles_build",
				want:  "partner: rebuilding shared backend bundles",
			},
			{
				name:  "web",
				event: pipeline.ProgressEvent{Stage: pipeline.ProgressStageWebBuildStarted},
				root:  "partner",
				apps:  2,
				stage: "pipeline.web_build",
				want:  "partner: rebuilding shared web assets",
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				for k := range stages {
					delete(stages, k)
				}
				ok := handlePipelineSharedProgress(tc.event, moduleplan.OpUninstall, tc.root, tc.apps, setSpinnerStage)
				if !ok {
					t.Fatal("expected handled")
				}
				if msg := stages[tc.stage]; msg != tc.want {
					t.Fatalf("unexpected message: %q want %q", msg, tc.want)
				}
			})
		}
	})
}
