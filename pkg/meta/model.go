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

// Entities returns the core model set for external use.
func Entities() []any {
	return []any{
		&Application{},
		&Module{},
		&Component{},
		&Model{},
		&Field{},
		&Service{},
		&TypeParameter{},
		&Parameter{},
		&Decorator{},
		&Argument{},
		&UiResource{},
		&UiResourceMenuRoute{},
		&UiResourceRouteAction{},
		// Declaration layer (EDS raw dual-store). Empty until Persist is rewired (EDS-2).
		&RawModel{},
		&RawField{},
		&RawService{},
		&RawTypeParameter{},
		&RawParameter{},
		&RawDecorator{},
		&RawArgument{},
	}
}

// DualStoreRawEntities returns declaration-layer tables only.
func DualStoreRawEntities() []any {
	return []any{
		&RawModel{},
		&RawField{},
		&RawService{},
		&RawTypeParameter{},
		&RawParameter{},
		&RawDecorator{},
		&RawArgument{},
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
