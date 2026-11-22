// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import * as VueRouter from 'vue-router';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ChoyListView from './ChoyListView.vue';
import ChoyPagination from './ChoyPagination.vue';
import ChoyTableHost from '@/web/web/components/internal/ChoyTableHost.vue';
import ChoyTableColumn from '@/web/web/components/table/ChoyTableColumn.vue';
import ListInlineEditScope from '@/web/web/components/view/ListInlineEditScope.vue';
import { useRecordSetStore } from '@/web/web/stores/recordSetStore';

const createFeStubRouter = (VueRouter as any).createFeStubRouter as (overrides?: object) => {
  router: {
    push: (to: unknown) => Promise<unknown>;
    currentRoute: { value: { path: string } };
  };
};

function makeStore() {
  return {
    modelName: 'base.Country',
    fullModelName: 'base.Country',
    storeId: 'list-detail-' + Math.random().toString(36).slice(2),
    metadata: { fields: { Name: { type: 'varchar' } } },
    state: {
      queryState: {
        keyword: '',
        appliedFilters: [],
        appliedGroups: [],
        keywordFields: [],
        pagination: { limit: 20, offset: 20 },
        orderBy: [{ field: 'Name', direction: 'asc' }],
      },
      result: {
        total: 40,
        rows: [
          { kind: 'record', key: 'a', payload: { Id: 'a', Name: 'A' } },
          { kind: 'record', key: 'b', payload: { Id: 'b', Name: 'B' } },
        ],
      },
      selection: [],
      planCache: new Map(),
    },
    setContext: () => {},
    getContext: () => ({}),
    withContext: async (_c: any, fn: any) => fn(),
    Search: async () => [],
  } as any;
}

test('detailPath row click seeds recordSet and pushes detail path', async () => {
  const pinia = createPinia();
  setActivePinia(pinia);
  const { router } = createFeStubRouter({
    route: { path: '/base/countries', fullPath: '/base/countries' },
  });
  const pushes: unknown[] = [];
  const origPush = router.push.bind(router);
  router.push = (to: unknown) => {
    pushes.push(to);
    return origPush(to);
  };

  stubSfc(ChoyTableHost, {
    name: 'ChoyTableHost',
    emits: ['row-click'],
    setup: () => () => h('div', { 'data-stub': 'ChoyTableHost' }),
  });
  stubSfc(ChoyTableColumn, { name: 'ChoyTableColumn', setup: () => () => null });
  stubSfc(ChoyPagination, {
    name: 'ChoyPagination',
    setup: () => () => h('div', { 'data-stub': 'Pagination' }),
  });
  stubSfc(ListInlineEditScope, {
    name: 'ListInlineEditScope',
    setup: (_p: any, { slots }: any) => () => h('div', slots.default?.()),
  });

  const mounted = mountApp(ChoyListView as any, {
    props: {
      store: makeStore(),
      detailPath: '/base/countries/:id',
      showHeader: false,
      showPaginate: false,
    },
    plugins: [pinia, router as any],
  });
  await flushPromises();

  const ss = mounted.setupState() as any;
  await ss.onRowClick({
    rowData: { kind: 'record', key: 'b', payload: { Id: 'b', Name: 'B' } },
    rowIndex: 1,
    rowKey: 'b',
  });
  await flushPromises();

  expect(pushes).toEqual(['/base/countries/b']);
  const entry = useRecordSetStore().getActiveForModel('base.Country');
  expect(entry?.index).toBe(21);
  expect(entry?.total).toBe(40);
  expect(entry?.ids).toEqual(['a', 'b']);

  mounted.unmount();
  restoreSfc(ChoyTableHost);
  restoreSfc(ChoyTableColumn);
  restoreSfc(ChoyPagination);
  restoreSfc(ListInlineEditScope);
});

test('detailPath grouped visibleNodes seed via pageIds when result.rows are groups', async () => {
  const pinia = createPinia();
  setActivePinia(pinia);
  const { router } = createFeStubRouter({
    route: { path: '/base/countries', fullPath: '/base/countries' },
  });
  const pushes: unknown[] = [];
  const origPush = router.push.bind(router);
  router.push = (to: unknown) => {
    pushes.push(to);
    return origPush(to);
  };

  stubSfc(ChoyTableHost, {
    name: 'ChoyTableHost',
    emits: ['row-click'],
    setup: () => () => h('div', { 'data-stub': 'ChoyTableHost' }),
  });
  stubSfc(ChoyTableColumn, { name: 'ChoyTableColumn', setup: () => () => null });
  stubSfc(ChoyPagination, {
    name: 'ChoyPagination',
    setup: () => () => h('div', { 'data-stub': 'Pagination' }),
  });
  stubSfc(ListInlineEditScope, {
    name: 'ListInlineEditScope',
    setup: (_p: any, { slots }: any) => () => h('div', slots.default?.()),
  });

  const store = makeStore();
  store.state.result = {
    kind: 'group',
    total: 40,
    rows: [{ kind: 'group', key: 'g1' }],
  };
  store.state.queryState.appliedGroups = [{ field: 'Name' }];

  const mounted = mountApp(ChoyListView as any, {
    props: {
      store,
      detailPath: '/base/countries/:id',
      showHeader: false,
      showPaginate: false,
    },
    plugins: [pinia, router as any],
  });
  await flushPromises();

  const ss = mounted.setupState() as any;
  // Simulate expanded group visible nodes (controller would set these after expand).
  ss.controller.vm.result = {
    kind: 'group',
    total: 40,
    rows: [{ kind: 'group', key: 'g1' }],
  };
  ss.controller.vm.visibleNodes = [
    { kind: 'group', key: 'g1' },
    { kind: 'record', key: 'a', payload: { Id: 'a', Name: 'A' } },
    { kind: 'record', key: 'b', payload: { Id: 'b', Name: 'B' } },
  ];
  ss.controller.vm.expandedKeys = new Set(['g1']);

  await ss.onRowClick({
    rowData: { kind: 'record', key: 'b', payload: { Id: 'b', Name: 'B' } },
    rowIndex: 2,
    rowKey: 'b',
  });
  await flushPromises();

  expect(pushes).toEqual(['/base/countries/b']);
  const entry = useRecordSetStore().getActiveForModel('base.Country');
  expect(entry?.ids).toEqual(['a', 'b']);
  expect(entry?.index).toBe(1);
  expect(entry?.windowStart).toBe(0);

  mounted.unmount();
  restoreSfc(ChoyTableHost);
  restoreSfc(ChoyTableColumn);
  restoreSfc(ChoyPagination);
  restoreSfc(ListInlineEditScope);
});

test('detailPath clears same-model session on mount and skips seed for missing id window', async () => {
  const pinia = createPinia();
  setActivePinia(pinia);
  const rs = useRecordSetStore();
  rs.seed({
    key: 'stale',
    model: 'base.Country',
    condition: {},
    total: 1,
    index: 0,
    ids: ['z'],
    windowStart: 0,
  });
  // Other model must be preserved by clearRecordSetSessionIfDetailHost.
  // Seed same-model only — mount clears it.
  const { router } = createFeStubRouter({
    route: { path: '/base/countries', fullPath: '/base/countries' },
  });
  const pushes: unknown[] = [];
  router.push = (to: unknown) => {
    pushes.push(to);
    return Promise.resolve(to);
  };

  stubSfc(ChoyTableHost, {
    name: 'ChoyTableHost',
    emits: ['row-click'],
    setup: () => () => h('div', { 'data-stub': 'ChoyTableHost' }),
  });
  stubSfc(ChoyTableColumn, { name: 'ChoyTableColumn', setup: () => () => null });
  stubSfc(ChoyPagination, {
    name: 'ChoyPagination',
    setup: () => () => h('div', { 'data-stub': 'Pagination' }),
  });
  stubSfc(ListInlineEditScope, {
    name: 'ListInlineEditScope',
    setup: (_p: any, { slots }: any) => () => h('div', slots.default?.()),
  });

  const store = makeStore();
  store.state.result = { total: 0, rows: [{ kind: 'group', key: 'g' }] };

  const mounted = mountApp(ChoyListView as any, {
    props: {
      store,
      detailPath: '/base/countries/:id',
      showHeader: false,
      showPaginate: false,
    },
    plugins: [pinia, router as any],
  });
  await flushPromises();
  expect(useRecordSetStore().active).toBeNull();

  const ss = mounted.setupState() as any;
  await ss.onRowClick({
    rowData: { kind: 'record', key: 'x', payload: { Id: 'x', Name: 'X' } },
    rowIndex: 0,
    rowKey: 'x',
  });
  await flushPromises();
  expect(pushes).toEqual(['/base/countries/x']);
  expect(useRecordSetStore().active).toBeNull();

  // canOpenDetail false → emit path (no push)
  pushes.length = 0;
  const blocked = mountApp(ChoyListView as any, {
    props: {
      store: makeStore(),
      detailPath: '/base/countries/:id',
      canOpenDetail: () => false,
      showHeader: false,
      showPaginate: false,
    },
    plugins: [pinia, router as any],
  });
  await flushPromises();
  await (blocked.setupState() as any).onRowClick({
    rowData: { kind: 'record', key: 'a', payload: { Id: 'a' } },
    rowIndex: 0,
    rowKey: 'a',
  });
  await flushPromises();
  expect(pushes).toEqual([]);

  mounted.unmount();
  blocked.unmount();
  restoreSfc(ChoyTableHost);
  restoreSfc(ChoyTableColumn);
  restoreSfc(ChoyPagination);
  restoreSfc(ListInlineEditScope);
});

test('detailPath clear preserves other-model recordSet session', async () => {
  const pinia = createPinia();
  setActivePinia(pinia);
  useRecordSetStore().seed({
    key: 'partner',
    model: 'partner.Partner',
    condition: {},
    total: 1,
    index: 0,
    ids: ['p1'],
    windowStart: 0,
  });
  const { router } = createFeStubRouter({
    route: { path: '/base/countries', fullPath: '/base/countries' },
  });
  stubSfc(ChoyTableHost, {
    name: 'ChoyTableHost',
    setup: () => () => h('div'),
  });
  stubSfc(ChoyTableColumn, { name: 'ChoyTableColumn', setup: () => () => null });
  stubSfc(ChoyPagination, { name: 'ChoyPagination', setup: () => () => h('div') });
  stubSfc(ListInlineEditScope, {
    name: 'ListInlineEditScope',
    setup: (_p: any, { slots }: any) => () => h('div', slots.default?.()),
  });

  const mounted = mountApp(ChoyListView as any, {
    props: {
      store: makeStore(),
      detailPath: '/base/countries/:id',
      showHeader: false,
      showPaginate: false,
    },
    plugins: [pinia, router as any],
  });
  await flushPromises();
  expect(useRecordSetStore().getActiveForModel('partner.Partner')?.ids).toEqual(['p1']);
  mounted.unmount();
  restoreSfc(ChoyTableHost);
  restoreSfc(ChoyTableColumn);
  restoreSfc(ChoyPagination);
  restoreSfc(ListInlineEditScope);
});

test('detailPath emits row-click for null path / blank id; snapshot throw still opens', async () => {
  const pinia = createPinia();
  setActivePinia(pinia);
  const { router } = createFeStubRouter({
    route: { path: '/base/countries', fullPath: '/base/countries' },
  });
  const pushes: unknown[] = [];
  router.push = (to: unknown) => {
    pushes.push(to);
    return Promise.resolve(to);
  };
  stubSfc(ChoyTableHost, {
    name: 'ChoyTableHost',
    setup: () => () => h('div'),
  });
  stubSfc(ChoyTableColumn, { name: 'ChoyTableColumn', setup: () => () => null });
  stubSfc(ChoyPagination, { name: 'ChoyPagination', setup: () => () => h('div') });
  stubSfc(ListInlineEditScope, {
    name: 'ListInlineEditScope',
    setup: (_p: any, { slots }: any) => () => h('div', slots.default?.()),
  });

  const nullPath = mountApp(ChoyListView as any, {
    props: {
      store: makeStore(),
      detailPath: () => null,
      showHeader: false,
      showPaginate: false,
    },
    plugins: [pinia, router as any],
  });
  await flushPromises();
  await (nullPath.setupState() as any).onRowClick({
    rowData: { kind: 'record', key: 'a', payload: { Id: 'a' } },
    rowIndex: 0,
    rowKey: 'a',
  });
  await flushPromises();
  expect(pushes).toEqual([]);

  await (nullPath.setupState() as any).onRowClick({
    rowData: { kind: 'record', key: 'x', payload: {} },
    rowIndex: 0,
    rowKey: 'x',
  });
  await flushPromises();
  expect(pushes).toEqual([]);
  nullPath.unmount();

  const boomStore = makeStore();
  Object.defineProperty(boomStore, 'state', {
    get() {
      throw new Error('snapshot boom');
    },
  });
  const boom = mountApp(ChoyListView as any, {
    props: {
      store: boomStore,
      detailPath: '/base/countries/:id',
      showHeader: false,
      showPaginate: false,
    },
    plugins: [pinia, router as any],
  });
  await flushPromises();
  await (boom.setupState() as any).onRowClick({
    rowData: { kind: 'record', key: 'b', payload: { Id: 'b' } },
    rowIndex: 0,
    rowKey: 'b',
  });
  await flushPromises();
  expect(pushes).toEqual(['/base/countries/b']);
  boom.unmount();

  restoreSfc(ChoyTableHost);
  restoreSfc(ChoyTableColumn);
  restoreSfc(ChoyPagination);
  restoreSfc(ListInlineEditScope);
});

test('detailPath clear catches missing Pinia on mount', async () => {
  const { router } = createFeStubRouter({
    route: { path: '/base/countries', fullPath: '/base/countries' },
  });
  stubSfc(ChoyTableHost, { name: 'ChoyTableHost', setup: () => () => h('div') });
  stubSfc(ChoyTableColumn, { name: 'ChoyTableColumn', setup: () => () => null });
  stubSfc(ChoyPagination, { name: 'ChoyPagination', setup: () => () => h('div') });
  stubSfc(ListInlineEditScope, {
    name: 'ListInlineEditScope',
    setup: (_p: any, { slots }: any) => () => h('div', slots.default?.()),
  });
  const mounted = mountApp(ChoyListView as any, {
    props: {
      store: makeStore(),
      detailPath: '/base/countries/:id',
      showHeader: false,
      showPaginate: false,
    },
    plugins: [router as any],
  });
  await flushPromises();
  expect(mounted.el != null).toBe(true);
  mounted.unmount();
  restoreSfc(ChoyTableHost);
  restoreSfc(ChoyTableColumn);
  restoreSfc(ChoyPagination);
  restoreSfc(ListInlineEditScope);
});
