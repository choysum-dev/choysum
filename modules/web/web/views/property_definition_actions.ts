// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineModelActions } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';

const { _lt } = createTranslate('web', { scope: 'web/views/PropertyDefinitionFormView' });

/**
 * Shared CRUD action ids for the Settings PropertyDefinition UI.
 * Model ACL remains per-app via PropertyDefinition logical packs; these ids are
 * the UI resource grants checked by hasAction.
 */
export const propertyDefinitionActions = defineModelActions('web.PropertyDefinition', {
  entityTitle: _lt('Property Definition'),
});
