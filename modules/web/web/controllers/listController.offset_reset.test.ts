// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createListController } from './listController';

function makeStore(overrides: Record<string, unknown> = {}) {
  const searchOffsets: number[] = [];
  const store = {
    fullModelName: 'base.Company',
    storeId: 'list-offset-reset',
    metadata: { fields: { Name: { type: 'varchar' } } },
    state: {
      queryState: {
        keyword: '',
        appliedFilters: [],
        appliedGroups: [],
        keywordFields: [],
        pagination: { limit: 20, offset: 0 },
      },
      result: null,
      selection: [],
      planCache: new Map(),
    },
    setContext: () => {},
    getContext: () => ({}),
    withContext: async (_c: unknown, fn: () => unknown) => fn(),
    Search: async (_domain: unknown, opts: { offset?: number } = {}) => {
      searchOffsets.push(Number(opts?.offset ?? 0));
      return [{ Id: '1', Name: 'Acme' }];
    },
    Count: async () => 40,
    ReadGroup: async () => ({ groups: [], total: 0 }),
    ReadGroupCount: async () => 0,
    ...overrides,
  } as any;
  return { store, searchOffsets };
}

async function seedPage2(ctrl: ReturnType<typeof createListController>, store: any) {
  await ctrl.apply();
  await ctrl.paginate({ limit: 20, offset: 20 });
  expect(store.state.queryState.pagination.offset).toBe(20);
}

test('listController: sort resets offset to 0', async () => {
  const { store, searchOffsets } = makeStore();
  const ctrl = createListController(store);
  await seedPage2(ctrl, store);
  searchOffsets.length = 0;
  await ctrl.sort([{ field: 'Name', direction: 'desc' }]);
  expect(store.state.queryState.pagination.offset).toBe(0);
  expect(store.state.queryState.pagination.limit).toBe(20);
  expect(searchOffsets[searchOffsets.length - 1]).toBe(0);
});

test('listController: setKeyword resets offset to 0', async () => {
  const { store } = makeStore();
  const ctrl = createListController(store);
  await seedPage2(ctrl, store);
  await ctrl.setKeyword('acme');
  expect(store.state.queryState.pagination.offset).toBe(0);
  expect(store.state.queryState.keyword).toBe('acme');
});

test('listController: setAppliedFilters resets offset to 0', async () => {
  const { store } = makeStore();
  const ctrl = createListController(store);
  await seedPage2(ctrl, store);
  await ctrl.setAppliedFilters([{ And: [['Name', 'ilike', 'x']] }] as any);
  expect(store.state.queryState.pagination.offset).toBe(0);
});

test('listController: setAppliedGroups resets offset to 0', async () => {
  const { store } = makeStore();
  const ctrl = createListController(store);
  await seedPage2(ctrl, store);
  await ctrl.setAppliedGroups(['Name'] as any);
  expect(store.state.queryState.pagination.offset).toBe(0);
});

test('listController: same forcedCondition soft apply keeps offset', async () => {
  const { store } = makeStore();
  const ctrl = createListController(store);
  await ctrl.apply({ forcedCondition: ['Active', '=', true] as any });
  await ctrl.paginate({ limit: 20, offset: 20 });
  expect(store.state.queryState.pagination.offset).toBe(20);
  await ctrl.apply({ forcedCondition: ['Active', '=', true] as any });
  expect(store.state.queryState.pagination.offset).toBe(20);
});

test('listController: different forcedCondition resets offset', async () => {
  const { store } = makeStore();
  const ctrl = createListController(store);
  await ctrl.apply({ forcedCondition: ['Active', '=', true] as any });
  await ctrl.paginate({ limit: 20, offset: 20 });
  await ctrl.apply({ forcedCondition: ['Active', '=', false] as any });
  expect(store.state.queryState.pagination.offset).toBe(0);
});

test('listController: explicit pagination with orderBy keeps offset', async () => {
  const { store, searchOffsets } = makeStore();
  const ctrl = createListController(store);
  await seedPage2(ctrl, store);
  searchOffsets.length = 0;
  await ctrl.apply({
    orderBy: [{ field: 'Name', direction: 'asc' }],
    pagination: { limit: 20, offset: 40 },
  });
  expect(store.state.queryState.pagination.offset).toBe(40);
  expect(searchOffsets[searchOffsets.length - 1]).toBe(40);
});

test('listController: paginate alone keeps requested offset', async () => {
  const { store } = makeStore();
  const ctrl = createListController(store);
  await ctrl.apply();
  await ctrl.paginate({ limit: 20, offset: 20 });
  expect(store.state.queryState.pagination.offset).toBe(20);
  await ctrl.paginate({ limit: 20, offset: 20 });
  expect(store.state.queryState.pagination.offset).toBe(20);
});

test('listController: grouped apply drops leftover List orderBy from ReadGroup only', async () => {
  const readGroupOrderBys: unknown[] = [];
  const { store } = makeStore({
    ReadGroup: async (_gb: unknown, _cond: unknown, opts: { orderBy?: unknown } = {}) => {
      readGroupOrderBys.push(opts?.orderBy);
      return { groups: [{ Revoked: false, __count: 1 }], total: 1 };
    },
  });
  const ctrl = createListController(store);
  await ctrl.apply();
  await ctrl.sort([{ field: 'CreatedAt', direction: 'asc' }]);
  expect(store.state.queryState.orderBy).toEqual([{ field: 'CreatedAt', direction: 'asc' }]);
  readGroupOrderBys.length = 0;
  await ctrl.apply({ appliedGroups: ['Revoked'] as any });
  // Collection orderBy stays in store; only the grouped RPC omits it.
  expect(store.state.queryState.orderBy).toEqual([{ field: 'CreatedAt', direction: 'asc' }]);
  expect(readGroupOrderBys.length).toBeGreaterThan(0);
  for (const ob of readGroupOrderBys) {
    expect(ob == null || (Array.isArray(ob) && ob.length === 0)).toBe(true);
  }
});
