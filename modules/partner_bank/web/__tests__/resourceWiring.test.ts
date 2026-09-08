// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { bankAccountActions } from '../views/bank_account_actions';

test('partner_bank resource wiring: shared bank_account_actions yields BankAccount ids', () => {
  expect(bankAccountActions.create).toBe('partner.action.bank_account_create');
  expect(bankAccountActions.edit).toBe('partner.action.bank_account_edit');
  expect(bankAccountActions.delete).toBe('partner.action.bank_account_delete');
});
