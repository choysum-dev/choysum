// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Dense QJS subset of ListView behaviors. Main suite mocked createListController
 * and drove table emits; here we use a real controller + stubbed table chrome
 * and assert expose/action-target / handle visibility contracts.
 */

import { defineComponent, h } from 'vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import { providePageContext } from '@/web/web/composables/usePageContext';
import ChoyListView from './ChoyListView.vue';
import ListPagination from './ListPagination.vue';
import ChoyTableHost from '@/web/web/components/internal/ChoyTableHost.vue';
import ChoyVColumn from '@/web/web/components/vtable/ChoyVColumn.vue';
import ListInlineEditScope from '@/web/web/components/view/ListInlineEditScope.vue';

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
  stubSfc(ChoyTableHost, {
    name: 'ChoyTableHost',
    emits: ['row-click', 'selection-change', 'sort-change'],
    setup(_p: any, { slots }: any) {
      return () => h('div', { 'data-stub': 'ChoyTableHost' }, [slots.default?.(), slots.empty?.()]);
    },
  });
  stubSfc(ChoyVColumn, {
    name: 'ChoyVColumn',
    setup: () => () => h('div', { 'data-stub': 'ChoyVColumn' }),
  });
  stubSfc(ListPagination, {
    name: 'ListPagination',
    setup: () => () => h('div', { 'data-stub': 'Pagination' }),
  });
  stubSfc(ListInlineEditScope, {
    name: 'ListInlineEditScope',
    setup(_p: any, { slots }: any) {
      return () => h('div', { 'data-stub': 'inline-scope' }, slots.default?.());
    },
  });
}

function restoreListChrome() {
  restoreSfc(ChoyTableHost);
  restoreSfc(ChoyVColumn);
  restoreSfc(ListPagination);
  restoreSfc(ListInlineEditScope);
}

describe('ListView', () => {
  beforeEach(() => {
    stubListChrome();
  });

  afterEach(() => {
    restoreListChrome();
  });

  test('hides handle column when not editable', async () => {
    const { plugins } = buildPageMountGlobal();
    const { unmount, qa, root } = mountApp(ChoyListView as any, {
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
    // Handle VColumn is gated by showHandleColumn; without editable it should not mount.
    expect(qa('[data-stub="ChoyVColumn"]').length).toBe(0);
    unmount();
  });

  test('exposes load and selectedItems', async () => {
    const { plugins } = buildPageMountGlobal();
    const { unmount, root } = mountApp(ChoyListView as any, {
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
    let ctx: ReturnType<typeof providePageContext> | null = null;
    const Host = defineComponent({
      setup(_, { slots }) {
        ctx = providePageContext({ store });
        return () => h('div', slots.default?.());
      },
    });
    const { plugins } = buildPageMountGlobal();
    const { unmount } = mountApp(Host, {
      plugins,
      slots: {
        default: () =>
          h(ChoyListView as any, {
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
