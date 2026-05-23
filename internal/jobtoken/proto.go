// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jobtoken

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/bufbuild/protocompile"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const jobTokenProtoPath = "auth/job_token_service.proto"

const jobTokenProtoContent = `syntax = "proto3";
package auth;
import "google/protobuf/struct.proto";

service JobTokenService {
  rpc IssueTaskJobToken(IssueTaskJobTokenReq) returns (IssueTaskJobTokenResp);
}

message IssueTaskJobTokenReq {
  string job_id = 1;
  string target_app = 2;
  string full_method = 3;
  string scheduler_user_id = 4;
  string triggered_by_user_id = 5;
  int64 attempt = 6;
  int64 ttl_ms = 7;
}

message IssueTaskJobTokenResp {
  string access_token = 1;
  int64 expires_at = 2;
}
`

type memoryResolver struct{}

func (r *memoryResolver) FindFileByPath(path string) (protocompile.SearchResult, error) {
	if path == jobTokenProtoPath {
		return protocompile.SearchResult{Source: strings.NewReader(jobTokenProtoContent)}, nil
	}
	return protocompile.SearchResult{}, fmt.Errorf("proto not found: %s", path)
}

var (
	jobTokenOnce sync.Once
	jobTokenErr  error

	jobTokenMethodDesc protoreflect.MethodDescriptor
	jobTokenReqDesc    protoreflect.MessageDescriptor
	jobTokenRespDesc   protoreflect.MessageDescriptor
)

func initJobTokenProto() error {
	jobTokenOnce.Do(func() {
		compiler := protocompile.Compiler{Resolver: protocompile.WithStandardImports(&memoryResolver{})}
		files, err := compiler.Compile(context.TODO(), jobTokenProtoPath)
		if err != nil {
			jobTokenErr = err
			return
		}
		fd := files[0]
		services := fd.Services()
		if services.Len() == 0 {
			jobTokenErr = fmt.Errorf("job token proto missing service")
			return
		}
		serviceDesc := services.Get(0)
		methods := serviceDesc.Methods()
		if methods.Len() == 0 {
			jobTokenErr = fmt.Errorf("job token proto missing method")
			return
		}
		jobTokenMethodDesc = methods.Get(0)
		jobTokenReqDesc = jobTokenMethodDesc.Input()
		jobTokenRespDesc = jobTokenMethodDesc.Output()
	})
	return jobTokenErr
}

// MethodDesc returns the compiled descriptor for the IssueTaskJobToken RPC method.
func MethodDesc() (protoreflect.MethodDescriptor, error) {
	if err := initJobTokenProto(); err != nil {
		return nil, err
	}
	return jobTokenMethodDesc, nil
}

// ReqDesc returns the compiled descriptor for the job token request message.
func ReqDesc() (protoreflect.MessageDescriptor, error) {
	if err := initJobTokenProto(); err != nil {
		return nil, err
	}
	return jobTokenReqDesc, nil
}

// RespDesc returns the compiled descriptor for the job token response message.
func RespDesc() (protoreflect.MessageDescriptor, error) {
	if err := initJobTokenProto(); err != nil {
		return nil, err
	}
	return jobTokenRespDesc, nil
}

// ServiceFullName returns the fully qualified protobuf service name.
func ServiceFullName() string {
	return "auth.JobTokenService"
}

// FullMethod returns the fully qualified gRPC method path for job token issuance.
func FullMethod() string {
	return "/auth.JobTokenService/IssueTaskJobToken"
}
