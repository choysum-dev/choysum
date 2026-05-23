// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"errors"
	"io"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func resetTaskWorkerProtoCaches() {
	taskWorkerMu.Lock()
	taskWorkerCache = map[string]protoreflect.MethodDescriptor{}
	taskWorkerErrs = map[string]error{}
	taskWorkerMu.Unlock()
}

func TestTaskWorkerProtoPathDefaultsAndTrims(t *testing.T) {
	if got := taskWorkerProtoPath(" auth "); got != "taskworker/auth.proto" {
		t.Fatalf("taskWorkerProtoPath(trimmed) = %q, want taskworker/auth.proto", got)
	}
	if got := taskWorkerProtoPath(""); got != "taskworker/taskworker.proto" {
		t.Fatalf("taskWorkerProtoPath(default) = %q, want taskworker/taskworker.proto", got)
	}
}

func TestTaskWorkerResolverFindFileByPath(t *testing.T) {
	resolver := &taskWorkerResolver{path: "taskworker/auth.proto", content: "syntax = \"proto3\";"}

	result, err := resolver.FindFileByPath("taskworker/auth.proto")
	if err != nil {
		t.Fatalf("FindFileByPath() error = %v", err)
	}
	body, err := io.ReadAll(result.Source)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if string(body) != "syntax = \"proto3\";" {
		t.Fatalf("resolver content = %q, want syntax decl", string(body))
	}

	if _, err := resolver.FindFileByPath("taskworker/missing.proto"); err == nil {
		t.Fatal("expected missing proto error")
	}
}

func TestTaskWorkerMethodBuildsAndCachesDescriptor(t *testing.T) {
	resetTaskWorkerProtoCaches()
	t.Cleanup(resetTaskWorkerProtoCaches)

	methodDesc, err := taskWorkerMethod(" auth ")
	if err != nil {
		t.Fatalf("taskWorkerMethod() error = %v", err)
	}
	if got := string(methodDesc.FullName()); got != "auth.TaskWorker.ExecuteJob" {
		t.Fatalf("method full name = %q, want auth.TaskWorker.ExecuteJob", got)
	}

	taskWorkerMu.Lock()
	_, cached := taskWorkerCache["auth"]
	taskWorkerMu.Unlock()
	if !cached {
		t.Fatal("expected trimmed app name to be cached")
	}

	defaultDesc, err := taskWorkerMethod("")
	if err != nil {
		t.Fatalf("taskWorkerMethod(default) error = %v", err)
	}
	if got := string(defaultDesc.FullName()); got != "taskworker.TaskWorker.ExecuteJob" {
		t.Fatalf("default method full name = %q, want taskworker.TaskWorker.ExecuteJob", got)
	}
}

func TestTaskWorkerStoreErrorClearsCacheAndPersistsError(t *testing.T) {
	resetTaskWorkerProtoCaches()
	t.Cleanup(resetTaskWorkerProtoCaches)

	methodDesc, err := taskWorkerMethod("auth")
	if err != nil {
		t.Fatalf("taskWorkerMethod() error = %v", err)
	}
	if methodDesc == nil {
		t.Fatal("expected descriptor before storing error")
	}

	sentinel := errors.New("boom")
	if _, err := taskWorkerStoreError("auth", sentinel); !errors.Is(err, sentinel) {
		t.Fatalf("taskWorkerStoreError() error = %v, want sentinel", err)
	}

	taskWorkerMu.Lock()
	_, cached := taskWorkerCache["auth"]
	storedErr := taskWorkerErrs["auth"]
	taskWorkerMu.Unlock()
	if cached {
		t.Fatal("expected cached descriptor to be cleared after store error")
	}
	if !errors.Is(storedErr, sentinel) {
		t.Fatalf("stored error = %v, want sentinel", storedErr)
	}

	if _, err := taskWorkerMethod("auth"); !errors.Is(err, sentinel) {
		t.Fatalf("taskWorkerMethod() cached error = %v, want sentinel", err)
	}
}
