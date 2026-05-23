// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package oerrors

import (
	"errors"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewAndWrapHelpers(t *testing.T) {
	base := New("auth", "E001", "bad token").WithMetadata("service", "auth").WithGrpcCode(codes.Unauthenticated)
	if base.Domain != "auth" || base.Code != "E001" || base.Message != "bad token" {
		t.Fatalf("unexpected base error info: %#v", base.ErrorInfo)
	}
	if base.Metadata["service"] != "auth" || base.GrpcCode != int32(codes.Unauthenticated) {
		t.Fatalf("unexpected metadata/grpc code: %#v", base.ErrorInfo)
	}

	formatted := Newf("auth", "E002", "bad %s", "request")
	if formatted.Message != "bad request" {
		t.Fatalf("formatted message = %q, want bad request", formatted.Message)
	}

	wrapped := Wrap(base, "transport", "E003", "send failed")
	if wrapped == nil {
		t.Fatal("expected wrapped error")
	}
	if !Is(wrapped, "transport", "E003") || !Is(wrapped, "transport", "") {
		t.Fatalf("wrapped error domain/code mismatch: %v", wrapped)
	}
	if As(wrapped) == nil || As(wrapped).Domain != "transport" {
		t.Fatalf("As(wrapped) = %#v, want transport error", As(wrapped))
	}
	if !Has(wrapped, base) {
		t.Fatal("expected wrapped error chain to contain base error")
	}

	w := As(wrapped)
	if w.ErrorId != base.ErrorId || w.GrpcCode != base.GrpcCode {
		t.Fatalf("expected wrapped error to inherit id/grpc code, got %#v", w.ErrorInfo)
	}
	if w.Metadata["service"] != "auth" {
		t.Fatalf("expected wrapped error to inherit metadata, got %#v", w.Metadata)
	}

	wrappedFmt := Wrapf(base, "transport", "E004", "request %s", "timed out")
	if !strings.Contains(wrappedFmt.Error(), "request timed out") {
		t.Fatalf("unexpected Wrapf error: %v", wrappedFmt)
	}

	if Wrap(nil, "x", "y", "z") != nil || Wrapf(nil, "x", "y", "z") != nil {
		t.Fatal("expected wrapping nil error to return nil")
	}
	if Is(nil, "", "") || As(nil) != nil || Has(nil, base) {
		t.Fatal("expected nil helper results")
	}
}

func TestFromGrpcStatus(t *testing.T) {
	if FromGrpcStatus(nil) != nil {
		t.Fatal("expected nil grpc status to return nil")
	}

	choysumErr := New("orders", "E777", "broken").WithGrpcCode(codes.InvalidArgument)
	st, err := choysumErr.ToGrpcStatus()
	if err != nil {
		t.Fatalf("ToGrpcStatus error: %v", err)
	}
	back := FromGrpcStatus(st)
	if back == nil || back.Domain != "orders" || back.Code != "E777" || back.Message != "broken" {
		t.Fatalf("unexpected FromGrpcStatus result: %#v", back)
	}

	fallback := FromGrpcStatus(status.New(codes.NotFound, "not found"))
	if fallback == nil || fallback.Domain != "grpc" || fallback.Code != "STATUS_ERROR" || fallback.Message != "not found" {
		t.Fatalf("unexpected fallback grpc error: %#v", fallback)
	}

	wrapped := Wrap(errors.New("io eof"), "orders", "E778", "stream failed")
	if !strings.Contains(wrapped.Error(), "stream failed") || !strings.Contains(wrapped.Error(), "io eof") {
		t.Fatalf("unexpected wrapped error: %q", wrapped.Error())
	}
}
