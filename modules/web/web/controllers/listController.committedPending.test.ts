// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createListController } from './listController';

function makeStore(overrides: Record<string, unknown> = {}) {
  return {
    fullModelName: 'base.Company',
    storeId: 'list-committed',
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
    Search: async () => [{ Id: '1', Name: 'Acme' }],
    Count: async () => 40,
    ...overrides,
  } as any;
}

test('listController: failed paginate keeps committed pagination and supports retry/cancel', async () => {
  let searchCalls = 0;
  const store = makeStore({
    Search: async (_domain: unknown, opts: { offset?: number }) => {
      searchCalls += 1;
      if ((opts?.offset ?? 0) > 0) throw new Error('page2 boom');
      return [{ Id: '1', Name: 'Acme' }];
    },
  });
  const ctrl = createListController(store);
  await ctrl.apply();
  expect(ctrl.vm.error).toBeNull();
  expect(store.state.queryState.pagination.offset).toBe(0);
  expect(searchCalls).toBe(1);

  let rejected: unknown;
  try {
    await ctrl.paginate({ limit: 20, offset: 20 });
  } catch (e) {
    rejected = e;
  }
  expect(rejected instanceof Error && /page2 boom/.test(rejected.message)).toBe(true);
  expect(ctrl.vm.error).not.toBeNull();
  // Strategy A: chrome stays on committed page until success.
  expect(store.state.queryState.pagination.offset).toBe(0);
  expect((ctrl.vm.result as any)?.rows?.length).toBe(1);

  ctrl.restoreCommitted();
  expect(ctrl.vm.error).toBeNull();
  expect(store.state.queryState.pagination.offset).toBe(0);

  // Re-fail then retry while still pending from a new attempt.
  try {
    await ctrl.paginate({ limit: 20, offset: 20 });
  } catch {
    /* expected */
  }
  expect(ctrl.vm.error).not.toBeNull();
  const callsBeforeRetry = searchCalls;
  try {
    await ctrl.retryPending();
  } catch {
    /* still boom */
  }
  expect(searchCalls).toBeGreaterThan(callsBeforeRetry);
  expect(store.state.queryState.pagination.offset).toBe(0);
});

test('listController: successful paginate commits new pagination', async () => {
  const store = makeStore({
    Search: async (_domain: unknown, opts: { offset?: number }) => {
      if ((opts?.offset ?? 0) > 0) return [{ Id: '2', Name: 'Beta' }];
      return [{ Id: '1', Name: 'Acme' }];
    },
  });
  const ctrl = createListController(store);
  await ctrl.apply();
  await ctrl.paginate({ limit: 20, offset: 20 });
  expect(ctrl.vm.error).toBeNull();
  expect(store.state.queryState.pagination.offset).toBe(20);
  expect((ctrl.vm.result as any)?.rows?.[0]?.payload?.Name || (ctrl.vm.result as any)?.rows?.[0]?.Name).toBeTruthy();
});
