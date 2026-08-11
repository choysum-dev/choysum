// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, reactive, ref } from 'vue';
import { mount, flushPromises } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const { applyMock, preloadLaneMock, awaitFieldSelectionMock, deferState, oSearchViewSentinel } = vi.hoisted(() => {
  const { markRaw } = require('vue') as typeof import('vue');
  return {
    applyMock: vi.fn(async () => {}),
    preloadLaneMock: vi.fn(async () => {}),
    awaitFieldSelectionMock: vi.fn(async () => {}),
    deferState: { defer: false },
    oSearchViewSentinel: markRaw({
      name: 'OSearchViewSentinel',
      setup: () => () => null,
    }),
  };
});

vi.mock('vue-router', () => ({ useRouter: () => ({ push: vi.fn() }) }));

vi.mock('@/web/web/controllers/kanbanController', () => ({
  createKanbanController: vi.fn(() => ({
    vm: reactive({
      result: { kind: 'search', total: 0, rows: [] },
    }),
    lanes: ref([]),
    laneRecords: ref({}),
    apply: applyMock,
    paginate: vi.fn(async () => {}),
    getLaneField: () => null,
    getLaneRemain: () => 0,
    preloadLane: preloadLaneMock,
    loadMoreLane: vi.fn(async () => {}),
  })),
}));

vi.mock('@/web/web/query/utils/registry/fieldReady', () => ({
  awaitFieldSelection: (...args: any[]) => awaitFieldSelectionMock(...args),
}));

vi.mock('@/web/web/components/view/OSearchView.vue', () => ({
  default: oSearchViewSentinel,
}));

vi.mock('@/web/web/components/view/kanbanFirstFrame', () => ({
  shouldDeferViewFirstFrame: (searchView: unknown, oSearchView: unknown) =>
    deferState.defer && searchView === oSearchView,
  shouldDeferKanbanFirstFrame: (searchView: unknown, oSearchView: unknown) =>
    deferState.defer && searchView === oSearchView,
}));

vi.mock('@/web/web/i18n', async () => {
  const actual = await vi.importActual<typeof import('@/web/web/i18n')>('@/web/web/i18n');
  return {
    ...actual,
    createTranslate: () => ({ _t: (msg: string) => msg, _lt: (msg: string) => msg }),
  };
});

vi.mock('element-plus', async () => {
  const actual = await vi.importActual<any>('element-plus');
  return {
    ...actual,
    ElMessage: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
  };
});

vi.mock('vuedraggable', () => ({
  default: defineComponent({
    name: 'DraggableStub',
    setup(_, { slots }) {
      return () => h('div', { class: 'draggable-stub' }, slots.item?.({ element: { key: '1', payload: { Id: '1' } }, index: 0 }));
    },
  }),
}));

import OKanbanView from './OKanbanView.vue';

function makeStore() {
  return {
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
      orderBy: undefined,
    },
  } as any;
}

const stubs = {
  OViewContainer: {
    template: `<div class="ovc"><slot name="header" /><slot /><slot name="fields" /></div>`,
  },
  OPagination: true,
  'el-button': true,
  'el-icon': true,
  OSearchView: true,
};

describe('OKanbanView first-frame load', () => {
  beforeEach(() => {
    applyMock.mockClear();
    preloadLaneMock.mockClear();
    awaitFieldSelectionMock.mockClear();
    awaitFieldSelectionMock.mockImplementation(async () => {});
    deferState.defer = false;
  });

  it('skips mount apply when first-frame should defer to OSearchView', async () => {
    deferState.defer = true;
    const wrapper = mount(OKanbanView as any, {
      props: {
        store: makeStore(),
        searchView: oSearchViewSentinel,
        showHeader: true,
        showActions: false,
        showPaginate: false,
      },
      global: { stubs },
    });
    try {
      await flushPromises();
      expect(awaitFieldSelectionMock).not.toHaveBeenCalled();
      expect(applyMock).not.toHaveBeenCalled();
    } finally {
      wrapper.unmount();
    }
  });

  it('still mounts apply for a custom searchView even when defer flag is set', async () => {
    deferState.defer = true;
    const SearchStub = defineComponent({
      name: 'SearchStub',
      setup() {
        return () => h('div');
      },
    });
    const wrapper = mount(OKanbanView as any, {
      props: {
        store: makeStore(),
        searchView: SearchStub,
        showHeader: true,
        showActions: false,
        showPaginate: false,
      },
      global: { stubs },
    });
    try {
      await flushPromises();
      expect(applyMock).toHaveBeenCalled();
    } finally {
      wrapper.unmount();
    }
  });

  it('runs mount apply when first-frame should not defer', async () => {
    deferState.defer = false;
    const CustomSearch = defineComponent({
      name: 'CustomSearch',
      setup() {
        return () => h('div', { class: 'custom-search' });
      },
    });
    const wrapper = mount(OKanbanView as any, {
      props: {
        store: makeStore(),
        searchView: CustomSearch,
        showHeader: true,
        showActions: false,
        showPaginate: false,
      },
      global: { stubs },
    });
    try {
      await flushPromises();
      expect(awaitFieldSelectionMock).toHaveBeenCalled();
      expect(applyMock).toHaveBeenCalled();
    } finally {
      wrapper.unmount();
    }
  });

  it('skips mount apply when custom search already emitted query-update', async () => {
    deferState.defer = false;
    // Hang field selection so onSearch never reaches apply; mount must still skip.
    awaitFieldSelectionMock.mockImplementationOnce(() => new Promise(() => {}));
    const SyncEmitSearch = defineComponent({
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
    });
    const wrapper = mount(OKanbanView as any, {
      props: {
        store: makeStore(),
        searchView: SyncEmitSearch,
        showHeader: true,
        showActions: false,
        showPaginate: false,
      },
      global: { stubs },
    });
    try {
      await flushPromises();
      expect(awaitFieldSelectionMock).toHaveBeenCalledTimes(1);
      expect(applyMock).not.toHaveBeenCalled();
    } finally {
      wrapper.unmount();
    }
  });

  it('onSearch with falsy payload does not apply and does not block mount fallback', async () => {
    deferState.defer = false;
    const FalsyEmitSearch = defineComponent({
      name: 'FalsyEmitSearch',
      emits: ['query-update'],
      setup(_, { emit }) {
        emit('query-update', null as any);
        return () => h('div');
      },
    });
    const wrapper = mount(OKanbanView as any, {
      props: {
        store: makeStore(),
        searchView: FalsyEmitSearch,
        showHeader: true,
        showActions: false,
        showPaginate: false,
      },
      global: { stubs },
    });
    try {
      await flushPromises();
      // null payload is ignored by onSearch, so mount still runs the fallback apply once.
      expect(applyMock).toHaveBeenCalledTimes(1);
    } finally {
      wrapper.unmount();
    }
  });
});
