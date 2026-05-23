// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package revocation

import (
	"testing"
	"time"

	"github.com/choysum-dev/choysum/pkg/auth"
)

func TestStandardTokenTimeGettersAndFactory(t *testing.T) {
	revokedAt := time.Now().Add(-time.Minute).Round(time.Second)
	expiresAt := time.Now().Add(time.Hour).Round(time.Second)
	token := &StandardToken{
		TokenID:   "token-1",
		UserID:    "user-1",
		TokenType: auth.AccessToken,
		RevokedAt: revokedAt,
		ExpiresAt: expiresAt,
		Reason:    "manual",
	}

	if got := token.GetRevokedAt(); !got.Equal(revokedAt) {
		t.Fatalf("GetRevokedAt() = %v, want %v", got, revokedAt)
	}
	if got := token.GetExpiresAt(); !got.Equal(expiresAt) {
		t.Fatalf("GetExpiresAt() = %v, want %v", got, expiresAt)
	}

	before := time.Now()
	created := NewToken("token-2", "user-2", auth.RefreshToken, expiresAt, "rotated")
	after := time.Now()
	if got := created.GetExpiresAt(); !got.Equal(expiresAt) {
		t.Fatalf("factory expiresAt = %v, want %v", got, expiresAt)
	}
	if got := created.GetRevokedAt(); got.Before(before.Add(-time.Second)) || got.After(after.Add(time.Second)) {
		t.Fatalf("factory revokedAt out of expected range: %v", got)
	}
}
