// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package revocation

import (
	"testing"
	"time"
)

func TestIsExpired(t *testing.T) {
	if !IsExpired(time.Now().Add(-time.Second)) {
		t.Fatal("expected past expiration time to be expired")
	}
	if IsExpired(time.Now().Add(time.Second)) {
		t.Fatal("expected future expiration time to be valid")
	}
}

func TestIsValidTokenID(t *testing.T) {
	if IsValidTokenID("") {
		t.Fatal("expected empty token id to be invalid")
	}
	if !IsValidTokenID("token-1") {
		t.Fatal("expected non-empty token id to be valid")
	}
}

func TestIsValidUserID(t *testing.T) {
	if IsValidUserID("") {
		t.Fatal("expected empty user id to be invalid")
	}
	if !IsValidUserID("user-1") {
		t.Fatal("expected non-empty user id to be valid")
	}
}
