// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineModelActions } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';

const { _lt } = createTranslate('web', { scope: 'web/views/UserFilterFormView' });

/**
 * Shared CRUD action ids for the Settings → web UserFilter admin List/Form.
 * End-user Favorites remain on OSearch; these ids gate the sys.admin ledger
 * (labeled Saved Filters in Settings chrome).
 */
export const userFilterActions = defineModelActions('web.UserFilter', {
  entityTitle: _lt('Saved Filter'),
});
