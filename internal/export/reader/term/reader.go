// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package term

import (
	"context"
	"fmt"
	"strings"

	exportplan "github.com/choysum-dev/choysum/internal/export/plan"
	"github.com/choysum-dev/choysum/internal/export/registry"
	"github.com/choysum-dev/choysum/internal/i18n/terms"
	"github.com/choysum-dev/choysum/pkg/auth"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// Reader exports terminology-profile units through TranslationTerm Search.
type Reader struct{}

// Read implements registry.Reader.
func (Reader) Read(ctx context.Context, runtimeScope scope.Scope, p exportplan.Plan) (registry.Result, error) {
	_ = runtimeScope

	accessToken, _ := auth.AccessTokenFromContext(ctx)
	if strings.TrimSpace(accessToken) == "" && !auth.IsAuthenticated(ctx) {
		return registry.Result{}, exportpkg.Errorf(exportpkg.CodeInvalidSpec, "authentication is required for terminology export")
	}

	app := strings.TrimSpace(p.Application)
	module := strings.TrimSpace(p.Module)
	lang := strings.TrimSpace(p.Lang)
	if app == "" || module == "" || lang == "" {
		return registry.Result{}, exportpkg.Errorf(exportpkg.CodeInvalidSpec, "application, module, and lang are required for terminology export")
	}

	items, truncated, err := terms.CollectAll(ctx, accessToken, app, lang, []string{module})
	if err != nil {
		return registry.Result{}, exportpkg.ErrorfWrap(exportpkg.CodeInvalidSpec, "terminology search failed", err)
	}

	entries := terms.BuildPOEntries(lang, items)
	count := len(items)
	result := registry.Result{
		UnitCount: count,
		POEntries: entries,
		Truncated: truncated,
		Outcomes: registry.Outcomes{
			Total: count,
			Ok:    count,
		},
	}
	if truncated {
		result.Messages = append(result.Messages, registry.Message{
			Type: "warning",
			Code: "truncated",
			Text: fmt.Sprintf("terminology export truncated at %d items", terms.ExportMaxItems),
		})
		result.Outcomes.Warning = 1
	}
	return result, nil
}
