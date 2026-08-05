// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package injectappmodel

func init() {
	Register(Spec{
		ModelName:        "FieldDefault",
		GeneratedRelPath: "service/models/__generated__/field_default.ts",
		DuplicateCode:    "FIELD_DEFAULT_DUPLICATE",
		BaseModelFile:    "core/service/orm/model/field_default_base_model.ts",
	})
	Register(Spec{
		ModelName:                   "AppSetting",
		GeneratedRelPath:            "service/models/__generated__/app_setting.ts",
		DuplicateCode:               "APP_SETTING_DUPLICATE",
		BaseModelFile:               "core/service/orm/model/app_setting_base_model.ts",
		SoftDeleteFalse:             true,
		ForeignClaimOnOwnerReinject: true,
	})
}
