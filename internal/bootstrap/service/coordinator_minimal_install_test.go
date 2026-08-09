// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package service

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/logger"
	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	origincontract "github.com/choysum-dev/choysum/internal/module/origin/contract"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func TestClassifyModuleInstallErrorMetaMessages(t *testing.T) {
	c, _ := newFreshnessTestCoordinator(t)
	progress := logger.NewProgressLine(io.Discard)

	err := c.classifyModuleInstallError(progress, time.Minute, context.DeadlineExceeded)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if got := bootstrapErrorCode(err); got != bootstrapErrCodeModuleInstallTimeout {
		t.Fatalf("code=%q, want timeout", got)
	}
	if !strings.Contains(err.Error(), "meta, the web shell, and their dependencies") {
		t.Fatalf("error=%q, want meta/web shell hint", err.Error())
	}

	err = c.classifyModuleInstallError(nil, time.Second, errors.New("boom"))
	if err == nil || bootstrapErrorCode(err) != bootstrapErrCodeRuntimePrepare {
		t.Fatalf("error=%v", err)
	}
}

func TestDefaultInstallMinimalModulesMarksMetaStage(t *testing.T) {
	c, _ := newFreshnessTestCoordinator(t)
	store := newMemoryStatusStore(time.Now)
	c.store = store
	opID := "op-meta-install"
	store.beginOperation(opID, "", 0)

	err := c.defaultInstallMinimalModules(context.Background(), opID)
	if err == nil {
		t.Fatal("expected install failure in test harness")
	}
	snap, ok := store.getOperation(opID)
	if !ok {
		t.Fatal("expected operation snapshot")
	}
	if !strings.Contains(snap.StageDetail, "meta module") {
		t.Fatalf("StageDetail=%q, want meta module stage text", snap.StageDetail)
	}
}

func TestApplyMinimalInstallFetchProgressEmptyModuleName(t *testing.T) {
	var details []string
	var messages []string
	markDetail := func(detail string) { details = append(details, detail) }
	setMessage := func(message string) { messages = append(messages, message) }

	applyMinimalInstallFetchProgress(markDetail, setMessage, origincontract.FetchProgressStageDownload, "  ")
	applyMinimalInstallFetchProgress(markDetail, setMessage, origincontract.FetchProgressStageVerify, "")
	applyMinimalInstallFetchProgress(markDetail, setMessage, origincontract.FetchProgressStageExtract, "partner")
	applyMinimalInstallFetchProgress(markDetail, setMessage, origincontract.FetchProgressStage("other"), "x")

	if len(details) != 3 || len(messages) != 3 {
		t.Fatalf("details=%v messages=%v, want 3 known stages", details, messages)
	}
	if !strings.Contains(details[0], "module") || strings.Contains(details[0], "core module") {
		t.Fatalf("download detail=%q, want empty name → module", details[0])
	}
	if !strings.HasPrefix(messages[0], "module:") {
		t.Fatalf("download message=%q", messages[0])
	}
	if !strings.Contains(details[1], "module") {
		t.Fatalf("verify detail=%q", details[1])
	}
	if !strings.Contains(details[2], "partner") || !strings.HasPrefix(messages[2], "partner:") {
		t.Fatalf("extract detail=%q message=%q", details[2], messages[2])
	}
}

func TestBindMinimalInstallFetchProgressReporter(t *testing.T) {
	var details []string
	var messages []string
	reporter := bindMinimalInstallFetchProgressReporter(
		func(detail string) { details = append(details, detail) },
		func(message string) { messages = append(messages, message) },
	)
	reporter(origincontract.FetchProgressStageDownload, "")
	if len(details) != 1 || !strings.Contains(details[0], "module") {
		t.Fatalf("details=%v", details)
	}
	if len(messages) != 1 || !strings.HasPrefix(messages[0], "module:") {
		t.Fatalf("messages=%v", messages)
	}
}

func TestMinimalInstallFetchDetailMarker(t *testing.T) {
	c, _ := newFreshnessTestCoordinator(t)
	store := newMemoryStatusStore(time.Now)
	c.store = store
	opID := "op-fetch-detail"
	store.beginOperation(opID, "", 0)
	marker := c.minimalInstallFetchDetailMarker(opID)
	marker("downloading module package: module...")
	snap, ok := store.getOperation(opID)
	if !ok || snap.StageDetail != "downloading module package: module..." {
		t.Fatalf("snap=%+v ok=%v", snap, ok)
	}
}

func TestDefaultInstallMinimalModulesWithFakeExecutor(t *testing.T) {
	c, _ := newFreshnessTestCoordinator(t)
	store := newMemoryStatusStore(time.Now)
	c.store = store
	opID := "op-meta-ok"
	store.beginOperation(opID, "", 0)

	prevExec := newMinimalInstallExecutor
	prevInstall := installMinimalModulesFn
	t.Cleanup(func() {
		newMinimalInstallExecutor = prevExec
		installMinimalModulesFn = prevInstall
	})
	newMinimalInstallExecutor = func(runtimeScope scope.Scope, opts ...jsexecutor.Option) (jsexecutor.JsExecutor, error) {
		return &noopMinimalInstallExecutor{}, nil
	}
	installMinimalModulesFn = func(ctx context.Context, runtimeScope scope.Scope, jsExecutor jsexecutor.ScriptExecutor, req lifecycle.InstallModuleRequest, opts ...lifecycle.Option) error {
		return nil
	}

	if err := c.defaultInstallMinimalModules(context.Background(), opID); err != nil {
		t.Fatalf("defaultInstallMinimalModules: %v", err)
	}
	snap, ok := store.getOperation(opID)
	if !ok || snap.StageDetail != "meta module installation completed" {
		t.Fatalf("StageDetail=%q ok=%v", snap.StageDetail, ok)
	}
}

func TestDefaultInstallMinimalModulesClassifiesInstallError(t *testing.T) {
	c, _ := newFreshnessTestCoordinator(t)
	store := newMemoryStatusStore(time.Now)
	c.store = store
	opID := "op-meta-fail"
	store.beginOperation(opID, "", 0)

	prevExec := newMinimalInstallExecutor
	prevInstall := installMinimalModulesFn
	t.Cleanup(func() {
		newMinimalInstallExecutor = prevExec
		installMinimalModulesFn = prevInstall
	})
	newMinimalInstallExecutor = func(runtimeScope scope.Scope, opts ...jsexecutor.Option) (jsexecutor.JsExecutor, error) {
		return &noopMinimalInstallExecutor{}, nil
	}
	installMinimalModulesFn = func(ctx context.Context, runtimeScope scope.Scope, jsExecutor jsexecutor.ScriptExecutor, req lifecycle.InstallModuleRequest, opts ...lifecycle.Option) error {
		return errors.New("meta install exploded")
	}

	err := c.defaultInstallMinimalModules(context.Background(), opID)
	if err == nil || !strings.Contains(err.Error(), "failed to install required system components") {
		t.Fatalf("err=%v", err)
	}
}

type noopMinimalInstallExecutor struct{}

func (e *noopMinimalInstallExecutor) Execute(context.Context, *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	return &jsengine.JsResponse{}, nil
}
func (e *noopMinimalInstallExecutor) GetJsScripts() []*jsengine.JsScript             { return nil }
func (e *noopMinimalInstallExecutor) SetJsScripts(scripts []*jsengine.JsScript)     {}
func (e *noopMinimalInstallExecutor) AppendJsScripts(scripts ...*jsengine.JsScript) {}
func (e *noopMinimalInstallExecutor) Reload(scripts ...*jsengine.JsScript) error    { return nil }
func (e *noopMinimalInstallExecutor) Start() error                                  { return nil }
func (e *noopMinimalInstallExecutor) Stop() error                                   { return nil }

func TestRunMinimalMetaModuleInstall(t *testing.T) {
	t.Run("success_marks_completed", func(t *testing.T) {
		var completed bool
		var sawReq lifecycle.InstallModuleRequest
		err := runMinimalMetaModuleInstall(
			context.Background(),
			nil,
			nil,
			func(ctx context.Context, runtimeScope scope.Scope, jsExecutor jsexecutor.ScriptExecutor, req lifecycle.InstallModuleRequest, opts ...lifecycle.Option) error {
				sawReq = req
				return nil
			},
			func() { completed = true },
			func(error) error { t.Fatal("classify should not run"); return nil },
		)
		if err != nil {
			t.Fatalf("err=%v", err)
		}
		if !completed {
			t.Fatal("expected markCompleted")
		}
		if sawReq.Input != "meta" || sawReq.WithDemo {
			t.Fatalf("req=%+v", sawReq)
		}
	})
	t.Run("failure_classifies", func(t *testing.T) {
		err := runMinimalMetaModuleInstall(
			context.Background(),
			nil,
			nil,
			func(ctx context.Context, runtimeScope scope.Scope, jsExecutor jsexecutor.ScriptExecutor, req lifecycle.InstallModuleRequest, opts ...lifecycle.Option) error {
				return errors.New("install boom")
			},
			func() { t.Fatal("markCompleted should not run") },
			func(installErr error) error {
				return errors.New("classified: " + installErr.Error())
			},
		)
		if err == nil || err.Error() != "classified: install boom" {
			t.Fatalf("err=%v", err)
		}
	})
}
