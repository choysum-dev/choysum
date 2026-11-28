// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { buildSearchOrGroupContext, buildUnifiedQuery } from './context';

function makeStore(queryState: Record<string, unknown> = {}) {
  return {
    storeId: 'demo.Task',
    metadata: { fields: {} },
    state: {
      queryState: {
        appliedFilters: [],
        pagination: { limit: 40, offset: 0 },
        ...queryState,
      },
    },
  } as any;
}

test('buildSearchOrGroupContext: explicit undefined orderBy wins over store for groups', () => {
  const store = makeStore({
    orderBy: [{ field: 'CreatedAt', direction: 'asc' }],
    appliedGroups: ['Revoked'],
  });
  const ctx = buildSearchOrGroupContext(store, {
    shape: 'groups',
    queryState: {
      appliedGroups: ['Revoked'],
      orderBy: undefined,
    },
  });
  expect(ctx.queryState.orderBy).toBeUndefined();
});

test('buildSearchOrGroupContext: omitted orderBy still inherits store orderBy', () => {
  const store = makeStore({
    orderBy: [{ field: 'CreatedAt', direction: 'asc' }],
  });
  const ctx = buildSearchOrGroupContext(store, {
    shape: 'collection',
    queryState: {},
  });
  expect(ctx.queryState.orderBy).toEqual([{ field: 'CreatedAt', direction: 'asc' }]);
});

test('buildUnifiedQuery condition merge: merges ui, forced, and parent conditions via combinePresentConditions', () => {
  const store = makeStore({
    forcedCondition: { Status: 'open' },
  });
  const ctx = buildUnifiedQuery(store, { parentCondition: { AssigneeId: 'u1' } });
  expect(ctx.filters).toEqual({
    And: [{ Status: 'open' }, { AssigneeId: 'u1' }],
  });
});

test('buildUnifiedQuery condition merge: preserves present false/0 operands when merging', () => {
  const store = makeStore({
    forcedCondition: false as any,
  });
  const ctx = buildUnifiedQuery(store, { parentCondition: { A: 1 } });
  expect(ctx.filters).toEqual({ And: [false, { A: 1 }] });
});

