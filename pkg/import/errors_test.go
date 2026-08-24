// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package importpkg_test

import (
	"errors"
	"testing"

	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

func TestError_ErrorAndMessage(t *testing.T) {
	var nilErr *importpkg.Error
	if nilErr.Error() != "" {
		t.Fatalf("nil Error().Error() = %q, want empty", nilErr.Error())
	}
	if (nilErr.Message() != importpkg.Message{}) {
		t.Fatal("nil Message() should be zero value")
	}

	err := &importpkg.Error{Code: importpkg.CodePolicyDenied}
	if err.Error() != importpkg.CodePolicyDenied {
		t.Fatalf("Error() = %q, want code fallback", err.Error())
	}

	err.Text = "denied"
	if err.Error() != "denied" {
		t.Fatalf("Error() = %q, want text", err.Error())
	}

	msg := err.Message()
	if msg.Type != importpkg.MessageError || msg.Code != importpkg.CodePolicyDenied || msg.Text != "denied" {
		t.Fatalf("Message() = %+v", msg)
	}
}

func TestAsError(t *testing.T) {
	cause := errors.New("cause")
	wrapped := importpkg.ErrorfWrap(importpkg.CodeInvalidFormat, "wrap", cause)

	got, ok := importpkg.AsError(wrapped)
	if !ok || got.Code != importpkg.CodeInvalidFormat {
		t.Fatalf("AsError() = %+v, %v", got, ok)
	}
	if !errors.Is(wrapped, cause) {
		t.Fatal("errors.Is should match wrapped cause")
	}
	var nilErr *importpkg.Error
	if nilErr.Unwrap() != nil {
		t.Fatal("nil Error.Unwrap() should be nil")
	}

	plain := errors.New("plain")
	if _, ok := importpkg.AsError(plain); ok {
		t.Fatal("AsError(plain) should be false")
	}
}

func TestErrorfWrap_nilCause(t *testing.T) {
	err := importpkg.ErrorfWrap(importpkg.CodeInvalidFormat, "text", nil)
	if err.Text != "text" || err.Unwrap() != nil {
		t.Fatalf("ErrorfWrap(nil) = %+v", err)
	}
}

func TestSentinelErrors(t *testing.T) {
	sentinels := []error{
		importpkg.ErrCallerProfileDenied,
		importpkg.ErrPolicyDenied,
		importpkg.ErrDryRunRequiresAtomic,
		importpkg.ErrWriterNotRegistered,
		importpkg.ErrPlanBuilderNotFound,
		importpkg.ErrAsyncNotSupported,
	}
	for _, err := range sentinels {
		if err.Error() == "" {
			t.Fatalf("sentinel %T has empty Error()", err)
		}
	}
}
