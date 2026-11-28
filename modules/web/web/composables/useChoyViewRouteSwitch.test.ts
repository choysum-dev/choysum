// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, nextTick } from 'vue';
import * as VueRouter from 'vue-router';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import {
  applyChoyViewRouteGroups,
  normalizeChoyViewRoutePath,
  resetChoyViewRoutePaginationOffset,
  resolveChoyViewRouteDefaultGroups,
  resolveChoyViewRouteKind,
  resolveChoyViewRoutePath,
  routeShouldKeepAlive,
  useChoyViewRouteSwitch,
} from './useChoyViewRouteSwitch';

const createFeStubRouter = (VueRouter as any).createFeStubRouter;

test('resolveChoyViewRouteKind reads meta.viewKind with list fallback', () => {
  expect(resolveChoyViewRouteKind({ viewKind: 'kanban' })).toBe('kanban');
  expect(resolveChoyViewRouteKind({ viewKind: '  chart  ' })).toBe('chart');
  expect(resolveChoyViewRouteKind({ viewKind: '   ' })).toBe('list');
  expect(resolveChoyViewRouteKind({})).toBe('list');
  expect(resolveChoyViewRouteKind(null)).toBe('list');
  expect(resolveChoyViewRouteKind(undefined, 'board')).toBe('board');
});

test('resolveChoyViewRouteDefaultGroups copies string groups only', () => {
  expect(resolveChoyViewRouteDefaultGroups({ defaultGroups: ['State', 'IsActive'] })).toEqual([
    'State',
    'IsActive',
  ]);
  expect(resolveChoyViewRouteDefaultGroups({ defaultGroups: ['A', 1 as unknown as string] })).toEqual([
    'A',
  ]);
  expect(resolveChoyViewRouteDefaultGroups({ defaultGroups: 'State' })).toEqual([]);
  expect(resolveChoyViewRouteDefaultGroups(null)).toEqual([]);
});

test('resolveChoyViewRoutePath looks up viewPaths by kind', () => {
  const meta = {
    viewPaths: {
      list: '/sandbox/records',
      kanban: '/sandbox/records/kanban',
      chart: '  /sandbox/records/chart  ',
    },
  };
  expect(resolveChoyViewRoutePath(meta, 'list')).toBe('/sandbox/records');
  expect(resolveChoyViewRoutePath(meta, 'chart')).toBe('/sandbox/records/chart');
  expect(resolveChoyViewRoutePath(meta, 'board')).toBe('');
  expect(resolveChoyViewRoutePath({}, 'list')).toBe('');
});

test('applyChoyViewRouteGroups overwrites appliedGroups and preserves siblings', () => {
  const store = {
    state: { queryState: { appliedGroups: ['Revoked'], keyword: 'keep-me' } },
  };
  applyChoyViewRouteGroups(store, ['TokenType', 'Revoked']);
  expect(store.state.queryState.appliedGroups).toEqual(['TokenType', 'Revoked']);
  expect(store.state.queryState.keyword).toBe('keep-me');

  applyChoyViewRouteGroups(store, []);
  expect(store.state.queryState.appliedGroups).toEqual([]);
});

test('applyChoyViewRouteGroups creates queryState when missing', () => {
  const store: { state?: { queryState?: { appliedGroups?: unknown } } } = {};
  applyChoyViewRouteGroups(store, ['State']);
  expect(store.state?.queryState?.appliedGroups).toEqual(['State']);
});

test('applyChoyViewRouteGroups ignores nullish or non-object store', () => {
  applyChoyViewRouteGroups(null, ['State']);
  applyChoyViewRouteGroups(undefined, ['State']);
  applyChoyViewRouteGroups('nope', ['State']);
});

test('resetChoyViewRoutePaginationOffset clears offset and keeps limit', () => {
  const store = {
    state: { queryState: { pagination: { limit: 40, offset: 80 }, keyword: 'x' } },
  };
  resetChoyViewRoutePaginationOffset(store);
  expect(store.state.queryState.pagination).toEqual({ limit: 40, offset: 0 });
  expect(store.state.queryState.keyword).toBe('x');

  resetChoyViewRoutePaginationOffset(store);
  expect(store.state.queryState.pagination.offset).toBe(0);

  resetChoyViewRoutePaginationOffset(null);
  resetChoyViewRoutePaginationOffset({ state: {} });
});

test('routeShouldKeepAlive honors keepAlive and pageKind collection', () => {
  expect(routeShouldKeepAlive({ keepAlive: true })).toBe(true);
  expect(routeShouldKeepAlive({ keepAlive: false, pageKind: 'collection' })).toBe(false);
  expect(routeShouldKeepAlive({ pageKind: 'collection' })).toBe(true);
  expect(routeShouldKeepAlive({ pageKind: '  collection  ' })).toBe(true);
  expect(routeShouldKeepAlive({ pageKind: 'record' })).toBe(false);
  // viewPaths alone must not imply KeepAlive.
  expect(routeShouldKeepAlive({ viewPaths: { list: '/a', kanban: '/b' } })).toBe(false);
  expect(routeShouldKeepAlive({})).toBe(false);
  expect(routeShouldKeepAlive(null)).toBe(false);
  expect(normalizeChoyViewRoutePath('/x/')).toBe('/x');
  expect(normalizeChoyViewRoutePath('/x///')).toBe('/x');
  expect(normalizeChoyViewRoutePath('/')).toBe('/');
});

test('useChoyViewRouteSwitch: equal defaultGroups arrays do not reset selection', async () => {
  const store = {
    state: { queryState: { appliedGroups: [] as string[] } },
  };
  const { router, route } = createFeStubRouter({
    route: {
      path: '/sandbox/records',
      fullPath: '/sandbox/records',
      meta: { viewKind: 'list', defaultGroups: [] as string[] },
    },
  });

  const Host = defineComponent({
    name: 'ViewSwitchHost',
    setup() {
      useChoyViewRouteSwitch({ store });
      return () => h('div', 'ok');
    },
  });
  const mounted = mountApp(Host as any, { plugins: [router] });
  await flushPromises();
  expect(store.state.queryState.appliedGroups).toEqual([]);

  store.state.queryState.appliedGroups = ['State'];
  // Fresh equal array (query-only navigation) must not wipe the selection.
  route.meta = { viewKind: 'list', defaultGroups: [] };
  await nextTick();
  await flushPromises();
  expect(store.state.queryState.appliedGroups).toEqual(['State']);

  route.meta = { viewKind: 'kanban', defaultGroups: ['Revoked'] };
  await nextTick();
  await flushPromises();
  expect(store.state.queryState.appliedGroups).toEqual(['Revoked']);

  mounted.unmount();
});

test('useChoyViewRouteSwitch: viewKind change clears pagination offset and keeps limit', async () => {
  const store = {
    state: {
      queryState: {
        appliedGroups: [] as string[],
        pagination: { limit: 40, offset: 80 },
      },
    },
  };
  const { router: listRouter } = createFeStubRouter({
    route: {
      path: '/sandbox/records',
      fullPath: '/sandbox/records',
      meta: { viewKind: 'list', defaultGroups: [] as string[] },
    },
  });
  const ListHost = defineComponent({
    name: 'ViewSwitchListHost',
    setup() {
      useChoyViewRouteSwitch({ store });
      return () => h('div', 'list');
    },
  });
  const listMounted = mountApp(ListHost as any, { plugins: [listRouter] });
  await flushPromises();
  expect(store.state.queryState.pagination).toEqual({ limit: 40, offset: 80 });

  const { router: kanbanRouter } = createFeStubRouter({
    route: {
      path: '/sandbox/records/kanban',
      fullPath: '/sandbox/records/kanban',
      meta: { viewKind: 'kanban', defaultGroups: ['Revoked'] },
    },
  });
  const KanbanHost = defineComponent({
    name: 'ViewSwitchKanbanHost',
    setup() {
      useChoyViewRouteSwitch({ store });
      return () => h('div', 'kanban');
    },
  });
  const kanbanMounted = mountApp(KanbanHost as any, { plugins: [kanbanRouter] });
  await flushPromises();
  expect(store.state.queryState.appliedGroups).toEqual(['Revoked']);
  expect(store.state.queryState.pagination).toEqual({ limit: 40, offset: 0 });

  listMounted.unmount();
  kanbanMounted.unmount();
});

test('useChoyViewRouteSwitch: same viewKind does not clear offset', async () => {
  const store = {
    state: {
      queryState: {
        appliedGroups: ['State'] as string[],
        pagination: { limit: 40, offset: 80 },
      },
    },
  };
  const { router, route } = createFeStubRouter({
    route: {
      path: '/sandbox/records',
      fullPath: '/sandbox/records',
      meta: { viewKind: 'list', defaultGroups: ['State'] },
    },
  });
  const Host = defineComponent({
    name: 'ViewSwitchSameKindHost',
    setup() {
      useChoyViewRouteSwitch({ store });
      return () => h('div', 'ok');
    },
  });
  const mounted = mountApp(Host as any, { plugins: [router] });
  await flushPromises();
  expect(store.state.queryState.pagination.offset).toBe(80);

  // Same kind, different groups content → applyGroups runs but kind latch unchanged → keep offset.
  route.meta = { viewKind: 'list', defaultGroups: ['Other'] };
  await nextTick();
  await flushPromises();
  expect(store.state.queryState.appliedGroups).toEqual(['Other']);
  expect(store.state.queryState.pagination).toEqual({ limit: 40, offset: 80 });

  mounted.unmount();
});
