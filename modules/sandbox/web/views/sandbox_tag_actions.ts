// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineModelActions } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';

const { _lt } = createTranslate('sandbox', { scope: 'web/views/SandboxTagFormView' });

/** Standard CRUD action ids for sandbox.SandboxTag. */
export const sandboxTagActions = defineModelActions('sandbox.SandboxTag', {
  entityTitle: _lt('Sandbox Tag'),
});
