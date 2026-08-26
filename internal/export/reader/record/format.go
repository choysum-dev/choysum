// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package record

import (
	"fmt"
	"strconv"
	"strings"
)

func formatCell(record map[string]any, field string) string {
	if strings.Contains(field, "/") {
		parts := strings.SplitN(field, "/", 2)
		if v, ok := record[field]; ok {
			return scalarString(v)
		}
		rel, ok := record[parts[0]].(map[string]any)
		if !ok {
			return ""
		}
		return scalarString(rel[parts[1]])
	}
	return scalarString(record[field])
}

func scalarString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	default:
		return fmt.Sprint(v)
	}
}

func recordRows(records []map[string]any, fields []string) [][]string {
	if len(records) == 0 {
		return nil
	}
	out := make([][]string, 0, len(records))
	for _, rec := range records {
		row := make([]string, len(fields))
		for i, field := range fields {
			row[i] = formatCell(rec, field)
		}
		out = append(out, row)
	}
	return out
}
