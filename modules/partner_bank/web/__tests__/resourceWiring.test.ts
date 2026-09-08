// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineModelActions } from '@/core/web/resource';
import { createTermReference } from '@/core/service/i18n';

test('partner_bank resource wiring: defineModelActions yields ids for BankAccount', () => {
  const actions = defineModelActions('partner.BankAccount', {
    entityTitle: createTermReference('partner', 'Bank Account', { scope: 'test' }),
  });
  expect(actions.create).toBeTruthy();
  expect(actions.edit).toBeTruthy();
  expect(actions.delete).toBeTruthy();
});
