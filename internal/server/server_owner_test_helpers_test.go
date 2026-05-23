// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"reflect"
	"testing"

	"github.com/choysum-dev/choysum/internal/server/runplan"
	"github.com/choysum-dev/choysum/internal/task"
	taskcontract "github.com/choysum-dev/choysum/pkg/task"
)

type taskRuntimeTestSnapshot struct {
	Dispatcher       *task.Dispatcher
	Scheduler        *task.Scheduler
	GarbageCollector taskcontract.GarbageCollector
}

type transportRuntimeStateTestSnapshot struct {
	HTTPServer     bool
	Listener       bool
	GRPCServer     bool
	GRPCClientPool bool
}

func (s *GRPCWebServer) taskRuntimeTestSnapshot() taskRuntimeTestSnapshot {
	return taskRuntimeTestSnapshot{
		Dispatcher:       s.taskRuntime.dispatcher,
		Scheduler:        s.taskRuntime.scheduler,
		GarbageCollector: s.taskRuntime.garbageCollector,
	}
}

func assertTaskRuntimeStopped(t *testing.T, srv *GRPCWebServer, context string) {
	t.Helper()
	got := srv.taskRuntimeTestSnapshot()
	if got.Dispatcher != nil || got.Scheduler != nil || got.GarbageCollector != nil {
		t.Fatalf("%s: expected task runtime to be stopped, got %#v", context, got)
	}
}

func assertTaskRuntimeStarted(t *testing.T, srv *GRPCWebServer, context string) taskRuntimeTestSnapshot {
	t.Helper()
	got := srv.taskRuntimeTestSnapshot()
	if got.Dispatcher == nil || got.Scheduler == nil || got.GarbageCollector == nil {
		t.Fatalf("%s: expected task runtime to be started, got %#v", context, got)
	}
	return got
}

func assertTaskRuntimeSnapshot(t *testing.T, got taskRuntimeTestSnapshot, want taskRuntimeTestSnapshot, context string) {
	t.Helper()
	if got.Dispatcher != want.Dispatcher || got.Scheduler != want.Scheduler || got.GarbageCollector != want.GarbageCollector {
		t.Fatalf("%s: task runtime snapshot = %#v, want %#v", context, got, want)
	}
}

func (s *GRPCWebServer) transportRuntimeStateTestSnapshot() transportRuntimeStateTestSnapshot {
	return transportRuntimeStateTestSnapshot{
		HTTPServer:     s.httpServer != nil,
		Listener:       s.listener != nil,
		GRPCServer:     s.server != nil,
		GRPCClientPool: s.grpcClientPool != nil,
	}
}

func assertServerReadyState(t *testing.T, srv *GRPCWebServer, want bool, context string) {
	t.Helper()
	if got := srv.ready.Load(); got != want {
		t.Fatalf("%s: server ready = %v, want %v", context, got, want)
	}
}

func assertTransportRuntimeState(t *testing.T, srv *GRPCWebServer, want transportRuntimeStateTestSnapshot, context string) {
	t.Helper()
	if got := srv.transportRuntimeStateTestSnapshot(); got != want {
		t.Fatalf("%s: transport runtime state = %#v, want %#v", context, got, want)
	}
}

func stopTaskRuntimeForTest(srv *GRPCWebServer) {
	if srv == nil {
		return
	}
	srv.taskRuntime.stop()
}

func restoreRunStateForTest(srv *GRPCWebServer, snapshot runStateSnapshot) {
	if srv == nil {
		return
	}
	srv.runState.restore(snapshot)
}

func assertRunStateMode(t *testing.T, srv *GRPCWebServer, want runplan.RunMode, context string) {
	t.Helper()
	if got := srv.runState.mode(); got != want {
		t.Fatalf("%s: run mode = %q, want %q", context, got, want)
	}
}

func assertRunStateTargets(t *testing.T, srv *GRPCWebServer, want []string, context string) {
	t.Helper()
	if got := srv.runState.serviceTargets(); !reflect.DeepEqual(got, want) {
		t.Fatalf("%s: run state targets = %#v, want %#v", context, got, want)
	}
}

func assertRunStateSnapshot(t *testing.T, srv *GRPCWebServer, want runStateSnapshot, context string) {
	t.Helper()
	got := srv.runState.snapshot()
	if got.distManifest != want.distManifest {
		t.Fatalf("%s: dist manifest = %#v, want %#v", context, got.distManifest, want.distManifest)
	}
	if got.runMode != want.runMode {
		t.Fatalf("%s: run mode = %q, want %q", context, got.runMode, want.runMode)
	}
	if got.runModeReason != want.runModeReason {
		t.Fatalf("%s: run mode reason = %q, want %q", context, got.runModeReason, want.runModeReason)
	}
	if got.compileBundleMode != want.compileBundleMode {
		t.Fatalf("%s: compile bundle mode = %q, want %q", context, got.compileBundleMode, want.compileBundleMode)
	}
	if !reflect.DeepEqual(got.applicationNames, want.applicationNames) {
		t.Fatalf("%s: application names = %#v, want %#v", context, got.applicationNames, want.applicationNames)
	}
}

func registrationInitScriptFilesForTest(result registrationBatchResult) []string {
	files := make([]string, 0, len(result.InitScripts))
	for _, script := range result.InitScripts {
		if script == nil {
			files = append(files, "")
			continue
		}
		files = append(files, script.FileName)
	}
	return files
}

func assertRegistrationInitScripts(t *testing.T, result registrationBatchResult, want []string, context string) {
	t.Helper()
	if got := registrationInitScriptFilesForTest(result); !reflect.DeepEqual(got, want) {
		t.Fatalf("%s: registration init scripts = %#v, want %#v", context, got, want)
	}
}

func assertRegistrationBindings(t *testing.T, result registrationBatchResult, want []registrationBinding, context string) {
	t.Helper()
	if !reflect.DeepEqual(result.Bindings, want) {
		t.Fatalf("%s: registration bindings = %#v, want %#v", context, result.Bindings, want)
	}
}

func requireSingleRegistrationBinding(t *testing.T, result registrationBatchResult, context string) registrationBinding {
	t.Helper()
	if len(result.Bindings) != 1 {
		t.Fatalf("%s: registration bindings len = %d, want 1 (%#v)", context, len(result.Bindings), result.Bindings)
	}
	return result.Bindings[0]
}

func assertRegisteredBindings(t *testing.T, srv *GRPCWebServer, want []registrationBinding, context string) {
	t.Helper()
	if got := srv.registration.registeredBindings(); !reflect.DeepEqual(got, want) {
		t.Fatalf("%s: registered bindings = %#v, want %#v", context, got, want)
	}
}

func assertRegistrationGRPCMethods(t *testing.T, result registrationBatchResult, want map[string]struct{}, context string) {
	t.Helper()
	assertRegistrationMethodSet(t, result.GRPCMethods, want, context+": registration grpc methods")
}

func assertRegisteredGRPCMethods(t *testing.T, srv *GRPCWebServer, want map[string]struct{}, context string) {
	t.Helper()
	assertRegistrationMethodSet(t, srv.registration.grpcMethodsSnapshot(), want, context+": registered grpc methods")
}

func assertRegistrationMethodSet(t *testing.T, got map[string]struct{}, want map[string]struct{}, context string) {
	t.Helper()
	if want == nil {
		if got != nil {
			t.Fatalf("%s = %#v, want nil", context, got)
		}
		return
	}
	if got == nil {
		t.Fatalf("%s = nil, want %#v", context, want)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", context, got, want)
	}
}

func recoveryDiagnosticsForTest(srv *GRPCWebServer) recoveryDiagnosticsSnapshot {
	if srv == nil {
		return recoveryDiagnosticsSnapshot{}
	}
	return srv.runtimeRecovery.snapshot()
}

func assertRecoveryActionDiagnostics(t *testing.T, snapshot recoveryDiagnosticsSnapshot, action recoveryAction, want recoveryActionDiagnostics, context string) {
	t.Helper()
	if got := snapshot.action(action); got != want {
		t.Fatalf("%s: recovery diagnostics for %q = %#v, want %#v", context, action, got, want)
	}
}

func assertRecoveryModeSwitchRollbacks(t *testing.T, snapshot recoveryDiagnosticsSnapshot, want uint64, context string) {
	t.Helper()
	if got := snapshot.ModeSwitchRollbacks; got != want {
		t.Fatalf("%s: mode-switch rollback count = %d, want %d", context, got, want)
	}
}

func assertHotreloadCounters(t *testing.T, srv *GRPCWebServer, wantDropped uint64, wantCoalesced uint64, context string) {
	t.Helper()
	gotDropped := srv.watchDroppedCount()
	gotCoalesced := srv.watchCoalescedCount()
	if gotDropped != wantDropped || gotCoalesced != wantCoalesced {
		t.Fatalf("%s: hotreload counters dropped/coalesced = %d/%d, want %d/%d", context, gotDropped, gotCoalesced, wantDropped, wantCoalesced)
	}
}

func assertStartupLifecycleStatus(t *testing.T, result startupLifecycleResult, wantSucceeded bool, wantReadyActivated bool, wantFailedPhase startupPhase, context string) {
	t.Helper()
	if result.Succeeded != wantSucceeded {
		t.Fatalf("%s: startup succeeded = %v, want %v", context, result.Succeeded, wantSucceeded)
	}
	if result.ReadyActivated != wantReadyActivated {
		t.Fatalf("%s: startup ready activated = %v, want %v", context, result.ReadyActivated, wantReadyActivated)
	}
	if result.FailedPhase != wantFailedPhase {
		t.Fatalf("%s: startup failed phase = %q, want %q", context, result.FailedPhase, wantFailedPhase)
	}
}

func assertAuthRuntimeSetupResult(t *testing.T, got authRuntimeSetupResult, wantEnabled bool, wantConfigured bool, wantDegraded bool, wantErr bool, context string) {
	t.Helper()
	if got.Enabled != wantEnabled || got.Configured != wantConfigured || got.Degraded != wantDegraded {
		t.Fatalf("%s: auth runtime setup result = %#v, want enabled=%v configured=%v degraded=%v", context, got, wantEnabled, wantConfigured, wantDegraded)
	}
	if (got.Err != nil) != wantErr {
		t.Fatalf("%s: auth runtime error presence = %v, want %v (%v)", context, got.Err != nil, wantErr, got.Err)
	}
}

func assertModeRuntimeSummaryFields(t *testing.T, got modeRuntimeStartupSummary, wantMode runplan.RunMode, wantServices int, wantInitScripts int, wantJSPrepared bool, wantJSActivated bool, context string) {
	t.Helper()
	if got.Mode != wantMode {
		t.Fatalf("%s: mode runtime mode = %q, want %q", context, got.Mode, wantMode)
	}
	if got.RegisteredServiceCount != wantServices {
		t.Fatalf("%s: registered service count = %d, want %d", context, got.RegisteredServiceCount, wantServices)
	}
	if got.InitScriptCount != wantInitScripts {
		t.Fatalf("%s: init script count = %d, want %d", context, got.InitScriptCount, wantInitScripts)
	}
	if got.JSRuntimePrepared != wantJSPrepared || got.JSRuntimeActivated != wantJSActivated {
		t.Fatalf("%s: mode runtime summary = %#v, want js prepared=%v activated=%v", context, got, wantJSPrepared, wantJSActivated)
	}
}

func assertTaskRuntimeSummaryFields(t *testing.T, got taskRuntimeStartupSummary, wantRequested bool, wantStarted bool, wantDispatcherStarted bool, wantSchedulerStarted bool, wantGarbageCollectorStarted bool, context string) {
	t.Helper()
	if got.Requested != wantRequested || got.Started != wantStarted || got.DispatcherStarted != wantDispatcherStarted || got.SchedulerStarted != wantSchedulerStarted || got.GarbageCollectorStarted != wantGarbageCollectorStarted {
		t.Fatalf("%s: task runtime summary = %#v, want requested=%v started=%v dispatcher=%v scheduler=%v gc=%v", context, got, wantRequested, wantStarted, wantDispatcherStarted, wantSchedulerStarted, wantGarbageCollectorStarted)
	}
}

func assertStartupCleanupState(t *testing.T, result startupLifecycleResult, wantAction recoveryAction, wantPlan startupRecoveryPlan, wantReport startupRecoveryReport, context string) {
	t.Helper()
	if result.Cleanup.Action != wantAction {
		t.Fatalf("%s: cleanup action = %q, want %q", context, result.Cleanup.Action, wantAction)
	}
	if result.CleanupPlan != wantPlan {
		t.Fatalf("%s: cleanup plan = %#v, want %#v", context, result.CleanupPlan, wantPlan)
	}
	if result.Cleanup.StartupCleanup != wantReport {
		t.Fatalf("%s: cleanup report = %#v, want %#v", context, result.Cleanup.StartupCleanup, wantReport)
	}
}
