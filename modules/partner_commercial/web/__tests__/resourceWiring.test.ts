// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineModelActions } from '@/core/web/resource';
import { createTermReference } from '@/core/service/i18n';

test('partner_commercial resource wiring: defineModelActions yields ids for PartnerIdentifier', () => {
  const actions = defineModelActions('partner.PartnerIdentifier', {
    entityTitle: createTermReference('partner', 'Identifier', { scope: 'test' }),
  });
  expect(actions.create).toBeTruthy();
  expect(actions.edit).toBeTruthy();
  expect(actions.delete).toBeTruthy();
});
