// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package autherrors

import (
	"errors"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/oerrors"
)

func TestErrorCodeStringAndNewHelpers(t *testing.T) {
	if ErrInvalidCredentials.String() != "INVALID_CREDENTIALS" {
		t.Fatalf("String() = %q", ErrInvalidCredentials.String())
	}

	err := NewAuthError(ErrMissingToken, "missing token")
	if err.Domain != Domain || err.Code != ErrMissingToken.String() || err.Message != "missing token" {
		t.Fatalf("NewAuthError() = %#v", err)
	}
	if !IsAuthError(err) || !IsAuthError(err, ErrMissingToken) || IsAuthError(err, ErrPermissionDenied) {
		t.Fatalf("unexpected IsAuthError results for %#v", err)
	}

	formatted := NewAuthErrorf(ErrPermissionDenied, "forbidden: %s", "scope")
	if formatted.Message != "forbidden: scope" {
		t.Fatalf("NewAuthErrorf message = %q", formatted.Message)
	}
}

func TestWrapHelpersIsAuthErrorAndAsAuthError(t *testing.T) {
	base := errors.New("db offline")
	wrapped := WrapAuthError(base, ErrConfigurationError, "init failed")
	if wrapped == nil || !strings.Contains(wrapped.Error(), "init failed") || !strings.Contains(wrapped.Error(), "db offline") {
		t.Fatalf("WrapAuthError() = %v", wrapped)
	}
	if !IsAuthError(wrapped, ErrConfigurationError) {
		t.Fatalf("expected wrapped error to match auth code: %v", wrapped)
	}

	formatted := WrapAuthErrorf(base, ErrTokenParsingFailed, "parse %s", "jwt")
	if formatted == nil || !strings.Contains(formatted.Error(), "parse jwt") {
		t.Fatalf("WrapAuthErrorf() = %v", formatted)
	}
	if !IsAuthError(formatted, ErrTokenParsingFailed) {
		t.Fatalf("expected formatted wrapped error to match auth code: %v", formatted)
	}

	choysumErr, ok := AsAuthError(formatted)
	if !ok || choysumErr == nil {
		t.Fatalf("AsAuthError() = %#v, %v", choysumErr, ok)
	}
	if choysumErr.Domain != Domain || choysumErr.Code != ErrTokenParsingFailed.String() {
		t.Fatalf("unexpected AsAuthError result: %#v", choysumErr)
	}

	nonAuth := oerrors.New("orders", "BAD_STATE", "broken")
	if converted, ok := AsAuthError(nonAuth); ok || converted != nil {
		t.Fatalf("expected non-auth error to be ignored, got %#v, %v", converted, ok)
	}
	if IsAuthError(nonAuth) {
		t.Fatalf("expected non-auth domain to fail IsAuthError: %v", nonAuth)
	}
	if converted, ok := AsAuthError(nil); ok || converted != nil {
		t.Fatalf("expected nil error to return nil,false, got %#v,%v", converted, ok)
	}
}