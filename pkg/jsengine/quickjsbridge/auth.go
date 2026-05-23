// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsbridge

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
)

func jsCreateTokens(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value, execCtx context.Context, authenticator auth.Authenticator) *quickjs.Value {
	if authenticator == nil {
		return ctx.ThrowError(fmt.Errorf("auth service is not enabled"))
	}
	if execCtx == nil {
		execCtx = context.Background()
	}

	if len(args) < 1 {
		return ctx.ThrowError(fmt.Errorf("createTokens requires userID"))
	}

	userID := args[0].String()
	metadata := make(map[string]interface{})
	if len(args) > 1 && !args[1].IsUndefined() && !args[1].IsNull() {
		metadataJSON := args[1].JSONStringify()
		if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
			return ctx.ThrowError(fmt.Errorf("invalid metadata format: %v", err))
		}
	}

	tokenPair, err := authenticator.CreateTokens(execCtx, userID, metadata)
	if err != nil {
		return ctx.ThrowError(err)
	}

	result := ctx.Object()
	result.Set("accessToken", ctx.String(tokenPair.AccessToken))
	result.Set("refreshToken", ctx.String(tokenPair.RefreshToken))
	result.Set("expiresAt", ctx.Int64(tokenPair.ExpiresAt))
	result.Set("refreshExpiresAt", ctx.Int64(tokenPair.RefreshExpiresAt))

	return result
}

func jsRefreshTokens(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value, execCtx context.Context, authenticator auth.Authenticator) *quickjs.Value {
	if authenticator == nil {
		return ctx.ThrowError(fmt.Errorf("auth service is not enabled"))
	}
	if execCtx == nil {
		execCtx = context.Background()
	}
	if len(args) < 1 {
		return ctx.ThrowError(fmt.Errorf("refreshTokens requires refreshToken"))
	}
	refreshToken := args[0].String()

	var metadata map[string]interface{}
	if len(args) > 1 && !args[1].IsUndefined() && !args[1].IsNull() {
		metadata = make(map[string]interface{})
		metadataJSON := args[1].JSONStringify()
		if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
			return ctx.ThrowError(fmt.Errorf("invalid metadata format: %v", err))
		}
	}

	tokenPair, err := authenticator.RefreshTokens(execCtx, refreshToken, metadata)
	if err != nil {
		return ctx.ThrowError(err)
	}

	result := ctx.Object()
	result.Set("accessToken", ctx.String(tokenPair.AccessToken))
	result.Set("refreshToken", ctx.String(tokenPair.RefreshToken))
	result.Set("expiresAt", ctx.Int64(tokenPair.ExpiresAt))
	result.Set("refreshExpiresAt", ctx.Int64(tokenPair.RefreshExpiresAt))
	return result
}

func jsRevokeToken(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value, execCtx context.Context, authenticator auth.Authenticator) *quickjs.Value {
	if authenticator == nil {
		return ctx.ThrowError(fmt.Errorf("auth service is not enabled"))
	}
	if execCtx == nil {
		execCtx = context.Background()
	}

	if len(args) < 1 {
		return ctx.ThrowError(fmt.Errorf("revokeToken requires token"))
	}

	token := args[0].String()
	reason := ""
	if len(args) > 1 && !args[1].IsUndefined() && !args[1].IsNull() {
		reason = args[1].String()
	}

	err := authenticator.RevokeToken(execCtx, token, reason)
	if err != nil {
		return ctx.ThrowError(err)
	}

	return ctx.Bool(true)
}

func jsRevokeAllUserTokens(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value, execCtx context.Context, authenticator auth.Authenticator) *quickjs.Value {
	if authenticator == nil {
		return ctx.ThrowError(fmt.Errorf("auth service is not enabled"))
	}
	if execCtx == nil {
		execCtx = context.Background()
	}

	if len(args) < 1 {
		return ctx.ThrowError(fmt.Errorf("revokeAllUserTokens requires userID"))
	}

	userID := args[0].String()
	exceptTokenID := ""
	reason := ""

	if len(args) > 1 && !args[1].IsUndefined() && !args[1].IsNull() {
		exceptTokenID = args[1].String()
	}
	if len(args) > 2 && !args[2].IsUndefined() && !args[2].IsNull() {
		reason = args[2].String()
	}

	count, err := authenticator.RevokeAllUserTokens(execCtx, userID, exceptTokenID, reason)
	if err != nil {
		return ctx.ThrowError(err)
	}

	return ctx.Int32(int32(count))
}

func jsValidateToken(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value, execCtx context.Context, authenticator auth.Authenticator) *quickjs.Value {
	if authenticator == nil {
		return ctx.ThrowError(fmt.Errorf("auth service is not enabled"))
	}
	if execCtx == nil {
		execCtx = context.Background()
	}

	if len(args) < 2 {
		return ctx.ThrowError(fmt.Errorf("validateToken requires token and tokenType"))
	}

	token := args[0].String()
	tokenTypeStr := args[1].String()

	var tokenType auth.TokenType
	if tokenTypeStr == "access" {
		tokenType = auth.AccessToken
	} else if tokenTypeStr == "refresh" {
		tokenType = auth.RefreshToken
	} else {
		return ctx.ThrowError(fmt.Errorf("invalid token type: %s", tokenTypeStr))
	}

	checkRevoked := false
	if len(args) > 2 && !args[2].IsUndefined() && !args[2].IsNull() {
		checkRevoked = args[2].Bool()
	}

	identity, err := authenticator.ValidateToken(execCtx, token, tokenType, checkRevoked)
	if err != nil {
		return ctx.ThrowError(err)
	}

	result := ctx.Object()
	result.Set("userId", ctx.String(identity.GetUserID()))
	result.Set("tokenId", ctx.String(identity.GetTokenID()))
	result.Set("isValid", ctx.Bool(identity.IsValid()))

	metadata := identity.GetMetadata()
	if metadata != nil {
		metadataJSON, err := json.Marshal(metadata)
		if err == nil {
			metadataObj := ctx.ParseJSON(string(metadataJSON))
			if !metadataObj.IsException() {
				result.Set("metadata", metadataObj)
			}
		}
	}

	return result
}

func WithAuth(authenticator auth.Authenticator) jsengine.JsEngineOption {
	return func(jsEngine jsengine.JsEngine) error {
		if authenticator == nil {
			return nil
		}

		jse := jsEngine.(*quickjsengine.QuickjsEngine)
		globalsObj := jse.Ctx.Globals()
		choysumObj := globalsObj.Get("$choysum")
		if choysumObj.IsUndefined() {
			choysumObj = jse.Ctx.Object()
		}

		authObj := jse.Ctx.Object()
		authObj.Set("enabled", jse.Ctx.Bool(true))

		authObj.Set("createTokens", jse.Ctx.Function(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			return jsCreateTokens(ctx, this, args, jse.ExecContext(), authenticator)
		}))
		authObj.Set("refreshTokens", jse.Ctx.Function(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			return jsRefreshTokens(ctx, this, args, jse.ExecContext(), authenticator)
		}))
		authObj.Set("revokeToken", jse.Ctx.Function(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			return jsRevokeToken(ctx, this, args, jse.ExecContext(), authenticator)
		}))
		authObj.Set("revokeAllUserTokens", jse.Ctx.Function(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			return jsRevokeAllUserTokens(ctx, this, args, jse.ExecContext(), authenticator)
		}))
		authObj.Set("validateToken", jse.Ctx.Function(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			return jsValidateToken(ctx, this, args, jse.ExecContext(), authenticator)
		}))

		choysumObj.Set("auth", authObj)
		globalsObj.Set("$choysum", choysumObj)

		return nil
	}
}