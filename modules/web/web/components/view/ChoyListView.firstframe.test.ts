// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, markRaw } from 'vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ListView from './ListView.vue';
import SearchView from './SearchView.vue';
import ListInlineEditScope from '@/web/web/components/view/ListInlineEditScope.vue';
import ViewContainer from '@/web/web/components/view/ViewContainer.vue';
import Pagination from './Pagination.vue';
import VTable from '@/web/web/components/vtable/VTable.vue';
import VColumn from '@/web/web/components/vtable/VColumn.vue';

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
  stubSfc(VTable, {
    name: 'VTable',
    setup: () => () => h('div', { 'data-stub': 'VTable' }),
  });
  stubSfc(VColumn, {
    name: 'VColumn',
    setup: () => () => null,
  });
  stubSfc(Pagination, {
    name: 'Pagination',
    setup: () => () => h('div', { 'data-stub': 'Pagination' }),
  });
  // Keep SearchView identity for shouldDeferViewFirstFrame; render a silent stub.
  stubSfc(SearchView, {
    name: 'SearchView',
    setup: () => () => h('div', { 'data-stub': 'SearchView' }),
  });
}

function restoreListChrome() {
  restoreSfc(ViewContainer);
  restoreSfc(ListInlineEditScope);
  restoreSfc(VTable);
  restoreSfc(VColumn);
  restoreSfc(Pagination);
  restoreSfc(SearchView);
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
    const { unmount } = mountApp(ListView as any, {
      props: {
        store: makeStore(search),
        // Same component reference ListView closes over as SearchView.
        searchView: SearchView,
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
    const { unmount } = mountApp(ListView as any, {
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
    const { unmount } = mountApp(ListView as any, {
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
