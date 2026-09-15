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

type errWrap struct{ err error }

func (e errWrap) Error() string { return "wrap" }
func (e errWrap) Unwrap() error { return e.err }

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

	wrapped := fmt.Errorf("failed to call function: %w", NormalizeError(qjsErr))
	if oerrors.GetErrorInfo(wrapped) == nil {
		t.Fatal("expected GetErrorInfo on wrapped normalized error")
	}

	if info := oerrors.GetErrorInfo(NormalizeError(fmt.Errorf("engine: %w", qjsErr))); info == nil || info.Code != "EJS" {
		t.Fatalf("expected wrapped quickjs error to normalize, got %#v", info)
	}
}

func TestNormalizeErrorFromMessageJSON(t *testing.T) {
	// Mirrors throw new Error(JSON.stringify(...)) where JSONString is "{}".
	got := NormalizeError(&quickjs.Error{
		Message:    `{"domain":"web","code":"EJS","message":"structured"}`,
		JSONString: `{}`,
	})
	info := oerrors.GetErrorInfo(got)
	if info == nil || info.Domain != "web" || info.Code != "EJS" || info.Message != "structured" {
		t.Fatalf("expected Message JSON payload, got %#v", info)
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

func TestNormalizeErrorTypedNilQuickJS(t *testing.T) {
	var typedNil *quickjs.Error
	got := NormalizeError(errWrap{err: typedNil})
	if got == nil {
		t.Fatal("expected typed-nil quickjs error to passthrough without panic")
	}
	if oerrors.GetErrorInfo(got) != nil {
		t.Fatalf("expected typed-nil not to become ChoysumError, got %v", got)
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

func TestNormalizeErrorPrefersJSONMessageOverEngineMessage(t *testing.T) {
	got := NormalizeError(&quickjs.Error{
		Message:    "Error: engine text",
		JSONString: `{"domain":"web","code":"EJS","message":"module text"}`,
	})
	info := oerrors.GetErrorInfo(got)
	if info == nil || info.Message != "module text" {
		t.Fatalf("expected structured JSON message to win, got %#v", info)
	}
}

func TestNormalizeErrorPreservesErrorIdWithoutDomain(t *testing.T) {
	got := NormalizeError(&quickjs.Error{
		Message:    "plain",
		JSONString: `{"errorId":"corr-1","metadata":{"k":"v","n":2,"obj":{"a":1}}}`,
	})
	info := oerrors.GetErrorInfo(got)
	if info == nil || info.ErrorId != "corr-1" || info.Domain != "js" || info.Code != "QUICKJS_ERROR" {
		t.Fatalf("expected errorId preserved with js/QUICKJS_ERROR defaults, got %#v", info)
	}
	if info.Metadata["k"] != "v" || info.Metadata["n"] != "2" || info.Metadata["obj"] != `{"a":1}` {
		t.Fatalf("expected metadata JSON-stringified, got %#v", info.Metadata)
	}
}

func TestNormalizeErrorMessageOnlyPayload(t *testing.T) {
	got := NormalizeError(&quickjs.Error{
		Message:    `{"message":"payment declined"}`,
		JSONString: `{}`,
	})
	info := oerrors.GetErrorInfo(got)
	if info == nil || info.Domain != "js" || info.Code != "QUICKJS_ERROR" || info.Message != "payment declined" {
		t.Fatalf("expected message-only payload preserved, got %#v", info)
	}
}

func TestNormalizeErrorPreservesGrpcCodeOnly(t *testing.T) {
	got := NormalizeError(&quickjs.Error{
		Message:    "plain",
		JSONString: `{"grpcCode":7}`,
	})
	info := oerrors.GetErrorInfo(got)
	if info == nil || info.Domain != "js" || info.Code != "QUICKJS_ERROR" || info.GrpcCode != 7 {
		t.Fatalf("expected grpcCode-only payload preserved, got %#v", info)
	}
}

func TestNormalizeErrorFallbackMessageFromJSONString(t *testing.T) {
	got := NormalizeError(&quickjs.Error{Message: "", JSONString: "{"})
	info := oerrors.GetErrorInfo(got)
	if info == nil || info.Message != "{" {
		t.Fatalf("expected JSONString fallback message, got %#v", info)
	}
}

func TestNormalizeErrorFallbackMessageFromErrorString(t *testing.T) {
	qjsErr := &quickjs.Error{Name: "TypeError", Message: "", JSONString: ""}
	got := NormalizeError(qjsErr)
	info := oerrors.GetErrorInfo(got)
	if info == nil || info.Message == "" {
		t.Fatalf("expected err.Error() fallback message, got %#v", info)
	}
}

func TestNormalizeErrorStructuredEmptyMessagesUseJSONString(t *testing.T) {
	got := NormalizeError(&quickjs.Error{
		Message:    "",
		JSONString: `{"domain":"web","code":"EJS"}`,
	})
	info := oerrors.GetErrorInfo(got)
	if info == nil || info.Message != `{"domain":"web","code":"EJS"}` {
		t.Fatalf("expected JSONString as message fallback, got %#v", info)
	}
}

func TestNormalizeExceptionNil(t *testing.T) {
	got := NormalizeException(nil, "exception without details")
	if !oerrors.Is(got, "js", "QUICKJS_ERROR") {
		t.Fatalf("expected categorized nil-exception error, got %v", got)
	}
	info := oerrors.GetErrorInfo(got)
	if info == nil || info.Message != "exception without details" {
		t.Fatalf("unexpected nil-exception info: %#v", info)
	}
	if NormalizeException(nil, "") == nil {
		t.Fatal("expected default missing-details message")
	}

	var typedNil *quickjs.Error
	got = NormalizeException(typedNil, "exception without details")
	if !oerrors.Is(got, "js", "QUICKJS_ERROR") {
		t.Fatalf("expected typed-nil exception categorized, got %v", got)
	}
}
