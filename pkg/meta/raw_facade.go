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
	RawModelTable         = "meta_raw_model"
	RawFieldTable         = "meta_raw_field"
	RawServiceTable       = "meta_raw_service"
	RawDecoratorTable     = "meta_raw_decorator"
	RawArgumentTable      = "meta_raw_argument"
	RawParameterTable     = "meta_raw_parameter"
	RawTypeParameterTable = "meta_raw_type_parameter"
)

// CountRawModelsByID counts declaration rows with the given id (including soft-deleted when unscoped).
func CountRawModelsByID(db *gorm.DB, id string, unscoped bool) (int64, error) {
	if db == nil {
		return 0, fmt.Errorf("db is nil")
	}
	id = strings.TrimSpace(id)
	q := db
	if unscoped {
		q = db.Unscoped()
	}
	var n int64
	err := q.Table(RawModelTable).Where("id = ?", id).Count(&n).Error
	return n, err
}

// DropRawModelTable drops meta_raw_model (tests that force list/load failures).
func DropRawModelTable(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	return db.Migrator().DropTable(RawModelTable)
}
