// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/bufbuild/protocompile"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const taskWorkerProtoPathFmt = "taskworker/%s.proto"

const taskWorkerProtoTemplate = `syntax = "proto3";
package %s;
import "google/protobuf/struct.proto";

service TaskWorker {
	rpc ExecuteJob(ExecuteJobReq) returns (ExecuteJobResp);
}

message ExecuteJobReq {
	string job_id = 1;
	int32 attempt = 2;
	string full_method = 3;
	google.protobuf.Struct payload = 4;
	string scheduler_user_id = 5;
	string triggered_by_user_id = 6;
	int64 timeout_ms = 7;
}

enum ExecuteJobStatus {
	EXECUTE_JOB_STATUS_UNSPECIFIED = 0;
	EXECUTE_JOB_STATUS_SUCCEEDED = 1;
	EXECUTE_JOB_STATUS_FAILED_NON_RETRYABLE = 2;
	EXECUTE_JOB_STATUS_FAILED_RETRYABLE = 3;
	EXECUTE_JOB_STATUS_ALREADY_RUNNING = 4;
	EXECUTE_JOB_STATUS_RESOURCE_BUSY = 5;
	EXECUTE_JOB_STATUS_CANCELLED = 6;
}

message ExecuteJobError {
	int32 grpc_code = 1;
	string message = 2;
	string domain = 3;
	string code = 4;
	google.protobuf.Struct details = 5;
}

message ExecuteJobResp {
	ExecuteJobStatus status = 1;
	google.protobuf.Value result = 2;
	ExecuteJobError error = 3;
	int64 retry_after_ms = 4;
}
`

type taskWorkerResolver struct {
	path    string
	content string
}

func (r *taskWorkerResolver) FindFileByPath(path string) (protocompile.SearchResult, error) {
	if path == r.path {
		return protocompile.SearchResult{Source: strings.NewReader(r.content)}, nil
	}
	return protocompile.SearchResult{}, fmt.Errorf("proto not found: %s", path)
}

var (
	taskWorkerMu    sync.Mutex
	taskWorkerCache = map[string]protoreflect.MethodDescriptor{}
	taskWorkerErrs  = map[string]error{}
)

func taskWorkerProtoPath(appName string) string {
	name := strings.TrimSpace(appName)
	if name == "" {
		name = "taskworker"
	}
	return fmt.Sprintf(taskWorkerProtoPathFmt, name)
}

func taskWorkerMethod(appName string) (protoreflect.MethodDescriptor, error) {
	name := strings.TrimSpace(appName)
	if name == "" {
		name = "taskworker"
	}
	path := taskWorkerProtoPath(name)

	taskWorkerMu.Lock()
	if md, ok := taskWorkerCache[name]; ok {
		taskWorkerMu.Unlock()
		return md, nil
	}
	if err, ok := taskWorkerErrs[name]; ok {
		taskWorkerMu.Unlock()
		return nil, err
	}
	taskWorkerMu.Unlock()

	content := fmt.Sprintf(taskWorkerProtoTemplate, name)
	resolver := &taskWorkerResolver{path: path, content: content}
	compiler := protocompile.Compiler{Resolver: protocompile.WithStandardImports(resolver)}
	files, err := compiler.Compile(context.TODO(), path)
	if err != nil {
		return taskWorkerStoreError(name, err)
	}
	fd := files[0]
	services := fd.Services()
	if services.Len() == 0 {
		return taskWorkerStoreError(name, fmt.Errorf("task worker proto missing service"))
	}
	serviceDesc := services.Get(0)
	methods := serviceDesc.Methods()
	if methods.Len() == 0 {
		return taskWorkerStoreError(name, fmt.Errorf("task worker proto missing method"))
	}
	methodDesc := methods.Get(0)

	taskWorkerMu.Lock()
	taskWorkerCache[name] = methodDesc
	delete(taskWorkerErrs, name)
	taskWorkerMu.Unlock()
	return methodDesc, nil
}

func taskWorkerStoreError(appName string, err error) (protoreflect.MethodDescriptor, error) {
	taskWorkerMu.Lock()
	taskWorkerErrs[appName] = err
	delete(taskWorkerCache, appName)
	taskWorkerMu.Unlock()
	return nil, err
}
