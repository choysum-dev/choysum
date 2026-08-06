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

func fetchAppSearchTerms(ctx context.Context, runtimeScope scope.Scope, accessToken, app, lang string, modules []string, q string, limit, offset int) (*searchTermsResult, error) {
	_ = runtimeScope
	app = strings.TrimSpace(app)
	if app == "" {
		return nil, fmt.Errorf("application is required")
	}

	reqMsg, err := i18nservice.NewSearchRequestMessage(app)
	if err != nil {
		return nil, err
	}
	modulesAny := make([]any, 0, len(modules))
	for _, m := range modules {
		modulesAny = append(modulesAny, m)
	}
	if err := converter.MapToMessage(map[string]any{
		"lang":    lang,
		"modules": modulesAny,
		"q":       q,
		"limit":   limit,
		"offset":  offset,
	}, reqMsg); err != nil {
		return nil, fmt.Errorf("build SearchTerms request: %w", err)
	}

	respMsg, err := i18nservice.NewSearchResponseMessage(app)
	if err != nil {
		return nil, err
	}

	rpcCtx := outgoingContextForUserRPC(ctx, accessToken)
	conn, err := client.Dial(rpcCtx, app+".I18n")
	if err != nil {
		return nil, client.ToStatusError(err)
	}
	if err := conn.Invoke(rpcCtx, i18nservice.FullMethod(app, i18nservice.MethodSearchTerms), reqMsg, respMsg); err != nil {
		return nil, client.ToStatusError(err)
	}

	out, err := converter.MessageToMap(respMsg)
	if err != nil {
		return nil, fmt.Errorf("decode SearchTerms response: %w", err)
	}
	return parseSearchTermsResult(app, out), nil
}

func parseSearchTermsResult(app string, out map[string]any) *searchTermsResult {
	result := &searchTermsResult{
		Lang:   strings.TrimSpace(fmt.Sprintf("%v", out["lang"])),
		Items:  nil,
		Total:  toInt64(out["total"]),
		Limit:  int(toInt64(out["limit"])),
		Offset: int(toInt64(out["offset"])),
	}
	rawItems, _ := out["items"].([]any)
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
		Application: strings.TrimSpace(fmt.Sprintf("%v", m["application"])),
		Module:      strings.TrimSpace(fmt.Sprintf("%v", m["module"])),
		Scope:       strings.TrimSpace(fmt.Sprintf("%v", m["scope"])),
		Src:         fmt.Sprintf("%v", m["src"]),
		Value:       fmt.Sprintf("%v", m["value"]),
		Kind:        strings.TrimSpace(fmt.Sprintf("%v", m["kind"])),
		Source:      strings.TrimSpace(fmt.Sprintf("%v", m["source"])),
		Status:      strings.TrimSpace(fmt.Sprintf("%v", m["status"])),
	}
	if item.Application == "" || item.Application == "<nil>" {
		item.Application = app
	}
	if item.Src == "<nil>" {
		item.Src = ""
	}
	if item.Value == "<nil>" {
		item.Value = ""
	}
	if item.Kind == "" || item.Kind == "<nil>" {
		item.Kind = "literal"
	}
	if item.Source == "<nil>" {
		item.Source = ""
	}
	if item.Status == "<nil>" {
		item.Status = ""
	}
	return item
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
