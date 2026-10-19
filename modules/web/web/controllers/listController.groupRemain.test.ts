// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createListController } from './listController';
import { createKanbanController } from './kanbanController';

function makeStore(search: () => Promise<unknown>, count: () => Promise<unknown>) {
  return {
    storeId: 'demo.Task',
    fieldsMetadata: {},
    state: {
      queryState: {
        pagination: { limit: 2, offset: 0 },
        orderBy: [],
        appliedGroups: ['Stage'],
      },
      planCache: new Map(),
    },
    Search: search,
    Count: count,
  } as any;
}

function seedGroup(list: ReturnType<typeof createListController>, count: number) {
  list.vm.result = {
    kind: 'group',
    rows: [
      {
        kind: 'group',
        key: 'todo',
        depth: 0,
        label: 'Todo',
        count,
        __condition: { Stage: 'todo' },
      },
    ],
    visibleNodes: [],
    expandedKeys: new Set<string>(),
  } as any;
}

test('fetchGroupRecords skipCount deletes remain; counted fetch publishes it', async () => {
  const list = createListController(
    makeStore(
      async () => [{ Id: '1' }, { Id: '2' }],
      async () => 5,
    ),
  );
  seedGroup(list, 5);

  const skipped = await list.fetchGroupRecords('todo', { limit: 2, offset: 0 }, { skipCount: true });
  expect(skipped?.length).toBe(2);
  expect(list.getGroupRecordRemain('todo')).toBeUndefined();
  expect(list.getGroupRecordRemain('missing')).toBeUndefined();

  const counted = await list.fetchGroupRecords('todo', { limit: 2, offset: 0 });
  expect(counted?.length).toBe(2);
  expect(list.getGroupRecordRemain('todo')).toBe(3);
});

test('loadMoreGroupRecords falls back to the group count when Count is not finite', async () => {
  const list = createListController(
    makeStore(
      async () => [{ Id: '3' }],
      async () => Number.NaN,
    ),
  );
  seedGroup(list, 9);
  await list.fetchGroupRecords('todo', { limit: 1, offset: 0 }, { skipCount: true });
  await list.loadMoreGroupRecords('todo');
  // skipCount page (1) plus the next page (1) against group count 9.
  expect(list.getGroupRecordRemain('todo')).toBe(7);
});

test('loadMoreGroupRecords uses the page length when neither Count nor group count is finite', async () => {
  const list = createListController(
    makeStore(
      async () => [{ Id: '3' }],
      async () => Number.NaN,
    ),
  );
  list.vm.result = {
    kind: 'group',
    rows: [
      {
        kind: 'group',
        key: 'todo',
        depth: 0,
        label: 'Todo',
        __condition: { Stage: 'todo' },
      },
    ],
    visibleNodes: [],
    expandedKeys: new Set<string>(),
  } as any;
  await list.loadMoreGroupRecords('todo');
  expect(list.getGroupRecordRemain('todo')).toBe(0);
});

test('kanban remain prefers published fetch remain and falls back after skipCount preload', async () => {
  const store = makeStore(
    async () => [{ Id: '1', Title: 'One' }],
    async () => 4,
  );
  const kanban = createKanbanController(store);
  seedGroup(kanban as any, 4);
  // createKanbanController wraps its own list; seed that vm.
  kanban.vm.result = {
    kind: 'group',
    rows: [
      {
        kind: 'group',
        key: 'todo',
        depth: 0,
        label: 'Todo',
        count: 4,
        __condition: { Stage: 'todo' },
      },
    ],
    visibleNodes: [],
    expandedKeys: new Set<string>(),
  } as any;

  await kanban.preloadLane('todo');
  await kanban.preloadLane('todo');
  const lane = kanban.lanes.value[0];
  expect(lane).toBeTruthy();
  expect(kanban.getLaneRemain(lane)).toBe(3);

  await kanban.loadMoreLane('todo');
  expect(kanban.getLaneRemain(kanban.lanes.value[0])).toBe(2);
});
