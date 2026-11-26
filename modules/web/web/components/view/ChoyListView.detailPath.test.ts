// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, KeepAlive, nextTick, ref } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import * as VueRouter from 'vue-router';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ChoyListView from './ChoyListView.vue';
import ChoyPagination from './ChoyPagination.vue';
import ChoyTableHost from '@/web/web/components/internal/ChoyTableHost.vue';
import ChoyTableColumn from '@/web/web/components/table/ChoyTableColumn.vue';
import ListInlineEditScope from '@/web/web/components/view/ListInlineEditScope.vue';
import { choyListPickerChromeProps } from '@/web/web/composables/choyListPickerChrome';
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

test('detailPath grouped List opens without pager seed (visible window ≠ Search offset)', async () => {
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
  expect(useRecordSetStore().active).toBeNull();

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

  // Throwing canOpenDetail must not break the click handler.
  pushes.length = 0;
  const clicks: unknown[] = [];
  const throwing = mountApp(ChoyListView as any, {
    props: {
      store: makeStore(),
      detailPath: '/base/countries/:id',
      canOpenDetail: () => {
        throw new Error('acl boom');
      },
      showHeader: false,
      showPaginate: false,
    },
    on: { onRowClick: (p: unknown) => clicks.push(p) },
    plugins: [pinia, router as any],
  });
  await flushPromises();
  await (throwing.setupState() as any).onRowClick({
    rowData: { kind: 'record', key: 'a', payload: { Id: 'a' } },
    rowIndex: 0,
    rowKey: 'a',
  });
  await flushPromises();
  expect(pushes).toEqual([]);
  expect(clicks.length).toBe(1);

  mounted.unmount();
  blocked.unmount();
  throwing.unmount();
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

  const clicks: unknown[] = [];
  const nullPath = mountApp(ChoyListView as any, {
    props: {
      store: makeStore(),
      detailPath: () => null,
      showHeader: false,
      showPaginate: false,
    },
    on: { onRowClick: (p: unknown) => clicks.push(p) },
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
  expect(clicks.length).toBe(1);

  // Named-route factory can resolve without an id; blank id still emits (no navigation).
  clicks.length = 0;
  const blankId = mountApp(ChoyListView as any, {
    props: {
      store: makeStore(),
      detailPath: () => ({ name: 'CountryDetail', params: {} }),
      showHeader: false,
      showPaginate: false,
    },
    on: { onRowClick: (p: unknown) => clicks.push(p) },
    plugins: [pinia, router as any],
  });
  await flushPromises();
  await (blankId.setupState() as any).openDetailOrEmitRowClick({
    row: {},
    rowIndex: 0,
    rowKey: 'x',
  });
  await flushPromises();
  expect(pushes).toEqual([]);
  expect(clicks.length).toBe(1);
  blankId.unmount();

  // No detailPath → always emit row-click.
  clicks.length = 0;
  const noPath = mountApp(ChoyListView as any, {
    props: {
      store: makeStore(),
      showHeader: false,
      showPaginate: false,
    },
    on: { onRowClick: (p: unknown) => clicks.push(p) },
    plugins: [pinia, router as any],
  });
  await flushPromises();
  await (noPath.setupState() as any).onRowClick({
    rowData: { kind: 'record', key: 'a', payload: { Id: 'a' } },
    rowIndex: 0,
    rowKey: 'a',
  });
  await flushPromises();
  expect(pushes).toEqual([]);
  expect(clicks.length).toBe(1);
  noPath.unmount();
  nullPath.unmount();

  // Keep store.state readable during mount; poison it only for the click snapshot.
  const boomStore = makeStore();
  const realState = boomStore.state;
  let boomOn = false;
  Object.defineProperty(boomStore, 'state', {
    configurable: true,
    get() {
      if (boomOn) throw new Error('snapshot boom');
      return realState;
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
  boomOn = true;
  await (boom.setupState() as any).onRowClick({
    rowData: { kind: 'record', key: 'b', payload: { Id: 'b' } },
    rowIndex: 0,
    rowKey: 'b',
  });
  await flushPromises();
  expect(pushes).toEqual(['/base/countries/b']);
  expect(useRecordSetStore().active).toBeNull();
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

test('detailPath without router emits row-click (chrome-style mount)', async () => {
  const pinia = createPinia();
  setActivePinia(pinia);
  stubSfc(ChoyTableHost, { name: 'ChoyTableHost', setup: () => () => h('div') });
  stubSfc(ChoyTableColumn, { name: 'ChoyTableColumn', setup: () => () => null });
  stubSfc(ChoyPagination, { name: 'ChoyPagination', setup: () => () => h('div') });
  stubSfc(ListInlineEditScope, {
    name: 'ListInlineEditScope',
    setup: (_p: any, { slots }: any) => () => h('div', slots.default?.()),
  });
  const clicks: unknown[] = [];
  // No vue-router plugin → useRouter throws and detailPath falls back to emit.
  const mounted = mountApp(ChoyListView as any, {
    props: {
      store: makeStore(),
      detailPath: '/base/countries/:id',
      showHeader: false,
      showPaginate: false,
    },
    on: { onRowClick: (p: unknown) => clicks.push(p) },
    plugins: [pinia],
  });
  await flushPromises();
  await (mounted.setupState() as any).onRowClick({
    rowData: { kind: 'record', key: 'a', payload: { Id: 'a' } },
    rowIndex: 0,
    rowKey: 'a',
  });
  await flushPromises();
  expect(clicks.length).toBe(1);
  mounted.unmount();
  restoreSfc(ChoyTableHost);
  restoreSfc(ChoyTableColumn);
  restoreSfc(ChoyPagination);
  restoreSfc(ListInlineEditScope);
});

test('picker chrome canOpenDetail survives domain detail-path override (no route push)', async () => {
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

  // Domain ListViews (e.g. CountryListView) bind detail-path after $attrs, so an
  // undefined chrome detailPath is overridden. canOpenDetail:false must still win.
  // clickToSelect:false isolates that gate from the picker select short-circuit.
  const chrome = choyListPickerChromeProps({ selectionMode: 'single' });
  const mounted = mountApp(ChoyListView as any, {
    props: {
      store: makeStore(),
      ...chrome,
      detailPath: '/base/countries/:id',
      clickToSelect: false,
      showHeader: false,
      showPaginate: false,
    },
    plugins: [pinia, router as any],
  });
  await flushPromises();

  await (mounted.setupState() as any).onRowClick({
    rowData: { kind: 'record', key: 'a', payload: { Id: 'a', Name: 'A' } },
    rowIndex: 0,
    rowKey: 'a',
  });
  await flushPromises();

  expect(pushes).toEqual([]);
  mounted.unmount();
  restoreSfc(ChoyTableHost);
  restoreSfc(ChoyTableColumn);
  restoreSfc(ChoyPagination);
  restoreSfc(ListInlineEditScope);
});

test('detailPath KeepAlive reactivate soft-refreshes via useDetailPathCollectionSoftRefresh', async () => {
  const pinia = createPinia();
  setActivePinia(pinia);
  const { router } = createFeStubRouter({
    route: { path: '/base/countries', fullPath: '/base/countries' },
  });
  const store = makeStore();
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

  const showList = ref(true);
  const searches: number[] = [];
  const origSearch = store.Search;
  store.Search = async (...args: unknown[]) => {
    searches.push(1);
    return origSearch(...args);
  };

  const ListPage = defineComponent({
    name: 'DetailPathListPage',
    setup() {
      return () =>
        h(ChoyListView as any, {
          store,
          detailPath: '/base/countries/:id',
          showHeader: false,
          showPaginate: false,
        });
    },
  });
  const Host = defineComponent({
    setup() {
      return () =>
        h(KeepAlive, null, {
          default: () => (showList.value ? h(ListPage) : h('div', { 'data-stub': 'form' }, 'form')),
        });
    },
  });

  const mounted = mountApp(Host as any, { plugins: [pinia, router as any] });
  await flushPromises();
  await nextTick();
  expect(searches.length).toBeGreaterThanOrEqual(1);

  showList.value = false;
  await nextTick();
  showList.value = true;
  await flushPromises();
  await nextTick();
  await nextTick();
  expect(searches.length).toBeGreaterThanOrEqual(2);

  mounted.unmount();
  restoreSfc(ChoyTableHost);
  restoreSfc(ChoyTableColumn);
  restoreSfc(ChoyPagination);
  restoreSfc(ListInlineEditScope);
});
