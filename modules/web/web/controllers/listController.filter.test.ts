// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createListController } from './listController';

const { buildPlan, execute, awaitFieldSelection, exportFieldSelection } = vi.hoisted(() => ({
  buildPlan: vi.fn(() => ({ kind: 'read' })),
  execute: vi.fn(async () => ({ kind: 'collection', rows: [], total: 0 })),
  awaitFieldSelection: vi.fn(async () => undefined),
  exportFieldSelection: vi.fn(() => ({})),
}));

vi.mock('@/web/web/query/planner', async () => {
  const actual = await vi.importActual<any>('@/web/web/query/planner');
  return { ...actual, buildPlan };
});

vi.mock('@/web/web/query/executor', () => ({ execute }));

vi.mock('@/web/web/query/utils/registry/fieldReady', () => ({
  awaitFieldSelection,
}));

vi.mock('@/web/web/query/utils/registry/field', () => ({
  exportFieldSelection,
}));

function makeStore(queryState: Record<string, unknown> = {}) {
  return {
    storeId: 'demo.Task',
    fullModelName: 'demo.Task',
    fieldsMetadata: {},
    state: {
      queryState: {
        appliedFilters: [],
        pagination: { limit: 40, offset: 0 },
        ...queryState,
      },
      planCache: new Map(),
    },
  } as any;
}

describe('listController filter normalize/combine', () => {
  beforeEach(() => {
    buildPlan.mockClear();
    execute.mockClear();
    execute.mockResolvedValue({ kind: 'collection', rows: [], total: 0 });
  });

  it('merges forcedCondition from state through combineFilters', async () => {
    const store = makeStore({
      forcedCondition: { Status: 'open' },
    });
    const ctrl = createListController(store);
    await ctrl.apply({ forcedCondition: { AssigneeId: 'u1' } } as any);
    expect(buildPlan).toHaveBeenCalled();
    const ctx = buildPlan.mock.calls[0]?.[0] as any;
    expect(ctx.filters).toEqual({
      And: [{ Status: 'open' }, { AssigneeId: 'u1' }],
    });
  });

  it('treats empty object forced filters as absent via normalizeFilter', async () => {
    const store = makeStore({
      forcedCondition: {},
    });
    const ctrl = createListController(store);
    await ctrl.apply({ forcedCondition: { A: 1 } } as any);
    const ctx = buildPlan.mock.calls[0]?.[0] as any;
    expect(ctx.filters).toEqual({ A: 1 });
  });

  it('preserves falsy forcedCondition values false and 0', async () => {
    const storeFalse = makeStore();
    const ctrlFalse = createListController(storeFalse);
    await ctrlFalse.apply({ forcedCondition: false as any } as any);
    expect(buildPlan.mock.calls[0]?.[0]?.filters).toBe(false);

    buildPlan.mockClear();
    const storeZero = makeStore();
    const ctrlZero = createListController(storeZero);
    await ctrlZero.apply({ forcedCondition: 0 as any } as any);
    expect(buildPlan.mock.calls[0]?.[0]?.filters).toBe(0);
  });
});
