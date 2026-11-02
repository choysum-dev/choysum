// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel } from '@/core/service';
import BankAccount from '../models/bank_account';

test('BankAccount: extends BaseModel (no MessageThread collaboration surface)', () => {
  expect(BankAccount.prototype instanceof BaseModel).toBe(true);
  expect((BankAccount as any).MessagePost).toBeUndefined();
  expect((BankAccount as any).MessageFollow).toBeUndefined();
  expect((BankAccount as any).MessageUnfollow).toBeUndefined();
  expect((BankAccount as any).MessageSearchByRecord).toBeUndefined();
});

test('BankAccount: validateEntity applies assertRequiredText and assertAccountType', async () => {
  const originalFill = (BankAccount as any).fillBankSnapshot;
  const originalSearch = (BankAccount as any).Search;
  (BankAccount as any).fillBankSnapshot = async (values: Record<string, any>) => {
    values.BankNameSnapshot = 'Mock Bank';
  };
  (BankAccount as any).Search = async () => [];
  try {
    const values: Record<string, any> = {
      PartnerId: 'p1',
      CompanyId: 'c1',
      BankId: 'b1',
      AccountName: '  Ops  ',
      AccountNo: '1234567890',
      AccountType: 'checking',
      AllowInbound: true,
      AllowOutbound: false,
    };
    await (BankAccount as any).validateEntity(values);
    expect(values.AccountName).toBe('Ops');
    expect(values.AccountNo).toBe('1234567890');
    expect(values.AccountType).toBe('checking');
    expect(values.AccountNoLast4).toBe('7890');
    expect(values.BankNameSnapshot).toBe('Mock Bank');
  } finally {
    (BankAccount as any).fillBankSnapshot = originalFill;
    (BankAccount as any).Search = originalSearch;
  }
});
