// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/rs/xid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// taskWorkerAdapter owns the ExecuteJob protocol entry, worker descriptor usage,
// and worker response envelope mapping for internal/service.
type taskWorkerAdapter struct {
	appName      string
	runtimeScope scope.Scope
	runtimeOpts  runtimeOptions
	runtime      invocationRuntime
	guard        serviceGuard
	emitMetric   func(string, map[string]any)
}

func (s *ApplicationService) taskWorkerAdapter() taskWorkerAdapter {
	return taskWorkerAdapter{
		appName:      s.name,
		runtimeScope: s.runtimeScope,
		runtimeOpts:  s.resolvedRuntimeOptions(),
		runtime:      s.runtime(),
		guard:        s.guard(),
		emitMetric:   s.emitTaskMetric,
	}
}

func (a taskWorkerAdapter) serviceDesc() (*grpc.ServiceDesc, error) {
	md, _, _, _, err := serviceCodec.taskWorker(a.appName)
	if err != nil {
		return nil, err
	}
	serviceName := a.appName + ".TaskWorker"
	return &grpc.ServiceDesc{
		ServiceName: serviceName,
		HandlerType: (*interface{})(nil),
		Methods: []grpc.MethodDesc{{
			MethodName: string(md.Name()),
			Handler:    a.methodHandler(),
		}},
		Streams:  []grpc.StreamDesc{},
		Metadata: taskWorkerProtoPath(a.appName),
	}, nil
}

func (a taskWorkerAdapter) methodHandler() func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	unaryHandler := a.unaryHandler()
	return func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
		_, reqDesc, _, _, err := serviceCodec.taskWorker(a.appName)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		reqMsg := serviceCodec.newMessage(reqDesc)
		if err := dec(reqMsg); err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid request format: %v", err))
		}
		if interceptor == nil {
			return unaryHandler(ctx, reqMsg)
		}
		info := &grpc.UnaryServerInfo{
			Server:     srv,
			FullMethod: "/" + a.appName + ".TaskWorker/ExecuteJob",
		}
		return interceptor(ctx, reqMsg, info, unaryHandler)
	}
}

func (a taskWorkerAdapter) unaryHandler() grpc.UnaryHandler {
	return func(ctx context.Context, req any) (any, error) {
		reqMsg, ok := req.(*dynamicpb.Message)
		if !ok {
			return nil, status.Error(codes.InvalidArgument, "invalid request type")
		}
		jobReq, err := a.parseExecuteJobReq(reqMsg)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		internalCaller := false
		ctx, internalCaller, err = a.guard.authorizeExecuteJobCaller(ctx, jobReq)
		if err != nil {
			return a.buildExecuteJobResp(executeJobResponse{
				Status: "FAILED_NON_RETRYABLE",
				Error:  errToTaskError(err),
			})
		}

		runtimeScope := a.runtimeScope.WithContext(ctx)
		jsCtx := a.runtime.buildJsContext(ctx)
		if reqMeta, ok := getJsReqMeta(jsCtx); ok {
			reqMeta["depth"] = 0
		}
		if ctxMap, ok := jsCtx["ctx"].(map[string]any); ok {
			ctxMap["schedulerUserId"] = jobReq.SchedulerUserId
			ctxMap["triggeredByUserId"] = jobReq.TriggeredByUserId
			ctxMap["jobId"] = jobReq.JobId
		}

		routing, err := a.guard.authorizeExecuteJobMethod(ctx, runtimeScope, jsCtx, jobReq, internalCaller)
		if err != nil {
			return a.buildExecuteJobResp(executeJobResponse{
				Status: "FAILED_NON_RETRYABLE",
				Error:  errToTaskError(err),
			})
		}

		execStore := taskExecutionStore{runtimeScope: a.runtimeScope, runtimeOpts: a.runtimeOpts, ctx: ctx}
		jobOwner := xid.New().String()
		leaseUntil := time.Now().UTC().Add(execStore.leaseDuration())
		startedAt := time.Now().UTC()

		startStatus, existing, err := execStore.tryStart(ctx, jobReq, jobOwner, leaseUntil, startedAt)
		if err != nil {
			return a.buildExecuteJobResp(executeJobResponse{
				Status: "FAILED_RETRYABLE",
				Error:  errToTaskError(err),
			})
		}
		if startStatus == "SUCCEEDED" || startStatus == "CANCELLED" || startStatus == "ALREADY_RUNNING" {
			return a.buildExecuteJobResp(existing)
		}

		ctxExec, cancel := context.WithCancel(ctx)
		defer cancel()
		headbeatStop := make(chan struct{})
		cancelled := make(chan struct{}, 1)
		leaseLost := make(chan struct{}, 1)
		go execStore.heartbeat(ctxExec, jobReq.JobId, jobOwner, headbeatStop, cancelled, cancel, leaseLost)
		go execStore.watchCancel(ctxExec, jobReq.JobId, cancelled, cancel, execStore.isCancelRequestedViaTask)
		defer close(headbeatStop)

		if cancelled, ok := execStore.isCancelRequestedViaTask(ctxExec, jobReq.JobId); ok && cancelled {
			cancel()
			execStore.finalizeCancelled(ctx, jobReq.JobId, jobOwner)
			return a.buildExecuteJobResp(executeJobResponse{Status: "CANCELLED"})
		}

		result, execErr := a.runtime.invokeTargetMethod(ctxExec, runtimeScope, jsCtx, routing, jobReq)
		select {
		case <-leaseLost:
			leaseErr := status.Error(codes.Aborted, "lease lost")
			execStore.finalizeFailure(ctx, jobReq.JobId, jobOwner, errToTaskError(leaseErr))
			return a.buildExecuteJobResp(executeJobResponse{Status: "FAILED_RETRYABLE", Error: errToTaskError(leaseErr)})
		case <-cancelled:
			execStore.finalizeCancelled(ctx, jobReq.JobId, jobOwner)
			return a.buildExecuteJobResp(executeJobResponse{Status: "CANCELLED"})
		default:
		}

		execDuration := time.Since(startedAt)
		a.emitTaskMetric("task_execute_duration", map[string]any{"target_app": a.appName, "full_method": jobReq.FullMethod, "duration_ms": execDuration.Milliseconds()})
		if execErr != nil {
			mapped := a.guard.mapExecutionError(execErr)
			a.emitTaskMetric("task_execute_result", map[string]any{"target_app": a.appName, "full_method": jobReq.FullMethod, "status": strings.ToLower(mapped.Status)})
			execStore.finalizeFailure(ctx, jobReq.JobId, jobOwner, mapped.Error)
			return a.buildExecuteJobResp(mapped)
		}

		a.emitTaskMetric("task_execute_result", map[string]any{"target_app": a.appName, "full_method": jobReq.FullMethod, "status": "succeeded"})
		execStore.finalizeSuccess(ctx, jobReq.JobId, jobOwner, result)
		return a.buildExecuteJobResp(executeJobResponse{Status: "SUCCEEDED", Result: result})
	}
}

func (a taskWorkerAdapter) parseExecuteJobReq(reqMsg *dynamicpb.Message) (*executeJobRequest, error) {
	reqMap, err := serviceCodec.messageToMap(reqMsg)
	if err != nil {
		return nil, err
	}
	jobID := strings.TrimSpace(fmt.Sprintf("%v", reqMap["job_id"]))
	fullMethod := strings.TrimSpace(fmt.Sprintf("%v", reqMap["full_method"]))
	schedulerUserID := strings.TrimSpace(fmt.Sprintf("%v", reqMap["scheduler_user_id"]))
	triggeredByUserID := strings.TrimSpace(fmt.Sprintf("%v", reqMap["triggered_by_user_id"]))
	if jobID == "" || fullMethod == "" || schedulerUserID == "" || triggeredByUserID == "" {
		return nil, fmt.Errorf("missing required fields")
	}
	attempt := 0
	if value, ok := reqMap["attempt"]; ok {
		switch typed := value.(type) {
		case int:
			attempt = typed
		case int32:
			attempt = int(typed)
		case int64:
			attempt = int(typed)
		case float64:
			attempt = int(typed)
		}
	}
	payload := map[string]interface{}{}
	if value, ok := reqMap["payload"]; ok {
		if payloadMap, ok := value.(map[string]interface{}); ok {
			payload = payloadMap
		}
	}
	timeoutMS := int64(0)
	if value, ok := reqMap["timeout_ms"]; ok {
		switch typed := value.(type) {
		case int64:
			timeoutMS = typed
		case int32:
			timeoutMS = int64(typed)
		case int:
			timeoutMS = int64(typed)
		case float64:
			timeoutMS = int64(typed)
		}
	}

	return &executeJobRequest{
		JobId:             jobID,
		Attempt:           attempt,
		FullMethod:        fullMethod,
		Payload:           payload,
		SchedulerUserId:   schedulerUserID,
		TriggeredByUserId: triggeredByUserID,
		TimeoutMs:         timeoutMS,
	}, nil
}

func (a taskWorkerAdapter) buildExecuteJobResp(resp executeJobResponse) (any, error) {
	_, _, respDesc, errDesc, err := serviceCodec.taskWorker(a.appName)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	out := serviceCodec.newMessage(respDesc)
	fields := respDesc.Fields()
	if field := fields.ByName("status"); field != nil {
		out.Set(field, protoreflect.ValueOfEnum(a.statusToEnum(resp.Status)))
	}
	if field := fields.ByName("retry_after_ms"); field != nil {
		out.Set(field, protoreflect.ValueOfInt64(resp.RetryAfterMs))
	}
	if resp.Result != nil {
		if field := fields.ByName("result"); field != nil {
			valueMsg := serviceCodec.newMessage(field.Message())
			if err := serviceCodec.anyToMessage(resp.Result, valueMsg); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "encode job result: %v", err)
			}
			out.Set(field, protoreflect.ValueOfMessage(valueMsg))
		}
	}
	if resp.Error != nil && errDesc != nil {
		if field := fields.ByName("error"); field != nil {
			errMsg := serviceCodec.newMessage(errDesc)
			if err := a.setTaskError(errMsg, resp.Error); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "encode job error: %v", err)
			}
			out.Set(field, protoreflect.ValueOfMessage(errMsg))
		}
	}
	return out, nil
}

func (a taskWorkerAdapter) setTaskError(msg *dynamicpb.Message, errMap map[string]any) error {
	fields := msg.Descriptor().Fields()
	if field := fields.ByName("grpc_code"); field != nil {
		if value, ok := errMap["grpc_code"]; ok {
			switch typed := value.(type) {
			case int32:
				msg.Set(field, protoreflect.ValueOfInt32(typed))
			case int64:
				msg.Set(field, protoreflect.ValueOfInt32(int32(typed)))
			case float64:
				msg.Set(field, protoreflect.ValueOfInt32(int32(typed)))
			}
		}
	}
	if field := fields.ByName("message"); field != nil {
		if value, ok := errMap["message"]; ok {
			msg.Set(field, protoreflect.ValueOfString(fmt.Sprintf("%v", value)))
		}
	}
	if field := fields.ByName("domain"); field != nil {
		if value, ok := errMap["domain"]; ok {
			msg.Set(field, protoreflect.ValueOfString(fmt.Sprintf("%v", value)))
		}
	}
	if field := fields.ByName("code"); field != nil {
		if value, ok := errMap["code"]; ok {
			msg.Set(field, protoreflect.ValueOfString(fmt.Sprintf("%v", value)))
		}
	}
	if field := fields.ByName("details"); field != nil {
		if value, ok := errMap["details"]; ok && value != nil {
			detailMap, ok := toAnyMap(value)
			if !ok {
				return fmt.Errorf("details expects a map, got %T", value)
			}
			detailsMsg := serviceCodec.newMessage(field.Message())
			if err := serviceCodec.anyToMessage(detailMap, detailsMsg); err != nil {
				return err
			}
			msg.Set(field, protoreflect.ValueOfMessage(detailsMsg))
		}
	}
	return nil
}

func (a taskWorkerAdapter) statusToEnum(status string) protoreflect.EnumNumber {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "SUCCEEDED":
		return 1
	case "FAILED_NON_RETRYABLE":
		return 2
	case "FAILED_RETRYABLE":
		return 3
	case "ALREADY_RUNNING":
		return 4
	case "RESOURCE_BUSY":
		return 5
	case "CANCELLED":
		return 6
	default:
		return 0
	}
}

func (a taskWorkerAdapter) emitTaskMetric(metric string, fields map[string]any) {
	if a.emitMetric != nil {
		a.emitMetric(metric, fields)
	}
}
