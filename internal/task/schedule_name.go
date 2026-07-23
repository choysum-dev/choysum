// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"encoding/json"
	"strings"

	"gorm.io/gorm"
)

// EncodeTranslatedScheduleName stores a schedule display name as a data-i18n
// lang map JSON document (base language en_US).
func EncodeTranslatedScheduleName(name string) string {
	name = strings.TrimSpace(name)
	raw, err := json.Marshal(map[string]string{"en_US": name})
	if err != nil {
		return `{"en_US":""}`
	}
	return string(raw)
}

// DecodeTranslatedScheduleName unwraps a stored schedule name.
// Accepts legacy plain strings and translated lang-map JSON.
func DecodeTranslatedScheduleName(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.HasPrefix(raw, "{") {
		return raw
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return raw
	}
	if v := strings.TrimSpace(m["en_US"]); v != "" {
		return v
	}
	if v := strings.TrimSpace(m["zh_CN"]); v != "" {
		return v
	}
	for _, v := range m {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return raw
}

// WhereScheduleNameEq matches task_schedule.name for either legacy varchar
// storage or translated jsonb/json lang maps (data-i18n).
func WhereScheduleNameEq(db *gorm.DB, name string) *gorm.DB {
	if db == nil {
		return db
	}
	name = strings.TrimSpace(name)
	encoded := EncodeTranslatedScheduleName(name)
	dialect := ""
	if db.Dialector != nil {
		dialect = strings.ToLower(strings.TrimSpace(db.Dialector.Name()))
	}
	switch dialect {
	case "postgres", "postgresql":
		// Avoid `name = <plain string>` on jsonb columns (SQLSTATE 22P02).
		return db.Where(`(name->>'en_US' = ? OR name = ?::jsonb)`, name, encoded)
	default:
		// TEXT/JSON dialects: match legacy plain string or exact lang-map document.
		// Do not call json_extract on plain strings (SQLite returns malformed JSON).
		return db.Where(`(name = ? OR name = ?)`, name, encoded)
	}
}
