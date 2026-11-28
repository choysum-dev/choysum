// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createListController } from './listController';

function makeStore(overrides: Record<string, unknown> = {}) {
  const readGroupOrderBys: unknown[] = [];
  const store = {
    fullModelName: 'message.Message',
    storeId: 'list-orderby-seed',
    metadata: {
      orderBy: [{ field: 'CreatedAt', order: 'asc' }],
      fields: { CreatedAt: { type: 'datetime' }, Name: { type: 'varchar' } },
    },
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
    Search: async () => [],
    ReadGroup: async (_gb: unknown, _cond: unknown, opts: { orderBy?: unknown } = {}) => {
      readGroupOrderBys.push(opts?.orderBy);
      return { groups: [], total: 0 };
    },
    ReadGroupCount: async () => 0,
    Count: async () => 0,
    ...overrides,
  } as any;
  store.__readGroupOrderBys = readGroupOrderBys;
  return store;
}

test('listController seeds queryState.orderBy from metadata.orderBy on first apply', async () => {
  const store = makeStore();
  const ctrl = createListController(store);
  await ctrl.apply();
  expect(store.state.queryState.orderBy).toEqual([{ field: 'CreatedAt', direction: 'asc' }]);
});

test('listController does not seed when metadata.orderBy is absent', async () => {
  const store = makeStore({
    metadata: { fields: { Name: { type: 'varchar' } } },
  });
  const ctrl = createListController(store);
  await ctrl.apply();
  expect(store.state.queryState.orderBy).toBeUndefined();
});

test('listController prefers explicit apply orderBy over metadata', async () => {
  const store = makeStore();
  const ctrl = createListController(store);
  await ctrl.apply({ orderBy: [{ field: 'Name', direction: 'desc' }] });
  expect(store.state.queryState.orderBy).toEqual([{ field: 'Name', direction: 'desc' }]);
});

test('listController reseeds metadata default when sort is cleared to empty', async () => {
  const store = makeStore();
  const ctrl = createListController(store);
  await ctrl.apply({ orderBy: [{ field: 'Name', direction: 'desc' }] });
  await ctrl.apply({ orderBy: [] });
  expect(store.state.queryState.orderBy).toEqual([{ field: 'CreatedAt', direction: 'asc' }]);
});

test('listController keeps existing queryState.orderBy across apply without override', async () => {
  const store = makeStore({
    metadata: { fields: { Name: { type: 'varchar' } } },
  });
  store.state.queryState.orderBy = [{ field: 'Name', direction: 'desc' }];
  const ctrl = createListController(store);
  await ctrl.apply();
  expect(store.state.queryState.orderBy).toEqual([{ field: 'Name', direction: 'desc' }]);
});

test('listController does not seed metadata.orderBy into grouped ReadGroup requests', async () => {
  const store = makeStore();
  const ctrl = createListController(store);
  await ctrl.apply({ appliedGroups: ['Name'] as any });
  expect(store.state.queryState.orderBy).toBeUndefined();
});

test('listController filters grouped RPC orderBy to group keys but keeps store collection orderBy', async () => {
  const store = makeStore();
  store.state.queryState.orderBy = [
    { field: 'CreatedAt', direction: 'asc' },
    { field: 'Name', direction: 'desc' },
  ];
  const ctrl = createListController(store);
  store.__readGroupOrderBys.length = 0;
  await ctrl.apply({ appliedGroups: ['Name'] as any });
  // Store keeps full collection orderBy across grouped apply without override.
  expect(store.state.queryState.orderBy).toEqual([
    { field: 'CreatedAt', direction: 'asc' },
    { field: 'Name', direction: 'desc' },
  ]);
  expect(store.__readGroupOrderBys.length).toBeGreaterThan(0);
  for (const ob of store.__readGroupOrderBys) {
    expect(ob).toEqual([{ field: 'Name', order: 'desc' }]);
  }
});

test('listController clears grouped sort when clear yields only non-group defaults', async () => {
  const store = makeStore();
  const ctrl = createListController(store);
  await ctrl.apply({
    appliedGroups: ['Name'] as any,
    orderBy: [],
  });
  expect(store.state.queryState.orderBy).toBeUndefined();
});

test('listController keeps explicit collection orderBy even when stored groups remain', async () => {
  const store = makeStore();
  store.state.queryState.appliedGroups = ['Name'];
  const ctrl = createListController(store);
  await ctrl.apply({
    kind: 'search',
    orderBy: [{ field: 'CreatedAt', direction: 'desc' }],
  });
  expect(store.state.queryState.orderBy).toEqual([{ field: 'CreatedAt', direction: 'desc' }]);
});

test('listController keeps group-key orderBy when apply({ kind: group }) reuses stored groups', async () => {
  const store = makeStore({
    metadata: { fields: { Name: { type: 'varchar' } } },
  });
  store.state.queryState.appliedGroups = ['Name'];
  store.state.queryState.orderBy = [{ field: 'Name', direction: 'desc' }];
  const ctrl = createListController(store);
  await ctrl.apply({ kind: 'group' });
  expect(store.state.queryState.orderBy).toEqual([{ field: 'Name', direction: 'desc' }]);
});

test('listController keeps store orderBy when grouped filter drops every field for RPC', async () => {
  const store = makeStore();
  store.state.queryState.orderBy = [{ field: 'CreatedAt', direction: 'asc' }];
  const ctrl = createListController(store);
  store.__readGroupOrderBys.length = 0;
  await ctrl.apply({ appliedGroups: ['Name'] as any });
  expect(store.state.queryState.orderBy).toEqual([{ field: 'CreatedAt', direction: 'asc' }]);
  expect(store.__readGroupOrderBys.length).toBeGreaterThan(0);
  for (const ob of store.__readGroupOrderBys) {
    expect(ob == null || (Array.isArray(ob) && ob.length === 0)).toBe(true);
  }
});

test('listController: List sort survives grouped apply then flat apply (Kanban round-trip)', async () => {
  const store = makeStore();
  const ctrl = createListController(store);
  await ctrl.apply({ orderBy: [{ field: 'CreatedAt', direction: 'desc' }] });
  expect(store.state.queryState.orderBy).toEqual([{ field: 'CreatedAt', direction: 'desc' }]);
  await ctrl.apply({ appliedGroups: ['Name'] as any });
  expect(store.state.queryState.orderBy).toEqual([{ field: 'CreatedAt', direction: 'desc' }]);
  await ctrl.apply({ kind: 'search', appliedGroups: [] as any });
  expect(store.state.queryState.orderBy).toEqual([{ field: 'CreatedAt', direction: 'desc' }]);
});

test('listController: explicit orderBy undefined override clears store even when grouped', async () => {
  const store = makeStore();
  store.state.queryState.orderBy = [{ field: 'CreatedAt', direction: 'asc' }];
  const ctrl = createListController(store);
  await ctrl.apply({ appliedGroups: ['Name'] as any, orderBy: undefined });
  expect(store.state.queryState.orderBy).toBeUndefined();
});
