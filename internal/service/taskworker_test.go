// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"
	"testing"
	"time"

	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/choysum-dev/choysum/internal/task"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/config"
	grpcclient "github.com/choysum-dev/choysum/pkg/grpc/client"
	"github.com/choysum-dev/choysum/pkg/grpc/converter"
	"github.com/choysum-dev/choysum/pkg/grpc/loader"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/oerrors"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/structpb"
)

func newTaskExecutionStoreTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:task_execution_store_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&task.Execution{}); err != nil {
		t.Fatalf("auto migrate execution: %v", err)
	}
	return db
}

type taskWorkerTestScope struct {
	ctx context.Context
	cfg *config.Config
	db  *gorm.DB
}

func (e *taskWorkerTestScope) Run(fn func(runtimeScope scope.Scope) error) error { return fn(e) }
func (e *taskWorkerTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}
func (e *taskWorkerTestScope) Session() *scope.Session { return &scope.Session{DB: e.db} }
func (e *taskWorkerTestScope) WithContext(ctx context.Context) scope.Scope {
	clone := *e
	clone.ctx = ctx
	return &clone
}
func (e *taskWorkerTestScope) Context() context.Context { return e.ctx }
func (e *taskWorkerTestScope) Logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
func (e *taskWorkerTestScope) Config() *config.Config { return e.cfg }

func (e *taskWorkerTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}

func TestTaskWorkerErrorDetails(t *testing.T) {
	choysumErr := oerrors.New("meta.lock", "LEASE_CONFLICT", "lease conflict").
		WithGrpcCode(codes.Aborted).
		WithMetadata("retry_after_ms", "5000").
		WithMetadata("reason", "lease_conflict")

	errMap := errToTaskError(choysumErr)
	if errMap == nil {
		t.Fatalf("expected error map")
	}

	details := requireAnyMap(t, errMap["details"])
	if details["retry_after_ms"] != "5000" {
		t.Fatalf("details retry_after_ms mismatch: %v", details["retry_after_ms"])
	}
	if details["reason"] != "lease_conflict" {
		t.Fatalf("details reason mismatch: %v", details["reason"])
	}

	svc := &ApplicationService{name: "task"}
	respMsg, respErr := svc.buildExecuteJobResp(executeJobResponse{
		Status: "FAILED_NON_RETRYABLE",
		Error:  errMap,
	})
	if respErr != nil {
		t.Fatalf("build response failed: %v", respErr)
	}
	protoMsg, ok := respMsg.(interface{ ProtoReflect() protoreflect.Message })
	if !ok {
		t.Fatalf("response does not implement ProtoReflect")
	}
	respMap, convErr := converter.MessageToMap(protoMsg.ProtoReflect())
	if convErr != nil {
		t.Fatalf("convert response failed: %v", convErr)
	}
	errVal, ok := respMap["error"].(map[string]any)
	if !ok {
		t.Fatalf("response error missing or invalid")
	}

	respDetails := requireAnyMap(t, errVal["details"])
	if respDetails["retry_after_ms"] != "5000" {
		t.Fatalf("response details retry_after_ms mismatch: %v", respDetails["retry_after_ms"])
	}
	if respDetails["reason"] != "lease_conflict" {
		t.Fatalf("response details reason mismatch: %v", respDetails["reason"])
	}
}

func TestTaskWorkerMapExecutionErrorResourceBusy(t *testing.T) {
	choysumErr := oerrors.New("meta.lock", "LEASE_CONFLICT", "lease conflict").
		WithGrpcCode(codes.Aborted).
		WithMetadata("retry_after_ms", "2500")

	mapped := mapExecutionError(choysumErr)
	if mapped.Status != "RESOURCE_BUSY" {
		t.Fatalf("status mismatch: want RESOURCE_BUSY, got %s", mapped.Status)
	}
	if mapped.RetryAfterMs != 2500 {
		t.Fatalf("retry_after_ms mismatch: want 2500, got %d", mapped.RetryAfterMs)
	}
	if mapped.Error == nil {
		t.Fatalf("expected error details")
	}
	if details := requireAnyMap(t, mapped.Error["details"]); details["retry_after_ms"] != "2500" {
		t.Fatalf("details retry_after_ms mismatch: %v", details["retry_after_ms"])
	}
}

func TestTaskWorkerHelperFunctions(t *testing.T) {
	for input, want := range map[string]int32{
		"SUCCEEDED":            1,
		"FAILED_NON_RETRYABLE": 2,
		"FAILED_RETRYABLE":     3,
		"ALREADY_RUNNING":      4,
		"RESOURCE_BUSY":        5,
		"CANCELLED":            6,
		"unknown":              0,
		"  failed_retryable  ": 3,
	} {
		if got := int32(statusToEnum(input)); got != want {
			t.Fatalf("statusToEnum(%q) = %d, want %d", input, got, want)
		}
	}

	if got, ok := toAnyMap(map[string]any{"a": 1}); !ok || got["a"] != 1 {
		t.Fatalf("unexpected toAnyMap(map[string]any) result: %#v ok=%v", got, ok)
	}
	if got, ok := toAnyMap(map[string]string{"a": "b"}); !ok || got["a"] != "b" {
		t.Fatalf("unexpected toAnyMap(map[string]string) result: %#v ok=%v", got, ok)
	}
	if _, ok := toAnyMap([]string{"bad"}); ok {
		t.Fatal("expected unsupported input toAnyMap to fail")
	}

	if got := jsonToAny(nil); got != nil {
		t.Fatalf("jsonToAny(nil) = %#v, want nil", got)
	}
	if got := jsonToAny(datatypes.JSON([]byte("not-json"))); got != nil {
		t.Fatalf("jsonToAny(invalid) = %#v, want nil", got)
	}
	decoded := jsonToAny(datatypes.JSON([]byte(`{"count":2,"name":"choysum"}`)))
	decodedMap, ok := decoded.(map[string]any)
	if !ok || decodedMap["name"] != "choysum" || decodedMap["count"] != float64(2) {
		t.Fatalf("unexpected jsonToAny(valid) result: %#v", decoded)
	}

	if errorsIsDuplicate(nil) {
		t.Fatal("expected nil error to not be duplicate")
	}
	if !errorsIsDuplicate(gorm.ErrDuplicatedKey) {
		t.Fatal("expected gorm duplicated key sentinel to be detected")
	}
	if !errorsIsDuplicate(errors.New("Duplicate entry")) {
		t.Fatal("expected duplicate substring to be detected")
	}
	if errorsIsDuplicate(errors.New("plain failure")) {
		t.Fatal("expected plain error to not be treated as duplicate")
	}

	ownerA := leaseOwner("job-1")
	ownerB := leaseOwner("job-1")
	if ownerA == "" || ownerB == "" || ownerA == ownerB {
		t.Fatalf("expected leaseOwner to generate distinct non-empty ids, got %q and %q", ownerA, ownerB)
	}

	for _, testCase := range []struct {
		name string
		in   any
		want bool
	}{
		{name: "string", in: "2024-01-02T03:04:05Z", want: true},
		{name: "blank string", in: "   ", want: false},
		{name: "float seconds", in: map[string]any{"seconds": float64(1)}, want: true},
		{name: "int seconds", in: map[string]any{"seconds": int64(2)}, want: true},
		{name: "zero seconds", in: map[string]any{"seconds": float64(0)}, want: false},
		{name: "other", in: 123, want: false},
	} {
		if got := hasTimestampValue(testCase.in); got != testCase.want {
			t.Fatalf("hasTimestampValue(%s) = %v, want %v", testCase.name, got, testCase.want)
		}
	}
}

func TestTaskWorkerBuildExecuteJobResp(t *testing.T) {
	svc := &ApplicationService{name: "task"}
	respMsg, err := svc.taskWorkerAdapter().buildExecuteJobResp(executeJobResponse{
		Status:       "SUCCEEDED",
		RetryAfterMs: 42,
		Result: map[string]any{
			"ok":    true,
			"count": 2,
		},
	})
	if err != nil {
		t.Fatalf("buildExecuteJobResp() error = %v", err)
	}
	protoMsg, ok := respMsg.(interface{ ProtoReflect() protoreflect.Message })
	if !ok {
		t.Fatalf("response does not implement ProtoReflect: %T", respMsg)
	}
	respMap, err := converter.MessageToMap(protoMsg.ProtoReflect())
	if err != nil {
		t.Fatalf("MessageToMap(response) error = %v", err)
	}
	if respMap["status"] != "EXECUTE_JOB_STATUS_SUCCEEDED" || respMap["retry_after_ms"] != "42" {
		t.Fatalf("unexpected response envelope: %#v", respMap)
	}
	resultMap, ok := respMap["result"].(map[string]any)
	if !ok || resultMap["ok"] != true || resultMap["count"] != float64(2) {
		t.Fatalf("unexpected response result: %#v", respMap["result"])
	}
}

func TestTaskWorkerBuildExecuteJobRespEncodeFailures(t *testing.T) {
	svc := &ApplicationService{name: "task"}
	adapter := svc.taskWorkerAdapter()

	_, err := adapter.buildExecuteJobResp(executeJobResponse{
		Status: "SUCCEEDED",
		Result: func() {},
	})
	if status.Code(err) != codes.InvalidArgument || !strings.Contains(status.Convert(err).Message(), "encode job result") {
		t.Fatalf("expected InvalidArgument for unsupported result, got %v", err)
	}

	_, err = adapter.buildExecuteJobResp(executeJobResponse{
		Status: "FAILED_NON_RETRYABLE",
		Error: map[string]any{
			"message": "boom",
			"details": map[string]any{"bad": func() {}},
		},
	})
	if status.Code(err) != codes.InvalidArgument || !strings.Contains(status.Convert(err).Message(), "encode job error") {
		t.Fatalf("expected InvalidArgument for unsupported error details, got %v", err)
	}
}

func TestTaskExecutionStoreAlreadyRunningRetryAfterMax(t *testing.T) {
	store := taskExecutionStore{}
	if got := store.alreadyRunningRetryAfterMax(); got != 60*time.Second {
		t.Fatalf("default alreadyRunningRetryAfterMax mismatch: %v", got)
	}
	store = taskExecutionStore{runtimeScope: &taskWorkerTestScope{
		cfg: &config.Config{Task: &config.TaskConfig{Worker: &config.TaskWorkerConfig{AlreadyRunningRetryAfterMaxMs: 75}}},
	}}
	if got := store.alreadyRunningRetryAfterMax(); got != 75*time.Millisecond {
		t.Fatalf("custom alreadyRunningRetryAfterMax mismatch: %v", got)
	}
}

func TestParseExecuteJobReqAndValidateJobToken(t *testing.T) {
	_, reqDesc, _, _, err := taskWorkerDescriptors("task")
	if err != nil {
		t.Fatalf("taskWorkerDescriptors() error = %v", err)
	}
	reqMsg := dynamicpb.NewMessage(reqDesc)
	if err := converter.MapToMessage(map[string]interface{}{
		"job_id":               "job-1",
		"attempt":              2,
		"full_method":          "task.Job/GetJob",
		"scheduler_user_id":    "scheduler-1",
		"triggered_by_user_id": "user-2",
		"timeout_ms":           5000,
		"payload": map[string]interface{}{
			"enabled": true,
		},
	}, reqMsg); err != nil {
		t.Fatalf("MapToMessage(request) error = %v", err)
	}
	parsed, err := parseExecuteJobReq(reqMsg)
	if err != nil {
		t.Fatalf("parseExecuteJobReq() error = %v", err)
	}
	if parsed.JobId != "job-1" || parsed.Attempt != 2 || parsed.FullMethod != "task.Job/GetJob" || parsed.SchedulerUserId != "scheduler-1" || parsed.TriggeredByUserId != "user-2" || parsed.Payload["enabled"] != true {
		t.Fatalf("unexpected parsed execute job request: %#v", parsed)
	}

	svc := &ApplicationService{name: "task"}
	validIdentity := &testIdentity{userID: "scheduler-1", tokenID: "tok-1", meta: map[string]any{
		"purpose":           "task_job",
		"jobId":             "job-1",
		"fullMethod":        "task.Job/GetJob",
		"targetApp":         "task",
		"schedulerUserId":   "scheduler-1",
		"triggeredByUserId": "user-2",
		"attempt":           float64(2),
	}}
	ctx := auth.ContextWithIdentity(context.Background(), validIdentity)
	if err := svc.validateJobToken(ctx, parsed); err != nil {
		t.Fatalf("validateJobToken(valid) error = %v", err)
	}

	mismatchIdentity := &testIdentity{userID: "scheduler-1", tokenID: "tok-1", meta: map[string]any{
		"purpose":           "task_job",
		"jobId":             "job-1",
		"fullMethod":        "task.Job/GetJob",
		"targetApp":         "task",
		"schedulerUserId":   "scheduler-1",
		"triggeredByUserId": "user-2",
		"attempt":           float64(3),
	}}
	ctx = auth.ContextWithIdentity(context.Background(), mismatchIdentity)
	if err := svc.validateJobToken(ctx, parsed); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected mismatched attempt to be unauthenticated, got %v", err)
	}
}

func TestParseExecuteJobReqAndValidateJobTokenFailurePaths(t *testing.T) {
	_, reqDesc, _, _, err := taskWorkerDescriptors("task")
	if err != nil {
		t.Fatalf("taskWorkerDescriptors() error = %v", err)
	}

	t.Run("parse rejects non-map messages", func(t *testing.T) {
		msg := dynamicpb.NewMessage(structpb.NewStringValue("not-a-map").ProtoReflect().Descriptor())
		if _, err := parseExecuteJobReq(msg); err == nil || !strings.Contains(err.Error(), "not a map") {
			t.Fatalf("expected non-map parse error, got %v", err)
		}
	})

	t.Run("parse rejects required fields that trim to empty", func(t *testing.T) {
		msg := dynamicpb.NewMessage(reqDesc)
		if err := converter.MapToMessage(map[string]any{
			"job_id":               " job-blank ",
			"full_method":          "   ",
			"scheduler_user_id":    "scheduler-1",
			"triggered_by_user_id": "user-2",
		}, msg); err != nil {
			t.Fatalf("MapToMessage() error = %v", err)
		}
		if _, err := parseExecuteJobReq(msg); err == nil || !strings.Contains(err.Error(), "missing required fields") {
			t.Fatalf("expected missing required field error, got %v", err)
		}
	})

	t.Run("parse keeps proto zero values when required fields are omitted", func(t *testing.T) {
		msg := dynamicpb.NewMessage(reqDesc)
		if err := converter.MapToMessage(map[string]any{
			"job_id":               "job-1",
			"full_method":          "task.Job/GetJob",
			"scheduler_user_id":    "scheduler-1",
			"triggered_by_user_id": "",
		}, msg); err != nil {
			t.Fatalf("MapToMessage() error = %v", err)
		}
		parsed, err := parseExecuteJobReq(msg)
		if err != nil {
			t.Fatalf("parseExecuteJobReq() error = %v", err)
		}
		if parsed.JobId != "job-1" || parsed.FullMethod != "task.Job/GetJob" || parsed.SchedulerUserId != "scheduler-1" {
			t.Fatalf("unexpected parsed zero-value request: %#v", parsed)
		}
	})

	t.Run("parse defaults payload when optional fields are omitted or normalized away", func(t *testing.T) {
		msg := dynamicpb.NewMessage(reqDesc)
		if err := converter.MapToMessage(map[string]any{
			"job_id":               "job-2",
			"attempt":              float64(4),
			"full_method":          "task.Job/GetJob",
			"scheduler_user_id":    "scheduler-1",
			"triggered_by_user_id": "user-2",
			"timeout_ms":           float64(2500),
		}, msg); err != nil {
			t.Fatalf("MapToMessage() error = %v", err)
		}
		parsed, err := parseExecuteJobReq(msg)
		if err != nil {
			t.Fatalf("parseExecuteJobReq() error = %v", err)
		}
		if parsed.Attempt != 4 || parsed.TimeoutMs != 0 || len(parsed.Payload) != 0 {
			t.Fatalf("unexpected parsed request: %#v", parsed)
		}
	})

	t.Run("parse ignores non map payload and accepts float timeout from alternate proto", func(t *testing.T) {
		loader.Global().RegisterProto("task/taskworker_parse_alt.proto", `syntax = "proto3";
package task;

service ParseAlt {
	rpc Run(ParseAltReq) returns (ParseAltResp);
}

message ParseAltReq {
	string job_id = 1;
	double attempt = 2;
	string full_method = 3;
	string scheduler_user_id = 4;
	string triggered_by_user_id = 5;
	int32 timeout_ms = 6;
	string payload = 7;
}

message ParseAltResp {}
`)
		md, err := loader.Global().GetMethodDescriptor("task.ParseAlt.Run")
		if err != nil {
			t.Fatalf("GetMethodDescriptor(task.ParseAlt.Run) error = %v", err)
		}

		msg := dynamicpb.NewMessage(md.Input())
		if err := converter.MapToMessage(map[string]any{
			"job_id":               "job-3",
			"attempt":              6,
			"full_method":          "task.Job/GetJob",
			"scheduler_user_id":    "scheduler-2",
			"triggered_by_user_id": "user-3",
			"timeout_ms":           1500,
			"payload":              "not-a-map",
		}, msg); err != nil {
			t.Fatalf("MapToMessage() error = %v", err)
		}

		parsed, err := parseExecuteJobReq(msg)
		if err != nil {
			t.Fatalf("parseExecuteJobReq() error = %v", err)
		}
		if parsed.JobId != "job-3" || parsed.Attempt != 6 || parsed.TimeoutMs != 1500 {
			t.Fatalf("unexpected parsed request core fields: %#v", parsed)
		}
		if len(parsed.Payload) != 0 {
			t.Fatalf("expected non-map payload to be ignored, got %#v", parsed.Payload)
		}
	})

	t.Run("validate job token rejects each metadata mismatch", func(t *testing.T) {
		req := &executeJobRequest{
			JobId:             "job-1",
			Attempt:           2,
			FullMethod:        "task.Job/GetJob",
			SchedulerUserId:   "scheduler-1",
			TriggeredByUserId: "user-2",
		}
		svc := &ApplicationService{name: "task"}
		baseMeta := map[string]any{
			"purpose":           "task_job",
			"jobId":             "job-1",
			"fullMethod":        "task.Job/GetJob",
			"targetApp":         "task",
			"schedulerUserId":   "scheduler-1",
			"triggeredByUserId": "user-2",
			"attempt":           float64(2),
		}

		cases := []struct {
			name string
			ctx  context.Context
		}{
			{name: "missing identity", ctx: context.Background()},
			{name: "user mismatch", ctx: auth.ContextWithIdentity(context.Background(), &testIdentity{userID: "other", tokenID: "tok", meta: baseMeta})},
			{name: "missing metadata", ctx: auth.ContextWithIdentity(context.Background(), &testIdentity{userID: "scheduler-1", tokenID: "tok", meta: nil})},
			{name: "purpose mismatch", ctx: auth.ContextWithIdentity(context.Background(), &testIdentity{userID: "scheduler-1", tokenID: "tok", meta: map[string]any{"purpose": "other"}})},
			{name: "job mismatch", ctx: auth.ContextWithIdentity(context.Background(), &testIdentity{userID: "scheduler-1", tokenID: "tok", meta: cloneAnyMap(baseMeta, "jobId", "job-x")})},
			{name: "method mismatch", ctx: auth.ContextWithIdentity(context.Background(), &testIdentity{userID: "scheduler-1", tokenID: "tok", meta: cloneAnyMap(baseMeta, "fullMethod", "task.Job/Other")})},
			{name: "target app mismatch", ctx: auth.ContextWithIdentity(context.Background(), &testIdentity{userID: "scheduler-1", tokenID: "tok", meta: cloneAnyMap(baseMeta, "targetApp", "auth")})},
			{name: "scheduler mismatch", ctx: auth.ContextWithIdentity(context.Background(), &testIdentity{userID: "scheduler-1", tokenID: "tok", meta: cloneAnyMap(baseMeta, "schedulerUserId", "scheduler-x")})},
			{name: "triggered mismatch", ctx: auth.ContextWithIdentity(context.Background(), &testIdentity{userID: "scheduler-1", tokenID: "tok", meta: cloneAnyMap(baseMeta, "triggeredByUserId", "user-x")})},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if err := svc.validateJobToken(tc.ctx, req); status.Code(err) != codes.Unauthenticated {
					t.Fatalf("expected unauthenticated, got %v", err)
				}
			})
		}
	})
}

func TestEnsureInternalIdentityAndCancelRequestedShortPaths(t *testing.T) {
	req := &executeJobRequest{
		JobId:             "job-1",
		Attempt:           2,
		FullMethod:        "task.Job/GetJob",
		SchedulerUserId:   "scheduler-1",
		TriggeredByUserId: "user-2",
	}
	existing := &testIdentity{userID: "existing", tokenID: "tok", meta: map[string]any{"k": "v"}}
	ctx := auth.ContextWithIdentity(context.Background(), existing)
	if got := auth.IdentityFromContext(ensureInternalIdentity(ctx, "task", req)); got != existing {
		t.Fatalf("expected existing identity to be preserved, got %#v", got)
	}

	newCtx := ensureInternalIdentity(context.Background(), "task", req)
	created := auth.IdentityFromContext(newCtx)
	if created == nil || created.GetUserID() != "scheduler-1" {
		t.Fatalf("expected ensureInternalIdentity to inject scheduler identity, got %#v", created)
	}
	meta := created.GetMetadata()
	if meta["purpose"] != "task_job" || meta["targetApp"] != "task" || meta["attempt"] != 2 {
		t.Fatalf("unexpected internal identity metadata: %#v", meta)
	}

	store := taskExecutionStore{runtimeScope: &taskWorkerTestScope{cfg: &config.Config{Auth: config.NewDefaultAuthConfig(), Server: config.NewDefaultServerConfig()}}}
	if cancelled, ok := store.isCancelRequestedViaTask(context.Background(), "job-1"); cancelled || ok {
		t.Fatalf("expected missing dialer to return false,false, got %v,%v", cancelled, ok)
	}

	ctxWithDialer := auth.ContextWithIdentity(context.Background(), existing)
	ctxWithDialer = metadata.NewIncomingContext(ctxWithDialer, metadata.Pairs("x-test", "v"))
	ctxWithDialer = grpcclient.ContextWithServiceDialer(ctxWithDialer, func(context.Context, string) (*grpc.ClientConn, error) {
		return nil, errors.New("dial failed")
	})
	if cancelled, ok := store.isCancelRequestedViaTask(ctxWithDialer, "job-1"); cancelled || ok {
		t.Fatalf("expected dial failure path to return false,false, got %v,%v", cancelled, ok)
	}
}

func TestEnsureInternalIdentityAndAuthorizeInternalCallerHelpers(t *testing.T) {
	t.Run("ensureInternalIdentity returns input context for nil or blank scheduler", func(t *testing.T) {
		baseCtx := context.WithValue(context.Background(), struct{}{}, "keep")
		if got := ensureInternalIdentity(baseCtx, "task", nil); got != baseCtx {
			t.Fatalf("expected nil req to preserve context, got %#v", got)
		}
		if got := ensureInternalIdentity(baseCtx, "task", &executeJobRequest{SchedulerUserId: "   "}); got != baseCtx {
			t.Fatalf("expected blank scheduler to preserve context, got %#v", got)
		}
	})

	t.Run("ensureInternalIdentity generates job id when missing", func(t *testing.T) {
		ctx := ensureInternalIdentity(context.Background(), "task", &executeJobRequest{
			JobId:             "   ",
			Attempt:           3,
			FullMethod:        "task.Job/GetJob",
			SchedulerUserId:   "scheduler-1",
			TriggeredByUserId: "user-2",
		})
		identity := auth.IdentityFromContext(ctx)
		if identity == nil || identity.GetUserID() != "scheduler-1" {
			t.Fatalf("expected generated identity, got %#v", identity)
		}
		meta := identity.GetMetadata()
		if strings.TrimSpace(fmt.Sprintf("%v", meta["jobId"])) == "" || meta["attempt"] != 3 {
			t.Fatalf("unexpected generated metadata: %#v", meta)
		}
	})

	t.Run("authorizeInternalCallerFromContext rejects missing or mismatched headers", func(t *testing.T) {
		runtimeScope := &taskWorkerTestScope{cfg: &config.Config{Auth: &config.AuthConfig{InternalKey: "secret"}, Server: &config.ServerConfig{Environment: "development"}}}
		if authorizeInternalCallerFromContext(context.Background(), runtimeScope) {
			t.Fatal("expected missing metadata to be rejected")
		}
		ctxNoValue := metadata.NewIncomingContext(context.Background(), metadata.MD{internalKeyHeader: {}})
		if authorizeInternalCallerFromContext(ctxNoValue, runtimeScope) {
			t.Fatal("expected empty header values to be rejected")
		}
		ctxMismatch := metadata.NewIncomingContext(context.Background(), metadata.Pairs(internalKeyHeader, "wrong"))
		if authorizeInternalCallerFromContext(ctxMismatch, runtimeScope) {
			t.Fatal("expected mismatched header to be rejected")
		}
	})
}

func requireAnyMap(t *testing.T, v any) map[string]any {
	t.Helper()
	switch tv := v.(type) {
	case map[string]any:
		return tv
	case map[string]string:
		out := make(map[string]any, len(tv))
		for k, v := range tv {
			out[k] = v
		}
		return out
	default:
		t.Fatalf("expected map, got %T", v)
		return nil
	}
}

func TestTaskExecutionStoreCancelWatchInterval(t *testing.T) {
	store := taskExecutionStore{}
	if got := store.cancelWatchInterval(); got != 2*time.Second {
		t.Fatalf("default cancel watch interval mismatch: %v", got)
	}

	runtimeScope := &taskWorkerTestScope{
		ctx: context.Background(),
		cfg: &config.Config{
			Task: &config.TaskConfig{
				Worker: &config.TaskWorkerConfig{
					CancelPollIntervalMs: 25,
				},
			},
		},
	}
	store = taskExecutionStore{runtimeScope: runtimeScope}
	if got := store.cancelWatchInterval(); got != 25*time.Millisecond {
		t.Fatalf("custom cancel watch interval mismatch: %v", got)
	}
}

func TestTaskExecutionStoreWatchCancelForcesInterrupt(t *testing.T) {
	runtimeScope := &taskWorkerTestScope{
		ctx: context.Background(),
		cfg: &config.Config{
			Task: &config.TaskConfig{
				Worker: &config.TaskWorkerConfig{
					CancelPollIntervalMs: 5,
				},
			},
		},
	}
	store := taskExecutionStore{runtimeScope: runtimeScope}
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	cancelled := make(chan struct{}, 1)
	cancelCalled := make(chan struct{}, 1)
	checks := 0
	checker := func(ctx context.Context, jobId string) (bool, bool) {
		checks++
		if checks >= 2 {
			return true, true
		}
		return false, true
	}
	go store.watchCancel(ctx, "job-1", cancelled, func() {
		ctxCancel()
		select {
		case cancelCalled <- struct{}{}:
		default:
		}
	}, checker)

	select {
	case <-cancelled:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected cancelled signal")
	}
	select {
	case <-cancelCalled:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected cancel callback")
	}
}

func TestTaskExecutionStoreHeartbeatLeaseLostCancels(t *testing.T) {
	dsn := "file:task_worker_heartbeat_lease_lost?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&task.Execution{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	now := time.Now().UTC()
	exec := task.Execution{
		JobId:       "job-1",
		Status:      "running",
		LeaseOwner:  "owner-2",
		LeaseUntil:  &now,
		StartedAt:   &now,
		CreatedAt:   now,
		UpdatedAt:   now,
		Attempt:     1,
		FullMethod:  "app.Service/Method",
		PayloadJson: datatypes.JSON([]byte(`{}`)),
	}
	if err := db.Create(&exec).Error; err != nil {
		t.Fatalf("insert execution: %v", err)
	}

	runtimeScope := &taskWorkerTestScope{
		ctx: context.Background(),
		cfg: &config.Config{
			Task: &config.TaskConfig{
				Worker: &config.TaskWorkerConfig{
					HeartbeatIntervalMs: 5,
					LeaseDurationMs:     10,
				},
			},
		},
		db: db,
	}
	store := taskExecutionStore{runtimeScope: runtimeScope}

	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	stop := make(chan struct{})
	defer close(stop)
	leaseLost := make(chan struct{}, 1)
	cancelCalled := make(chan struct{}, 1)

	go store.heartbeat(ctx, "job-1", "owner-1", stop, nil, func() {
		ctxCancel()
		select {
		case cancelCalled <- struct{}{}:
		default:
		}
	}, leaseLost)

	select {
	case <-leaseLost:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected lease lost signal")
	}
	select {
	case <-cancelCalled:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected cancel callback")
	}
}

func TestTaskExecutionStoreHeartbeatCancelsWhenTaskRequestsCancel(t *testing.T) {
	loader.Global().RegisterProto("task/taskworker_cancel_getjob.proto", `syntax = "proto3";
package task;

service Job {
	rpc GetJob(GetJobReq) returns (GetJobResp);
}

message GetJobReq {
	string job_id = 1;
	repeated string fields = 2;
}

message JobRecord {
	string cancelRequestedAt = 1;
	string status = 2;
}

message GetJobResp {
	JobRecord job = 1;
}
`)
	md, err := loader.Global().GetMethodDescriptor("task.Job.GetJob")
	if err != nil {
		t.Fatalf("GetMethodDescriptor(task.Job.GetJob) error = %v", err)
	}

	dsn := "file:task_worker_heartbeat_cancelled?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&task.Execution{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	now := time.Now().UTC()
	exec := task.Execution{
		JobId:       "job-cancelled",
		Status:      "running",
		LeaseOwner:  "owner-1",
		LeaseUntil:  &now,
		StartedAt:   &now,
		CreatedAt:   now,
		UpdatedAt:   now,
		Attempt:     1,
		FullMethod:  "app.Service/Method",
		PayloadJson: datatypes.JSON([]byte(`{}`)),
	}
	if err := db.Create(&exec).Error; err != nil {
		t.Fatalf("insert execution: %v", err)
	}

	lis := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	var seenMD metadata.MD
	serviceDesc := &grpc.ServiceDesc{
		ServiceName: "task.Job",
		HandlerType: (*interface{})(nil),
		Methods: []grpc.MethodDesc{{
			MethodName: "GetJob",
			Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
				if mdIn, ok := metadata.FromIncomingContext(ctx); ok {
					seenMD = mdIn
				}
				reqMsg := dynamicpb.NewMessage(md.Input())
				if err := dec(reqMsg); err != nil {
					return nil, err
				}
				respMsg := dynamicpb.NewMessage(md.Output())
				jobField := respMsg.Descriptor().Fields().ByName("job")
				jobMsg := dynamicpb.NewMessage(jobField.Message())
				if err := converter.MapToMessage(map[string]any{
					"cancelRequestedAt": now.Format(time.RFC3339),
					"status":            "running",
				}, jobMsg); err != nil {
					return nil, err
				}
				respMsg.Set(jobField, protoreflect.ValueOfMessage(jobMsg))
				return respMsg, nil
			},
		}},
	}
	server.RegisterService(serviceDesc, nil)
	go func() { _ = server.Serve(lis) }()
	t.Cleanup(server.Stop)

	var conn *grpc.ClientConn
	dialer := func(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
		if serviceName != "task.Job" {
			t.Fatalf("serviceName = %q, want task.Job", serviceName)
		}
		if conn != nil {
			return conn, nil
		}
		c, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, err
		}
		conn = c
		return conn, nil
	}
	t.Cleanup(func() {
		if conn != nil {
			_ = conn.Close()
		}
	})

	runtimeScope := &taskWorkerTestScope{
		ctx: grpcclient.ContextWithServiceDialer(context.Background(), dialer),
		cfg: &config.Config{
			Auth:   &config.AuthConfig{InternalKey: "secret"},
			Server: &config.ServerConfig{Environment: "development"},
			Task: &config.TaskConfig{Worker: &config.TaskWorkerConfig{
				HeartbeatIntervalMs: 5,
				LeaseDurationMs:     20,
			}},
		},
		db: db,
	}
	store := taskExecutionStore{runtimeScope: runtimeScope}

	ctx, ctxCancel := context.WithCancel(runtimeScope.ctx)
	defer ctxCancel()
	stop := make(chan struct{})
	defer close(stop)
	cancelled := make(chan struct{}, 1)
	cancelCalled := make(chan struct{}, 1)
	leaseLost := make(chan struct{}, 1)

	go store.heartbeat(ctx, "job-cancelled", "owner-1", stop, cancelled, func() {
		ctxCancel()
		select {
		case cancelCalled <- struct{}{}:
		default:
		}
	}, leaseLost)

	select {
	case <-cancelled:
	case <-time.After(300 * time.Millisecond):
		t.Fatalf("expected cancelled signal")
	}
	select {
	case <-cancelCalled:
	case <-time.After(300 * time.Millisecond):
		t.Fatalf("expected cancel callback")
	}
	select {
	case <-leaseLost:
		t.Fatalf("did not expect lease lost when cancellation is detected via task service")
	default:
	}
	if got := seenMD.Get(internalKeyHeader); len(got) != 1 || got[0] != "secret" {
		t.Fatalf("internal auth metadata = %#v", got)
	}
}

func TestTaskExecutionStoreTryStartAndFinalizeLifecycle(t *testing.T) {
	db := newTaskExecutionStoreTestDB(t)
	runtimeScope := &taskWorkerTestScope{
		ctx: context.Background(),
		cfg: &config.Config{
			Auth:   config.NewDefaultAuthConfig(),
			Server: config.NewDefaultServerConfig(),
		},
		db: db,
	}
	store := taskExecutionStore{runtimeScope: runtimeScope}
	now := time.Now().UTC().Truncate(time.Millisecond)
	req := &executeJobRequest{
		JobId:             "job-start",
		Attempt:           2,
		SchedulerUserId:   "scheduler-1",
		TriggeredByUserId: "user-1",
		FullMethod:        "task.Job/GetJob",
		Payload:           map[string]any{"enabled": true},
	}

	state, resp, err := store.tryStart(context.Background(), req, "owner-1", now.Add(time.Minute), now)
	if err != nil {
		t.Fatalf("tryStart() error = %v", err)
	}
	if state != "RUNNING" || resp.Status != "" {
		t.Fatalf("unexpected tryStart result: state=%q resp=%#v", state, resp)
	}

	var rec task.Execution
	if err := db.Where("job_id = ?", req.JobId).First(&rec).Error; err != nil {
		t.Fatalf("load created execution: %v", err)
	}
	if rec.Status != "running" || rec.LeaseOwner != "owner-1" || rec.Attempt != 2 || rec.FullMethod != req.FullMethod {
		t.Fatalf("unexpected created execution: %#v", rec)
	}
	if payload := jsonToAny(rec.PayloadJson); payload.(map[string]any)["enabled"] != true {
		t.Fatalf("unexpected stored payload: %#v", payload)
	}

	store.finalizeSuccess(context.Background(), req.JobId, "owner-1", map[string]any{"ok": true})
	if err := db.Where("job_id = ?", req.JobId).First(&rec).Error; err != nil {
		t.Fatalf("reload succeeded execution: %v", err)
	}
	if rec.Status != "succeeded" || rec.FinishedAt == nil {
		t.Fatalf("unexpected success finalization: %#v", rec)
	}
	if result := jsonToAny(rec.ResultJson); result.(map[string]any)["ok"] != true {
		t.Fatalf("unexpected stored result: %#v", result)
	}

	failed := task.Execution{
		JobId:      "job-fail",
		Status:     "running",
		LeaseOwner: "owner-2",
		LeaseUntil: ptrTime(now.Add(time.Minute)),
		StartedAt:  ptrTime(now),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := db.Create(&failed).Error; err != nil {
		t.Fatalf("insert failed execution: %v", err)
	}
	store.finalizeFailure(context.Background(), "job-fail", "owner-2", map[string]any{"reason": "boom"})
	var failedRec task.Execution
	if err := db.Where("job_id = ?", "job-fail").First(&failedRec).Error; err != nil {
		t.Fatalf("reload failed execution: %v", err)
	}
	if failedRec.Status != "failed" || failedRec.FinishedAt == nil {
		t.Fatalf("unexpected failure finalization: %#v", failedRec)
	}
	if errMap := jsonToAny(failedRec.ErrorJson); errMap.(map[string]any)["reason"] != "boom" {
		t.Fatalf("unexpected stored error: %#v", errMap)
	}

	cancelled := task.Execution{
		JobId:      "job-cancel",
		Status:     "running",
		LeaseOwner: "owner-3",
		LeaseUntil: ptrTime(now.Add(time.Minute)),
		StartedAt:  ptrTime(now),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := db.Create(&cancelled).Error; err != nil {
		t.Fatalf("insert cancelled execution: %v", err)
	}
	store.finalizeCancelled(context.Background(), "job-cancel", "owner-3")
	var cancelledRec task.Execution
	if err := db.Where("job_id = ?", "job-cancel").First(&cancelledRec).Error; err != nil {
		t.Fatalf("reload cancelled execution: %v", err)
	}
	if cancelledRec.Status != "cancelled" || cancelledRec.CancelledAt == nil || cancelledRec.FinishedAt == nil {
		t.Fatalf("unexpected cancel finalization: %#v", cancelledRec)
	}
}

func TestTaskExecutionStoreUsesContextScopeBeforeRuntimeScopeSession(t *testing.T) {
	baseDB := newTaskExecutionStoreTestDB(t)
	ctxDB := newTaskExecutionStoreTestDB(t)
	runtimeScope := &taskWorkerTestScope{
		ctx: context.Background(),
		cfg: &config.Config{
			Auth:   config.NewDefaultAuthConfig(),
			Server: config.NewDefaultServerConfig(),
		},
		db: baseDB,
	}
	ctx := scope.ContextWithScope(context.Background(), &taskWorkerTestScope{ctx: context.Background(), cfg: runtimeScope.cfg, db: ctxDB})
	store := taskExecutionStore{runtimeScope: runtimeScope, ctx: ctx}
	now := time.Now().UTC().Truncate(time.Millisecond)
	req := &executeJobRequest{
		JobId:             "job-context-bound",
		Attempt:           1,
		SchedulerUserId:   "scheduler-ctx",
		TriggeredByUserId: "user-ctx",
		FullMethod:        "task.Job/GetJob",
		Payload:           map[string]any{"source": "ctx"},
	}

	state, resp, err := store.tryStart(ctx, req, "owner-ctx", now.Add(time.Minute), now)
	if err != nil {
		t.Fatalf("tryStart() error = %v", err)
	}
	if state != "RUNNING" || resp.Status != "" {
		t.Fatalf("unexpected tryStart result: state=%q resp=%#v", state, resp)
	}

	var ctxCount int64
	if err := ctxDB.Model(&task.Execution{}).Where("job_id = ?", req.JobId).Count(&ctxCount).Error; err != nil {
		t.Fatalf("count ctx execution rows: %v", err)
	}
	if ctxCount != 1 {
		t.Fatalf("ctx execution row count = %d, want 1", ctxCount)
	}

	var baseCount int64
	if err := baseDB.Model(&task.Execution{}).Where("job_id = ?", req.JobId).Count(&baseCount).Error; err != nil {
		t.Fatalf("count base execution rows: %v", err)
	}
	if baseCount != 0 {
		t.Fatalf("base execution row count = %d, want 0", baseCount)
	}

	store.finalizeSuccess(ctx, req.JobId, "owner-ctx", map[string]any{"ok": true})

	var ctxRec task.Execution
	if err := ctxDB.Where("job_id = ?", req.JobId).First(&ctxRec).Error; err != nil {
		t.Fatalf("reload ctx execution: %v", err)
	}
	if ctxRec.Status != "succeeded" || ctxRec.FinishedAt == nil {
		t.Fatalf("unexpected ctx finalization: %#v", ctxRec)
	}

	if err := baseDB.Where("job_id = ?", req.JobId).First(&task.Execution{}).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected base db to remain untouched, got %v", err)
	}
}

func TestTaskExecutionStoreTryStartFallbackPaths(t *testing.T) {
	t.Run("duplicate create falls back to existing execution", func(t *testing.T) {
		db := newTaskExecutionStoreTestDB(t)
		now := time.Now().UTC().Truncate(time.Millisecond)
		leaseUntil := now.Add(time.Minute)
		if err := db.Create(&task.Execution{
			JobId:      "job-duplicate",
			Status:     "running",
			LeaseOwner: "owner-existing",
			LeaseUntil: &leaseUntil,
			CreatedAt:  now,
			UpdatedAt:  now,
		}).Error; err != nil {
			t.Fatalf("insert existing execution: %v", err)
		}
		if err := db.Callback().Create().Before("gorm:create").Register("taskworker_force_duplicate", func(tx *gorm.DB) {
			tx.AddError(gorm.ErrDuplicatedKey)
		}); err != nil {
			t.Fatalf("register duplicate callback: %v", err)
		}

		store := taskExecutionStore{runtimeScope: &taskWorkerTestScope{
			ctx: context.Background(),
			cfg: &config.Config{
				Auth:   config.NewDefaultAuthConfig(),
				Server: config.NewDefaultServerConfig(),
				Task: &config.TaskConfig{Worker: &config.TaskWorkerConfig{
					AlreadyRunningRetryAfterMaxMs: 250,
				}},
			},
			db: db,
		}}

		state, resp, err := store.tryStart(context.Background(), &executeJobRequest{JobId: "job-duplicate"}, "owner-next", now.Add(2*time.Minute), now)
		if err != nil {
			t.Fatalf("tryStart() duplicate error = %v", err)
		}
		if state != "ALREADY_RUNNING" || resp.Status != "ALREADY_RUNNING" {
			t.Fatalf("unexpected duplicate tryStart result: state=%q resp=%#v", state, resp)
		}
		if resp.RetryAfterMs <= 0 || resp.RetryAfterMs > 250 {
			t.Fatalf("unexpected duplicate retry_after_ms: %d", resp.RetryAfterMs)
		}
	})

	t.Run("create error returns failed retryable", func(t *testing.T) {
		dsn := fmt.Sprintf("file:task_execution_store_nomigrate_%d?mode=memory&cache=shared", time.Now().UnixNano())
		db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
		if err != nil {
			t.Fatalf("open sqlite: %v", err)
		}

		store := taskExecutionStore{runtimeScope: &taskWorkerTestScope{
			ctx: context.Background(),
			cfg: &config.Config{Auth: config.NewDefaultAuthConfig(), Server: config.NewDefaultServerConfig()},
			db:  db,
		}}
		state, resp, err := store.tryStart(context.Background(), &executeJobRequest{JobId: "job-no-table"}, "owner-1", time.Now().Add(time.Minute), time.Now().UTC())
		if err == nil {
			t.Fatal("expected tryStart() to return create error")
		}
		if state != "FAILED_RETRYABLE" || resp.Status != "" || resp.Result != nil || resp.Error != nil || resp.RetryAfterMs != 0 {
			t.Fatalf("unexpected failed create result: state=%q resp=%#v err=%v", state, resp, err)
		}
	})
}

func TestTaskExecutionStoreHandleExistingPaths(t *testing.T) {
	db := newTaskExecutionStoreTestDB(t)
	runtimeScope := &taskWorkerTestScope{
		ctx: context.Background(),
		cfg: &config.Config{
			Auth:   config.NewDefaultAuthConfig(),
			Server: config.NewDefaultServerConfig(),
			Task: &config.TaskConfig{Worker: &config.TaskWorkerConfig{
				AlreadyRunningRetryAfterMaxMs: 50,
				LeaseDurationMs:               75,
			}},
		},
		db: db,
	}
	store := taskExecutionStore{runtimeScope: runtimeScope}
	now := time.Now().UTC().Truncate(time.Millisecond)

	succeeded := task.Execution{
		JobId:      "job-succeeded",
		Status:     "succeeded",
		LeaseOwner: "owner-a",
		ResultJson: datatypes.JSON([]byte(`{"done":true}`)),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	running := task.Execution{
		JobId:      "job-running",
		Status:     "running",
		LeaseOwner: "owner-b",
		LeaseUntil: ptrTime(now.Add(time.Minute)),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	failed := task.Execution{
		JobId:      "job-failed",
		Status:     "failed",
		LeaseOwner: "owner-c",
		LeaseUntil: ptrTime(now.Add(-time.Minute)),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	expiredRunning := task.Execution{
		JobId:      "job-expired",
		Status:     "running",
		LeaseOwner: "owner-d",
		LeaseUntil: ptrTime(now.Add(-time.Minute)),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	cancelled := task.Execution{
		JobId:      "job-cancelled",
		Status:     "cancelled",
		LeaseOwner: "owner-e",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	for _, rec := range []task.Execution{succeeded, running, failed, expiredRunning, cancelled} {
		if err := db.Create(&rec).Error; err != nil {
			t.Fatalf("insert execution %s: %v", rec.JobId, err)
		}
	}

	state, resp, err := store.handleExisting(context.Background(), &executeJobRequest{JobId: "job-succeeded"}, "owner-next")
	if err != nil || state != "SUCCEEDED" || resp.Status != "SUCCEEDED" {
		t.Fatalf("unexpected succeeded handleExisting result: state=%q resp=%#v err=%v", state, resp, err)
	}
	if result, ok := resp.Result.(map[string]any); !ok || result["done"] != true {
		t.Fatalf("unexpected succeeded result payload: %#v", resp.Result)
	}

	state, resp, err = store.handleExisting(context.Background(), &executeJobRequest{JobId: "job-cancelled"}, "owner-next")
	if err != nil || state != "CANCELLED" || resp.Status != "CANCELLED" {
		t.Fatalf("unexpected cancelled handleExisting result: state=%q resp=%#v err=%v", state, resp, err)
	}

	state, resp, err = store.handleExisting(context.Background(), &executeJobRequest{JobId: "job-running"}, "owner-next")
	if err != nil || state != "ALREADY_RUNNING" || resp.Status != "ALREADY_RUNNING" {
		t.Fatalf("unexpected running handleExisting result: state=%q resp=%#v err=%v", state, resp, err)
	}
	if resp.RetryAfterMs <= 0 || resp.RetryAfterMs > 50 {
		t.Fatalf("expected retry_after_ms to be capped to 50ms, got %d", resp.RetryAfterMs)
	}

	retryReq := &executeJobRequest{JobId: "job-failed", Attempt: 3}
	state, resp, err = store.handleExisting(context.Background(), retryReq, "owner-retry")
	if err != nil || state != "RUNNING" || resp.Status != "" {
		t.Fatalf("unexpected failed->running result: state=%q resp=%#v err=%v", state, resp, err)
	}
	var rec task.Execution
	if err := db.Where("job_id = ?", retryReq.JobId).First(&rec).Error; err != nil {
		t.Fatalf("reload retried execution: %v", err)
	}
	if rec.Status != "running" || rec.LeaseOwner != "owner-retry" || rec.Attempt != 3 || rec.LeaseUntil == nil {
		t.Fatalf("unexpected retried execution row: %#v", rec)
	}

	expiredReq := &executeJobRequest{JobId: "job-expired", Attempt: 4}
	state, resp, err = store.handleExisting(context.Background(), expiredReq, "owner-expired")
	if err != nil || state != "RUNNING" || resp.Status != "" {
		t.Fatalf("unexpected expired running retry result: state=%q resp=%#v err=%v", state, resp, err)
	}
	var expiredRec task.Execution
	if err := db.Where("job_id = ?", expiredReq.JobId).First(&expiredRec).Error; err != nil {
		t.Fatalf("reload expired execution: %v", err)
	}
	if expiredRec.Status != "running" || expiredRec.LeaseOwner != "owner-expired" || expiredRec.Attempt != 4 || expiredRec.LeaseUntil == nil {
		t.Fatalf("unexpected expired execution row: %#v", expiredRec)
	}
}

func TestInvokeTargetMethodValidationPaths(t *testing.T) {
	svc := &ApplicationService{}
	runtimeScope := newHelperScope(t.TempDir())
	jsCtx := map[string]interface{}{}

	if _, err := svc.invokeTargetMethod(context.Background(), runtimeScope, jsCtx, nil, &executeJobRequest{FullMethod: "missing.Service/Call"}); status.Code(err) != codes.Unimplemented {
		t.Fatalf("expected missing method to be unimplemented, got %v", err)
	}

	loader.Global().RegisterProto("taskworker_stream/chat.proto", `syntax = "proto3";
package taskworker_stream;

service Chat {
  rpc Talk(stream ChatRequest) returns (ChatReply);
}

message ChatRequest { string message = 1; }
message ChatReply { string message = 1; }
`)
	_, err := svc.invokeTargetMethod(context.Background(), runtimeScope, jsCtx, nil, &executeJobRequest{FullMethod: "taskworker_stream.Chat/Talk"})
	if status.Code(err) != codes.Unimplemented {
		t.Fatalf("expected streaming method to be unsupported, got %v", err)
	}
}

func TestTaskWorkerMethodHandlerPaths(t *testing.T) {
	resetTaskWorkerProtoCaches()
	t.Cleanup(resetTaskWorkerProtoCaches)

	t.Run("descriptor errors map to internal status", func(t *testing.T) {
		svc := &ApplicationService{name: "task"}
		if _, _, _, _, err := taskWorkerStoreError("task", errors.New("boom")); err == nil {
			t.Fatal("expected taskWorkerStoreError to return error")
		}

		resp, err := svc.taskWorkerAdapter().methodHandler()(nil, context.Background(), func(any) error {
			t.Fatal("decoder should not be called when descriptors fail")
			return nil
		}, nil)
		if resp != nil || status.Code(err) != codes.Internal || status.Convert(err).Message() != "boom" {
			t.Fatalf("expected internal descriptor error, got resp=%#v err=%v", resp, err)
		}
	})

	resetTaskWorkerProtoCaches()

	t.Run("decode errors map to invalid argument", func(t *testing.T) {
		svc := &ApplicationService{name: "task"}
		resp, err := svc.taskWorkerAdapter().methodHandler()(nil, context.Background(), func(any) error {
			return errors.New("bad body")
		}, nil)
		if resp != nil || status.Code(err) != codes.InvalidArgument || !strings.Contains(status.Convert(err).Message(), "bad body") {
			t.Fatalf("expected invalid argument decode error, got resp=%#v err=%v", resp, err)
		}
	})

	t.Run("interceptor receives task worker full method", func(t *testing.T) {
		svc := &ApplicationService{name: "task"}
		intercepted := false
		resp, err := svc.taskWorkerAdapter().methodHandler()(nil, context.Background(), func(msg any) error {
			if _, ok := msg.(*dynamicpb.Message); !ok {
				t.Fatalf("expected dynamic request message, got %T", msg)
			}
			return nil
		}, func(ctx context.Context, req any, info *grpc.UnaryServerInfo, next grpc.UnaryHandler) (any, error) {
			intercepted = true
			if info.FullMethod != "/task.TaskWorker/ExecuteJob" {
				t.Fatalf("unexpected full method: %s", info.FullMethod)
			}
			return "intercepted", nil
		})
		if err != nil || resp != "intercepted" || !intercepted {
			t.Fatalf("unexpected interceptor result: resp=%#v err=%v intercepted=%v", resp, err, intercepted)
		}
	})
}

func TestTaskWorkerServiceDescPaths(t *testing.T) {
	resetTaskWorkerProtoCaches()
	t.Cleanup(resetTaskWorkerProtoCaches)

	t.Run("builds service desc with execute job method metadata", func(t *testing.T) {
		svc := &ApplicationService{name: "task"}
		desc, err := svc.taskWorkerAdapter().serviceDesc()
		if err != nil {
			t.Fatalf("taskWorkerServiceDesc() error = %v", err)
		}
		if desc.ServiceName != "task.TaskWorker" || desc.Metadata != "taskworker/task.proto" {
			t.Fatalf("unexpected service desc header: %#v", desc)
		}
		if len(desc.Methods) != 1 || desc.Methods[0].MethodName != "ExecuteJob" || desc.Methods[0].Handler == nil {
			t.Fatalf("unexpected service desc methods: %#v", desc.Methods)
		}
		if len(desc.Streams) != 0 {
			t.Fatalf("expected no streams, got %#v", desc.Streams)
		}
	})

	t.Run("descriptor errors propagate", func(t *testing.T) {
		svc := &ApplicationService{name: "task-service-desc-error"}
		if _, _, _, _, err := taskWorkerStoreError("task-service-desc-error", errors.New("desc boom")); err == nil {
			t.Fatal("expected taskWorkerStoreError to return error")
		}
		if desc, err := svc.taskWorkerAdapter().serviceDesc(); desc != nil || err == nil || err.Error() != "desc boom" {
			t.Fatalf("unexpected desc error result: desc=%#v err=%v", desc, err)
		}
	})
}

func TestTaskWorkerUnaryHandlerPaths(t *testing.T) {
	resetTaskWorkerProtoCaches()
	t.Cleanup(resetTaskWorkerProtoCaches)

	t.Run("rejects wrong request type", func(t *testing.T) {
		svc := &ApplicationService{name: "task", runtimeScope: &taskWorkerTestScope{ctx: context.Background(), cfg: &config.Config{Auth: config.NewDefaultAuthConfig(), Server: config.NewDefaultServerConfig()}}}
		resp, err := svc.taskWorkerAdapter().unaryHandler()(context.Background(), "bad")
		if resp != nil || status.Code(err) != codes.InvalidArgument {
			t.Fatalf("expected invalid request type error, got resp=%#v err=%v", resp, err)
		}
	})

	t.Run("invalid job token returns non retryable response", func(t *testing.T) {
		runtimeScope := &taskWorkerTestScope{ctx: context.Background(), cfg: &config.Config{Auth: config.NewDefaultAuthConfig(), Server: config.NewDefaultServerConfig()}, db: newTaskExecutionStoreTestDB(t)}
		svc := &ApplicationService{name: "task", runtimeScope: runtimeScope}
		resp, err := svc.taskWorkerAdapter().unaryHandler()(context.Background(), buildTaskWorkerReqMsg(t, "task", map[string]any{
			"job_id":               "job-invalid-token",
			"attempt":              1,
			"full_method":          "task.Job/GetJob",
			"scheduler_user_id":    "scheduler-1",
			"triggered_by_user_id": "user-1",
			"payload":              map[string]any{"job_id": "job-invalid-token"},
		}))
		if err != nil {
			t.Fatalf("unexpected unary handler error: %v", err)
		}
		respMap := taskWorkerRespMap(t, resp)
		if respMap["status"] != "EXECUTE_JOB_STATUS_FAILED_NON_RETRYABLE" {
			t.Fatalf("unexpected status: %#v", respMap)
		}
		errMap := requireAnyMap(t, respMap["error"])
		if fmt.Sprintf("%v", errMap["grpc_code"]) != fmt.Sprintf("%d", codes.Unauthenticated) {
			t.Fatalf("unexpected grpc code: %#v", errMap)
		}
	})

	t.Run("non internal caller with valid token executes target method", func(t *testing.T) {
		loader.Global().RegisterProto("task/taskworker_success_echo.proto", `syntax = "proto3";
package task;

service WorkerEcho {
	rpc Run(WorkerEchoRunReq) returns (WorkerEchoRunResp);
}

message WorkerEchoRunReq { string message = 1; }
message WorkerEchoRunResp { string result = 1; }
`)

		cfg := &config.Config{Auth: config.NewDefaultAuthConfig(), Server: config.NewDefaultServerConfig()}
		cfg.Auth.Enabled = false
		runtimeScope := &taskWorkerTestScope{ctx: context.Background(), cfg: cfg, db: newTaskExecutionStoreTestDB(t)}
		executor := newACLTestExecutor(t, runtimeScope, func(ctx context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
			if req.Service != "task.WorkerEcho.Run" {
				return nil, fmt.Errorf("unexpected JS service: %s", req.Service)
			}
			return &jsengine.JsResponse{Id: req.Id, Result: "done-non-internal"}, nil
		})
		svc := &ApplicationService{name: "task", runtimeScope: runtimeScope, jsExecutor: executor}
		req := &executeJobRequest{
			JobId:             "job-valid-token-success",
			Attempt:           2,
			FullMethod:        "task.WorkerEcho/Run",
			SchedulerUserId:   "scheduler-1",
			TriggeredByUserId: "user-1",
		}
		ctx := auth.ContextWithIdentity(context.Background(), &testIdentity{userID: "scheduler-1", tokenID: "tok-1", meta: map[string]any{
			"purpose":           "task_job",
			"jobId":             req.JobId,
			"fullMethod":        req.FullMethod,
			"targetApp":         "task",
			"schedulerUserId":   req.SchedulerUserId,
			"triggeredByUserId": req.TriggeredByUserId,
			"attempt":           float64(req.Attempt),
		}})

		resp, err := svc.taskWorkerAdapter().unaryHandler()(ctx, buildTaskWorkerReqMsg(t, "task", map[string]any{
			"job_id":               req.JobId,
			"attempt":              req.Attempt,
			"full_method":          req.FullMethod,
			"scheduler_user_id":    req.SchedulerUserId,
			"triggered_by_user_id": req.TriggeredByUserId,
			"payload":              map[string]any{"message": "hello"},
		}))
		if err != nil {
			t.Fatalf("unexpected unary handler error: %v", err)
		}
		respMap := taskWorkerRespMap(t, resp)
		if respMap["status"] != "EXECUTE_JOB_STATUS_SUCCEEDED" {
			t.Fatalf("unexpected status: %#v", respMap)
		}
		result := requireAnyMap(t, respMap["result"])
		if result["result"] != "done-non-internal" {
			t.Fatalf("unexpected result payload: %#v", result)
		}
	})

	t.Run("internal caller rejects target app mismatch", func(t *testing.T) {
		cfg := &config.Config{Auth: config.NewDefaultAuthConfig(), Server: config.NewDefaultServerConfig()}
		cfg.Auth.InternalKey = "secret"
		runtimeScope := &taskWorkerTestScope{ctx: context.Background(), cfg: cfg, db: newTaskExecutionStoreTestDB(t)}
		svc := &ApplicationService{name: "task", runtimeScope: runtimeScope}
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(internalKeyHeader, "secret"))
		resp, err := svc.taskWorkerAdapter().unaryHandler()(ctx, buildTaskWorkerReqMsg(t, "task", map[string]any{
			"job_id":               "job-app-mismatch",
			"attempt":              1,
			"full_method":          "other.Job/GetJob",
			"scheduler_user_id":    "scheduler-1",
			"triggered_by_user_id": "user-1",
			"payload":              map[string]any{"job_id": "job-app-mismatch"},
		}))
		if err != nil {
			t.Fatalf("unexpected unary handler error: %v", err)
		}
		respMap := taskWorkerRespMap(t, resp)
		if respMap["status"] != "EXECUTE_JOB_STATUS_FAILED_NON_RETRYABLE" {
			t.Fatalf("unexpected status: %#v", respMap)
		}
		errMap := requireAnyMap(t, respMap["error"])
		if fmt.Sprintf("%v", errMap["grpc_code"]) != fmt.Sprintf("%d", codes.InvalidArgument) {
			t.Fatalf("unexpected grpc code: %#v", errMap)
		}
	})

	t.Run("internal caller executes target method and finalizes success", func(t *testing.T) {
		loader.Global().RegisterProto("task/taskworker_success_echo.proto", `syntax = "proto3";
package task;

service WorkerEcho {
	rpc Run(WorkerEchoRunReq) returns (WorkerEchoRunResp);
}

message WorkerEchoRunReq { string message = 1; }
message WorkerEchoRunResp { string result = 1; }
`)

		cfg := &config.Config{Auth: config.NewDefaultAuthConfig(), Server: config.NewDefaultServerConfig()}
		cfg.Auth.Enabled = false
		cfg.Auth.InternalKey = "secret"
		runtimeScope := &taskWorkerTestScope{ctx: context.Background(), cfg: cfg, db: newTaskExecutionStoreTestDB(t)}
		executor := newACLTestExecutor(t, runtimeScope, func(ctx context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
			if req.Service != "task.WorkerEcho.Run" {
				return nil, fmt.Errorf("unexpected JS service: %s", req.Service)
			}
			if len(req.Args) != 1 {
				return nil, fmt.Errorf("unexpected JS args: %#v", req.Args)
			}
			arg, ok := req.Args[0].(string)
			if !ok || arg != "hello" {
				return nil, fmt.Errorf("unexpected JS payload: %#v", req.Args)
			}
			return &jsengine.JsResponse{Id: req.Id, Result: "done"}, nil
		})
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(internalKeyHeader, "secret"))
		svc := &ApplicationService{name: "task", runtimeScope: runtimeScope, jsExecutor: executor}

		resp, err := svc.taskWorkerAdapter().unaryHandler()(ctx, buildTaskWorkerReqMsg(t, "task", map[string]any{
			"job_id":               "job-success",
			"attempt":              1,
			"full_method":          "task.WorkerEcho/Run",
			"scheduler_user_id":    "scheduler-1",
			"triggered_by_user_id": "user-1",
			"payload":              map[string]any{"message": "hello"},
		}))
		if err != nil {
			t.Fatalf("unexpected unary handler error: %v", err)
		}
		respMap := taskWorkerRespMap(t, resp)
		if respMap["status"] != "EXECUTE_JOB_STATUS_SUCCEEDED" {
			t.Fatalf("unexpected status: %#v", respMap)
		}
		result := requireAnyMap(t, respMap["result"])
		if result["result"] != "done" {
			t.Fatalf("unexpected result payload: %#v", result)
		}

		var rec task.Execution
		if err := runtimeScope.db.Where("job_id = ?", "job-success").First(&rec).Error; err != nil {
			t.Fatalf("load execution: %v", err)
		}
		if rec.Status != "succeeded" || rec.FinishedAt == nil {
			t.Fatalf("unexpected execution state: %#v", rec)
		}
	})

	t.Run("internal caller returns already running existing response", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Millisecond)
		leaseUntil := now.Add(time.Minute)
		db := newTaskExecutionStoreTestDB(t)
		if err := db.Create(&task.Execution{
			JobId:      "job-already-running",
			Status:     "running",
			LeaseOwner: "owner-existing",
			LeaseUntil: &leaseUntil,
			CreatedAt:  now,
			UpdatedAt:  now,
		}).Error; err != nil {
			t.Fatalf("insert existing execution: %v", err)
		}
		if err := db.Callback().Create().Before("gorm:create").Register("taskworker_force_duplicate_unary", func(tx *gorm.DB) {
			tx.AddError(gorm.ErrDuplicatedKey)
		}); err != nil {
			t.Fatalf("register duplicate callback: %v", err)
		}

		cfg := &config.Config{Auth: config.NewDefaultAuthConfig(), Server: config.NewDefaultServerConfig()}
		cfg.Auth.InternalKey = "secret"
		cfg.Task = &config.TaskConfig{Worker: &config.TaskWorkerConfig{AlreadyRunningRetryAfterMaxMs: 250}}
		runtimeScope := &taskWorkerTestScope{ctx: context.Background(), cfg: cfg, db: db}
		svc := &ApplicationService{name: "task", runtimeScope: runtimeScope}
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(internalKeyHeader, "secret"))

		resp, err := svc.taskWorkerAdapter().unaryHandler()(ctx, buildTaskWorkerReqMsg(t, "task", map[string]any{
			"job_id":               "job-already-running",
			"attempt":              1,
			"full_method":          "task.Job/GetJob",
			"scheduler_user_id":    "scheduler-1",
			"triggered_by_user_id": "user-1",
			"payload":              map[string]any{"job_id": "job-already-running"},
		}))
		if err != nil {
			t.Fatalf("unexpected unary handler error: %v", err)
		}
		respMap := taskWorkerRespMap(t, resp)
		if respMap["status"] != "EXECUTE_JOB_STATUS_ALREADY_RUNNING" {
			t.Fatalf("unexpected status: %#v", respMap)
		}
		if retryAfter := fmt.Sprintf("%v", respMap["retry_after_ms"]); retryAfter != "250" {
			t.Fatalf("unexpected retry_after_ms: %#v", respMap["retry_after_ms"])
		}
	})

	t.Run("internal caller maps tryStart errors to failed retryable response", func(t *testing.T) {
		dsn := fmt.Sprintf("file:taskworker_unary_no_table_%d?mode=memory&cache=shared", time.Now().UnixNano())
		db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
		if err != nil {
			t.Fatalf("open sqlite: %v", err)
		}
		cfg := &config.Config{Auth: config.NewDefaultAuthConfig(), Server: config.NewDefaultServerConfig()}
		cfg.Auth.InternalKey = "secret"
		runtimeScope := &taskWorkerTestScope{ctx: context.Background(), cfg: cfg, db: db}
		svc := &ApplicationService{name: "task", runtimeScope: runtimeScope}
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(internalKeyHeader, "secret"))

		resp, err := svc.taskWorkerAdapter().unaryHandler()(ctx, buildTaskWorkerReqMsg(t, "task", map[string]any{
			"job_id":               "job-try-start-error",
			"attempt":              1,
			"full_method":          "task.Job/GetJob",
			"scheduler_user_id":    "scheduler-1",
			"triggered_by_user_id": "user-1",
			"payload":              map[string]any{"job_id": "job-try-start-error"},
		}))
		if err != nil {
			t.Fatalf("unexpected unary handler error: %v", err)
		}
		respMap := taskWorkerRespMap(t, resp)
		if respMap["status"] != "EXECUTE_JOB_STATUS_FAILED_RETRYABLE" {
			t.Fatalf("unexpected status: %#v", respMap)
		}
		errMap := requireAnyMap(t, respMap["error"])
		if fmt.Sprintf("%v", errMap["grpc_code"]) != fmt.Sprintf("%d", codes.Unknown) {
			t.Fatalf("unexpected grpc code: %#v", errMap)
		}
	})

	t.Run("internal caller cancels before invoke when task service reports cancelled", func(t *testing.T) {
		loader.Global().RegisterProto("task/taskworker_cancel_getjob.proto", `syntax = "proto3";
package task;

service Job {
	rpc GetJob(GetJobReq) returns (GetJobResp);
}

message GetJobReq {
	string job_id = 1;
	repeated string fields = 2;
}

message JobRecord {
	string cancelRequestedAt = 1;
	string status = 2;
}

message GetJobResp {
	JobRecord job = 1;
}
`)
		md, err := loader.Global().GetMethodDescriptor("task.Job.GetJob")
		if err != nil {
			t.Fatalf("GetMethodDescriptor(task.Job.GetJob) error = %v", err)
		}

		lis := bufconn.Listen(1024 * 1024)
		server := grpc.NewServer()
		serviceDesc := &grpc.ServiceDesc{
			ServiceName: "task.Job",
			HandlerType: (*interface{})(nil),
			Methods: []grpc.MethodDesc{{
				MethodName: "GetJob",
				Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
					reqMsg := dynamicpb.NewMessage(md.Input())
					if err := dec(reqMsg); err != nil {
						return nil, err
					}
					respMsg := dynamicpb.NewMessage(md.Output())
					jobField := respMsg.Descriptor().Fields().ByName("job")
					jobMsg := dynamicpb.NewMessage(jobField.Message())
					if err := converter.MapToMessage(map[string]any{"status": "cancelled"}, jobMsg); err != nil {
						return nil, err
					}
					respMsg.Set(jobField, protoreflect.ValueOfMessage(jobMsg))
					return respMsg, nil
				},
			}},
		}
		server.RegisterService(serviceDesc, nil)
		go func() { _ = server.Serve(lis) }()
		t.Cleanup(server.Stop)

		var conn *grpc.ClientConn
		dialer := func(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
			if serviceName != "task.Job" {
				t.Fatalf("serviceName = %q, want task.Job", serviceName)
			}
			if conn != nil {
				return conn, nil
			}
			c, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return lis.Dial()
			}), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				return nil, err
			}
			conn = c
			return conn, nil
		}
		t.Cleanup(func() {
			if conn != nil {
				_ = conn.Close()
			}
		})

		cfg := &config.Config{Auth: config.NewDefaultAuthConfig(), Server: config.NewDefaultServerConfig()}
		cfg.Auth.InternalKey = "secret"
		runtimeScope := &taskWorkerTestScope{ctx: grpcclient.ContextWithServiceDialer(context.Background(), dialer), cfg: cfg, db: newTaskExecutionStoreTestDB(t)}
		svc := &ApplicationService{name: "task", runtimeScope: runtimeScope}
		ctx := metadata.NewIncomingContext(runtimeScope.ctx, metadata.Pairs(internalKeyHeader, "secret"))

		resp, err := svc.taskWorkerAdapter().unaryHandler()(ctx, buildTaskWorkerReqMsg(t, "task", map[string]any{
			"job_id":               "job-cancel-before-invoke",
			"attempt":              1,
			"full_method":          "task.WorkerEcho/Run",
			"scheduler_user_id":    "scheduler-1",
			"triggered_by_user_id": "user-1",
			"payload":              map[string]any{"message": "hello"},
		}))
		if err != nil {
			t.Fatalf("unexpected unary handler error: %v", err)
		}
		respMap := taskWorkerRespMap(t, resp)
		if respMap["status"] != "EXECUTE_JOB_STATUS_CANCELLED" {
			t.Fatalf("unexpected status: %#v", respMap)
		}

		var rec task.Execution
		if err := runtimeScope.db.Where("job_id = ?", "job-cancel-before-invoke").First(&rec).Error; err != nil {
			t.Fatalf("load execution: %v", err)
		}
		if rec.Status != "cancelled" || rec.CancelledAt == nil || rec.FinishedAt == nil {
			t.Fatalf("unexpected cancelled execution state: %#v", rec)
		}
	})

	t.Run("internal caller maps execution errors to retryable response and finalizes failure", func(t *testing.T) {
		loader.Global().RegisterProto("task/taskworker_exec_error_echo.proto", `syntax = "proto3";
package task;

service WorkerEchoRetry {
	rpc Run(WorkerEchoRetryRunReq) returns (WorkerEchoRetryRunResp);
}

message WorkerEchoRetryRunReq { string message = 1; }
message WorkerEchoRetryRunResp { string result = 1; }
`)

		cfg := &config.Config{Auth: config.NewDefaultAuthConfig(), Server: config.NewDefaultServerConfig()}
		cfg.Auth.Enabled = false
		cfg.Auth.InternalKey = "secret"
		runtimeScope := &taskWorkerTestScope{ctx: context.Background(), cfg: cfg, db: newTaskExecutionStoreTestDB(t)}
		executor := newACLTestExecutor(t, runtimeScope, func(ctx context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
			return nil, status.Error(codes.Unavailable, "upstream unavailable")
		})
		svc := &ApplicationService{name: "task", runtimeScope: runtimeScope, jsExecutor: executor}
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(internalKeyHeader, "secret"))

		resp, err := svc.taskWorkerAdapter().unaryHandler()(ctx, buildTaskWorkerReqMsg(t, "task", map[string]any{
			"job_id":               "job-exec-error",
			"attempt":              1,
			"full_method":          "task.WorkerEchoRetry/Run",
			"scheduler_user_id":    "scheduler-1",
			"triggered_by_user_id": "user-1",
			"payload":              map[string]any{"message": "hello"},
		}))
		if err != nil {
			t.Fatalf("unexpected unary handler error: %v", err)
		}
		respMap := taskWorkerRespMap(t, resp)
		if respMap["status"] != "EXECUTE_JOB_STATUS_FAILED_RETRYABLE" {
			t.Fatalf("unexpected status: %#v", respMap)
		}
		errMap := requireAnyMap(t, respMap["error"])
		if fmt.Sprintf("%v", errMap["grpc_code"]) != fmt.Sprintf("%d", codes.Unavailable) {
			t.Fatalf("unexpected grpc code: %#v", errMap)
		}

		var rec task.Execution
		if err := runtimeScope.db.Where("job_id = ?", "job-exec-error").First(&rec).Error; err != nil {
			t.Fatalf("load execution: %v", err)
		}
		if rec.Status != "failed" || rec.FinishedAt == nil {
			t.Fatalf("unexpected failed execution state: %#v", rec)
		}
	})
}

func TestIsCancelRequestedViaTaskResponsePaths(t *testing.T) {
	loader.Global().RegisterProto("task/taskworker_cancel_getjob.proto", `syntax = "proto3";
package task;

service Job {
	rpc GetJob(GetJobReq) returns (GetJobResp);
}

message GetJobReq {
	string job_id = 1;
	repeated string fields = 2;
}

message JobRecord {
	string cancelRequestedAt = 1;
	string status = 2;
}

message GetJobResp {
	JobRecord job = 1;
}
`)
	md, err := loader.Global().GetMethodDescriptor("task.Job.GetJob")
	if err != nil {
		t.Fatalf("GetMethodDescriptor(task.Job.GetJob) error = %v", err)
	}

	newCancelCheckStore := func(t *testing.T, envName string, response map[string]any, invokeErr error) (taskExecutionStore, context.Context, *metadata.MD) {
		t.Helper()

		lis := bufconn.Listen(1024 * 1024)
		server := grpc.NewServer()
		var seenMD metadata.MD
		serviceDesc := &grpc.ServiceDesc{
			ServiceName: "task.Job",
			HandlerType: (*interface{})(nil),
			Methods: []grpc.MethodDesc{{
				MethodName: "GetJob",
				Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
					if mdIn, ok := metadata.FromIncomingContext(ctx); ok {
						seenMD = mdIn
					}
					reqMsg := dynamicpb.NewMessage(md.Input())
					if err := dec(reqMsg); err != nil {
						return nil, err
					}
					if invokeErr != nil {
						return nil, invokeErr
					}
					respMsg := dynamicpb.NewMessage(md.Output())
					if err := converter.MapToMessage(response, respMsg); err != nil {
						return nil, err
					}
					return respMsg, nil
				},
			}},
		}
		server.RegisterService(serviceDesc, nil)
		go func() { _ = server.Serve(lis) }()
		t.Cleanup(server.Stop)

		var conn *grpc.ClientConn
		dialer := func(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
			if serviceName != "task.Job" {
				t.Fatalf("serviceName = %q, want task.Job", serviceName)
			}
			if conn != nil {
				return conn, nil
			}
			c, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return lis.Dial()
			}), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				return nil, err
			}
			conn = c
			return conn, nil
		}
		t.Cleanup(func() {
			if conn != nil {
				_ = conn.Close()
			}
		})

		runtimeScope := &taskWorkerTestScope{
			ctx: grpcclient.ContextWithServiceDialer(context.Background(), dialer),
			cfg: &config.Config{
				Auth:   &config.AuthConfig{InternalKey: "secret"},
				Server: &config.ServerConfig{Environment: envName},
			},
		}
		return taskExecutionStore{runtimeScope: runtimeScope}, runtimeScope.ctx, &seenMD
	}

	t.Run("cancelled status returns true true without internal header in production", func(t *testing.T) {
		store, ctx, seenMD := newCancelCheckStore(t, "production", map[string]any{
			"job": map[string]any{"status": "cancelled"},
		}, nil)

		cancelled, ok := store.isCancelRequestedViaTask(ctx, "job-status-cancelled")
		if !ok || !cancelled {
			t.Fatalf("expected cancelled status to return true,true, got %v,%v", cancelled, ok)
		}
		if got := seenMD.Get(internalKeyHeader); len(got) != 0 {
			t.Fatalf("did not expect internal auth metadata in production, got %#v", got)
		}
	})

	t.Run("cancel requested timestamp returns true true and injects internal header outside production", func(t *testing.T) {
		store, ctx, seenMD := newCancelCheckStore(t, "development", map[string]any{
			"job": map[string]any{"cancelRequestedAt": time.Now().UTC().Format(time.RFC3339)},
		}, nil)

		cancelled, ok := store.isCancelRequestedViaTask(ctx, "job-cancel-requested-at")
		if !ok || !cancelled {
			t.Fatalf("expected cancelRequestedAt to return true,true, got %v,%v", cancelled, ok)
		}
		if got := seenMD.Get(internalKeyHeader); len(got) != 1 || got[0] != "secret" {
			t.Fatalf("expected internal auth metadata in development, got %#v", got)
		}
	})

	t.Run("invoke error returns false false", func(t *testing.T) {
		store, ctx, _ := newCancelCheckStore(t, "development", nil, status.Error(codes.NotFound, "missing"))

		cancelled, ok := store.isCancelRequestedViaTask(ctx, "job-invoke-error")
		if cancelled || ok {
			t.Fatalf("expected invoke error to return false,false, got %v,%v", cancelled, ok)
		}
	})

	t.Run("response without cancellation markers returns false true", func(t *testing.T) {
		store, ctx, _ := newCancelCheckStore(t, "development", map[string]any{
			"job": map[string]any{},
		}, nil)

		cancelled, ok := store.isCancelRequestedViaTask(ctx, "job-not-cancelled")
		if cancelled || !ok {
			t.Fatalf("expected no cancellation markers to return false,true, got %v,%v", cancelled, ok)
		}
	})
}

func TestInvokeTargetMethodSuccessPath(t *testing.T) {
	loader.Global().RegisterProto("task/taskworker_invoke_echo.proto", `syntax = "proto3";
package task;

service InvokeEcho {
	rpc Run(InvokeEchoRunReq) returns (InvokeEchoRunResp);
}

message InvokeEchoRunReq { string message = 1; }
message InvokeEchoRunResp { string result = 1; }
`)

	runtimeScope := newHelperScope(t.TempDir())
	executor := newACLTestExecutor(t, runtimeScope, func(ctx context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
		if req.Service != "task.InvokeEcho.Run" {
			return nil, fmt.Errorf("unexpected JS service: %s", req.Service)
		}
		return &jsengine.JsResponse{Id: req.Id, Result: "done"}, nil
	})
	svc := &ApplicationService{runtimeScope: runtimeScope, jsExecutor: executor}
	jsCtx := map[string]any{"req": map[string]any{"depth": 0}}
	resp, err := svc.invokeTargetMethod(context.Background(), runtimeScope, jsCtx, nil, &executeJobRequest{
		FullMethod: "task.InvokeEcho/Run",
		Payload:    map[string]any{"message": "hello"},
		TimeoutMs:  5,
	})
	if err != nil {
		t.Fatalf("invokeTargetMethod() error = %v", err)
	}
	result, ok := resp.(map[string]any)
	if !ok || result["result"] != "done" {
		t.Fatalf("unexpected invokeTargetMethod result: %#v", resp)
	}
}

func buildTaskWorkerReqMsg(t *testing.T, appName string, fields map[string]any) *dynamicpb.Message {
	t.Helper()
	_, reqDesc, _, _, err := taskWorkerDescriptors(appName)
	if err != nil {
		t.Fatalf("taskWorkerDescriptors(%q) error = %v", appName, err)
	}
	msg := dynamicpb.NewMessage(reqDesc)
	if err := converter.MapToMessage(fields, msg); err != nil {
		t.Fatalf("MapToMessage(task worker req) error = %v", err)
	}
	return msg
}

func taskWorkerRespMap(t *testing.T, resp any) map[string]any {
	t.Helper()
	protoMsg, ok := resp.(interface{ ProtoReflect() protoreflect.Message })
	if !ok {
		t.Fatalf("response does not implement ProtoReflect: %T", resp)
	}
	respMap, err := converter.MessageToMap(protoMsg.ProtoReflect())
	if err != nil {
		t.Fatalf("converter.MessageToMap(response) error = %v", err)
	}
	return respMap
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func cloneAnyMap(in map[string]any, key string, value any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	out[key] = value
	return out
}
