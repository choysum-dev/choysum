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
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/dynamicpb"
)

const (
	translationTermSearch = "Search"
	translationTermCount  = "Count"
)

type termItem struct {
	Application string `json:"application"`
	Module      string `json:"module"`
	Scope       string `json:"scope"`
	Src         string `json:"src"`
	Value       string `json:"value"`
	Kind        string `json:"kind"`
	Source      string `json:"source,omitempty"`
	Status      string `json:"status,omitempty"`
}

type searchTermsResult struct {
	Lang   string
	Items  []termItem
	Total  int64
	Limit  int
	Offset int
}

// fetchAppSearchTerms dials {app}.TranslationTerm/Count then Search with the caller's token (PO export).
func fetchAppSearchTerms(ctx context.Context, runtimeScope scope.Scope, accessToken, app, lang string, modules []string, q string, limit, offset int) (*searchTermsResult, error) {
	_ = runtimeScope
	app = strings.TrimSpace(app)
	lang = strings.TrimSpace(lang)
	q = strings.TrimSpace(q)
	if app == "" {
		return nil, fmt.Errorf("application is required")
	}
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	service := app + "." + translationTermModelName
	condition := buildTermSearchCondition(lang, modules, q)

	rpcCtx := outgoingContextForUserRPC(ctx, accessToken)
	conn, err := client.Dial(rpcCtx, service)
	if err != nil {
		return nil, client.ToStatusError(err)
	}

	total, err := invokeTranslationTermCount(rpcCtx, conn, service, condition)
	if err != nil {
		return nil, err
	}
	return searchTranslationTermPage(rpcCtx, conn, service, app, lang, condition, total, limit, offset)
}

// countAppTerms dials {app}.TranslationTerm/Count with the caller's token.
func countAppTerms(ctx context.Context, accessToken, app, lang string, modules []string, q string) (int64, error) {
	app = strings.TrimSpace(app)
	if app == "" {
		return 0, fmt.Errorf("application is required")
	}
	service := app + "." + translationTermModelName
	condition := buildTermSearchCondition(strings.TrimSpace(lang), modules, strings.TrimSpace(q))
	rpcCtx := outgoingContextForUserRPC(ctx, accessToken)
	conn, err := client.Dial(rpcCtx, service)
	if err != nil {
		return 0, client.ToStatusError(err)
	}
	return invokeTranslationTermCount(rpcCtx, conn, service, condition)
}

// searchAppTermsPage dials Search only (caller supplies total from Count).
func searchAppTermsPage(ctx context.Context, accessToken, app, lang string, modules []string, q string, total int64, limit, offset int) (*searchTermsResult, error) {
	app = strings.TrimSpace(app)
	lang = strings.TrimSpace(lang)
	q = strings.TrimSpace(q)
	if app == "" {
		return nil, fmt.Errorf("application is required")
	}
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	service := app + "." + translationTermModelName
	condition := buildTermSearchCondition(lang, modules, q)
	rpcCtx := outgoingContextForUserRPC(ctx, accessToken)
	conn, err := client.Dial(rpcCtx, service)
	if err != nil {
		return nil, client.ToStatusError(err)
	}
	return searchTranslationTermPage(rpcCtx, conn, service, app, lang, condition, total, limit, offset)
}

func searchTranslationTermPage(ctx context.Context, conn *grpc.ClientConn, service, app, lang string, condition map[string]any, total int64, limit, offset int) (*searchTermsResult, error) {
	searchMD, err := loader.Global().GetMethodDescriptor(service + "." + translationTermSearch)
	if err != nil {
		return nil, fmt.Errorf("load Search descriptor: %w", err)
	}
	reqMsg := dynamicpb.NewMessage(searchMD.Input())
	if err := converter.MapToMessage(map[string]any{
		"condition": condition,
		"options": map[string]any{
			"fields": []any{"Application", "Module", "Scope", "Src", "Value", "Kind", "Source", "Comments"},
			"limit":  limit,
			"offset": offset,
			"orderBy": []any{
				map[string]any{"field": "Module", "order": "asc"},
				map[string]any{"field": "Scope", "order": "asc"},
				map[string]any{"field": "Src", "order": "asc"},
				map[string]any{"field": "Kind", "order": "asc"},
			},
		},
	}, reqMsg); err != nil {
		return nil, fmt.Errorf("build Search request: %w", err)
	}

	respMsg := dynamicpb.NewMessage(searchMD.Output())
	if err := conn.Invoke(ctx, "/"+service+"/"+translationTermSearch, reqMsg, respMsg); err != nil {
		return nil, client.ToStatusError(err)
	}

	out, err := converter.MessageToMap(respMsg)
	if err != nil {
		return nil, fmt.Errorf("decode Search response: %w", err)
	}
	return parseSearchResult(app, lang, limit, offset, total, out), nil
}

func buildTermSearchCondition(lang string, modules []string, q string) map[string]any {
	and := []any{
		[]any{"Lang", "=", lang},
	}
	wantedModules := make([]string, 0, len(modules))
	for _, m := range modules {
		m = strings.TrimSpace(m)
		if m != "" {
			wantedModules = append(wantedModules, m)
		}
	}
	switch len(wantedModules) {
	case 0:
		// language-wide
	case 1:
		and = append(and, []any{"Module", "=", wantedModules[0]})
	default:
		ors := make([]any, 0, len(wantedModules))
		for _, m := range wantedModules {
			ors = append(ors, []any{"Module", "=", m})
		}
		and = append(and, map[string]any{"Or": ors})
	}
	if q != "" {
		like := "%" + q + "%"
		and = append(and, map[string]any{"Or": []any{
			[]any{"Module", "ilike", like},
			[]any{"Scope", "ilike", like},
			[]any{"Src", "ilike", like},
			[]any{"Value", "ilike", like},
		}})
	}
	return map[string]any{"And": and}
}

func invokeTranslationTermCount(ctx context.Context, conn *grpc.ClientConn, service string, condition map[string]any) (int64, error) {
	md, err := loader.Global().GetMethodDescriptor(service + "." + translationTermCount)
	if err != nil {
		return 0, fmt.Errorf("load Count descriptor: %w", err)
	}
	reqMsg := dynamicpb.NewMessage(md.Input())
	if err := converter.MapToMessage(map[string]any{
		"condition": condition,
		"options":   map[string]any{},
	}, reqMsg); err != nil {
		return 0, fmt.Errorf("build Count request: %w", err)
	}
	respMsg := dynamicpb.NewMessage(md.Output())
	if err := conn.Invoke(ctx, "/"+service+"/"+translationTermCount, reqMsg, respMsg); err != nil {
		return 0, client.ToStatusError(err)
	}
	out, err := converter.MessageToMap(respMsg)
	if err != nil {
		return 0, fmt.Errorf("decode Count response: %w", err)
	}
	return toInt64(out["result"]), nil
}

func parseSearchResult(app, lang string, limit, offset int, total int64, out map[string]any) *searchTermsResult {
	result := &searchTermsResult{
		Lang:   lang,
		Limit:  limit,
		Offset: offset,
		Total:  total,
	}
	rawItems, _ := out["result"].([]any)
	for _, raw := range rawItems {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		result.Items = append(result.Items, parseTermItem(app, m))
	}
	return result
}

func parseTermItem(app string, m map[string]any) termItem {
	if m == nil {
		return termItem{Application: app}
	}
	item := termItem{
		Application: mapString(m, "Application", "application"),
		Module:      mapString(m, "Module", "module"),
		Scope:       mapString(m, "Scope", "scope"),
		Src:         mapString(m, "Src", "src"),
		Value:       mapString(m, "Value", "value"),
		Kind:        mapString(m, "Kind", "kind"),
		Source:      mapString(m, "Source", "source"),
	}
	if item.Application == "" {
		item.Application = app
	}
	if item.Kind == "" {
		item.Kind = "literal"
	}
	// Comments hold PO file:line refs only; TranslationTerm has no fuzzy flag field.
	item.Status = termStatus(item.Value)
	return item
}

func mapString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := m[key]; ok {
			s := strings.TrimSpace(fmt.Sprintf("%v", v))
			if s == "" || s == "<nil>" {
				return ""
			}
			return s
		}
	}
	return ""
}

func termStatus(value string) string {
	if strings.TrimSpace(value) == "" {
		return "missing"
	}
	return "translated"
}

func toInt64(v any) int64 {
	switch typed := v.(type) {
	case int:
		return int64(typed)
	case int32:
		return int64(typed)
	case int64:
		return typed
	case float64:
		return int64(typed)
	case float32:
		return int64(typed)
	default:
		s := strings.TrimSpace(fmt.Sprintf("%v", v))
		if s == "" || s == "<nil>" {
			return 0
		}
		var n int64
		_, _ = fmt.Sscanf(s, "%d", &n)
		return n
	}
}
