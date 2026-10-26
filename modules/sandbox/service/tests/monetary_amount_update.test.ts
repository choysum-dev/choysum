// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import SandboxRecord from '@/sandbox/service/models/sandbox_record';
import { MetadataStorage } from '@/core/service/api/metadata';
import { getCurrencyFieldName } from '@/core/service/orm/metadata/decimal_like';

test('sandbox.SandboxRecord Amount metadata exposes currencyField CurrencyId', () => {
  const meta = MetadataStorage.instance.getModelMetadata(SandboxRecord as any);
  const amount = meta?.fields?.get('Amount');
  expect(amount?.type).toBe('monetary');
  expect(getCurrencyFieldName(amount?.column)).toBe('CurrencyId');
});

test('sandbox.SandboxRecord Update Amount with existing CurrencyId does not require currency in patch', async () => {
  const row = await SandboxRecord.Search([], {
    fields: ['Id', 'Amount', 'CurrencyId'],
    limit: 5,
  });
  const record = row.find(r => r.CurrencyId != null && r.CurrencyId !== '');
  expect(record).toBeTruthy();

  const nextAmount = '12.34';
  await SandboxRecord.UpdateById(String(record!.Id), { Amount: nextAmount } as any);

  const refreshed = await SandboxRecord.Browse(String(record!.Id), ['Amount', 'CurrencyId']);
  expect(String(refreshed.Amount)).toBe(nextAmount);
  expect(refreshed.CurrencyId).toBeTruthy();
});
