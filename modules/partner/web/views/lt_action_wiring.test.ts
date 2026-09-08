// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { partnerActions, partnerOpenDetailAction } from './partner_actions';

test('partner view action wiring: shared partner_actions yields Partner CRUD ids', () => {
  expect(partnerActions.create).toBe('partner.action.partner_create');
  expect(partnerActions.edit).toBe('partner.action.partner_edit');
  expect(partnerActions.copy).toBe('partner.action.partner_copy');
  expect(partnerActions.delete).toBe('partner.action.partner_delete');
});

test('partner view action wiring: shared partner_actions yields open_detail id', () => {
  expect(partnerOpenDetailAction).toBe('partner.action.partner_open_detail');
});
