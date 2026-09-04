// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createKanbanController } from './kanbanController';

const { exportMetrics, buildPlan, execute } = vi.hoisted(() => ({
  exportMetrics: vi.fn(() => [{ field: 'Amount', agg: 'sum', alias: 'Amount__sum' }]),
  buildPlan: vi.fn(() => ({ kind: 'readGroup' })),
  execute: vi.fn(async () => ({ kind: 'group', rows: [] })),
}));

vi.mock('@/web/web/query/utils/registry/metric', () => ({
  exportMetrics,
}));

vi.mock('@/web/web/query/planner', async () => {
  const actual = await vi.importActual<any>('@/web/web/query/planner');
  return {
    ...actual,
    buildPlan,
  };
});

vi.mock('@/web/web/query/executor', () => ({
  execute,
}));

vi.mock('@/web/web/query/utils/registry/field', () => ({
  exportFieldSelection: vi.fn(() => ({})),
  pathsToFieldSelection: vi.fn(() => ({})),
  ensureRootId: vi.fn((x: any) => x),
}));

vi.mock('@/web/web/query/utils/registry/fieldReady', () => ({
  awaitFieldSelection: vi.fn(async () => undefined),
}));

function makeStore() {
  return {
    storeId: 'demo.Task',
    fullModelName: 'demo.Task',
    fieldsMetadata: {},
    state: {
      queryState: {
        appliedGroups: ['Stage'],
        appliedFilters: [],
        pagination: { limit: 40, offset: 0 },
      },
      planCache: new Map(),
    },
  } as any;
}

describe('kanbanController refreshLaneAggregates conditions', () => {
  beforeEach(() => {
    exportMetrics.mockClear();
    buildPlan.mockClear();
    execute.mockClear();
    execute.mockResolvedValue({ kind: 'group', rows: [] });
  });

  it('uses single-lane present condition', async () => {
    const store = makeStore();
    const ctrl = createKanbanController(store);
    (ctrl.vm as any).result = {
      kind: 'group',
      rows: [
        {
          kind: 'group',
          depth: 0,
          key: 'todo',
          label: 'Todo',
          __condition: { Stage: 'todo' },
          raw: { labels: {} },
        },
      ],
    };

    await ctrl.refreshLaneAggregates(['todo']);
    expect(buildPlan).toHaveBeenCalled();
    const ctx = buildPlan.mock.calls[0]?.[0] as any;
    expect(ctx.filters).toEqual({ Stage: 'todo' });
  });

  it('drops combined Or when any selected lane is unconditioned', async () => {
    const store = makeStore();
    const ctrl = createKanbanController(store);
    (ctrl.vm as any).result = {
      kind: 'group',
      rows: [
        {
          kind: 'group',
          depth: 0,
          key: 'todo',
          label: 'Todo',
          __condition: { Stage: 'todo' },
          raw: { labels: {} },
        },
        {
          kind: 'group',
          depth: 0,
          key: 'all',
          label: 'All',
          __condition: undefined,
          raw: { labels: {} },
        },
      ],
    };

    await ctrl.refreshLaneAggregates(['todo', 'all']);
    expect(buildPlan).toHaveBeenCalled();
    const ctx = buildPlan.mock.calls[0]?.[0] as any;
    // Unconditioned lane means no parent constraint on the batch refresh.
    expect(ctx.filters).toBeUndefined();
  });

  it('Or-combines present conditions when every lane is conditioned', async () => {
    const store = makeStore();
    const ctrl = createKanbanController(store);
    (ctrl.vm as any).result = {
      kind: 'group',
      rows: [
        {
          kind: 'group',
          depth: 0,
          key: 'todo',
          label: 'Todo',
          __condition: { Stage: 'todo' },
          raw: { labels: {} },
        },
        {
          kind: 'group',
          depth: 0,
          key: 'done',
          label: 'Done',
          __condition: { Stage: 'done' },
          raw: { labels: {} },
        },
      ],
    };

    await ctrl.refreshLaneAggregates(['todo', 'done']);
    expect(buildPlan).toHaveBeenCalled();
    const ctx = buildPlan.mock.calls[0]?.[0] as any;
    expect(ctx.filters).toEqual({
      Or: [{ Stage: 'todo' }, { Stage: 'done' }],
    });
  });
});
