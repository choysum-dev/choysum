// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package i18ngateway

import (
	"context"

	"github.com/choysum-dev/choysum/internal/i18n/terms"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type termItem = terms.Item
type searchTermsResult = terms.SearchResult

func fetchAppSearchTerms(ctx context.Context, runtimeScope scope.Scope, accessToken, app, lang string, modules []string, q string, limit, offset int) (*searchTermsResult, error) {
	return terms.FetchAppSearchTerms(ctx, runtimeScope, accessToken, app, lang, modules, q, limit, offset)
}

func countAppTerms(ctx context.Context, accessToken, app, lang string, modules []string, q string) (int64, error) {
	return terms.CountAppTerms(ctx, accessToken, app, lang, modules, q)
}

func searchAppTermsPage(ctx context.Context, accessToken, app, lang string, modules []string, q string, total int64, limit, offset int) (*searchTermsResult, error) {
	return terms.SearchAppTermsPage(ctx, accessToken, app, lang, modules, q, total, limit, offset)
}

func buildTermSearchCondition(lang string, modules []string, q string) map[string]any {
	return terms.BuildSearchCondition(lang, modules, q)
}
