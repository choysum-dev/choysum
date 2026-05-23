// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsruntime

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type documentBridgeTestScope struct {
	ctx    context.Context
	logger *slog.Logger
	cfg    *config.Config
}

func (e *documentBridgeTestScope) Run(fn func(runtimeScope scope.Scope) error) error { return fn(e) }
func (e *documentBridgeTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}
func (e *documentBridgeTestScope) Session() *scope.Session { return nil }
func (e *documentBridgeTestScope) WithContext(ctx context.Context) scope.Scope {
	return &documentBridgeTestScope{ctx: ctx, logger: e.logger, cfg: e.cfg}
}
func (e *documentBridgeTestScope) Context() context.Context { return e.ctx }
func (e *documentBridgeTestScope) Logger() *slog.Logger     { return e.logger }
func (e *documentBridgeTestScope) Config() *config.Config   { return e.cfg }
func (e *documentBridgeTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.cfg)
}

func newDocumentBridgeTestScope() *documentBridgeTestScope {
	return &documentBridgeTestScope{
		ctx:    context.Background(),
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg: &config.Config{
			Document: config.NewDefaultDocumentConfig(),
		},
	}
}

func TestWithDocumentStorageExposesDeleteStoredContent(t *testing.T) {
	runtimeScope := newDocumentBridgeTestScope()
	engine := newTestQuickjsEngine(t, WithDocumentStorage(runtimeScope))

	if !evalBool(t, engine, `typeof $choysum.document.deleteStoredContent === "function"`) {
		t.Fatal("expected $choysum.document.deleteStoredContent to be a function")
	}
}

func TestWithDocumentStorageDeleteStoredContentPropagatesPayloadErrors(t *testing.T) {
	runtimeScope := newDocumentBridgeTestScope()
	engine := newTestQuickjsEngine(t, WithDocumentStorage(runtimeScope))

	errText := evalString(t, engine, `(() => {
		try {
			$choysum.document.deleteStoredContent({ storedContentId: "sc_001" });
			return "no-error";
		} catch (e) {
			return String(e);
		}
	})()`)
	if !strings.Contains(errText, "document database session is required") {
		t.Fatalf("expected payload deletion error, got %q", errText)
	}
}

func TestWithDocumentStorageDeleteStoredContentValidatesPayload(t *testing.T) {
	runtimeScope := newDocumentBridgeTestScope()
	engine := newTestQuickjsEngine(t, WithDocumentStorage(runtimeScope))

	missingArg := evalString(t, engine, `(() => {
		try {
			$choysum.document.deleteStoredContent();
			return "no-error";
		} catch (e) {
			return String(e);
		}
	})()`)
	if !strings.Contains(missingArg, "need 1 arg: payload") {
		t.Fatalf("expected missing payload error, got %q", missingArg)
	}

	missingKey := evalString(t, engine, `(() => {
		try {
			$choysum.document.deleteStoredContent({});
			return "no-error";
		} catch (e) {
			return String(e);
		}
	})()`)
	if !strings.Contains(missingKey, "storedContentId is required") {
		t.Fatalf("expected storedContentId validation error, got %q", missingKey)
	}
}

func TestWithDocumentStorageProviderUsesExecContext(t *testing.T) {
	type ctxKey struct{}
	runtimeCtx := context.WithValue(context.Background(), ctxKey{}, "runtime")
	baseScope := newDocumentBridgeTestScope()
	var seenCtx context.Context

	engine := newTestQuickjsEngine(t, WithDocumentStorageProvider(func(ctx context.Context) scope.Scope {
		seenCtx = ctx
		if ctx == nil {
			return baseScope
		}
		return baseScope.WithContext(ctx)
	}))

	evalString(t, engine, `(function() {
		globalThis.$choysum = globalThis.$choysum || {};
		globalThis.$choysum.__rpc__ = async function(req) {
			try {
				$choysum.document.deleteStoredContent({ storedContentId: "sc_001" });
				return { id: req.id, result: { ok: true }, context: {} };
			} catch (e) {
				return { id: req.id, result: { error: String(e) }, context: {} };
			}
		};
		return "ok";
	})()`)

	resp, err := engine.Execute(runtimeCtx, &jsengine.JsRequest{Id: "req-1", Service: "demo"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected result object, got %#v", resp.Result)
	}
	if _, exists := result["error"]; !exists {
		t.Fatalf("expected document delete to surface an error result, got %#v", result)
	}
	if seenCtx == nil || seenCtx.Value(ctxKey{}) != "runtime" {
		t.Fatalf("expected provider to receive runtime exec ctx, got %#v", seenCtx)
	}
}
