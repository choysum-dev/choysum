// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package terms

import (
	"context"
)

// Tunable for tests; production defaults keep PO downloads bounded.
var (
	ExportPageSize = 500
	ExportMaxItems = 10000
)

// CollectAll pages TranslationTerm rows up to ExportMaxItems.
func CollectAll(ctx context.Context, accessToken, app, lang string, modules []string) ([]Item, bool, error) {
	if ExportMaxItems <= 0 {
		return nil, true, nil
	}

	hooks, hasHooks := collectHooksFromContext(ctx)

	var (
		total int64
		err   error
	)
	if hasHooks && hooks.search != nil {
		probe, err := hooks.search(ctx, accessToken, app, lang, modules, "", 1, 0)
		if err != nil {
			return nil, false, err
		}
		if probe != nil {
			total = probe.Total
		}
	} else {
		total, err = CountAppTerms(ctx, accessToken, app, lang, modules, "")
		if err != nil {
			return nil, false, err
		}
	}

	var all []Item
	offset := 0
	truncated := false
	for {
		remaining := ExportMaxItems - len(all)
		page := ExportPageSize
		if page > remaining {
			page = remaining
		}
		var result *SearchResult
		if hasHooks && hooks.search != nil {
			result, err = hooks.search(ctx, accessToken, app, lang, modules, "", page, offset)
		} else {
			result, err = SearchAppTermsPage(ctx, accessToken, app, lang, modules, "", total, page, offset)
		}
		if err != nil {
			return nil, false, err
		}
		if result == nil || len(result.Items) == 0 {
			break
		}
		all = append(all, result.Items...)
		offset += len(result.Items)
		if len(all) >= ExportMaxItems {
			if total <= 0 || total > int64(len(all)) {
				truncated = true
			}
			break
		}
		if total <= 0 && result.Total > 0 {
			total = result.Total
		}
		if len(result.Items) < page {
			break
		}
		if total > 0 && int64(offset) >= total {
			break
		}
	}
	return all, truncated, nil
}
