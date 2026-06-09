// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import UoMCategory from '@/base/service/models/uom_category';

import { uid, updateInstanceWithRetry } from './_helpers';

test('base.proxy_symbol_binding: instance update persists changed scalar field', async () => {
  const row = await UoMCategory.Create(
    {
      Name: uid('ProxySymbolBindingCategory'),
      Code: uid('PSB').slice(-16),
      IsActive: true,
    } as any,
    ['Id', 'Name'] as any
  );

  const id = String((row as any).Id);

  const inst = await UoMCategory.Browse(id, ['Id', 'Name'] as any);
  const updatedName = uid('ProxySymbolBindingUpdatedName');
  await updateInstanceWithRetry(inst as any, { Name: updatedName });

  const after = await UoMCategory.Browse(id, ['Id', 'Name'] as any);
  expect(String((after as any).Name)).toBe(updatedName);
});
