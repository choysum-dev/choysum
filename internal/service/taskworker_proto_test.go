// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package service

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestTaskWorkerProtoPathAndResolver(t *testing.T) {
	if got := taskWorkerProtoPath(""); got != "taskworker/taskworker.proto" {
		t.Fatalf("taskWorkerProtoPath(\"\") = %q", got)
	}
	if got := taskWorkerProtoPath(" task "); got != "taskworker/task.proto" {
		t.Fatalf("taskWorkerProtoPath(trimmed) = %q", got)
	}

	resolver := &taskWorkerResolver{path: "taskworker/task.proto", content: "syntax = \"proto3\";"}
	result, err := resolver.FindFileByPath("taskworker/task.proto")
	if err != nil {
		t.Fatalf("FindFileByPath(match) error = %v", err)
	}
	srcBytes, readErr := io.ReadAll(result.Source)
	if readErr != nil || !strings.Contains(string(srcBytes), "syntax = \"proto3\"") {
		t.Fatalf("unexpected resolver source read result: %q err=%v", string(srcBytes), readErr)
	}
	if _, err := resolver.FindFileByPath("taskworker/missing.proto"); err == nil {
		t.Fatal("expected missing proto path to return error")
	}
}

func TestTaskWorkerDescriptorsCachesByApp(t *testing.T) {
	resetTaskWorkerProtoCaches()
	t.Cleanup(resetTaskWorkerProtoCaches)

	methodA, reqA, respA, errA, err := taskWorkerDescriptors(" task ")
	if err != nil {
		t.Fatalf("taskWorkerDescriptors(first) error = %v", err)
	}
	if string(methodA.Name()) != "ExecuteJob" || reqA.Name() != "ExecuteJobReq" || respA.Name() != "ExecuteJobResp" || errA == nil || errA.Name() != "ExecuteJobError" {
		t.Fatalf("unexpected task worker descriptors: method=%v req=%v resp=%v err=%v", methodA.Name(), reqA.Name(), respA.Name(), errA)
	}

	methodB, _, _, _, err := taskWorkerDescriptors("task")
	if err != nil {
		t.Fatalf("taskWorkerDescriptors(cached) error = %v", err)
	}
	if methodA != methodB {
		t.Fatal("expected taskWorkerDescriptors to reuse cached descriptor set for trimmed app name")
	}
}

func TestTaskWorkerStoreErrorClearsCache(t *testing.T) {
	resetTaskWorkerProtoCaches()
	t.Cleanup(resetTaskWorkerProtoCaches)
	appName := "taskworker_more_test"

	if _, _, _, _, err := taskWorkerDescriptors(appName); err != nil {
		t.Fatalf("taskWorkerDescriptors(seed) error = %v", err)
	}
	_, _, _, _, storeErr := taskWorkerStoreError(appName, errors.New("boom"))
	if storeErr == nil || storeErr.Error() != "boom" {
		t.Fatalf("unexpected taskWorkerStoreError return: %v", storeErr)
	}

	taskWorkerMu.Lock()
	defer taskWorkerMu.Unlock()
	if err := taskWorkerErrs[appName]; err == nil || err.Error() != "boom" {
		t.Fatalf("expected cached task worker error, got %#v", err)
	}
	if _, ok := taskWorkerCache[appName]; ok {
		t.Fatalf("expected task worker cache entry to be cleared")
	}
}

func resetTaskWorkerProtoCaches() {
	taskWorkerMu.Lock()
	defer taskWorkerMu.Unlock()
	taskWorkerCache = map[string]*taskWorkerDescriptorSet{}
	taskWorkerErrs = map[string]error{}
}
