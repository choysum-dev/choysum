// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, markRaw } from 'vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ChoyListView from './ChoyListView.vue';
import ChoySearchView from './ChoySearchView.vue';
import ListInlineEditScope from '@/web/web/components/view/ListInlineEditScope.vue';
import ViewContainer from '@/web/web/components/view/ViewContainer.vue';
import ChoyPagination from './ChoyPagination.vue';
import ChoyTableHost from '@/web/web/components/internal/ChoyTableHost.vue';
import ChoyTableColumn from '@/web/web/components/table/ChoyTableColumn.vue';

function makeStore(search?: ReturnType<typeof fnRecorder>) {
  return {
    fullModelName: 'partner.Partner',
    storeId: 'list-ff-' + Math.random().toString(36).slice(2),
    fieldsMetadata: { Name: { type: 'varchar' } },
    state: {
      queryState: {
        keyword: '',
        appliedFilters: [],
        appliedGroups: [],
        keywordFields: [],
        pagination: { limit: 20, offset: 0 },
      },
      result: { total: 0 },
      selection: [],
      planCache: new Map(),
      orderBy: undefined,
    },
    setContext: () => {},
    getContext: () => ({}),
    withContext: async (_c: any, fn: any) => fn(),
    Search: search || (async () => []),
  } as any;
}

function stubListChrome() {
  stubSfc(ViewContainer, {
    name: 'ViewContainer',
    setup(_p: any, { slots }: any) {
      return () => h('div', { class: 'ovc' }, [slots.header?.(), slots.fields?.(), slots.default?.()]);
    },
  });
  stubSfc(ListInlineEditScope, {
    name: 'ListInlineEditScope',
    setup(_p: any, { slots }: any) {
      return () => h('div', { class: 'inline-edit-scope' }, slots.default?.());
    },
  });
  stubSfc(ChoyTableHost, {
    name: 'ChoyTableHost',
    setup: () => () => h('div', { 'data-stub': 'ChoyTableHost' }),
  });
  stubSfc(ChoyTableColumn, {
    name: 'ChoyTableColumn',
    setup: () => () => null,
  });
  stubSfc(ChoyPagination, {
    name: 'ChoyPagination',
    setup: () => () => h('div', { 'data-stub': 'Pagination' }),
  });
  // Keep SearchView identity for shouldDeferViewFirstFrame; render a silent stub.
  stubSfc(ChoySearchView, {
    name: 'ChoySearchView',
    setup: () => () => h('div', { 'data-stub': 'SearchView' }),
  });
}

function restoreListChrome() {
  restoreSfc(ViewContainer);
  restoreSfc(ListInlineEditScope);
  restoreSfc(ChoyTableHost);
  restoreSfc(ChoyTableColumn);
  restoreSfc(ChoyPagination);
  restoreSfc(ChoySearchView);
}

describe('ListView first-frame load', () => {
  beforeEach(() => {
    stubListChrome();
  });

  afterEach(() => {
    restoreListChrome();
  });

  test('skips mount apply when first-frame should defer to SearchView', async () => {
    const search = fnRecorder(async () => []);
    const { plugins } = buildPageMountGlobal();
    const { unmount } = mountApp(ChoyListView as any, {
      props: {
        store: makeStore(search),
        // Same component reference ListView closes over as SearchView.
        searchView: ChoySearchView,
        showPaginate: false,
        refreshAction: false,
        deleteAction: false,
      },
      plugins,
      stubs: { ElButton: true, ElIcon: true },
    });
    await flushPromises();
    expect(search.calls.length).toBe(0);
    unmount();
  });

  test('still mounts apply for a custom searchView even when SearchView would defer', async () => {
    const search = fnRecorder(async () => []);
    const SearchStub = markRaw(
      defineComponent({
        name: 'SearchStub',
        setup: () => () => h('div', { class: 'custom-search' }),
      })
    );
    const { plugins } = buildPageMountGlobal();
    const { unmount } = mountApp(ChoyListView as any, {
      props: {
        store: makeStore(search),
        searchView: SearchStub,
        showPaginate: false,
        refreshAction: false,
        deleteAction: false,
      },
      plugins,
      stubs: { ElButton: true, ElIcon: true },
    });
    await flushPromises();
    expect(search.calls.length).toBeGreaterThan(0);
    unmount();
  });

  test('runs mount apply when first-frame should not defer', async () => {
    const search = fnRecorder(async () => []);
    const { plugins } = buildPageMountGlobal();
    const { unmount } = mountApp(ChoyListView as any, {
      props: {
        store: makeStore(search),
        showPaginate: false,
        refreshAction: false,
        deleteAction: false,
      },
      plugins,
      stubs: { ElButton: true, ElIcon: true },
    });
    await flushPromises();
    expect(search.calls.length).toBeGreaterThan(0);
    unmount();
  });
});
