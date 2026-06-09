// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Bank from '@/base/service/models/bank';

import { uid } from './_helpers';

test('base.bank: Code is normalized (trim+upper)', async () => {
  const created = await Bank.Create(
    {
      Name: uid('Bank'),
      Code: ' ab12 ',
      IsActive: true,
    } as any,
    ['Id', 'Code'] as any
  );

  expect(String((created as any).Code)).toBe('AB12');
});

test('base.bank: Code empty becomes null', async () => {
  const created = await Bank.Create(
    {
      Name: uid('BankEmptyCode'),
      Code: '   ',
      IsActive: true,
    } as any,
    ['Id', 'Code'] as any
  );

  expect((created as any).Code).toBe(null);
});
