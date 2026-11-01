// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineModelActions } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';

const { _lt } = createTranslate('web', { scope: 'web/views/AppSettingFormView' });

/**
 * Shared CRUD action ids for the Settings AppSetting (Parameters) UI.
 * Model ACL remains per-app via AppSetting logical packs; these ids are
 * the UI resource grants checked by hasAction.
 */
export const appSettingActions = defineModelActions('web.AppSetting', {
  entityTitle: _lt('Parameter'),
});
