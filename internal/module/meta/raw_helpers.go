// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// Raw model / child table names (declaration layer). Prefer these over exporting Raw* types.
const (
	rawModelTable         = "meta_raw_model"
	rawFieldTable         = "meta_raw_field"
	rawServiceTable       = "meta_raw_service"
	rawDecoratorTable     = "meta_raw_decorator"
	rawArgumentTable      = "meta_raw_argument"
	rawParameterTable     = "meta_raw_parameter"
	rawTypeParameterTable = "meta_raw_type_parameter"
)

// countRawModelsByID counts declaration rows with the given id (including soft-deleted when unscoped).
func countRawModelsByID(db *gorm.DB, id string, unscoped bool) (int64, error) {
	if db == nil {
		return 0, fmt.Errorf("db is nil")
	}
	id = strings.TrimSpace(id)
	q := db
	if unscoped {
		q = db.Unscoped()
	}
	var n int64
	err := q.Table(rawModelTable).Where("id = ?", id).Count(&n).Error
	return n, err
}

// dropRawModelTable drops meta_raw_model (tests that force list/load failures).
func dropRawModelTable(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	return db.Migrator().DropTable(rawModelTable)
}
