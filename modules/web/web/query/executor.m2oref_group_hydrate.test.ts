// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { execute, __hydrateGroupRefLabelsForTest } from './executor';
import { asyncFnRecorder } from '@/web/web/__tests__/mountApp';
import { baseFieldFromGroupAlias } from './utils/relation/displayNameBatch';

test('baseFieldFromGroupAlias strips temporal suffix', () => {
  expect(baseFieldFromGroupAlias('CompanyId')).toBe('CompanyId');
  expect(baseFieldFromGroupAlias('CompanyId__month')).toBe('CompanyId');
  expect(baseFieldFromGroupAlias('')).toBe('');
});

test('execute readGroup hydrates ManyToOneRef group labels via createModelStore', async () => {
  const companySearch = asyncFnRecorder(async () => [
    { Id: 'dbtb6jfhsa67kjrja0qg', DisplayName: 'Main Company' },
  ]);
  const createModelStore = (modelName: string) => {
    if (modelName === 'base.Company') {
      return { storeId: 'base.Company', Search: companySearch } as any;
    }
    throw new Error(`unexpected model: ${modelName}`);
  };

  const store = {
    storeId: 'auth.User',
    fullModelName: 'auth.User',
    metadata: { fields: {
      CompanyId: { type: 'ManyToOneRef', relationModel: 'base.Company' },
      LanguageId: { type: 'ManyToOneRef', relationModel: 'base.Language' },
    } },
    ReadGroup: asyncFnRecorder(async () => [
      {
        depth: 0,
        keys: { CompanyId: 'dbtb6jfhsa67kjrja0qg' },
        labels: { CompanyId: 'dbtb6jfhsa67kjrja0qg' },
        metrics: {},
        count: 1,
        condition: ['CompanyId', '=', 'dbtb6jfhsa67kjrja0qg'],
      },
    ]),
  } as any;

  const snapshot = await execute(
    {
      main: { kind: 'readGroup', params: { groupby: [{ field: 'CompanyId' }] }, hash: 'rg-m2o-ref' },
      auxiliary: [],
    } as any,
    store,
    'list',
    { createModelStore }
  );

  expect(store.ReadGroup.calls.length).toBe(1);
  expect(companySearch.calls.length).toBe(1);
  expect(companySearch.calls[0]?.[0]).toEqual(['Id', 'in', ['dbtb6jfhsa67kjrja0qg']]);

  const row = snapshot.rows[0] as any;
  expect(row?.kind).toBe('group');
  expect(row?.keys?.CompanyId).toBe('dbtb6jfhsa67kjrja0qg');
  expect(row?.labels?.CompanyId).toBe('Main Company');
  expect(row?.label).toBe('Main Company');
  expect(row?.raw?.labels?.CompanyId).toBe('Main Company');
});

test('execute readGroup keeps raw id when ManyToOneRef target row is missing', async () => {
  const languageSearch = asyncFnRecorder(async () => []);
  const createModelStore = (modelName: string) => {
    if (modelName === 'base.Language') {
      return { storeId: 'base.Language', Search: languageSearch } as any;
    }
    throw new Error(`unexpected model: ${modelName}`);
  };

  const missingId = 'lang-missing-id';
  const store = {
    storeId: 'auth.User',
    fullModelName: 'auth.User',
    metadata: { fields: {
      LanguageId: { type: 'ManyToOneRef', relationModel: 'base.Language' },
    } },
    ReadGroup: asyncFnRecorder(async () => [
      {
        depth: 0,
        keys: { LanguageId: missingId },
        labels: { LanguageId: missingId },
        metrics: {},
        count: 2,
        condition: ['LanguageId', '=', missingId],
      },
    ]),
  } as any;

  const snapshot = await execute(
    {
      main: { kind: 'readGroup', params: { groupby: [{ field: 'LanguageId' }] }, hash: 'rg-m2o-ref-miss' },
      auxiliary: [],
    } as any,
    store,
    'list',
    { createModelStore }
  );

  const row = snapshot.rows[0] as any;
  expect(row?.keys?.LanguageId).toBe(missingId);
  expect(row?.labels?.LanguageId).toBe(missingId);
  expect(row?.label).toBe(missingId);
});

test('execute readGroup hydrates nested ManyToOneRef child group labels', async () => {
  const companySearch = asyncFnRecorder(async () => [
    { Id: 'c-parent', DisplayName: 'Parent Co' },
    { Id: 'c-child', DisplayName: 'Child Co' },
  ]);
  const createModelStore = (modelName: string) => {
    if (modelName === 'base.Company') {
      return { storeId: 'base.Company', Search: companySearch } as any;
    }
    throw new Error(`unexpected model: ${modelName}`);
  };

  const store = {
    storeId: 'auth.User',
    fullModelName: 'auth.User',
    metadata: { fields: {
      CompanyId: { type: 'ManyToOneRef', relationModel: 'base.Company' },
    } },
    ReadGroup: asyncFnRecorder(async () => [
      {
        depth: 0,
        keys: { CompanyId: 'c-parent' },
        labels: { CompanyId: 'c-parent' },
        metrics: {},
        count: 1,
        condition: ['CompanyId', '=', 'c-parent'],
        children: [
          {
            depth: 1,
            keys: { CompanyId: 'c-child' },
            labels: { CompanyId: 'c-child' },
            metrics: {},
            count: 1,
            condition: ['CompanyId', '=', 'c-child'],
          },
        ],
      },
    ]),
  } as any;

  const snapshot = await execute(
    {
      main: { kind: 'readGroup', params: { groupby: [{ field: 'CompanyId' }] }, hash: 'rg-nested' },
      auxiliary: [],
    } as any,
    store,
    'list',
    { createModelStore }
  );

  const parent = snapshot.rows[0] as any;
  expect(parent?.label).toBe('Parent Co');
  expect(parent?.children?.[0]?.label).toBe('Child Co');
  expect(parent?.children?.[0]?.keys?.CompanyId).toBe('c-child');
  expect(companySearch.calls.length).toBe(1);
  const cond = companySearch.calls[0]?.[0] as unknown;
  const ids = Array.isArray(cond) ? (cond[2] as string[]) : [];
  expect(new Set(ids)).toEqual(new Set(['c-parent', 'c-child']));
});

test('execute readGroup leaves ManyToOne labels untouched', async () => {
  const createModelStore = () => {
    throw new Error('createModelStore should not be called for ManyToOne-only groups');
  };

  const store = {
    storeId: 'demo.Order',
    fullModelName: 'demo.Order',
    metadata: { fields: {
      OwnerId: { type: 'ManyToOne', relationModel: 'demo.User' },
    } },
    ReadGroup: asyncFnRecorder(async () => [
      {
        depth: 0,
        keys: { OwnerId: 'u1' },
        labels: { OwnerId: 'Alice' },
        metrics: {},
        count: 3,
        condition: ['OwnerId', '=', 'u1'],
      },
    ]),
  } as any;

  const snapshot = await execute(
    {
      main: { kind: 'readGroup', params: { groupby: [{ field: 'OwnerId' }] }, hash: 'rg-m2o' },
      auxiliary: [],
    } as any,
    store,
    'list',
    { createModelStore }
  );

  const row = snapshot.rows[0] as any;
  expect(row?.label).toBe('Alice');
  expect(row?.labels?.OwnerId).toBe('Alice');
});

test('hydrateGroupRefLabels no-ops on empty rows and swallows apply errors', async () => {
  await __hydrateGroupRefLabelsForTest({ metadata: { fields: {} } } as any, [], {
    createModelStore: () => {
      throw new Error('should not create');
    },
  });

  // Throw while collecting nested children so the outer try/catch is exercised.
  const boomRow = {
    keys: { CompanyId: 'c1' },
    labels: { CompanyId: 'c1' },
    get children() {
      throw new Error('children boom');
    },
  };
  await __hydrateGroupRefLabelsForTest(
    {
      metadata: { fields: {
        CompanyId: { type: 'ManyToOneRef', relationModel: 'base.Company' },
      } },
    } as any,
    [boomRow as any],
    {
      createModelStore: (() => ({
        Search: async () => [{ Id: 'c1', DisplayName: 'Acme' }],
      })) as any,
    }
  );
});

test('hydrateGroupRefLabels only rewrites Ref aliases when mixed with ManyToOne', async () => {
  const companySearch = asyncFnRecorder(async () => [{ Id: 'c1', DisplayName: 'Acme' }]);
  const rows = [
    {
      kind: 'group' as const,
      key: 'c1|u1',
      depth: 0,
      label: 'Alice',
      keys: { CompanyId: 'c1', OwnerId: 'u1' },
      labels: { CompanyId: 'c1', OwnerId: 'Alice' },
      raw: { labels: { CompanyId: 'c1', OwnerId: 'Alice' } },
    },
  ];

  await __hydrateGroupRefLabelsForTest(
    {
      metadata: { fields: {
        CompanyId: { type: 'ManyToOneRef', relationModel: 'base.Company' },
        OwnerId: { type: 'ManyToOne', relationModel: 'demo.User' },
      } },
    } as any,
    rows as any,
    {
      createModelStore: ((model: string) => {
        if (model !== 'base.Company') throw new Error(model);
        return { Search: companySearch };
      }) as any,
    }
  );

  // Sorted first alias is CompanyId → primary label becomes Acme.
  expect(rows[0]?.labels?.CompanyId).toBe('Acme');
  expect(rows[0]?.labels?.OwnerId).toBe('Alice');
  expect(rows[0]?.raw?.labels?.CompanyId).toBe('Acme');
  expect(rows[0]?.raw?.labels?.OwnerId).toBe('Alice');
  expect(rows[0]?.label).toBe('Acme');
});
