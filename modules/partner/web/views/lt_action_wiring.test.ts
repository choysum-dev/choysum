// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineAction, defineModelActions } from '@/core/web/resource';
import { createTermReference, createTranslate } from '@/core/service/i18n';

const { _lt } = createTranslate('partner', { scope: 'web/views' });

test('partner view action wiring: defineModelActions yields ids for Partner', () => {
  const actions = defineModelActions('partner.Partner', {
    entityTitle: createTermReference('partner', 'Partner', { scope: 'test' }),
  });
  expect(actions.create).toBeTruthy();
  expect(actions.edit).toBeTruthy();
  expect(actions.copy).toBeTruthy();
  expect(actions.delete).toBeTruthy();
});

test('partner view action wiring: defineAction returns open_detail id', () => {
  expect(defineAction('partner.action.partner_open_detail', { title: _lt('Open Partner Detail') })).toBe(
    'partner.action.partner_open_detail'
  );
});
