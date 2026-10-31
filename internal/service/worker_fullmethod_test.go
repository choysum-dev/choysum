// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/pkg/jsengine"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestIsWorkerOnlyFullMethod(t *testing.T) {
	t.Parallel()
	if !isWorkerOnlyFullMethod("document.AttachmentContent/RunGarbageCollection") {
		t.Fatal("expected document GC routing key")
	}
	if isWorkerOnlyFullMethod("document.AttachmentContent/PrepareUpload") {
		t.Fatal("interactive method must not be worker-only")
	}
}

func TestWorkerJsServicePath(t *testing.T) {
	t.Parallel()
	got, err := workerJsServicePath("document.AttachmentContent/RunGarbageCollection")
	if err != nil || got != "document.AttachmentContent.RunGarbageCollection" {
		t.Fatalf("got %q err=%v", got, err)
	}
	_, err = workerJsServicePath("  ")
	if err == nil || err.Error() != "empty full method" {
		t.Fatalf("empty full method err=%v", err)
	}
}

func TestWorkerPayloadArgs(t *testing.T) {
	t.Parallel()
	gc := "document.AttachmentContent/RunGarbageCollection"
	if args := workerPayloadArgs(gc, nil); args != nil {
		t.Fatalf("nil payload: %#v", args)
	}
	if args := workerPayloadArgs(gc, map[string]interface{}{"nowISO": "2020-01-01T00:00:00Z"}); len(args) != 1 || args[0] != "2020-01-01T00:00:00Z" {
		t.Fatalf("nowISO: %#v", args)
	}
	if args := workerPayloadArgs(gc, map[string]interface{}{"NowISO": "2020-01-02T00:00:00Z"}); len(args) != 1 || args[0] != "2020-01-02T00:00:00Z" {
		t.Fatalf("NowISO: %#v", args)
	}
	payload := map[string]interface{}{"nowISO": 42, "other": "x"}
	mapArgs := workerPayloadArgs(gc, payload)
	if len(mapArgs) != 1 {
		t.Fatalf("non-string nowISO: %#v", mapArgs)
	}
	if m, ok := mapArgs[0].(map[string]interface{}); !ok || m["other"] != "x" {
		t.Fatalf("expected map payload, got %#v", mapArgs[0])
	}
	// Non-GC worker methods must keep the full payload map even with nowISO.
	other := workerPayloadArgs("document.AttachmentContent/Other", map[string]interface{}{"nowISO": "2020-01-01T00:00:00Z"})
	if len(other) != 1 {
		t.Fatalf("non-GC args: %#v", other)
	}
	if m, ok := other[0].(map[string]interface{}); !ok || m["nowISO"] != "2020-01-01T00:00:00Z" {
		t.Fatalf("non-GC expected map payload, got %#v", other[0])
	}
}

type workerOnlyExec struct {
	last    *jsengine.JsRequest
	lastCtx context.Context
}

func (e *workerOnlyExec) Execute(ctx context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	e.last = req
	e.lastCtx = ctx
	return &jsengine.JsResponse{Result: map[string]any{"ok": true}}, nil
}
func (e *workerOnlyExec) GetJsScripts() []*jsengine.JsScript         { return nil }
func (e *workerOnlyExec) SetJsScripts(scripts []*jsengine.JsScript)  {}
func (e *workerOnlyExec) Reload(scripts ...*jsengine.JsScript) error { return nil }

func TestInvokeTargetMethodWorkerOnlyBypassesDescriptor(t *testing.T) {
	t.Parallel()
	exec := &workerOnlyExec{}
	rt := invocationRuntime{jsExecutor: exec}
	out, err := rt.invokeTargetMethod(context.Background(), nil, map[string]interface{}{}, nil, &executeJobRequest{
		FullMethod: "document.AttachmentContent/RunGarbageCollection",
		Payload:    map[string]interface{}{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if exec.last == nil || exec.last.Service != "document.AttachmentContent.RunGarbageCollection" {
		t.Fatalf("last request = %#v", exec.last)
	}
	m, _ := out.(map[string]any)
	if m["ok"] != true {
		t.Fatalf("out = %#v", out)
	}
}

type workerExecErr struct{}

func (workerExecErr) Execute(context.Context, *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	return nil, errors.New("execute failed")
}
func (workerExecErr) GetJsScripts() []*jsengine.JsScript         { return nil }
func (workerExecErr) SetJsScripts(scripts []*jsengine.JsScript)  {}
func (workerExecErr) Reload(scripts ...*jsengine.JsScript) error { return nil }

type workerExecNilResp struct{}

func (workerExecNilResp) Execute(context.Context, *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	return nil, nil
}
func (workerExecNilResp) GetJsScripts() []*jsengine.JsScript         { return nil }
func (workerExecNilResp) SetJsScripts(scripts []*jsengine.JsScript)  {}
func (workerExecNilResp) Reload(scripts ...*jsengine.JsScript) error { return nil }

func TestInvokeWorkerOnlyFullMethodEdgeCases(t *testing.T) {
	t.Parallel()
	req := &executeJobRequest{
		FullMethod: "document.AttachmentContent/RunGarbageCollection",
		Payload:    map[string]interface{}{},
	}
	rt := invocationRuntime{jsExecutor: workerExecErr{}}
	_, err := rt.invokeWorkerOnlyFullMethod(context.Background(), nil, nil, nil, req)
	if err == nil || err.Error() != "execute failed" {
		t.Fatalf("Execute error = %v", err)
	}

	rt = invocationRuntime{jsExecutor: workerExecNilResp{}}
	out, err := rt.invokeWorkerOnlyFullMethod(context.Background(), nil, nil, nil, req)
	if err != nil || out != nil {
		t.Fatalf("nil response out=%#v err=%v", out, err)
	}

	_, err = rt.invokeWorkerOnlyFullMethod(context.Background(), nil, nil, nil, &executeJobRequest{FullMethod: " "})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("empty method code=%v err=%v", status.Code(err), err)
	}

	timeoutExec := &workerOnlyExec{}
	rt = invocationRuntime{jsExecutor: timeoutExec}
	parent, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	_, err = rt.invokeWorkerOnlyFullMethod(parent, nil, nil, nil, &executeJobRequest{
		FullMethod: req.FullMethod,
		TimeoutMs:  50,
	})
	if err != nil {
		t.Fatalf("timeout invoke: %v", err)
	}
	childDeadline, ok := timeoutExec.lastCtx.Deadline()
	if !ok {
		t.Fatal("expected shorter worker deadline under long parent")
	}
	if rem := time.Until(childDeadline); rem > 200*time.Millisecond {
		t.Fatalf("worker deadline too long under TimeoutMs=50: remaining=%v", rem)
	}
	parentDeadline, ok := parent.Deadline()
	if !ok || time.Until(parentDeadline) < 30*time.Second {
		t.Fatal("expected parent minute deadline unchanged")
	}

	// Parent already shorter than TimeoutMs: keep parent deadline.
	shortParent, shortCancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer shortCancel()
	wantDeadline, _ := shortParent.Deadline()
	_, err = rt.invokeWorkerOnlyFullMethod(shortParent, nil, nil, nil, &executeJobRequest{
		FullMethod: req.FullMethod,
		TimeoutMs:  5000,
	})
	if err != nil {
		t.Fatalf("short parent invoke: %v", err)
	}
	gotDeadline, ok := timeoutExec.lastCtx.Deadline()
	if !ok || !gotDeadline.Equal(wantDeadline) {
		t.Fatalf("expected parent deadline kept when shorter, got %v want %v", gotDeadline, wantDeadline)
	}

	_, err = rt.invokeWorkerOnlyFullMethod(context.Background(), nil, nil, nil, &executeJobRequest{
		FullMethod: req.FullMethod,
		TimeoutMs:  50,
	})
	if err != nil {
		t.Fatalf("child timeout invoke: %v", err)
	}
}
