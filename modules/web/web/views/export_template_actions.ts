// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineModelActions } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';

const { _lt } = createTranslate('web', { scope: 'web/views/ExportTemplateFormView' });

/**
 * Shared CRUD action ids for the Settings → web ExportTemplate admin List/Form.
 * End-user template CRUD remains on ExportPanel; these ids gate the sys.admin ledger.
 */
export const exportTemplateActions = defineModelActions('web.ExportTemplate', {
  entityTitle: _lt('Export Template'),
});
