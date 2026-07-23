// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Address from '@/base/service/models/address';
import Bank from '@/base/service/models/bank';
import Sequence from '@/base/service/models/sequence';
import UoMCategory from '@/base/service/models/uom_category';
import { MetadataStorage } from '@/core/service/api/metadata';
import { withContext } from '@/core/service/api/context';

import { uid } from './_helpers';

function expectTranslateTrigram(model: any, fieldName: string, size: number) {
  const field = MetadataStorage.instance.getModelMetadata(model).fields.get(fieldName);
  expect(field?.translate).toBe(true);
  expect(field?.column?.index).toBe('trigram');
  expect(field?.storageHints?.size).toBe(size);
}

test('base catalog display fields expose translate+trigram metadata', () => {
  expectTranslateTrigram(Bank, 'Name', 120);
  expectTranslateTrigram(UoMCategory, 'Name', 100);
  expectTranslateTrigram(Sequence, 'Name', 100);
  expectTranslateTrigram(Address, 'Label', 120);
});

test('base.bank: Name bilingual write/read unwraps by lang', async () => {
  const enName = uid('BankEn');
  const zhName = uid('BankZh');
  const created = await Bank.Create(
    {
      Name: { en_US: enName, zh_CN: zhName } as any,
      IsActive: true,
    } as any,
    ['Id', 'Name'] as any
  );
  expect(String((created as any).Name)).toBe(enName);
  const id = String((created as any).Id);
  const zh = await withContext({ lang: 'zh_CN' }, () => Bank.Browse(id, ['Id', 'Name'] as any));
  expect(String((zh as any).Name)).toBe(zhName);
});
