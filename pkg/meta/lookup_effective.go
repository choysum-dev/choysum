// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// LookupEffectiveModel returns the live effective meta_model row for (application, name).
// Prefer empty module_id (E2 projection) when legacy declaration shells still coexist.
// No tip Order — callers must not use created_at/id DESC to pick among same-name rows.
func LookupEffectiveModel(db *gorm.DB, application, name string) (*Model, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	application = strings.TrimSpace(application)
	name = strings.TrimSpace(name)
	if application == "" || name == "" {
		return nil, fmt.Errorf("lookup effective requires application and name")
	}

	var rows []Model
	if err := db.Where("application = ? AND name = ?", application, name).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("lookup effective %s.%s: %w", application, name, err)
	}
	if len(rows) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	picked := pickEffectiveAmong(rows)
	return &picked, nil
}

func pickEffectiveAmong(rows []Model) Model {
	if len(rows) == 1 {
		return rows[0]
	}
	var best *Model
	for i := range rows {
		row := &rows[i]
		emptyModule := !row.ModuleId.Valid || strings.TrimSpace(row.ModuleId.String) == ""
		if best == nil {
			best = row
			continue
		}
		bestEmpty := !best.ModuleId.Valid || strings.TrimSpace(best.ModuleId.String) == ""
		if emptyModule && !bestEmpty {
			best = row
			continue
		}
		if emptyModule == bestEmpty {
			if row.UpdatedAt.After(best.UpdatedAt) ||
				(row.UpdatedAt.Equal(best.UpdatedAt) && row.Id.String > best.Id.String) {
				best = row
			}
		}
	}
	return *best
}

// IsEffectiveModelNotFound reports whether err is a missing effective model.
func IsEffectiveModelNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
