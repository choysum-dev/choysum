// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, markRaw } from 'vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import { provideOPageContext } from '@/web/web/composables/usePageContext';
import OKanbanView from './OKanbanView.vue';
import OSearchView from './OSearchView.vue';
import OViewContainer from '@/web/web/components/view/OViewContainer.vue';
import OPagination from './OPagination.vue';

function makeStore(search?: ReturnType<typeof fnRecorder>) {
  return {
    fullModelName: 'partner.Partner',
    storeId: 'kanban-ff-' + Math.random().toString(36).slice(2),
    fieldsMetadata: {},
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

function stubKanbanChrome() {
  stubSfc(OViewContainer, {
    name: 'OViewContainer',
    setup(_p: any, { slots }: any) {
      return () => h('div', { class: 'ovc' }, [slots.header?.(), slots.fields?.(), slots.default?.()]);
    },
  });
  stubSfc(OPagination, {
    name: 'OPagination',
    setup: () => () => h('div', { 'data-stub': 'OPagination' }),
  });
  stubSfc(OSearchView, {
    name: 'OSearchView',
    setup: () => () => h('div', { 'data-stub': 'OSearchView' }),
  });
}

function restoreKanbanChrome() {
  restoreSfc(OViewContainer);
  restoreSfc(OPagination);
  restoreSfc(OSearchView);
}

describe('OKanbanView first-frame load', () => {
  beforeEach(() => {
    stubKanbanChrome();
  });

  afterEach(() => {
    restoreKanbanChrome();
  });

  test('skips mount apply when first-frame should defer to OSearchView', async () => {
    const search = fnRecorder(async () => []);
    const { plugins } = buildPageMountGlobal();
    const { unmount } = mountApp(OKanbanView as any, {
      props: {
        store: makeStore(search),
        searchView: OSearchView,
        showHeader: true,
        showActions: false,
        showPaginate: false,
      },
      plugins,
      stubs: { ElButton: true, ElIcon: true },
    });
    try {
      await flushPromises();
      expect(search.calls.length).toBe(0);
    } finally {
      unmount();
    }
  });

  test('still mounts apply for a custom searchView even when defer flag would apply', async () => {
    const search = fnRecorder(async () => []);
    const SearchStub = markRaw(
      defineComponent({
        name: 'SearchStub',
        setup: () => () => h('div'),
      })
    );
    const { plugins } = buildPageMountGlobal();
    const { unmount } = mountApp(OKanbanView as any, {
      props: {
        store: makeStore(search),
        searchView: SearchStub,
        showHeader: true,
        showActions: false,
        showPaginate: false,
      },
      plugins,
      stubs: { ElButton: true, ElIcon: true },
    });
    try {
      await flushPromises();
      expect(search.calls.length).toBeGreaterThan(0);
    } finally {
      unmount();
    }
  });

  test('runs mount apply when first-frame should not defer', async () => {
    const search = fnRecorder(async () => []);
    const CustomSearch = markRaw(
      defineComponent({
        name: 'CustomSearch',
        setup: () => () => h('div', { class: 'custom-search' }),
      })
    );
    const { plugins } = buildPageMountGlobal();
    const { unmount, q } = mountApp(OKanbanView as any, {
      props: {
        store: makeStore(search),
        searchView: CustomSearch,
        showHeader: true,
        showActions: false,
        showPaginate: false,
      },
      plugins,
      stubs: { ElButton: true, ElIcon: true },
    });
    try {
      await flushPromises();
      expect(q('.custom-search')).toBeTruthy();
      expect(search.calls.length).toBeGreaterThan(0);
    } finally {
      unmount();
    }
  });

  test('covers resolvedSearchView null branch when searchView is omitted', async () => {
    const search = fnRecorder(async () => []);
    const { plugins } = buildPageMountGlobal();
    const { unmount, q } = mountApp(OKanbanView as any, {
      props: {
        store: makeStore(search),
        showHeader: true,
        showActions: false,
        showPaginate: false,
      },
      plugins,
      stubs: { ElButton: true, ElIcon: true },
    });
    try {
      await flushPromises();
      expect(q('.o-kanban__search')).toBeFalsy();
      expect(search.calls.length).toBeGreaterThan(0);
    } finally {
      unmount();
    }
  });

  test('skips mount apply when custom search already emitted query-update', async () => {
    const search = fnRecorder(async () => []);
    const SyncEmitSearch = markRaw(
      defineComponent({
        name: 'SyncEmitSearch',
        emits: ['query-update'],
        setup(_, { emit }) {
          emit('query-update', {
            keyword: 'pre',
            appliedFilters: [],
            appliedGroups: [],
          });
          return () => h('div', { class: 'sync-emit-search' });
        },
      })
    );
    const { plugins } = buildPageMountGlobal();
    const { unmount } = mountApp(OKanbanView as any, {
      props: {
        store: makeStore(search),
        searchView: SyncEmitSearch,
        showHeader: true,
        showActions: false,
        showPaginate: false,
      },
      plugins,
      stubs: { ElButton: true, ElIcon: true },
    });
    try {
      await flushPromises();
      // onSearch marks firstApplied and schedules apply; mount fallback must not double-apply.
      // With a real controller both paths share Search — assert at most one Search after settle.
      expect(search.calls.length).toBeLessThanOrEqual(1);
    } finally {
      unmount();
    }
  });

  test('onSearch with falsy payload does not apply and does not block mount fallback', async () => {
    const search = fnRecorder(async () => []);
    const FalsyEmitSearch = markRaw(
      defineComponent({
        name: 'FalsyEmitSearch',
        emits: ['query-update'],
        setup(_, { emit }) {
          emit('query-update', null as any);
          return () => h('div');
        },
      })
    );
    const { plugins } = buildPageMountGlobal();
    const { unmount } = mountApp(OKanbanView as any, {
      props: {
        store: makeStore(search),
        searchView: FalsyEmitSearch,
        showHeader: true,
        showActions: false,
        showPaginate: false,
      },
      plugins,
      stubs: { ElButton: true, ElIcon: true },
    });
    try {
      await flushPromises();
      expect(search.calls.length).toBe(1);
    } finally {
      unmount();
    }
  });
});

describe('OKanbanView page action target', () => {
  beforeEach(() => {
    stubKanbanChrome();
  });

  afterEach(() => {
    restoreKanbanChrome();
  });

  test('auto-registers refresh when omitted registerActionTarget stays undefined', async () => {
    const store = makeStore(fnRecorder(async () => []));
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
          h(OKanbanView as any, {
            showHeader: false,
            showActions: false,
            showPaginate: false,
          }),
      },
      stubs: { ElButton: true, ElIcon: true },
    });
    try {
      await flushPromises();
      const target = ctx!.actionTarget.value;
      expect(target).toBeTruthy();
      expect(target!.selectedItems).toEqual([]);
      const search = store.Search as ReturnType<typeof fnRecorder>;
      search.mockClear();
      await target!.refresh?.();
      await flushPromises();
      expect(search.calls.length).toBeGreaterThan(0);
    } finally {
      unmount();
    }
  });
});
