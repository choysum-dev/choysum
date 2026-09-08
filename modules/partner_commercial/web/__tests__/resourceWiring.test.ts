// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { partnerIdentifierActions } from '../views/partner_identifier_actions';

test('partner_commercial resource wiring: shared partner_identifier_actions yields Identifier ids', () => {
  expect(partnerIdentifierActions.create).toBe('partner.action.partner_identifier_create');
  expect(partnerIdentifierActions.edit).toBe('partner.action.partner_identifier_edit');
  expect(partnerIdentifierActions.delete).toBe('partner.action.partner_identifier_delete');
});
