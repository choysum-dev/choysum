// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/oerrors"
	"github.com/choysum-dev/choysum/pkg/scope"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/trace"
	xfmt "golang.org/x/exp/errors/fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// invocationRuntime owns JS context building, method invocation, result mapping,
// and error normalization for internal/service.
type invocationRuntime struct {
	runtimeScope scope.Scope
	runtimeOpts  runtimeOptions
	jsExecutor   jsexecutor.ScriptExecutor
}

func (s *ApplicationService) runtime() invocationRuntime {
	return invocationRuntime{
		runtimeScope: s.runtimeScope,
		runtimeOpts:  s.resolvedRuntimeOptions(),
		jsExecutor:   s.jsExecutor,
	}
}

func (r invocationRuntime) handleError(err error) (any, error) {
	if err == nil {
		return nil, nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil, status.FromContextError(err).Err()
	}
	if st, ok := status.FromError(err); ok {
		return nil, st.Err()
	}

	errorInfo := oerrors.GetErrorInfo(err)
	if errorInfo != nil {
		r.runtimeScope.Logger().Error("js interpreter failed",
			"error", err,
			"error_id", errorInfo.ErrorId,
			"domain", errorInfo.Domain,
			"code", errorInfo.Code,
			"grpc_code", errorInfo.GrpcCode,
		)

		if errorInfo.GrpcCode == 0 {
			r.runtimeScope.Logger().Warn("js interpreter status mapping degraded",
				"reason", "invalid_grpc_code",
				"grpc_code", errorInfo.GrpcCode,
			)
			return nil, status.Error(codes.Internal, errorInfo.Message)
		}

		st := status.New(codes.Code(errorInfo.GrpcCode), errorInfo.Message)
		if stWithDetails, detailsErr := st.WithDetails(errorInfo); detailsErr != nil {
			r.runtimeScope.Logger().Warn("js interpreter status mapping degraded",
				"reason", "status_details_attach_failed",
				"grpc_code", errorInfo.GrpcCode,
				"error", detailsErr,
			)
			return nil, st.Err()
		} else {
			return nil, stWithDetails.Err()
		}
	}

	r.runtimeScope.Logger().Error("js interpreter failed", "reason", "missing_error_info", "error", err)
	return nil, status.Error(codes.Internal, err.Error())
}

func (r invocationRuntime) executeUnary(
	ctx context.Context,
	runtimeScope scope.Scope,
	jsCtx map[string]interface{},
	routing *jsengine.JsExecutionRouting,
	packageName protoreflect.Name,
	serviceName protoreflect.Name,
	methodName protoreflect.Name,
	outputMsgDesc protoreflect.MessageDescriptor,
	reqMsg *dynamicpb.Message,
) (*dynamicpb.Message, error) {
	span := trace.SpanFromContext(ctx)
	sc := span.SpanContext()
	traceID := sc.TraceID().String()
	spanID := sc.SpanID().String()

	_ = grpc.SetHeader(ctx, metadata.Pairs("Server-Timing", fmt.Sprintf(`traceparent;desc="00-%s-%s-01"`, traceID, spanID)))

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		r.runtimeScope.Logger().Debug("unary handler executed",
			"full_method", fmt.Sprintf("%s.%s.%s", packageName, serviceName, methodName),
			"trace_id", traceID,
			"span_id", spanID,
			"duration_ms", duration.Milliseconds(),
		)
	}()

	execCtx := scope.ContextWithScope(ctx, runtimeScope)
	if _, ok := auth.AccessTokenFromContext(execCtx); !ok {
		if key := strings.TrimSpace(r.runtimeOpts.authInternalKey); key != "" {
			execCtx = auth.ContextWithInternalKey(execCtx, key)
		}
	}
	jsRequest := &jsengine.JsRequest{
		Id:      spanID,
		Service: fmt.Sprintf("%s.%s.%s", packageName, serviceName, methodName),
		Context: jsCtx,
		Routing: routing,
	}

	for index := 0; index < reqMsg.Descriptor().Fields().Len(); index++ {
		field := reqMsg.Descriptor().Fields().Get(index)
		value := reqMsg.Get(field)
		message := field.Message()

		if message != nil {
			msgJSON, err := serviceCodec.messageToAny(value.Message())
			if err != nil {
				return nil, xfmt.Errorf("Error converting message to Any: %w", err)
			}
			if msgJSON != nil {
				jsRequest.Args = append(jsRequest.Args, msgJSON)
			}
		} else {
			jsRequest.Args = append(jsRequest.Args, value.Interface())
		}
	}

	jsResponse, err := r.jsExecutor.Execute(execCtx, jsRequest)
	if err != nil {
		return nil, err
	}

	outMsg := serviceCodec.newMessage(outputMsgDesc)
	resultField := outMsg.Descriptor().Fields().ByTextName("result")
	if resultField != nil {
		if resultField.Message() != nil {
			resultMsg := serviceCodec.newMessage(resultField.Message())
			if err := serviceCodec.anyToMessage(jsResponse.Result, resultMsg); err != nil {
				return nil, xfmt.Errorf("Error converting result to message: %w", err)
			}
			outMsg.Set(resultField, protoreflect.ValueOf(resultMsg))
		} else {
			protoValue, err := serviceCodec.convertToProtoValue(jsResponse.Result, resultField)
			if err != nil {
				return nil, xfmt.Errorf("Error converting result to proto value: %w", err)
			}
			outMsg.Set(resultField, protoValue)
		}
	}
	return outMsg, nil
}

// buildJsContext constructs the hardened JS request context.
func (r invocationRuntime) buildJsContext(ctx context.Context) map[string]interface{} {
	span := trace.SpanFromContext(ctx)
	sc := span.SpanContext()
	traceID := ""
	spanID := ""
	if sc.IsValid() {
		traceID = sc.TraceID().String()
		spanID = sc.SpanID().String()
	} else {
		r.runtimeScope.Logger().Warn("service trace context degraded", "reason", "invalid_span_context")
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	}
	getFirst := func(key string) string {
		values := md.Get(key)
		if len(values) == 0 {
			return ""
		}
		return values[0]
	}

	kind := "grpc"
	if value := getFirst("x-grpc-web"); value == "1" {
		kind = "grpc-web"
	}

	depth := 0
	if value := getFirst("x-choysum-depth"); value != "" {
		if parsed, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && parsed >= 0 {
			depth = parsed
		}
	}

	identitySnap := map[string]any{}
	allowedCompanyIDs := []string(nil)
	baseCtx := map[string]any{}

	identity := auth.IdentityFromContext(ctx)
	if identity != nil {
		identitySnap["userId"] = identity.GetUserID()
		identitySnap["tokenId"] = identity.GetTokenID()

		meta := identity.GetMetadata()
		if meta != nil {
			if value, ok := meta["lang"].(string); ok && value != "" {
				baseCtx["lang"] = value
			} else if value, ok := meta["language"].(string); ok && value != "" {
				baseCtx["lang"] = value
			}
			if value, ok := meta["tz"].(string); ok {
				if normalized, ok := normalizeIANATimezone(value); ok {
					baseCtx["tz"] = normalized
				}
			} else if value, ok := meta["timezone"].(string); ok {
				if normalized, ok := normalizeIANATimezone(value); ok {
					baseCtx["tz"] = normalized
				}
			}
			if value, ok := meta["companyTimezone"].(string); ok {
				if normalized, ok := normalizeIANATimezone(value); ok {
					baseCtx["companyTz"] = normalized
				}
			} else if value, ok := meta["companyTz"].(string); ok {
				if normalized, ok := normalizeIANATimezone(value); ok {
					baseCtx["companyTz"] = normalized
				}
			} else if value, ok := meta["activeCompanyTimezone"].(string); ok {
				if normalized, ok := normalizeIANATimezone(value); ok {
					baseCtx["companyTz"] = normalized
				}
			}

			toStringSlice := func(value any) []string {
				items := []string{}
				switch typed := value.(type) {
				case []string:
					for _, item := range typed {
						if item != "" {
							items = append(items, item)
						}
					}
				case []any:
					for _, item := range typed {
						if str, ok := item.(string); ok && str != "" {
							items = append(items, str)
						}
					}
				}
				return items
			}
			if value, ok := meta["allowedCompanyIds"]; ok {
				allowedCompanyIDs = toStringSlice(value)
				if len(allowedCompanyIDs) > 0 {
					identitySnap["allowedCompanyIds"] = allowedCompanyIDs
				}
			}

			if value, ok := meta["activeCompanyId"].(string); ok && value != "" {
				baseCtx["activeCompanyId"] = value
			}
			if value, ok := meta["enabledCompanyIds"]; ok {
				ids := toStringSlice(value)
				if len(ids) > 0 {
					baseCtx["enabledCompanyIds"] = ids
				}
			}
		}
	}

	clientKV := map[string]string{}
	if bag := baggage.FromContext(ctx); bag.Len() > 0 {
		for _, member := range bag.Members() {
			clientKV[member.Key()] = member.Value()
		}
	}
	getAllowed := func(id string) bool {
		if id == "" {
			return false
		}
		for _, item := range allowedCompanyIDs {
			if item == id {
				return true
			}
		}
		return false
	}

	active := ""
	if value, ok := baseCtx["activeCompanyId"].(string); ok {
		active = strings.TrimSpace(value)
	}
	enabled := []string(nil)
	if value, ok := baseCtx["enabledCompanyIds"].([]string); ok {
		enabled = value
	}
	if len(enabled) == 0 && active != "" {
		enabled = []string{active}
	}
	if active == "" && len(enabled) > 0 {
		active = enabled[0]
	}
	if len(allowedCompanyIDs) > 0 {
		if active != "" && !getAllowed(active) {
			active = allowedCompanyIDs[0]
		}
		filtered := make([]string, 0, len(enabled))
		seen := map[string]bool{}
		for _, id := range enabled {
			id = strings.TrimSpace(id)
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true
			if getAllowed(id) {
				filtered = append(filtered, id)
			}
		}
		enabled = filtered
		if len(enabled) == 0 && active != "" {
			enabled = []string{active}
		}
		if len(enabled) == 0 {
			active = allowedCompanyIDs[0]
			enabled = []string{active}
		}
	}
	if active != "" {
		baseCtx["activeCompanyId"] = active
	}
	if len(enabled) > 0 {
		baseCtx["enabledCompanyIds"] = enabled
	}
	for key, value := range clientKV {
		lowerKey := strings.ToLower(strings.TrimSpace(key))
		if !strings.HasPrefix(lowerKey, "ctx.") {
			continue
		}
		ctxKey := strings.TrimPrefix(lowerKey, "ctx.")
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		switch ctxKey {
		case "lang":
			if len(value) <= 32 {
				baseCtx["lang"] = value
			}
		case "tz":
			// Display tz: baggage only fills when user preference is empty (D6/D7).
			if _, hasUserTz := baseCtx["tz"]; !hasUserTz {
				if normalized, ok := normalizeIANATimezone(value); ok && len(normalized) <= 64 {
					baseCtx["tz"] = normalized
				}
			}
		}
	}

	if companyTz, ok := baseCtx["companyTz"].(string); ok && companyTz != "" {
		if _, hasUserTz := baseCtx["tz"]; !hasUserTz {
			baseCtx["tz"] = companyTz
		}
	}
	if _, hasTz := baseCtx["tz"]; !hasTz {
		baseCtx["tz"] = "UTC"
	}

	if companyID, ok := baseCtx["activeCompanyId"].(string); ok && companyID != "" {
		if companyIDs, ok := baseCtx["enabledCompanyIds"].([]string); ok {
			found := false
			for _, item := range companyIDs {
				if item == companyID {
					found = true
					break
				}
			}
			if !found {
				baseCtx["enabledCompanyIds"] = append([]string{companyID}, companyIDs...)
			}
		}
	}

	reqMeta := map[string]any{
		"requestId": spanID,
		"traceId":   traceID,
		"spanId":    spanID,
		"kind":      kind,
		"depth":     depth,
	}

	return map[string]interface{}{
		"ctx":      baseCtx,
		"identity": identitySnap,
		"req":      reqMeta,
	}
}

// normalizeIANATimezone trims and validates an IANA timezone id via time.LoadLocation.
// Empty or invalid values return ok=false.
func normalizeIANATimezone(value string) (string, bool) {
	normalized := strings.TrimSpace(value)
	if normalized == "" || len(normalized) > 64 {
		return "", false
	}
	if _, err := time.LoadLocation(normalized); err != nil {
		return "", false
	}
	return normalized, true
}

func (r invocationRuntime) invokeTargetMethod(ctx context.Context, runtimeScope scope.Scope, jsCtx map[string]interface{}, routing *jsengine.JsExecutionRouting, req *executeJobRequest) (any, error) {
	methodFullName := strings.ReplaceAll(req.FullMethod, "/", ".")
	md, err := serviceCodec.methodDescriptor(methodFullName)
	if err != nil {
		return nil, status.Error(codes.Unimplemented, "target method does not exist")
	}
	if md.IsPlaceholder() || md.IsStreamingClient() || md.IsStreamingServer() {
		return nil, status.Error(codes.Unimplemented, "unsupported target method")
	}

	if req.TimeoutMs > 0 {
		if _, ok := ctx.Deadline(); !ok {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, time.Duration(req.TimeoutMs)*time.Millisecond)
			defer cancel()
		}
	}

	reqMsg := serviceCodec.newMessage(md.Input())
	if err := serviceCodec.mapToMessage(req.Payload, reqMsg); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid payload format")
	}

	pkgName := md.ParentFile().Package().Name()
	serviceName := md.Parent().Name()
	methodName := md.Name()

	outMsg, err := r.executeUnary(ctx, runtimeScope, jsCtx, routing, pkgName, serviceName, methodName, md.Output(), reqMsg)
	if err != nil {
		return nil, err
	}
	return serviceCodec.messageToAny(outMsg)
}
