// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jobtoken

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/grpc/converter"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// Service issues internal job tokens over gRPC.
type Service struct {
	runtimeScope  scope.Scope
	authenticator auth.Authenticator
}

type accessTokenIssuer interface {
	CreateAccessTokenWithTTL(ctx context.Context, userID string, metadata map[string]interface{}, ttl time.Duration) (string, int64, error)
}

// NewService creates a job token service bound to the given runtime scope and authenticator.
func NewService(runtimeScope scope.Scope, authenticator auth.Authenticator) *Service {
	return &Service{runtimeScope: runtimeScope, authenticator: authenticator}
}

// ServiceDesc builds the gRPC service descriptor for the job token service.
func (s *Service) ServiceDesc() (*grpc.ServiceDesc, error) {
	md, err := MethodDesc()
	if err != nil {
		return nil, err
	}
	serviceName := string(md.Parent().FullName())
	return &grpc.ServiceDesc{
		ServiceName: serviceName,
		HandlerType: (*interface{})(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: string(md.Name()),
				Handler:    s.methodHandler(md),
			},
		},
		Streams:  []grpc.StreamDesc{},
		Metadata: jobTokenProtoPath,
	}, nil
}

func (s *Service) methodHandler(methodDesc protoreflect.MethodDescriptor) func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	unaryHandler := s.unaryHandler(methodDesc)
	return func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
		reqMsg := dynamicpb.NewMessage(methodDesc.Input())
		if err := dec(reqMsg); err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid request payload: %v", err))
		}
		if interceptor == nil {
			return unaryHandler(ctx, reqMsg)
		}
		info := &grpc.UnaryServerInfo{
			Server:     srv,
			FullMethod: FullMethod(),
		}
		return interceptor(ctx, reqMsg, info, unaryHandler)
	}
}

func (s *Service) unaryHandler(methodDesc protoreflect.MethodDescriptor) grpc.UnaryHandler {
	return func(ctx context.Context, req any) (any, error) {
		if s.authenticator == nil {
			return nil, status.Error(codes.Unavailable, "auth service unavailable")
		}
		md, _ := metadata.FromIncomingContext(ctx)
		if err := authorizeInternalCallerFromContext(ctx, md, s.runtimeScope); err != nil {
			return nil, err
		}

		reqMsg, ok := req.(*dynamicpb.Message)
		if !ok {
			return nil, status.Error(codes.InvalidArgument, "unexpected request type")
		}

		reqMap, err := converter.MessageToMap(reqMsg)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "failed to parse request")
		}

		jobId := requiredString(reqMap, "job_id")
		targetApp := requiredString(reqMap, "target_app")
		fullMethod := requiredString(reqMap, "full_method")
		schedulerUserId := requiredString(reqMap, "scheduler_user_id")
		triggeredByUserId := requiredString(reqMap, "triggered_by_user_id")
		attempt := int64(0)
		if v, ok := reqMap["attempt"]; ok {
			switch tv := v.(type) {
			case int64:
				attempt = tv
			case float64:
				attempt = int64(tv)
			case int:
				attempt = int64(tv)
			case string:
				if parsed, err := strconv.ParseInt(strings.TrimSpace(tv), 10, 64); err == nil {
					attempt = parsed
				}
			}
		}
		ttlMs := int64(0)
		if v, ok := reqMap["ttl_ms"]; ok {
			switch tv := v.(type) {
			case int64:
				ttlMs = tv
			case float64:
				ttlMs = int64(tv)
			case int:
				ttlMs = int64(tv)
			case string:
				if parsed, err := strconv.ParseInt(strings.TrimSpace(tv), 10, 64); err == nil {
					ttlMs = parsed
				}
			}
		}

		if jobId == "" || targetApp == "" || fullMethod == "" || schedulerUserId == "" || triggeredByUserId == "" {
			return nil, status.Error(codes.InvalidArgument, "missing required fields")
		}

		metadata := map[string]interface{}{
			"purpose":           "task_job",
			"jobId":             jobId,
			"targetApp":         targetApp,
			"fullMethod":        fullMethod,
			"schedulerUserId":   schedulerUserId,
			"triggeredByUserId": triggeredByUserId,
			"attempt":           attempt,
			"ttlMs":             ttlMs,
		}

		if ttlMs <= 0 {
			ttlMs = 2 * 60 * 1000
		}

		// Issue access token for the scheduler user.
		var accessToken string
		var expiresAt int64
		if issuer, ok := s.authenticator.(accessTokenIssuer); ok {
			token, exp, err := issuer.CreateAccessTokenWithTTL(ctx, schedulerUserId, metadata, time.Duration(ttlMs)*time.Millisecond)
			if err != nil {
				return nil, status.Error(codes.Internal, fmt.Sprintf("failed to issue task token: %v", err))
			}
			accessToken = token
			expiresAt = exp
		} else {
			pair, err := s.authenticator.CreateTokens(ctx, schedulerUserId, metadata)
			if err != nil {
				return nil, status.Error(codes.Internal, fmt.Sprintf("failed to issue task token: %v", err))
			}
			accessToken = pair.AccessToken
			expiresAt = pair.ExpiresAt
		}

		respDesc := methodDesc.Output()
		respMsg := dynamicpb.NewMessage(respDesc)
		fields := respDesc.Fields()
		if f := fields.ByName("access_token"); f != nil {
			respMsg.Set(f, protoreflect.ValueOfString(accessToken))
		}
		if f := fields.ByName("expires_at"); f != nil {
			respMsg.Set(f, protoreflect.ValueOfInt64(expiresAt))
		}
		return respMsg, nil
	}
}

func requiredString(values map[string]interface{}, key string) string {
	v, ok := values[key]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}
