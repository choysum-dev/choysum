// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package i18ngateway

import (
	"context"
	"fmt"
	"strings"

	"github.com/choysum-dev/choysum/pkg/grpc/client"
	"github.com/choysum-dev/choysum/pkg/grpc/converter"
	"github.com/choysum-dev/choysum/pkg/grpc/loader"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/dynamicpb"
)

const (
	internalKeyHeader              = "x-choysum-internal-key"
	translationTermModelName       = "TranslationTerm"
	translationTermGetTranslations = "GetTranslations"
)

type appTranslations struct {
	Hash  string
	Terms map[string]map[string]map[string]string
}

// fetchAppTranslations dials {app}.TranslationTerm/GetTranslations with internal identity (D1).
func fetchAppTranslations(ctx context.Context, runtimeScope scope.Scope, app, lang string, moduleNames []string) (*appTranslations, error) {
	app = strings.TrimSpace(app)
	if app == "" {
		return nil, fmt.Errorf("application is required")
	}

	service := app + "." + translationTermModelName
	descriptorMethod := service + "." + translationTermGetTranslations
	invokeMethod := "/" + service + "/" + translationTermGetTranslations

	md, err := loader.Global().GetMethodDescriptor(descriptorMethod)
	if err != nil {
		return nil, fmt.Errorf("load GetTranslations descriptor: %w", err)
	}

	moduleNamesAny := make([]any, 0, len(moduleNames))
	for _, m := range moduleNames {
		moduleNamesAny = append(moduleNamesAny, m)
	}
	// Generated protos wrap the single TS param as google.protobuf.Value req
	// (TranslationTerm_GetTranslations_Req), matching document gateway's {"req": ...} shape.
	reqMsg := dynamicpb.NewMessage(md.Input())
	if err := converter.MapToMessage(map[string]any{
		"req": map[string]any{
			"lang":         lang,
			"module_names": moduleNamesAny,
			"hash":         "",
		},
	}, reqMsg); err != nil {
		return nil, fmt.Errorf("build GetTranslations request: %w", err)
	}

	rpcCtx := outgoingContextForInternalRPC(ctx, runtimeScope)
	conn, err := client.Dial(rpcCtx, service)
	if err != nil {
		return nil, client.ToStatusError(err)
	}
	respMsg := dynamicpb.NewMessage(md.Output())
	if err := conn.Invoke(rpcCtx, invokeMethod, reqMsg, respMsg); err != nil {
		return nil, client.ToStatusError(err)
	}

	out, err := converter.MessageToMap(respMsg)
	if err != nil {
		return nil, fmt.Errorf("decode GetTranslations response: %w", err)
	}
	payload, err := unwrapGetTranslationsPayload(out)
	if err != nil {
		return nil, err
	}
	return parseAppTranslations(payload), nil
}

// unwrapGetTranslationsPayload accepts Resp{ Value result = 1 } or a legacy
// unwrapped body. A present non-object result is a decode error (not empty catalog).
func unwrapGetTranslationsPayload(out map[string]any) (map[string]any, error) {
	payload, ok := out["result"].(map[string]any)
	if ok {
		return payload, nil
	}
	if _, hasResult := out["result"]; hasResult {
		return nil, fmt.Errorf("decode GetTranslations response: result must be an object")
	}
	return out, nil
}

func parseAppTranslations(out map[string]any) *appTranslations {
	result := &appTranslations{
		Hash:  strings.TrimSpace(fmt.Sprintf("%v", out["hash"])),
		Terms: map[string]map[string]map[string]string{},
	}
	if result.Hash == "<nil>" {
		result.Hash = ""
	}
	raw, ok := out["terms_by_module"].(map[string]any)
	if !ok || raw == nil {
		return result
	}
	for mod, byScopeAny := range raw {
		byScope, ok := byScopeAny.(map[string]any)
		if !ok {
			continue
		}
		modMap := make(map[string]map[string]string, len(byScope))
		for scopeKey, bySrcAny := range byScope {
			bySrc, ok := bySrcAny.(map[string]any)
			if !ok {
				continue
			}
			srcMap := make(map[string]string, len(bySrc))
			for src, val := range bySrc {
				srcMap[src] = fmt.Sprintf("%v", val)
			}
			modMap[scopeKey] = srcMap
		}
		result.Terms[mod] = modMap
	}
	return result
}

func outgoingContextForInternalRPC(ctx context.Context, runtimeScope scope.Scope) context.Context {
	md := metadata.MD{}
	if authOpts, ok := scope.AuthRuntimeOptionsFromScope(runtimeScope); ok {
		key := strings.TrimSpace(authOpts.InternalKey)
		env := ""
		if serverOpts, ok := scope.ServerRuntimeOptionsFromScope(runtimeScope); ok {
			env = strings.TrimSpace(serverOpts.Environment)
		}
		if key != "" && !strings.EqualFold(env, "production") {
			md.Set(internalKeyHeader, key)
		}
	}
	md.Set("x-choysum-depth", "1")
	return metadata.NewOutgoingContext(ctx, md)
}
