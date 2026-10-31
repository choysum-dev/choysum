// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package service

import (
	"context"
	"testing"

	"github.com/choysum-dev/choysum/pkg/jsengine"
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
}

func TestWorkerPayloadArgs(t *testing.T) {
	t.Parallel()
	if args := workerPayloadArgs(nil); args != nil {
		t.Fatalf("nil payload: %#v", args)
	}
	if args := workerPayloadArgs(map[string]interface{}{"nowISO": "2020-01-01T00:00:00Z"}); len(args) != 1 || args[0] != "2020-01-01T00:00:00Z" {
		t.Fatalf("nowISO: %#v", args)
	}
}

type workerOnlyExec struct {
	last *jsengine.JsRequest
}

func (e *workerOnlyExec) Execute(ctx context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	e.last = req
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
