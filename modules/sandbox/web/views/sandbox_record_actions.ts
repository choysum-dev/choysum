// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineAction, defineModelActions } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';

const { _lt: listLt } = createTranslate('sandbox', { scope: 'web/views/SandboxRecordListView' });
const { _lt: formLt } = createTranslate('sandbox', { scope: 'web/views/SandboxRecordFormView' });

/** Open sandbox record detail from the list. */
export const sandboxRecordOpenDetailAction = defineAction('sandbox.action.sandbox_record_open_detail', {
  title: listLt('Open Sandbox Record Detail'),
  requires: [{ model: 'sandbox.SandboxRecord' }],
});

/** Standard CRUD action ids for sandbox.SandboxRecord. */
export const sandboxRecordActions = defineModelActions('sandbox.SandboxRecord', {
  entityTitle: formLt('Sandbox Record'),
});
