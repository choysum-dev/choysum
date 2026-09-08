// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, markRaw } from 'vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import OListView from './OListView.vue';
import OSearchView from './OSearchView.vue';
import OListInlineEditScope from '@/web/web/components/view/OListInlineEditScope.vue';
import OViewContainer from '@/web/web/components/view/OViewContainer.vue';
import OPagination from './OPagination.vue';
import OVTable from '@/web/web/components/vtable/OVTable.vue';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';

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
  stubSfc(OViewContainer, {
    name: 'OViewContainer',
    setup(_p: any, { slots }: any) {
      return () => h('div', { class: 'ovc' }, [slots.header?.(), slots.fields?.(), slots.default?.()]);
    },
  });
  stubSfc(OListInlineEditScope, {
    name: 'OListInlineEditScope',
    setup(_p: any, { slots }: any) {
      return () => h('div', { class: 'inline-edit-scope' }, slots.default?.());
    },
  });
  stubSfc(OVTable, {
    name: 'OVTable',
    setup: () => () => h('div', { 'data-stub': 'OVTable' }),
  });
  stubSfc(OVColumn, {
    name: 'OVColumn',
    setup: () => () => null,
  });
  stubSfc(OPagination, {
    name: 'OPagination',
    setup: () => () => h('div', { 'data-stub': 'OPagination' }),
  });
  // Keep OSearchView identity for shouldDeferViewFirstFrame; render a silent stub.
  stubSfc(OSearchView, {
    name: 'OSearchView',
    setup: () => () => h('div', { 'data-stub': 'OSearchView' }),
  });
}

function restoreListChrome() {
  restoreSfc(OViewContainer);
  restoreSfc(OListInlineEditScope);
  restoreSfc(OVTable);
  restoreSfc(OVColumn);
  restoreSfc(OPagination);
  restoreSfc(OSearchView);
}

describe('OListView first-frame load', () => {
  beforeEach(() => {
    stubListChrome();
  });

  afterEach(() => {
    restoreListChrome();
  });

  test('skips mount apply when first-frame should defer to OSearchView', async () => {
    const search = fnRecorder(async () => []);
    const { plugins } = buildPageMountGlobal();
    const { unmount } = mountApp(OListView as any, {
      props: {
        store: makeStore(search),
        // Same component reference OListView closes over as OSearchView.
        searchView: OSearchView,
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

  test('still mounts apply for a custom searchView even when OSearchView would defer', async () => {
    const search = fnRecorder(async () => []);
    const SearchStub = markRaw(
      defineComponent({
        name: 'SearchStub',
        setup: () => () => h('div', { class: 'custom-search' }),
      })
    );
    const { plugins } = buildPageMountGlobal();
    const { unmount } = mountApp(OListView as any, {
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
    const { unmount } = mountApp(OListView as any, {
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
