// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package record

import (
	"context"
	"fmt"

	exportplan "github.com/choysum-dev/choysum/internal/export/plan"
	"github.com/choysum-dev/choysum/internal/export/registry"
	importcaller "github.com/choysum-dev/choysum/internal/import/caller"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// Reader exports record-profile units through ORM Search (no raw SQL).
type Reader struct{}

// Read implements registry.Reader.
func (Reader) Read(ctx context.Context, runtimeScope scope.Scope, p exportplan.Plan) (registry.Result, error) {
	_ = runtimeScope

	fields, err := exportplan.ResolveExportFields(p, DefaultExportFields)
	if err != nil {
		return registry.Result{}, err
	}

	if p.Mode == exportpkg.ModeTemplate {
		return registry.Result{Headers: fields}, nil
	}

	caller, ok := importcaller.CallerFromContext(ctx)
	if !ok {
		return registry.Result{}, exportpkg.Errorf(exportpkg.CodeInvalidSpec, "ORM caller is required for record export")
	}

	cond, err := exportplan.SearchCondition(p)
	if err != nil {
		return registry.Result{}, err
	}

	opts := map[string]any{"fields": fields}
	if p.Limit > 0 {
		opts["limit"] = p.Limit
	}
	if p.Offset > 0 {
		opts["offset"] = p.Offset
	}

	raw, err := caller.Call(ctx, importcaller.CallRequest{
		Model:  p.Model,
		Method: "Search",
		Args:   []any{cond, opts},
	})
	if err != nil {
		return registry.Result{}, exportpkg.ErrorfWrap(exportpkg.CodeInvalidSpec, "record search failed", err)
	}

	records, err := parseSearchRecords(raw)
	if err != nil {
		return registry.Result{}, err
	}

	rows := recordRows(records, fields)
	count := len(rows)
	return registry.Result{
		UnitCount: count,
		Headers:   fields,
		Rows:      rows,
		Outcomes: registry.Outcomes{
			Total: count,
			Ok:    count,
		},
	}, nil
}

func parseSearchRecords(raw any) ([]map[string]any, error) {
	items, ok := raw.([]any)
	if !ok {
		return nil, exportpkg.Errorf(exportpkg.CodeInvalidSpec, "unexpected search result type")
	}
	out := make([]map[string]any, 0, len(items))
	for i, item := range items {
		rec, ok := item.(map[string]any)
		if !ok {
			return nil, exportpkg.Errorf(exportpkg.CodeInvalidSpec, fmt.Sprintf("unexpected record at index %d", i))
		}
		out = append(out, rec)
	}
	return out, nil
}
