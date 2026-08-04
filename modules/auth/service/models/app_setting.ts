// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Model, AppSettingBaseModel } from '@/core/service';

/**
 * Auth AppSetting store (handwritten; supersedes C2 for this app).
 * Business code must still resolve via `pool<AppSettingModelCtor>('AppSetting')`.
 */
@Model('AppSetting', { softDelete: false })
export default class AppSetting extends AppSettingBaseModel {}
