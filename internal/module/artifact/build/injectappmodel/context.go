// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package injectappmodel

import (
	"github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/gorm"
)

// BuildCtx is the read-only build context for Decide / Materialize / Supersede.
// Callers apply returned Effects (virtual sources, entry imports) themselves.
type BuildCtx struct {
	Module      *meta.Module
	DB          *gorm.DB // may be nil
	ModulesPath string
}
