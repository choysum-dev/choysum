// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createListController } from './listController';

function makeStore() {
  return {
    fullModelName: 'base.Company',
    storeId: 'list-applied-once',
    fieldsMetadata: { Name: { type: 'varchar' } },
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
    Count: async () => 0,
  } as any;
}

test('listController: hasAppliedOnce defaults to false', () => {
  const ctrl = createListController(makeStore());
  expect(ctrl.vm.hasAppliedOnce).toBe(false);
});

test('listController: apply finally sets hasAppliedOnce even when the query path fails', async () => {
  const ctrl = createListController(makeStore());
  expect(ctrl.vm.hasAppliedOnce).toBe(false);
  // Incomplete FE-unit store may fail inside execute; finally must still settle the gate.
  await ctrl.apply().catch(() => undefined);
  expect(ctrl.vm.hasAppliedOnce).toBe(true);
  expect(ctrl.vm.loading).toBe(false);
});
