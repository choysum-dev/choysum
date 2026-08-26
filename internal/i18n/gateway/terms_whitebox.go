// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package i18ngateway

import (
	"context"

	"github.com/choysum-dev/choysum/internal/i18n/terms"
	"google.golang.org/grpc"
)

const (
	translationTermSearch = terms.TranslationTermSearch
	translationTermCount  = terms.TranslationTermCount
)

func parseTermItem(app string, m map[string]any) termItem {
	return terms.ParseTermItem(app, m)
}

func parseSearchResult(app, lang string, limit, offset int, total int64, out map[string]any) *searchTermsResult {
	return terms.ParseSearchResult(app, lang, limit, offset, total, out)
}

func toInt64(v any) int64 {
	return terms.ToInt64(v)
}

func termStatus(value string) string {
	return terms.TermStatus(value)
}

func invokeTranslationTermCount(ctx context.Context, conn *grpc.ClientConn, service string, condition map[string]any) (int64, error) {
	return terms.InvokeTranslationTermCount(ctx, conn, service, condition)
}

func searchTranslationTermPage(ctx context.Context, conn *grpc.ClientConn, service, app, lang string, condition map[string]any, total int64, limit, offset int) (*searchTermsResult, error) {
	return terms.SearchTranslationTermPage(ctx, conn, service, app, lang, condition, total, limit, offset)
}
