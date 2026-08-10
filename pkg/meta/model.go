// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"
	"time"

	"github.com/rs/xid"
	"gorm.io/gorm"
)

type BaseModel struct {
	Id         sql.NullString `gorm:"primaryKey;type:char(20)"`
	CreatedAt  time.Time      `gorm:"index"`
	UpdatedAt  time.Time      `gorm:"index"`
	DeletedAt  gorm.DeletedAt `gorm:"index"`
	CreatedUid sql.NullString `gorm:"size:20;index"`
	UpdatedUid sql.NullString `gorm:"size:20;index"`
	DeletedUid sql.NullString `gorm:"size:20;index"`
}

func (m *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if m.Id.String == "" {
		m.Id = sql.NullString{
			String: xid.New().String(),
			Valid:  true,
		}
	}
	return nil
}

// Entities returns AutoMigrate models for non-declaration catalog tables plus the
// effective dual-store projection. Declaration-layer tables are registered via
// internal/module/meta.DualStoreRawEntities (callers that need both should append both slices).
func Entities() []any {
	out := []any{
		&Application{},
		&Module{},
		&Component{},
		&UiResource{},
		&UiResourceMenuRoute{},
		&UiResourceRouteAction{},
	}
	return append(out, dualStoreEffectiveEntities()...)
}

// dualStoreEffectiveEntities returns effective projection tables (current meta_* names).
func dualStoreEffectiveEntities() []any {
	return []any{
		&Model{},
		&Field{},
		&Service{},
		&TypeParameter{},
		&Parameter{},
		&Decorator{},
		&Argument{},
	}
}
