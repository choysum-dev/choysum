// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { defineComponent, h } from 'vue';
import { useChoyKanbanStoreEngine } from './useChoyKanbanStoreEngine';

function store() {
  return {
    storeId: 'demo.Task',
    fieldsMetadata: {},
    state: {
      queryState: { pagination: { limit: 20, offset: 0 }, appliedGroups: ['Stage'] },
      planCache: new Map(),
    },
    Search: async () => [],
    Count: async () => 0,
    ReadGroup: async () => ({ groups: [] }),
  } as any;
}

function groupResult(keys: Array<{ key: string; label: string }>) {
  return {
    kind: 'group',
    rows: keys.map((lane) => ({
      kind: 'group',
      key: lane.key,
      depth: 0,
      label: lane.label,
      count: 1,
      __condition: { Stage: lane.key },
    })),
    visibleNodes: [],
    expandedKeys: new Set<string>(),
  };
}

test('store engine orders lanes and keeps the board idle while a lane preload fails', async () => {
  const Harness = defineComponent({
    name: 'KanbanEngineHarness',
    setup() {
      return useChoyKanbanStoreEngine({
        store: store(),
        autoBootstrap: false,
        laneOrder: ['done', 'todo'],
        flatLaneLabel: 'Everything',
      });
    },
    render() {
      return h('div');
    },
  });

  const mounted = mountApp(Harness as any);
  await flushPromises();
  const engine = mounted.root;

  engine.controller.vm.result = groupResult([
    { key: 'todo', label: 'Todo' },
    { key: 'done', label: 'Done' },
  ]);
  const originalPreload = engine.controller.preloadLane.bind(engine.controller);
  engine.controller.preloadLane = async () => {
    throw new Error('lane preload failed');
  };
  const logged: string[] = [];
  const originalError = console.error;
  console.error = (...args: unknown[]) => {
    logged.push(args.map((item) => String(item)).join(' '));
  };
  try {
    await engine.syncLanesFromController();
  } finally {
    console.error = originalError;
    engine.controller.preloadLane = originalPreload;
  }
  expect(logged.some((line) => line.includes('lane preload failed'))).toBe(true);
  await engine.syncLanesFromController();
  expect(engine.lanes.map((lane: { key: string }) => lane.key)).toEqual(['done', 'todo']);
  expect(engine.boardBusy).toBe(false);

  const TokenHarness = defineComponent({
    name: 'KanbanEngineTokenHarness',
    setup() {
      return useChoyKanbanStoreEngine({
        store: store(),
        autoBootstrap: false,
        laneOrder: ['Done', 'Todo'],
        laneOrderToken: (lane) => String(lane.label || ''),
      });
    },
    render() {
      return h('div');
    },
  });
  const tokenMounted = mountApp(TokenHarness as any);
  await flushPromises();
  tokenMounted.root.controller.vm.result = groupResult([
    { key: 'todo', label: 'Todo' },
    { key: 'done', label: 'Done' },
  ]);
  await tokenMounted.root.syncLanesFromController();
  expect(tokenMounted.root.lanes.map((lane: { label: string }) => lane.label)).toEqual(['Done', 'Todo']);
  tokenMounted.unmount();

  engine.controller.vm.result = { kind: 'search', rows: [{ Id: '1', Title: 'Flat' }] };
  await engine.syncLanesFromController();
  expect(engine.lanes[0].label).toBe('Everything');
  expect(engine.lanes[0].cards.length).toBe(1);

  mounted.unmount();
});

test('load-more patches the target lane without rematerializing siblings', async () => {
  const Harness = defineComponent({
    name: 'KanbanEngineLoadMoreHarness',
    setup() {
      return useChoyKanbanStoreEngine({
        store: store(),
        autoBootstrap: false,
        mapRowToCard: (row: any, _index, laneKey) => ({
          id: String(row.payload?.Id ?? row.key ?? ''),
          title: String(row.payload?.Title ?? row.key ?? ''),
          laneKey,
        }),
      });
    },
    render() {
      return h('div');
    },
  });

  const mounted = mountApp(Harness as any);
  await flushPromises();
  const engine = mounted.root;

  engine.controller.vm.result = {
    ...groupResult([
      { key: 'active', label: 'Active' },
      { key: 'revoked', label: 'Revoked' },
    ]),
    rows: [
      {
        kind: 'group',
        key: 'active',
        depth: 0,
        label: 'Active',
        count: 1,
        __condition: { Stage: 'active' },
      },
      {
        kind: 'group',
        key: 'revoked',
        depth: 0,
        label: 'Revoked',
        count: 3,
        __condition: { Stage: 'revoked' },
      },
    ],
    groupRecords: new Map([
      [
        'active',
        [{ kind: 'record', key: 'a1', payload: { Id: 'a1', Title: 'Active 1' } }],
      ],
      [
        'revoked',
        [{ kind: 'record', key: 'r1', payload: { Id: 'r1', Title: 'Revoked 1' } }],
      ],
    ]),
  };
  engine.controller.laneRecords.value = {
    active: [{ kind: 'record', key: 'a1', payload: { Id: 'a1', Title: 'Active 1' } }],
    revoked: [{ kind: 'record', key: 'r1', payload: { Id: 'r1', Title: 'Revoked 1' } }],
  };
  engine.controller.preloadLane = async () => undefined;
  await engine.syncLanesFromController();

  const siblingBefore = engine.lanes.find((lane: { key: string }) => lane.key === 'active');
  const targetBefore = engine.lanes.find((lane: { key: string }) => lane.key === 'revoked');
  expect(siblingBefore?.cards.length).toBe(1);
  expect(targetBefore?.cards.length).toBe(1);

  engine.controller.loadMoreLane = async (key: string) => {
    const next = [
      ...(engine.controller.laneRecords.value[key] || []),
      { kind: 'record', key: 'r2', payload: { Id: 'r2', Title: 'Revoked 2' } },
    ];
    engine.controller.laneRecords.value = {
      ...engine.controller.laneRecords.value,
      [key]: next,
    };
  };

  await engine.onLaneLoadMore({ laneKey: 'revoked' });
  await flushPromises();

  const siblingAfter = engine.lanes.find((lane: { key: string }) => lane.key === 'active');
  const targetAfter = engine.lanes.find((lane: { key: string }) => lane.key === 'revoked');
  expect(siblingAfter).toBe(siblingBefore);
  expect(targetAfter).not.toBe(targetBefore);
  expect(targetAfter?.cards.map((card: { id: string }) => card.id)).toEqual(['r1', 'r2']);
  expect(engine.loadMorePending).toBe(false);
  expect(engine.loadMoreLaneKey).toBeNull();
  expect(engine.boardBusy).toBe(false);

  mounted.unmount();
});
