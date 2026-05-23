// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jobtoken

import (
	"context"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/grpc/converter"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/dynamicpb"
)

func TestServiceUnaryHandler(t *testing.T) {
	methodDesc, err := MethodDesc()
	if err != nil {
		t.Fatalf("MethodDesc error: %v", err)
	}
	runtimeScope := newJobTokenScope("development")
	runtimeScope.cfg.Auth.InternalKey = "secret"

	t.Run("service description", func(t *testing.T) {
		svc := NewService(runtimeScope, &fakeAuthenticator{})
		desc, err := svc.ServiceDesc()
		if err != nil {
			t.Fatalf("ServiceDesc error: %v", err)
		}
		if desc.ServiceName != ServiceFullName() || len(desc.Methods) != 1 || desc.Methods[0].MethodName != string(methodDesc.Name()) {
			t.Fatalf("unexpected service desc: %#v", desc)
		}
	})

	t.Run("missing authenticator", func(t *testing.T) {
		handler := NewService(runtimeScope, nil).unaryHandler(methodDesc)
		if _, err := handler(metadata.NewIncomingContext(context.Background(), metadata.Pairs(internalKeyHeader, "secret")), dynamicpb.NewMessage(methodDesc.Input())); status.Code(err) != codes.Unavailable {
			t.Fatalf("expected unavailable error, got %v", err)
		}
	})

	t.Run("invalid request type and missing fields", func(t *testing.T) {
		handler := NewService(runtimeScope, &fakeAuthenticator{}).unaryHandler(methodDesc)
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(internalKeyHeader, "secret"))
		if _, err := handler(ctx, "bad-request"); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("expected invalid argument for bad request type, got %v", err)
		}
		if _, err := handler(ctx, dynamicpb.NewMessage(methodDesc.Input())); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("expected invalid argument for missing fields, got %v", err)
		}
	})

	t.Run("issues token via ttl issuer", func(t *testing.T) {
		fakeAuth := &fakeAuthenticator{issueToken: "issued", issueExp: 999}
		handler := NewService(runtimeScope, fakeAuth).unaryHandler(methodDesc)
		req := dynamicpb.NewMessage(methodDesc.Input())
		payload := map[string]interface{}{
			"job_id":               "job-1",
			"target_app":           "auth",
			"full_method":          "/auth.Task/Run",
			"scheduler_user_id":    "u-1",
			"triggered_by_user_id": "u-2",
			"attempt":              int64(2),
			"ttl_ms":               int64(5000),
		}
		if err := converter.MapToMessage(payload, req); err != nil {
			t.Fatalf("MapToMessage error: %v", err)
		}
		resp, err := handler(metadata.NewIncomingContext(context.Background(), metadata.Pairs(internalKeyHeader, "secret")), req)
		if err != nil {
			t.Fatalf("unaryHandler error: %v", err)
		}
		respMsg, ok := resp.(*dynamicpb.Message)
		if !ok {
			t.Fatalf("unexpected response type: %T", resp)
		}
		respMap, err := converter.MessageToMap(respMsg)
		if err != nil {
			t.Fatalf("MessageToMap error: %v", err)
		}
		if respMap["access_token"] != "issued" || respMap["expires_at"] != "999" {
			t.Fatalf("unexpected response map: %#v", respMap)
		}
		if fakeAuth.lastUserID != "u-1" || fakeAuth.lastTTL != 5*time.Second {
			t.Fatalf("unexpected issue args: user=%q ttl=%s meta=%#v", fakeAuth.lastUserID, fakeAuth.lastTTL, fakeAuth.lastMeta)
		}
	})

	t.Run("falls back to token pair auth", func(t *testing.T) {
		fakeAuth := &pairOnlyAuthenticator{pair: &auth.TokenPair{AccessToken: "pair", ExpiresAt: 888}}
		handler := NewService(runtimeScope, fakeAuth).unaryHandler(methodDesc)
		req := dynamicpb.NewMessage(methodDesc.Input())
		if err := converter.MapToMessage(map[string]interface{}{
			"job_id":               "job-2",
			"target_app":           "auth",
			"full_method":          "/auth.Task/Run",
			"scheduler_user_id":    "u-3",
			"triggered_by_user_id": "u-4",
		}, req); err != nil {
			t.Fatalf("MapToMessage error: %v", err)
		}
		resp, err := handler(metadata.NewIncomingContext(context.Background(), metadata.Pairs(internalKeyHeader, "secret")), req)
		if err != nil {
			t.Fatalf("unaryHandler fallback error: %v", err)
		}
		respMsg := resp.(*dynamicpb.Message)
		respMap, err := converter.MessageToMap(respMsg)
		if err != nil {
			t.Fatalf("MessageToMap error: %v", err)
		}
		if respMap["access_token"] != "pair" || respMap["expires_at"] != "888" {
			t.Fatalf("unexpected fallback response map: %#v", respMap)
		}
	})
}
