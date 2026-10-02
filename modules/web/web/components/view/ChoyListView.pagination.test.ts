// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, markRaw, reactive } from 'vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ChoyListView from './ChoyListView.vue';
import ChoyPagination from './ChoyPagination.vue';
import ChoySearchView from './ChoySearchView.vue';
import ChoyTableHost from '@/web/web/components/internal/ChoyTableHost.vue';
import ChoyTableColumn from '@/web/web/components/table/ChoyTableColumn.vue';
import ListInlineEditScope from '@/web/web/components/view/ListInlineEditScope.vue';

function makeStore(pagination: { limit: number; offset: number }, total = 100) {
  return {
    fullModelName: 'partner.Partner',
    storeId: 'list-page-' + Math.random().toString(36).slice(2),
    fieldsMetadata: { Name: { type: 'varchar' } },
    state: {
      queryState: {
        keyword: '',
        appliedFilters: [],
        appliedGroups: [],
        keywordFields: [],
        pagination: { ...pagination },
      },
      result: { total },
      selection: [],
      planCache: new Map(),
      orderBy: undefined,
    },
    setContext: () => {},
    getContext: () => ({}),
    withContext: async (_c: any, fn: any) => fn(),
    Search: fnRecorder(async () => []),
    UpdateById: fnRecorder(async () => ({})),
  } as any;
}

describe('ListView pagination models', () => {
  beforeEach(() => {
    stubSfc(ChoyTableHost, {
      name: 'ChoyTableHost',
      setup(_p: any, { slots }: any) {
        return () => h('div', { 'data-stub': 'ChoyTableHost' }, [slots.default?.(), slots.empty?.()]);
      },
    });
    stubSfc(ChoyTableColumn, {
      name: 'ChoyTableColumn',
      setup: () => () => h('div', { 'data-stub': 'ChoyTableColumn' }),
    });
    stubSfc(ChoyPagination, {
      name: 'ChoyPagination',
      setup: () => () => h('div', { 'data-stub': 'Pagination' }),
    });
    stubSfc(ListInlineEditScope, {
      name: 'ListInlineEditScope',
      setup(_p: any, { slots }: any) {
        return () => h('div', { 'data-stub': 'inline-scope' }, slots.default?.());
      },
    });
    stubSfc(ChoySearchView as any, {
      name: 'ChoySearchView',
      setup: () => () => h('div', { 'data-stub': 'ChoySearchView' }),
    });
  });

  afterEach(() => {
    restoreSfc(ChoyTableHost);
    restoreSfc(ChoyTableColumn);
    restoreSfc(ChoyPagination);
    restoreSfc(ListInlineEditScope);
    restoreSfc(ChoySearchView as any);
  });

  test('page/pageSize models paginate once; size change suppresses stale page reset', async () => {
    const store = reactive(makeStore({ limit: 20, offset: 40 }));
    const paginateEvents: Array<{ page: number; pageSize: number }> = [];
    const { plugins } = buildPageMountGlobal();
    const mounted = mountApp(ChoyListView as any, {
      props: {
        store,
        searchView: markRaw(ChoySearchView),
        // Keep ChoyPagination out of the tree (host crash under FE-QJS); drive exposed models.
        showPaginate: false,
        refreshAction: false,
        deleteAction: false,
      },
      plugins,
      on: {
        onPaginate: (payload: { page: number; pageSize: number }) => {
          paginateEvents.push({ ...payload });
          store.state.queryState.pagination = {
            limit: payload.pageSize,
            offset: Math.max(0, (payload.page - 1) * payload.pageSize),
          };
        },
      },
      stubs: { ElButton: true, ElIcon: true },
    });
    await flushPromises();

    const root = mounted.root as {
      listPageModel: number;
      listPageSizeModel: number;
      canMountPagination: boolean;
    };
    expect(root.canMountPagination).toBe(true);
    expect(root.listPageModel).toBe(3);
    expect(root.listPageSizeModel).toBe(20);

    const beforeNoop = paginateEvents.length;
    root.listPageModel = 3;
    await flushPromises();
    expect(paginateEvents.length).toBe(beforeNoop);

    root.listPageModel = 1;
    await flushPromises();
    expect(paginateEvents.some((e) => e.page === 1 && e.pageSize === 20)).toBe(true);

    store.state.queryState.pagination = { limit: 20, offset: 40 };
    await flushPromises();
    const afterRestore = paginateEvents.length;
    root.listPageSizeModel = 50;
    // Simulate ChoyPagination resetting page→1 in the same turn as pageSize.
    root.listPageModel = 1;
    await flushPromises();
    const sizeEvents = paginateEvents.slice(afterRestore);
    expect(sizeEvents.some((e) => e.pageSize === 50 && e.page === 1)).toBe(true);
    expect(sizeEvents.some((e) => e.pageSize === 20)).toBe(false);
    expect(store.state.queryState.pagination.limit).toBe(50);

    const beforeSame = paginateEvents.length;
    root.listPageSizeModel = 50;
    root.listPageSizeModel = 0;
    await flushPromises();
    expect(paginateEvents.length).toBe(beforeSame);

    mounted.unmount();
  });

  test('canMountPagination is false while total is unknown and offset is restored', async () => {
    const store = reactive(makeStore({ limit: 20, offset: 40 }, 0));
    const { plugins } = buildPageMountGlobal();
    const mounted = mountApp(ChoyListView as any, {
      props: {
        store,
        searchView: markRaw(ChoySearchView),
        showPaginate: false,
        refreshAction: false,
        deleteAction: false,
      },
      plugins,
      stubs: { ElButton: true, ElIcon: true },
    });
    await flushPromises();
    expect((mounted.root as { canMountPagination: boolean }).canMountPagination).toBe(false);
    mounted.unmount();
  });
});
