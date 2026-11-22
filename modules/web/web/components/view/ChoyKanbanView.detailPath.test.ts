// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import * as VueRouter from 'vue-router';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyKanbanView from './ChoyKanbanView.vue';
import { useRecordSetStore } from '@/web/web/stores/recordSetStore';
import type { ChoyKanbanCard } from './kanbanViewHelpers';

const createFeStubRouter = (VueRouter as any).createFeStubRouter as (overrides?: object) => {
  router: {
    push: (to: unknown) => Promise<unknown>;
    currentRoute: { value: { path: string } };
  };
};

function makeStore(rows: unknown[]) {
  return {
    modelName: 'sandbox.Record',
    storeId: 'kanban-detail-' + Math.random().toString(36).slice(2),
    state: {
      queryState: {
        appliedFilters: [],
        pagination: { limit: 20, offset: 0 },
        orderBy: [],
      },
      result: { total: rows.length, rows },
    },
  } as any;
}

test('detailPath card click seeds recordSet from unwrapped rows', async () => {
  const pinia = createPinia();
  setActivePinia(pinia);
  const { router } = createFeStubRouter({
    route: { path: '/sandbox/records/kanban', fullPath: '/sandbox/records/kanban' },
  });
  const pushes: unknown[] = [];
  const origPush = router.push.bind(router);
  router.push = (to: unknown) => {
    pushes.push(to);
    return origPush(to);
  };

  const mounted = mountApp(ChoyKanbanView as any, {
    props: {
      store: makeStore([
        { kind: 'record', payload: { Id: 'a', Name: 'A' } },
        { kind: 'record', payload: { Id: 'b', Name: 'B' } },
      ]),
      autoBootstrap: false,
      detailPath: '/sandbox/records/:id',
    },
    plugins: [pinia, router as any],
  });
  await flushPromises();

  const state = mounted.setupState() as {
    onCardClick: (card: ChoyKanbanCard) => void;
  };
  state.onCardClick({
    id: 'b',
    title: 'B',
    laneKey: 'todo',
    payload: { Id: 'b', Name: 'B' },
  });
  await flushPromises();

  expect(pushes).toEqual(['/sandbox/records/b']);
  const entry = useRecordSetStore().getActiveForModel('sandbox.Record');
  expect(entry?.ids).toEqual(['a', 'b']);
  expect(entry?.index).toBe(1);
  mounted.unmount();
});

test('detailPath card click clears session and pushes when id not in window', async () => {
  const pinia = createPinia();
  setActivePinia(pinia);
  const rs = useRecordSetStore();
  rs.seed({
    key: 'stale',
    model: 'sandbox.Record',
    condition: {},
    total: 1,
    index: 0,
    ids: ['old'],
    windowStart: 0,
  });

  const { router } = createFeStubRouter({
    route: { path: '/sandbox/records/kanban', fullPath: '/sandbox/records/kanban' },
  });
  const pushes: unknown[] = [];
  router.push = (to: unknown) => {
    pushes.push(to);
    return Promise.resolve(to);
  };

  const mounted = mountApp(ChoyKanbanView as any, {
    props: {
      // Group rows only — snapshot ids empty.
      store: makeStore([{ kind: 'group', key: 'g1' }]),
      autoBootstrap: false,
      detailPath: '/sandbox/records/:id',
    },
    plugins: [pinia, router as any],
  });
  await flushPromises();

  const state = mounted.setupState() as {
    onCardClick: (card: ChoyKanbanCard) => void;
  };
  state.onCardClick({
    id: 'x',
    title: 'X',
    laneKey: 'todo',
    payload: { Id: 'x' },
  });
  await flushPromises();

  expect(pushes).toEqual(['/sandbox/records/x']);
  expect(useRecordSetStore().active).toBeNull();
  mounted.unmount();
});

test('detailPath respects canOpenDetail gate and emits card-click when blocked', async () => {
  const pinia = createPinia();
  setActivePinia(pinia);
  const { router } = createFeStubRouter({
    route: { path: '/sandbox/records/kanban', fullPath: '/sandbox/records/kanban' },
  });
  const pushes: unknown[] = [];
  router.push = (to: unknown) => {
    pushes.push(to);
    return Promise.resolve(to);
  };
  const clicks: ChoyKanbanCard[] = [];

  const mounted = mountApp(ChoyKanbanView as any, {
    props: {
      store: makeStore([{ Id: 'a' }]),
      autoBootstrap: false,
      detailPath: '/sandbox/records/:id',
      canOpenDetail: false,
    },
    on: { onCardClick: (c: ChoyKanbanCard) => clicks.push(c) },
    plugins: [pinia, router as any],
  });
  await flushPromises();

  const card = { id: 'a', title: 'A', laneKey: 'todo', payload: { Id: 'a' } };
  (mounted.setupState() as any).onCardClick(card);
  await flushPromises();

  expect(pushes).toEqual([]);
  expect(clicks).toEqual([card]);
  mounted.unmount();
});

test('detailPath without router emits card-click; blank id / null path emit', async () => {
  const clicks: ChoyKanbanCard[] = [];
  // No vue-router plugin → router stays null (chrome-style mount).
  const mounted = mountApp(ChoyKanbanView as any, {
    props: {
      store: makeStore([{ Id: 'a' }]),
      autoBootstrap: false,
      detailPath: '/sandbox/records/:id',
    },
    on: { onCardClick: (c: ChoyKanbanCard) => clicks.push(c) },
  });
  await flushPromises();
  const card = { id: 'a', title: 'A', laneKey: 'todo', payload: { Id: 'a' } };
  (mounted.setupState() as any).onCardClick(card);
  expect(clicks).toEqual([card]);
  mounted.unmount();

  const pinia = createPinia();
  setActivePinia(pinia);
  const { router } = createFeStubRouter({
    route: { path: '/sandbox/records/kanban', fullPath: '/sandbox/records/kanban' },
  });
  const pushes: unknown[] = [];
  router.push = (to: unknown) => {
    pushes.push(to);
    return Promise.resolve(to);
  };
  const clicks2: ChoyKanbanCard[] = [];
  const withRouter = mountApp(ChoyKanbanView as any, {
    props: {
      store: makeStore([{ Id: 'a' }]),
      autoBootstrap: false,
      detailPath: () => null,
    },
    on: { onCardClick: (c: ChoyKanbanCard) => clicks2.push(c) },
    plugins: [pinia, router as any],
  });
  await flushPromises();
  const blank = { id: '', title: 'X', laneKey: 'todo', payload: {} };
  (withRouter.setupState() as any).onCardClick(blank);
  (withRouter.setupState() as any).onCardClick({
    id: 'a',
    title: 'A',
    laneKey: 'todo',
    payload: { Id: 'a' },
  });
  await flushPromises();
  expect(pushes).toEqual([]);
  expect(clicks2.length).toBe(2);
  withRouter.unmount();
});

test('detailPath clear catches missing Pinia on mount', async () => {
  const { router } = createFeStubRouter({
    route: { path: '/sandbox/records/kanban', fullPath: '/sandbox/records/kanban' },
  });
  // detailPath set but no pinia → clearRecordSetSessionIfDetailHost catch path.
  const mounted = mountApp(ChoyKanbanView as any, {
    props: {
      store: makeStore([]),
      autoBootstrap: false,
      detailPath: '/sandbox/records/:id',
    },
    plugins: [router as any],
  });
  await flushPromises();
  expect(mounted.q('[data-anchor="choy.kanban-view"]') || mounted.el).toBeTruthy();
  mounted.unmount();
});
