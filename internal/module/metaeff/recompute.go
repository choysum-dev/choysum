// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

// Package metaeff owns incremental effective-catalog recompute for EDS-2+.
package metaeff

import (
	"github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/gorm"
)

// LogicalKey identifies one effective catalog row.
type LogicalKey = meta.LogicalKey

// RecomputeEffective rebuilds one effective projection from live raw rows.
func RecomputeEffective(tx *gorm.DB, application, name string) error {
	return meta.RecomputeEffective(tx, application, name)
}

// RecomputeKeys rebuilds effective projections for each logical key.
func RecomputeKeys(tx *gorm.DB, keys []LogicalKey) error {
	return meta.RecomputeKeys(tx, keys)
}
