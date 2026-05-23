// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsengine

import (
	"strconv"
	"strings"
	"testing"
)

func TestWithCryptoExposesHashVerifyAndTokenHelpers(t *testing.T) {
	engine := newTestQuickjsEngine(t, WithCrypto())

	hash := evalString(t, engine, `$choysum.crypto.hashPassword("secret")`)
	if hash == "secret" || hash == "" {
		t.Fatalf("unexpected password hash %q", hash)
	}
	if !evalBool(t, engine, `$choysum.crypto.verifyPassword("secret", `+strconv.Quote(hash)+`)`) {
		t.Fatal("expected verifyPassword to accept matching password")
	}
	if evalBool(t, engine, `$choysum.crypto.verifyPassword("wrong", `+strconv.Quote(hash)+`)`) {
		t.Fatal("expected verifyPassword to reject mismatched password")
	}

	tokenA := evalString(t, engine, `$choysum.crypto.generateToken()`)
	tokenB := evalString(t, engine, `$choysum.crypto.generateToken()`)
	if len(tokenA) != 64 || len(tokenB) != 64 || tokenA == tokenB {
		t.Fatalf("unexpected generated tokens: %q %q", tokenA, tokenB)
	}

	errText := evalString(t, engine, `(() => { try { $choysum.crypto.hashPassword(); return ''; } catch (e) { return String(e); } })()`)
	if !strings.Contains(errText, "hashPassword requires a password argument") {
		t.Fatalf("unexpected hashPassword error: %q", errText)
	}
	errText = evalString(t, engine, `(() => { try { $choysum.crypto.verifyPassword('secret'); return ''; } catch (e) { return String(e); } })()`)
	if !strings.Contains(errText, "verifyPassword requires password and hash arguments") {
		t.Fatalf("unexpected verifyPassword error: %q", errText)
	}
}
