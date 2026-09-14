// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsengine

import (
	"errors"
	"fmt"
	"testing"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/pkg/oerrors"
)

func TestNormalizeErrorFromStructuredQuickJS(t *testing.T) {
	qjsErr := &quickjs.Error{
		Message:    "js exploded",
		JSONString: `{"errorId":"err-1","domain":"web","code":"EJS","grpcCode":7,"metadata":{"tenant":"acme"}}`,
	}

	got := NormalizeError(qjsErr)
	info := oerrors.GetErrorInfo(got)
	if info == nil {
		t.Fatal("expected GetErrorInfo to work after NormalizeError")
	}
	if info.ErrorId != "err-1" || info.Domain != "web" || info.Code != "EJS" || info.Message != "js exploded" || info.GrpcCode != 7 {
		t.Fatalf("unexpected error info: %#v", info)
	}
	if info.Metadata["tenant"] != "acme" {
		t.Fatalf("expected metadata preserved, got %#v", info.Metadata)
	}

	// Wrapped at the boundary still exposes ChoysumError via errors.As.
	wrapped := fmt.Errorf("failed to call function: %w", NormalizeError(qjsErr))
	if oerrors.GetErrorInfo(wrapped) == nil {
		t.Fatal("expected GetErrorInfo on wrapped normalized error")
	}

	// errors.As must also traverse wrappers around the raw QuickJS error.
	if info := oerrors.GetErrorInfo(NormalizeError(fmt.Errorf("engine: %w", qjsErr))); info == nil || info.Code != "EJS" {
		t.Fatalf("expected wrapped quickjs error to normalize, got %#v", info)
	}
}

func TestNormalizeErrorInvalidJSONFallback(t *testing.T) {
	qjsErr := &quickjs.Error{Message: "bad json", JSONString: "{"}
	got := NormalizeError(qjsErr)
	if !oerrors.Is(got, "js", "QUICKJS_ERROR") {
		t.Fatalf("expected js/QUICKJS_ERROR fallback, got %v", got)
	}
	info := oerrors.GetErrorInfo(got)
	if info == nil || info.Message != "bad json" {
		t.Fatalf("unexpected fallback info: %#v", info)
	}
}

func TestNormalizeErrorUnstructuredJSONFallsBack(t *testing.T) {
	for _, payload := range []string{`null`, `{}`} {
		got := NormalizeError(&quickjs.Error{Message: "plain js error", JSONString: payload})
		if !oerrors.Is(got, "js", "QUICKJS_ERROR") {
			t.Fatalf("expected js/QUICKJS_ERROR fallback for %q, got %v", payload, got)
		}
	}
}

func TestNormalizeErrorPassthrough(t *testing.T) {
	plain := errors.New("plain")
	if NormalizeError(plain) != plain {
		t.Fatal("expected plain errors unchanged")
	}
	ce := oerrors.New("billing", "E100", "payment failed")
	if NormalizeError(ce) != ce {
		t.Fatal("expected ChoysumError unchanged")
	}
	if NormalizeError(nil) != nil {
		t.Fatal("expected nil unchanged")
	}
}

func TestErrorInfoFromQuickJSNil(t *testing.T) {
	if info := errorInfoFromQuickJS(nil); info != nil {
		t.Fatalf("expected nil for nil quickjs error, got %#v", info)
	}
}

func TestNormalizeErrorKeepsJSONMessageWhenQuickJSMessageEmpty(t *testing.T) {
	got := NormalizeError(&quickjs.Error{
		Message:    "",
		JSONString: `{"domain":"web","code":"EJS","message":"from-json"}`,
	})
	info := oerrors.GetErrorInfo(got)
	if info == nil || info.Message != "from-json" {
		t.Fatalf("expected JSON message preserved when QuickJS Message empty, got %#v", info)
	}
}

func TestNormalizeErrorFallbackMessageFromJSONString(t *testing.T) {
	got := NormalizeError(&quickjs.Error{Message: "", JSONString: "{"})
	info := oerrors.GetErrorInfo(got)
	if info == nil || info.Message != "{" {
		t.Fatalf("expected JSONString fallback message, got %#v", info)
	}
}

func TestNormalizeExceptionNil(t *testing.T) {
	got := NormalizeException(nil, "exception without details")
	if got == nil || got.Error() != "exception without details" {
		t.Fatalf("unexpected nil-exception result: %v", got)
	}
	if NormalizeException(nil, "") == nil {
		t.Fatal("expected default missing-details message")
	}
}
