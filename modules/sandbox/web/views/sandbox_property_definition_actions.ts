// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineModelActions } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';

const { _lt } = createTranslate('sandbox', { scope: 'web/views/SandboxPropertyDefinitionFormView' });

/** Standard CRUD action ids for sandbox.PropertyDefinition (C2-injected logical model). */
export const sandboxPropertyDefinitionActions = defineModelActions('sandbox.PropertyDefinition', {
  entityTitle: _lt('Property Definition'),
});
