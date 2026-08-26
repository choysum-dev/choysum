// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package plan

import (
	"encoding/json"
	"strings"

	exportpkg "github.com/choysum-dev/choysum/pkg/export"
)

// SearchCondition builds the ORM Search condition from a record export plan.
// When Ids are set they take precedence over Domain (EX22).
func SearchCondition(p Plan) (map[string]any, error) {
	if len(p.Ids) > 0 {
		return map[string]any{
			"And": []any{[]any{"Id", "in", append([]string(nil), p.Ids...)}},
		}, nil
	}
	domain := strings.TrimSpace(p.Domain)
	if domain == "" {
		return map[string]any{"And": []any{}}, nil
	}
	var decoded any
	if err := json.Unmarshal([]byte(domain), &decoded); err != nil {
		return nil, exportpkg.Errorf(exportpkg.CodeInvalidSpec, "domain must be valid JSON")
	}
	switch v := decoded.(type) {
	case nil:
		return map[string]any{"And": []any{}}, nil
	case map[string]any:
		return v, nil
	case []any:
		if len(v) > 0 {
			if _, ok := v[0].(string); ok {
				return map[string]any{"And": []any{v}}, nil
			}
		}
		return map[string]any{"And": v}, nil
	default:
		return nil, exportpkg.Errorf(exportpkg.CodeInvalidSpec, "domain must be an object or condition tuple")
	}
}

// ResolveExportFields returns explicit plan fields or the model default export set.
func ResolveExportFields(p Plan, defaultFields func(string) ([]string, error)) ([]string, error) {
	if len(p.Fields) > 0 {
		out := make([]string, len(p.Fields))
		copy(out, p.Fields)
		return out, nil
	}
	if defaultFields == nil {
		return nil, exportpkg.Errorf(exportpkg.CodeInvalidSpec, "export fields are required")
	}
	return defaultFields(p.Model)
}
