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
	Id        sql.NullString `gorm:"primaryKey;type:char(20)"`
	CreatedAt time.Time      `gorm:"index"`
	UpdatedAt time.Time      `gorm:"index"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
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
// DualStoreRawEntities (callers that need both should append both slices).
func Entities() []any {
	out := []any{
		&Application{},
		&Module{},
		&Component{},
		&UiResource{},
		&UiResourceMenuRoute{},
		&UiResourceRouteAction{},
	}
	return append(out, DualStoreEffectiveEntities()...)
}

// CatalogEntities returns Entities plus DualStoreRawEntities for AutoMigrate of the
// full dual-store catalog (effective projection + declaration tables).
func CatalogEntities() []any {
	out := append([]any{}, Entities()...)
	return append(out, DualStoreRawEntities()...)
}

// DualStoreRawEntities returns declaration-layer tables only.
func DualStoreRawEntities() []any {
	return []any{
		&rawModel{},
		&rawField{},
		&rawService{},
		&rawTypeParameter{},
		&rawParameter{},
		&rawDecorator{},
		&rawArgument{},
	}
}

// DualStoreEffectiveEntities returns effective projection tables (current meta_* names).
func DualStoreEffectiveEntities() []any {
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
