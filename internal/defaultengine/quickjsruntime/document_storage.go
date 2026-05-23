// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsruntime

import (
	"context"
	"fmt"
	"strings"

	"github.com/buke/quickjs-go"
	documentpayload "github.com/choysum-dev/choysum/internal/document/payload"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type documentDeleteStoredContentRequest struct {
	StoredContentID string `json:"storedContentId"`
}

// WithDocumentStorageProvider installs the document storage bridge for a scope provider.
func WithDocumentStorageProvider(scopeProvider jsengine.ScopeProvider) jsengine.JsEngineOption {
	return func(jsEngine jsengine.JsEngine) error {
		jse := jsEngine.(*quickjsengine.QuickjsEngine)
		globalsObj := jse.Ctx.Globals()

		choysumObj := globalsObj.Get("$choysum")
		if choysumObj.IsUndefined() {
			choysumObj = jse.Ctx.Object()
		}

		documentObj := jse.Ctx.Object()
		documentObj.Set("deleteStoredContent", jse.Ctx.Function(documentDeleteStoredContentFactory(scopeProvider, jse)))
		choysumObj.Set("document", documentObj)

		globalsObj.Set("$choysum", choysumObj)
		return nil
	}
}

// WithDocumentStorage installs the document storage bridge for a fixed runtime scope.
func WithDocumentStorage(runtimeScope scope.Scope) jsengine.JsEngineOption {
	return WithDocumentStorageProvider(jsengine.StaticScopeProvider(runtimeScope))
}

func documentDeleteStoredContentFactory(scopeProvider jsengine.ScopeProvider, engine *quickjsengine.QuickjsEngine) func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	return func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("need 1 arg: payload"))
		}

		var req documentDeleteStoredContentRequest
		if err := ctx.Unmarshal(args[0], &req); err != nil {
			return ctx.ThrowError(fmt.Errorf("invalid deleteStoredContent payload: %w", err))
		}
		req.StoredContentID = strings.TrimSpace(req.StoredContentID)
		if req.StoredContentID == "" {
			return ctx.ThrowError(fmt.Errorf("storedContentId is required"))
		}

		execCtx := resolveQuickjsExecContext(engine)
		runtimeScope := jsengine.ResolveScope(scopeProvider, execCtx)
		if err := deleteStoredContentFromDocument(execCtx, runtimeScope, req); err != nil {
			return ctx.ThrowError(err)
		}
		return ctx.Null()
	}
}

func resolveQuickjsExecContext(engine *quickjsengine.QuickjsEngine) context.Context {
	if engine != nil {
		if execCtx := engine.ExecContext(); execCtx != nil {
			return execCtx
		}
	}
	return context.Background()
}

func deleteStoredContentFromDocument(ctx context.Context, runtimeScope scope.Scope, req documentDeleteStoredContentRequest) error {
	return documentpayload.DeleteStoredContent(ctx, runtimeScope, req.StoredContentID, documentpayload.Options{})
}
