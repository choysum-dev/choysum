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
