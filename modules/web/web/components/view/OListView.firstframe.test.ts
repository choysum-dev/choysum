// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, reactive, ref } from 'vue';
import { mount, flushPromises } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const { applyMock, awaitFieldSelectionMock, deferState, oSearchViewSentinel } = vi.hoisted(() => {
  // Plain objects passed as searchView get proxied; markRaw keeps === stable with the SFC import.
  const { markRaw } = require('vue') as typeof import('vue');
  return {
    applyMock: vi.fn(async () => {}),
    awaitFieldSelectionMock: vi.fn(async () => {}),
    deferState: { defer: false },
    oSearchViewSentinel: markRaw({
      name: 'OSearchViewSentinel',
      setup: () => () => null,
    }),
  };
});

vi.mock('vue-router', () => ({ useRouter: () => ({ push: vi.fn() }) }));

vi.mock('@/web/web/controllers/listController', () => ({
  createListController: vi.fn(() => ({
    vm: reactive({
      visibleNodes: [],
      result: { kind: 'search', total: 0 },
      expandedKeys: new Set(),
    }),
    apply: applyMock,
    paginate: vi.fn(async () => {}),
    sort: vi.fn(async () => {}),
    expandGroup: vi.fn(),
    loadMoreGroupChildren: vi.fn(async () => {}),
    loadMoreGroupRecords: vi.fn(async () => {}),
  })),
}));

vi.mock('@/web/web/composables/useAdaptiveHeight', () => ({
  useAdaptiveHeight: () => ({
    height: ref(400),
    pxHeight: ref('400px'),
    recompute: vi.fn(),
  }),
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
    ElMessageBox: { confirm: vi.fn() },
  };
});

import OListView from './OListView.vue';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

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
  OVTable: true,
  OVColumn: true,
  OPagination: true,
  'el-button': true,
  'el-icon': true,
  OSearchView: true,
};

describe('OListView first-frame load', () => {
  beforeEach(() => {
    applyMock.mockClear();
    awaitFieldSelectionMock.mockClear();
    awaitFieldSelectionMock.mockImplementation(async () => {});
    deferState.defer = false;
  });

  it('waits for OSearchView first-frame query-update instead of mount apply', () => {
    const dir = dirname(fileURLToPath(import.meta.url));
    const src = readFileSync(join(dir, 'OListView.vue'), 'utf8');
    expect(src).toContain('shouldDeferViewFirstFrame(resolvedSearchView.value, OSearchView)');
  });

  it('skips mount apply when first-frame should defer to OSearchView', async () => {
    deferState.defer = true;
    const wrapper = mount(OListView as any, {
      props: {
        store: makeStore(),
        // Same sentinel OListView imports as OSearchView (mocked above).
        searchView: oSearchViewSentinel,
        showPaginate: false,
        refreshAction: false,
        deleteAction: false,
      },
      global: { stubs },
    });
    await flushPromises();
    expect(applyMock).not.toHaveBeenCalled();
    expect(awaitFieldSelectionMock).not.toHaveBeenCalled();
    wrapper.unmount();
  });

  it('still mounts apply for a custom searchView even when defer flag is set', async () => {
    deferState.defer = true;
    const SearchStub = defineComponent({
      name: 'SearchStub',
      setup() {
        return () => h('div');
      },
    });
    const wrapper = mount(OListView as any, {
      props: {
        store: makeStore(),
        searchView: SearchStub,
        showPaginate: false,
        refreshAction: false,
        deleteAction: false,
      },
      global: { stubs },
    });
    await flushPromises();
    expect(awaitFieldSelectionMock).toHaveBeenCalled();
    expect(applyMock).toHaveBeenCalled();
    wrapper.unmount();
  });

  it('runs mount apply when first-frame should not defer', async () => {
    deferState.defer = false;
    const wrapper = mount(OListView as any, {
      props: {
        store: makeStore(),
        showPaginate: false,
        refreshAction: false,
        deleteAction: false,
      },
      global: { stubs },
    });
    await flushPromises();
    expect(awaitFieldSelectionMock).toHaveBeenCalled();
    expect(applyMock).toHaveBeenCalled();
    wrapper.unmount();
  });
});
