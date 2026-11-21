// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { batchLoadRefRecordsByModel, baseFieldFromGroupAlias } from './displayNameBatch';

test('batchLoadRefRecordsByModel maps Id to records and skips empty sets', async () => {
  const searches: Array<{ model: string; ids: string[] }> = [];
  const createModelStore = (modelName: string) =>
    ({
      Search: async (cond: any) => {
        const ids = Array.isArray(cond?.[2]) ? cond[2].map(String) : [];
        searches.push({ model: modelName, ids });
        if (modelName === 'base.Company') {
          return ids.filter((id: string) => id === 'c1').map((Id: string) => ({ Id, DisplayName: 'Acme' }));
        }
        return [];
      },
    }) as any;

  const batch = new Map<string, Set<string>>([
    ['base.Company', new Set(['c1', 'c2'])],
    ['base.Language', new Set()],
  ]);

  const lookups = await batchLoadRefRecordsByModel(batch, createModelStore);
  expect(searches).toEqual([{ model: 'base.Company', ids: ['c1', 'c2'] }]);
  expect(lookups.get('base.Company')?.get('c1')).toEqual({ Id: 'c1', DisplayName: 'Acme' });
  expect(lookups.get('base.Company')?.has('c2')).toBe(false);
  expect(lookups.has('base.Language')).toBe(false);
});

test('batchLoadRefRecordsByModel tolerates target Search failures', async () => {
  const createModelStore = () =>
    ({
      Search: async () => {
        throw new Error('boom');
      },
    }) as any;

  const batch = new Map<string, Set<string>>([['base.Company', new Set(['c1'])]]);
  const lookups = await batchLoadRefRecordsByModel(batch, createModelStore);
  expect(lookups.size).toBe(0);
});

test('batchLoadRefRecordsByModel chunks large id sets', async () => {
  const searches: string[][] = [];
  const createModelStore = () =>
    ({
      Search: async (cond: any) => {
        const ids = Array.isArray(cond?.[2]) ? cond[2].map(String) : [];
        searches.push(ids);
        return ids.map((Id: string) => ({ Id, DisplayName: Id }));
      },
    }) as any;

  const ids = Array.from({ length: 201 }, (_, i) => `id-${i}`);
  const batch = new Map<string, Set<string>>([['base.Company', new Set(ids)]]);
  const lookups = await batchLoadRefRecordsByModel(batch, createModelStore);
  expect(searches.length).toBe(2);
  expect(searches[0]?.length).toBe(200);
  expect(searches[1]?.length).toBe(1);
  expect(lookups.get('base.Company')?.size).toBe(201);
});

test('batchLoadRefRecordsByModel skips stores without Search', async () => {
  const lookups = await batchLoadRefRecordsByModel(
    new Map([['base.Company', new Set(['c1'])]]),
    () => ({}) as any
  );
  expect(lookups.size).toBe(0);
});

test('baseFieldFromGroupAlias', () => {
  expect(baseFieldFromGroupAlias('LanguageId__day')).toBe('LanguageId');
});
