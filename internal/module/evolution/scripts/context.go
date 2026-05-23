// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package scripts

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/rs/xid"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
)

type execCtxPayload struct {
	requestId string
	execCtx   context.Context
	payload   map[string]any
}

func BuildJsContext(ctx context.Context, runtimeScope scope.Scope, module *meta.IrModule, version string, fromVersion string) execCtxPayload {
	traceId := ""
	spanId := ""
	span := trace.SpanFromContext(ctx)
	if span != nil {
		sc := span.SpanContext()
		if sc.IsValid() {
			traceId = sc.TraceID().String()
			spanId = sc.SpanID().String()
		}
	}
	if traceId == "" {
		traceId = xid.New().String()
	}
	if spanId == "" {
		spanId = xid.New().String()
	}

	userId := "admin"
	if id := auth.IdentityFromContext(ctx); id != nil && id.IsValid() {
		if v := strings.TrimSpace(id.GetUserID()); v != "" {
			userId = v
		}
	}

	payload := map[string]any{
		"ctx": map[string]any{},
		"identity": map[string]any{
			"userId": userId,
		},
		"req": map[string]any{
			"requestId": spanId,
			"traceId":   traceId,
			"depth":     0,
			"kind":      "grpc",
		},
		"module": map[string]any{
			"name":        module.Name,
			"version":     version,
			"fromVersion": fromVersion,
		},
	}

	return execCtxPayload{
		requestId: spanId,
		execCtx:   BuildExecContext(ctx, runtimeScope),
		payload:   payload,
	}
}

func contextWithEffectiveScope(ctx context.Context, runtimeScope scope.Scope) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := scope.SessionFromContext(ctx); ok {
		return ctx
	}
	if _, ok := scope.ScopeFromContext(ctx); ok {
		return ctx
	}
	if runtimeScope == nil {
		return ctx
	}
	return scope.ContextWithScope(ctx, runtimeScope)
}

func BuildExecContext(ctx context.Context, runtimeScope scope.Scope) context.Context {
	execCtx := contextWithEffectiveScope(ctx, runtimeScope)
	if _, ok := auth.AccessTokenFromContext(execCtx); !ok {
		if key := strings.TrimSpace(runtimeOptionsFromScope(runtimeScope).authInternalKey); key != "" {
			execCtx = auth.ContextWithInternalKey(execCtx, key)
		}
	}

	md := metadata.New(nil)
	md.Set("x-choysum-depth", "0")
	if span := trace.SpanFromContext(ctx); span != nil {
		sc := span.SpanContext()
		if sc.IsValid() {
			traceparent := fmt.Sprintf("00-%s-%s-01", sc.TraceID().String(), sc.SpanID().String())
			md.Set("traceparent", traceparent)
		}
	}
	execCtx = metadata.NewIncomingContext(execCtx, md)
	return execCtx
}

func LoadRuntimeScripts(runtimeScope scope.Scope, module *meta.IrModule) ([]*jsengine.JsScript, error) {
	if runtimeScope == nil || module == nil {
		return nil, fmt.Errorf("missing env or module")
	}
	if strings.TrimSpace(module.ServiceEntryPoint) == "" {
		return nil, nil
	}
	runtimeOpts := runtimeOptionsFromScope(runtimeScope)
	mode := strings.ToLower(strings.TrimSpace(runtimeOpts.compileBundleMode))
	if mode == "" {
		mode = "bundle"
	}
	var scriptPath string
	if mode == "bundle" {
		scriptPath = config.BundlesIndexJS(runtimeOpts.distPath)
	} else {
		app := strings.TrimSpace(module.ApplicationStr)
		if app == "" {
			app = strings.TrimSpace(module.Name)
		}
		if app == "" {
			return nil, fmt.Errorf("module application is empty")
		}
		scriptPath = config.AppIndexJS(runtimeOpts.distPath, app)
	}
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		if os.IsNotExist(err) {
			if strings.TrimSpace(module.ServiceEntryPoint) == "" {
				return nil, nil
			}
		}
		return nil, err
	}
	return []*jsengine.JsScript{{FileName: scriptPath, Content: string(content)}}, nil
}
