// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package exportpkg_test

import (
	"errors"
	"testing"

	exportpkg "github.com/choysum-dev/choysum/pkg/export"
)

func TestError_ErrorAndUnwrap(t *testing.T) {
	var nilErr *exportpkg.Error
	if nilErr.Error() != "" {
		t.Fatalf("nil Error().Error() = %q, want empty", nilErr.Error())
	}
	if nilErr.Unwrap() != nil {
		t.Fatal("nil Error.Unwrap() should be nil")
	}

	err := &exportpkg.Error{Code: exportpkg.CodeProfileNotApproved}
	if err.Error() != exportpkg.CodeProfileNotApproved {
		t.Fatalf("Error() = %q, want code fallback", err.Error())
	}

	err.Text = "denied"
	if err.Error() != "denied" {
		t.Fatalf("Error() = %q, want text", err.Error())
	}
}

func TestAsError(t *testing.T) {
	cause := errors.New("cause")
	wrapped := exportpkg.ErrorfWrap(exportpkg.CodeInvalidFormat, "wrap", cause)

	got, ok := exportpkg.AsError(wrapped)
	if !ok || got.Code != exportpkg.CodeInvalidFormat {
		t.Fatalf("AsError() = %+v, %v", got, ok)
	}
	if !errors.Is(wrapped, cause) {
		t.Fatal("errors.Is should match wrapped cause")
	}

	plain := errors.New("plain")
	if _, ok := exportpkg.AsError(plain); ok {
		t.Fatal("AsError(plain) should be false")
	}
}

func TestErrorfWrap_nilCause(t *testing.T) {
	err := exportpkg.ErrorfWrap(exportpkg.CodeInvalidFormat, "text", nil)
	if err.Text != "text" || err.Unwrap() != nil {
		t.Fatalf("ErrorfWrap(nil) = %+v", err)
	}
}

func TestSentinelErrors(t *testing.T) {
	sentinels := []error{
		exportpkg.ErrCallerProfileDenied,
		exportpkg.ErrProfileNotApproved,
		exportpkg.ErrReaderNotRegistered,
		exportpkg.ErrAsyncNotSupported,
	}
	for _, err := range sentinels {
		if err.Error() == "" {
			t.Fatalf("sentinel %T has empty Error()", err)
		}
	}
}
