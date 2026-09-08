// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineModelActions } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';

const { _lt } = createTranslate('partner_commercial', { scope: 'web/views/PartnerFormView' });

/** CRUD action ids for partner.PartnerIdentifier on the partner form extension. */
export const partnerIdentifierActions = defineModelActions('partner.PartnerIdentifier', {
  entityTitle: _lt('Identifier'),
});
