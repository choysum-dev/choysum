// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsengine

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"golang.org/x/crypto/bcrypt"
)

// hashPassword hashes a password with bcrypt.
func hashPassword(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	if len(args) < 1 {
		return ctx.ThrowError(fmt.Errorf("hashPassword requires a password argument"))
	}

	password := args[0].String()

	// Generate the hash with bcrypt at cost factor 10.
	// Cost 10: ~100ms
	// Cost 12: ~400ms
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return ctx.ThrowError(err)
	}

	// Return the hash string.
	return ctx.String(string(hashedBytes))
}

// verifyPassword uses bcrypt to compare a password against a hash.
func verifyPassword(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	if len(args) < 2 {
		return ctx.ThrowError(fmt.Errorf("verifyPassword requires password and hash arguments"))
	}

	password := args[0].String()
	hashedPassword := args[1].String()

	// Compare the password and hash with bcrypt.
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return ctx.Bool(err == nil)
}

// generateToken generates a secure random token.
func generateToken(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	// Generate 32 random bytes.
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return ctx.ThrowError(err)
	}

	// Encode the bytes as a hex string.
	token := hex.EncodeToString(tokenBytes)
	return ctx.String(token)
}

// WithCrypto registers all crypto helpers in the JavaScript environment.
func WithCrypto() jsengine.JsEngineOption {
	return func(jsEngine jsengine.JsEngine) error {
		jse := jsEngine.(*QuickjsEngine)
		globalsObj := jse.Ctx.Globals()

		choysumObj := globalsObj.Get("$choysum")
		if choysumObj.IsUndefined() {
			choysumObj = jse.Ctx.Object()
		}

		cryptoObj := jse.Ctx.Object()
		cryptoObj.Set("hashPassword", jse.Ctx.Function(hashPassword))
		cryptoObj.Set("verifyPassword", jse.Ctx.Function(verifyPassword))
		cryptoObj.Set("generateToken", jse.Ctx.Function(generateToken))

		choysumObj.Set("crypto", cryptoObj)
		globalsObj.Set("$choysum", choysumObj)
		return nil
	}
}
