// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createKanbanController, combineLaneAggregateConditions } from './kanbanController';

type CallRecorder = { calls: unknown[][] };

function fnRecorder<T = undefined, A extends unknown[] = unknown[]>(
  impl?: (...args: A) => T | Promise<T>
): CallRecorder & ((...args: A) => T | Promise<T>) {
  const rec: CallRecorder & ((...args: A) => T | Promise<T>) = Object.assign(
    (...args: A) => {
      rec.calls.push(args);
      return impl ? impl(...args) : (undefined as T);
    },
    { calls: [] as unknown[][] }
  );
  return rec;
}

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
  const Search = fnRecorder(async () => []);
  const ReadGroup = fnRecorder(async () => ({
    groups: [
      {
        key: 'todo',
        label: 'Todo',
        __condition: { Stage: 'todo' },
        count: 1,
      },
      {
        key: 'done',
        label: 'Done',
        __condition: { Stage: 'done' },
        count: 2,
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
    Count: fnRecorder(async () => 0),
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

  // Batch aggregate path must have attempted at least one store read.
  expect(ReadGroup.calls.length + Search.calls.length).toBeGreaterThan(0);
});
