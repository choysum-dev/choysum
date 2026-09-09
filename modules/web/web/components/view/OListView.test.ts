// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Dense QJS subset of OListView behaviors. Main suite mocked createListController
 * and drove OVTable emits; here we use a real controller + stubbed table chrome
 * and assert expose/action-target / handle visibility contracts.
 */

import { defineComponent, h } from 'vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import { provideOPageContext } from '@/web/web/composables/usePageContext';
import OListView from './OListView.vue';
import OPagination from './OPagination.vue';
import OVTable from '@/web/web/components/vtable/OVTable.vue';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import OListInlineEditScope from '@/web/web/components/view/OListInlineEditScope.vue';

function makeStore(extra?: Record<string, unknown>) {
  return {
    fullModelName: 'partner.Partner',
    storeId: 'list-main-' + Math.random().toString(36).slice(2),
    fieldsMetadata: { Name: { type: 'varchar' }, Sequence: { type: 'integer' } },
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
    Search: async () => [],
    UpdateById: fnRecorder(async () => ({})),
    ...extra,
  } as any;
}

function stubListChrome() {
  stubSfc(OVTable, {
    name: 'OVTable',
    emits: ['row-click', 'selection-change', 'sort-change'],
    setup(_p: any, { slots }: any) {
      return () => h('div', { 'data-stub': 'OVTable' }, [slots.default?.(), slots.empty?.()]);
    },
  });
  stubSfc(OVColumn, {
    name: 'OVColumn',
    setup: () => () => h('div', { 'data-stub': 'OVColumn' }),
  });
  stubSfc(OPagination, {
    name: 'OPagination',
    setup: () => () => h('div', { 'data-stub': 'OPagination' }),
  });
  stubSfc(OListInlineEditScope, {
    name: 'OListInlineEditScope',
    setup(_p: any, { slots }: any) {
      return () => h('div', { 'data-stub': 'inline-scope' }, slots.default?.());
    },
  });
}

function restoreListChrome() {
  restoreSfc(OVTable);
  restoreSfc(OVColumn);
  restoreSfc(OPagination);
  restoreSfc(OListInlineEditScope);
}

describe('OListView', () => {
  beforeEach(() => {
    stubListChrome();
  });

  afterEach(() => {
    restoreListChrome();
  });

  test('hides handle column when not editable', async () => {
    const { plugins } = buildPageMountGlobal();
    const { unmount, qa, root } = mountApp(OListView as any, {
      props: {
        store: makeStore(),
        editable: false,
        showHandle: true,
        showPaginate: false,
        refreshAction: false,
        deleteAction: false,
      },
      plugins,
      stubs: { ElButton: true, ElIcon: true },
    });
    await flushPromises();
    expect(root).toBeTruthy();
    // Handle OVColumn is gated by showHandleColumn; without editable it should not mount.
    expect(qa('[data-stub="OVColumn"]').length).toBe(0);
    unmount();
  });

  test('exposes load and selectedItems', async () => {
    const { plugins } = buildPageMountGlobal();
    const { unmount, root } = mountApp(OListView as any, {
      props: {
        store: makeStore(),
        showPaginate: false,
        refreshAction: false,
        deleteAction: false,
      },
      plugins,
      stubs: { ElButton: true, ElIcon: true },
    });
    await flushPromises();
    expect(typeof root.load).toBe('function');
    expect(Array.isArray(root.selectedItems)).toBe(true);
    unmount();
  });

  test('auto-registers selectedItems and refresh when omitted registerActionTarget stays undefined', async () => {
    const store = makeStore();
    let ctx: ReturnType<typeof provideOPageContext> | null = null;
    const Host = defineComponent({
      setup(_, { slots }) {
        ctx = provideOPageContext({ store });
        return () => h('div', slots.default?.());
      },
    });
    const { plugins } = buildPageMountGlobal();
    const { unmount } = mountApp(Host, {
      plugins,
      slots: {
        default: () =>
          h(OListView as any, {
            showHeader: false,
            showActions: false,
            showPaginate: false,
            refreshAction: false,
            deleteAction: false,
          }),
      },
      stubs: { ElButton: true, ElIcon: true },
    });
    await flushPromises();
    const target = ctx!.actionTarget.value;
    expect(target).toBeTruthy();
    expect(target!.selectedItems).toEqual([]);
    const search = store.Search as any;
    // Search may already have run on mount; clear by replacing.
    const callsBefore = typeof search.calls === 'object' ? search.calls.length : 0;
    store.Search = fnRecorder(async () => []);
    await target!.refresh?.();
    await flushPromises();
    expect((store.Search as ReturnType<typeof fnRecorder>).calls.length).toBeGreaterThan(0);
    void callsBefore;
    unmount();
  });

});
