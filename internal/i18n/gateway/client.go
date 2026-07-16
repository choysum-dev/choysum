// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package i18ngateway

import (
	"context"
	"fmt"
	"strings"

	i18nservice "github.com/choysum-dev/choysum/internal/i18n/service"
	"github.com/choysum-dev/choysum/pkg/grpc/client"
	"github.com/choysum-dev/choysum/pkg/grpc/converter"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc/metadata"
)

const internalKeyHeader = "x-choysum-internal-key"

type appTranslations struct {
	Hash  string
	Terms map[string]map[string]map[string]string
}

// fetchAppTranslations dials {app}.I18n/GetTranslations with internal identity (D1).
func fetchAppTranslations(ctx context.Context, runtimeScope scope.Scope, app, lang string, moduleNames []string) (*appTranslations, error) {
	app = strings.TrimSpace(app)
	if app == "" {
		return nil, fmt.Errorf("application is required")
	}

	reqMsg, err := i18nservice.NewRequestMessage(app)
	if err != nil {
		return nil, err
	}
	moduleNamesAny := make([]any, 0, len(moduleNames))
	for _, m := range moduleNames {
		moduleNamesAny = append(moduleNamesAny, m)
	}
	if err := converter.MapToMessage(map[string]any{
		"lang":         lang,
		"module_names": moduleNamesAny,
		"hash":         "",
	}, reqMsg); err != nil {
		return nil, fmt.Errorf("build GetTranslations request: %w", err)
	}

	respMsg, err := i18nservice.NewResponseMessage(app)
	if err != nil {
		return nil, err
	}

	rpcCtx := outgoingContextForInternalRPC(ctx, runtimeScope)
	conn, err := client.Dial(rpcCtx, app+".I18n")
	if err != nil {
		return nil, client.ToStatusError(err)
	}
	if err := conn.Invoke(rpcCtx, i18nservice.FullMethod(app), reqMsg, respMsg); err != nil {
		return nil, client.ToStatusError(err)
	}

	out, err := converter.MessageToMap(respMsg)
	if err != nil {
		return nil, fmt.Errorf("decode GetTranslations response: %w", err)
	}
	return parseAppTranslations(out), nil
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
