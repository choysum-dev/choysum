// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsbridge

import (
	"context"
	"net"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/oerrors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

func TestBuildTrustedOutgoingMetadataUsesTrustedContextValues(t *testing.T) {
	execCtx := context.Background()
	execCtx = auth.ContextWithAccessToken(execCtx, "access-token")
	execCtx = metadata.NewIncomingContext(execCtx, metadata.Pairs(
		"traceparent", "00-abc-123-01",
		"tracestate", "vendor=test",
		"baggage", "tenant=acme",
		"x-choysum-depth", "1",
	))
	execCtx = peer.NewContext(execCtx, &peer.Peer{Addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9123}})

	md, err := buildTrustedOutgoingMetadata(execCtx)
	if err != nil {
		t.Fatalf("buildTrustedOutgoingMetadata: %v", err)
	}
	if got := md.Get("authorization"); len(got) != 1 || got[0] != "Bearer access-token" {
		t.Fatalf("authorization = %#v", got)
	}
	if got := md.Get("traceparent"); len(got) != 1 || got[0] != "00-abc-123-01" {
		t.Fatalf("traceparent = %#v", got)
	}
	if got := md.Get("tracestate"); len(got) != 1 || got[0] != "vendor=test" {
		t.Fatalf("tracestate = %#v", got)
	}
	if got := md.Get("baggage"); len(got) != 1 || got[0] != "tenant=acme" {
		t.Fatalf("baggage = %#v", got)
	}
	if got := md.Get("x-choysum-depth"); len(got) != 1 || got[0] != "2" {
		t.Fatalf("x-choysum-depth = %#v", got)
	}
	if got := md.Get("x-choysum-client-ip"); len(got) != 1 || got[0] != "127.0.0.1:9123" {
		t.Fatalf("x-choysum-client-ip = %#v", got)
	}
	if got := md.Get("user-agent"); len(got) != 1 || got[0] != "choysum-quickjs/1.0" {
		t.Fatalf("user-agent = %#v", got)
	}
	if got := md.Get("x-choysum-jsclient"); len(got) != 1 || got[0] != "1" {
		t.Fatalf("x-choysum-jsclient = %#v", got)
	}
}

func TestBuildTrustedOutgoingMetadataRejectsInvalidOrMissingCredentials(t *testing.T) {
	if _, err := buildTrustedOutgoingMetadata(context.Background()); err == nil || !strings.Contains(err.Error(), "missing access token") {
		t.Fatalf("expected missing access token error, got %v", err)
	}

	ctxWithKey := auth.ContextWithInternalKey(context.Background(), "internal-key")
	md, err := buildTrustedOutgoingMetadata(ctxWithKey)
	if err != nil {
		t.Fatalf("buildTrustedOutgoingMetadata(internal key): %v", err)
	}
	if got := md.Get("x-choysum-internal-key"); len(got) != 1 || got[0] != "internal-key" {
		t.Fatalf("x-choysum-internal-key = %#v", got)
	}

	badTokenCtx := auth.ContextWithAccessToken(context.Background(), "bad\nvalue")
	if _, err := buildTrustedOutgoingMetadata(badTokenCtx); err == nil || !strings.Contains(err.Error(), "invalid authorization header") {
		t.Fatalf("expected invalid authorization header error, got %v", err)
	}

	badKeyCtx := auth.ContextWithInternalKey(context.Background(), "bad\nvalue")
	if _, err := buildTrustedOutgoingMetadata(badKeyCtx); err == nil || !strings.Contains(err.Error(), "invalid internal key header") {
		t.Fatalf("expected invalid internal key header error, got %v", err)
	}
}

func TestGrpcHelperUtilities(t *testing.T) {
	cloned := cloneMetadata(metadata.MD{"traceparent": {"root"}})
	cloned["traceparent"][0] = "changed"
	if original := (metadata.MD{"traceparent": {"root"}}); false {
		t.Fatalf("unreachable: %#v", original)
	}

	source := metadata.MD{"traceparent": {"root"}, "baggage": {"tenant=acme"}}
	dup := cloneMetadata(source)
	dup["traceparent"][0] = "updated"
	if source["traceparent"][0] != "root" {
		t.Fatalf("expected source metadata to remain unchanged, got %#v", source)
	}
	if len(cloneMetadata(nil)) != 0 {
		t.Fatal("expected cloneMetadata(nil) to return empty metadata")
	}

	names, err := methodNamesFromServiceParts(" /choysum.auth.Service/ ", " Login ")
	if err != nil {
		t.Fatalf("methodNamesFromServiceParts: %v", err)
	}
	if names.descriptor != "choysum.auth.Service.Login" || names.invoke != "/choysum.auth.Service/Login" {
		t.Fatalf("unexpected method names: %#v", names)
	}
	if _, err := methodNamesFromServiceParts(" ", "Login"); err == nil || !strings.Contains(err.Error(), "service name cannot be empty") {
		t.Fatalf("expected empty service error, got %v", err)
	}
	if _, err := methodNamesFromServiceParts("choysum.auth.Service", " "); err == nil || !strings.Contains(err.Error(), "method name cannot be empty") {
		t.Fatalf("expected empty method error, got %v", err)
	}
}

func TestAttachChoysumErrorInfoAddsStructuredFields(t *testing.T) {
	engine := newTestQuickjsEngine(t)
	errObj := engine.Ctx.Object()

	st, err := oerrors.New("billing", "PAYMENT_FAILED", "payment failed").
		WithMetadata("tenant", "acme").
		WithGrpcCode(codes.InvalidArgument).
		ToGrpcStatus()
	if err != nil {
		t.Fatalf("ToGrpcStatus: %v", err)
	}
	attachChoysumErrorInfo(engine.Ctx, errObj, st.Err())
	engine.Ctx.Globals().Set("__choysumErr", errObj)

	json := evalString(t, engine, `JSON.stringify(__choysumErr)`)
	for _, fragment := range []string{`"domain":"billing"`, `"code":"PAYMENT_FAILED"`, `"grpcCode":3`, `"tenant":"acme"`} {
		if !strings.Contains(json, fragment) {
			t.Fatalf("expected attached error info to contain %q, got %s", fragment, json)
		}
	}

	plain := engine.Ctx.Object()
	attachChoysumErrorInfo(engine.Ctx, plain, context.Canceled)
	engine.Ctx.Globals().Set("__plainErr", plain)
	if got := evalString(t, engine, `JSON.stringify(__plainErr)`); got != "{}" {
		t.Fatalf("expected plain error object to remain empty, got %s", got)
	}

	attachChoysumErrorInfo(nil, nil, nil)
	attachChoysumErrorInfo(engine.Ctx, nil, st.Err())
}