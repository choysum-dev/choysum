// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createKanbanController, combineLaneAggregateConditions } from './kanbanController';
import { asyncFnRecorder } from '@/web/web/__tests__/mountApp';

test('combineLaneAggregateConditions: uses the single-lane present condition', () => {
  expect(combineLaneAggregateConditions([{ Stage: 'todo' }])).toEqual({ Stage: 'todo' });
  expect(combineLaneAggregateConditions([])).toBeUndefined();
});

test('combineLaneAggregateConditions: drops combined Or when any selected lane is unconditioned', () => {
  expect(combineLaneAggregateConditions([{ Stage: 'todo' }, undefined])).toBeUndefined();
  expect(combineLaneAggregateConditions([{ Stage: 'todo' }, {}])).toBeUndefined();
});

test('combineLaneAggregateConditions: Or-combines present conditions when every lane is conditioned', () => {
  expect(combineLaneAggregateConditions([{ Stage: 'todo' }, { Stage: 'done' }])).toEqual({
    Or: [{ Stage: 'todo' }, { Stage: 'done' }],
  });
});

test('refreshLaneAggregates combines lane conditions for the batch query', async () => {
  const Search = asyncFnRecorder(async () => []);
  const ReadGroup = asyncFnRecorder(async () => ({
    groups: [
      {
        key: 'todo',
        label: 'Todo',
        __condition: { Stage: 'todo' },
        count: 10,
        Amount__sum: 100,
      },
      {
        key: 'done',
        label: 'Done',
        __condition: { Stage: 'done' },
        count: 20,
        Amount__sum: 200,
      },
    ],
  }));

  const store = {
    storeId: 'demo.Task',
    fullModelName: 'demo.Task',
    fieldsMetadata: {
      Stage: { type: 'varchar' },
      Amount: { type: 'float' },
    },
    state: {
      result: null,
      queryState: {
        appliedGroups: ['Stage'],
        appliedFilters: [],
        keyword: '',
        keywordFields: [],
        forcedCondition: null,
        pagination: { limit: 80, offset: 0 },
        orderBy: [],
      },
      planCache: new Map(),
    },
    getContext: () => ({}),
    Search,
    ReadGroup,
    Count: asyncFnRecorder(async () => 0),
  } as any;

  const controller = createKanbanController(store);
  // Seed a group snapshot so lanes() exposes conditioned lanes.
  controller.vm.result = {
    kind: 'group',
    rows: [
      {
        kind: 'group',
        key: 'todo',
        depth: 0,
        label: 'Todo',
        count: 1,
        __condition: { Stage: 'todo' },
        raw: { key: 'todo', __condition: { Stage: 'todo' }, labels: {} },
      },
      {
        kind: 'group',
        key: 'done',
        depth: 0,
        label: 'Done',
        count: 2,
        __condition: { Stage: 'done' },
        raw: { key: 'done', __condition: { Stage: 'done' }, labels: {} },
      },
    ],
    total: 2,
    ts: Date.now(),
  } as any;

  await controller.refreshLaneAggregates(['todo', 'done']);

  expect(ReadGroup.calls.length).toBe(1);
  expect(Search.calls.length).toBe(0);
  const [, condition] = ReadGroup.calls[0] as [unknown, unknown, unknown];
  expect(condition).toEqual({
    Or: [{ Stage: 'todo' }, { Stage: 'done' }],
  });

  const rows = (controller.vm.result as any).rows as Array<{ key: string; count?: number; Amount__sum?: number }>;
  const todo = rows.find(r => r.key === 'todo');
  const done = rows.find(r => r.key === 'done');
  expect(todo?.count).toBe(10);
  expect(todo?.Amount__sum).toBe(100);
  expect(done?.count).toBe(20);
  expect(done?.Amount__sum).toBe(200);
});
