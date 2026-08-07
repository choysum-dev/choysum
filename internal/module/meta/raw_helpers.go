// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

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
