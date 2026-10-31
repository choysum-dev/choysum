// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/scope"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// workerOnlyFullMethods are TaskWorker routing keys that are invoked in-process
// via the JS pool without requiring a public gRPC methodDescriptor.
// Extensible for later hardcuts (FanOut / Execute* / CleanExpired*).
var workerOnlyFullMethods = map[string]struct{}{
	"document.AttachmentContent/RunGarbageCollection": {},
}

func isWorkerOnlyFullMethod(fullMethod string) bool {
	_, ok := workerOnlyFullMethods[strings.TrimSpace(fullMethod)]
	return ok
}

func workerJsServicePath(fullMethod string) (string, error) {
	fullMethod = strings.TrimSpace(fullMethod)
	if fullMethod == "" {
		return "", fmt.Errorf("empty full method")
	}
	// FullMethod uses Service/Method; JS pool uses Service.Method.
	return strings.ReplaceAll(fullMethod, "/", "."), nil
}

func workerPayloadArgs(payload map[string]interface{}) []interface{} {
	if len(payload) == 0 {
		return nil
	}
	// Prefer a single nowISO string when present (GC contract).
	if raw, ok := payload["nowISO"]; ok {
		if s, ok := raw.(string); ok {
			return []interface{}{s}
		}
	}
	if raw, ok := payload["NowISO"]; ok {
		if s, ok := raw.(string); ok {
			return []interface{}{s}
		}
	}
	return []interface{}{payload}
}

func (r invocationRuntime) invokeWorkerOnlyFullMethod(
	ctx context.Context,
	runtimeScope scope.Scope,
	jsCtx map[string]interface{},
	routing *jsengine.JsExecutionRouting,
	req *executeJobRequest,
) (any, error) {
	servicePath, err := workerJsServicePath(req.FullMethod)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if req.TimeoutMs > 0 {
		if _, ok := ctx.Deadline(); !ok {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, time.Duration(req.TimeoutMs)*time.Millisecond)
			defer cancel()
		}
	}

	span := trace.SpanFromContext(ctx)
	spanID := span.SpanContext().SpanID().String()
	execCtx := scope.ContextWithScope(ctx, runtimeScope)
	jsRequest := &jsengine.JsRequest{
		Id:      spanID,
		Service: servicePath,
		Context: jsCtx,
		Routing: routing,
		Args:    workerPayloadArgs(req.Payload),
	}
	jsResponse, err := r.jsExecutor.Execute(execCtx, jsRequest)
	if err != nil {
		return nil, err
	}
	if jsResponse == nil {
		return nil, nil
	}
	return jsResponse.Result, nil
}
