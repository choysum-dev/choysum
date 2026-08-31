// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach } from 'vitest';
import { flushPromises, mount } from '@vue/test-utils';
import { reactive, ref } from 'vue';

const routerPush = vi.fn(async () => undefined);

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: routerPush,
    resolve: vi.fn((loc: { name?: string }) => ({
      name: loc?.name,
      matched: loc?.name ? [{ path: '/x' }] : [],
    })),
  }),
  useRoute: () => ({ name: 'PartnerList', fullPath: '/partner/partners', params: {}, query: {} }),
}));

vi.mock('@/web/web/controllers/listController', () => ({
  createListController: vi.fn(() => ({
    vm: reactive({
      visibleNodes: [],
      result: { kind: 'search', total: 0 },
      expandedKeys: new Set(),
    }),
    apply: vi.fn(async () => {}),
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
  awaitFieldSelection: vi.fn(async () => {}),
}));

vi.mock('@/web/web/i18n', () => ({
  createTranslate: () => ({ _t: (msg: string) => msg, _lt: (msg: string) => msg }),
}));

vi.mock('element-plus', async () => {
  const actual = await vi.importActual<any>('element-plus');
  return {
    ...actual,
    ElMessage: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
    ElMessageBox: { confirm: vi.fn() },
  };
});

import OListView from './OListView.vue';

function makeStore() {
  return {
    fullModelName: 'partner.Partner',
    storeId: 's',
    fieldsMetadata: {},
    state: {
      queryState: { pagination: { limit: 20, offset: 0 } },
      result: { total: 0 },
      selection: [],
      planCache: new Map(),
    },
    setContext: vi.fn(),
    getContext: vi.fn(() => ({})),
    withContext: vi.fn(async (_c: any, fn: any) => fn()),
  };
}

const stubs = {
  OViewContainer: { template: '<div><slot name="header" /><slot /></div>' },
  OPagination: true,
  OVTable: true,
  OVColumn: true,
  'el-button': { template: '<button type="button" v-bind="$attrs"><slot /></button>' },
  'el-icon': true,
};

describe('OListView create action', () => {
  beforeEach(() => {
    routerPush.mockReset();
    routerPush.mockImplementation(async () => undefined);
  });

  it('pushes explicit createAction when New is clicked', async () => {
    const wrapper = mount(OListView as any, {
      props: {
        store: makeStore(),
        createAction: '/partner/partners/new',
        showHeader: true,
        showActions: true,
        refreshAction: false,
        deleteAction: false,
        showPaginate: false,
      },
      global: { stubs },
    });
    await flushPromises();

    const newBtn = wrapper.findAll('button').find(b => b.text().includes('New'));
    expect(newBtn).toBeTruthy();
    await newBtn!.trigger('click');
    await flushPromises();
    expect(routerPush).toHaveBeenCalledWith('/partner/partners/new');
    wrapper.unmount();
  });

  it('emits action-error when create navigation fails', async () => {
    routerPush.mockRejectedValueOnce(new Error('nav failed'));
    const wrapper = mount(OListView as any, {
      props: {
        store: makeStore(),
        createAction: '/partner/partners/new',
        showHeader: true,
        showActions: true,
        refreshAction: false,
        deleteAction: false,
        showPaginate: false,
      },
      global: { stubs },
    });
    await flushPromises();

    const newBtn = wrapper.findAll('button').find(b => b.text().includes('New'));
    expect(newBtn).toBeTruthy();
    await newBtn!.trigger('click');
    await flushPromises();
    expect(wrapper.emitted('action-error')?.[0]?.[0]).toMatchObject({
      action: 'create',
      error: expect.objectContaining({ message: 'nav failed' }),
    });
    wrapper.unmount();
  });

  it('hides New when createAction resolves to empty', async () => {
    const wrapper = mount(OListView as any, {
      props: {
        store: makeStore(),
        createAction: '',
        showHeader: true,
        showActions: true,
        refreshAction: false,
        deleteAction: false,
        showPaginate: false,
      },
      global: { stubs },
    });
    await flushPromises();
    expect(wrapper.findAll('button').some(b => b.text().includes('New'))).toBe(false);
    // Defensive early-return in handleCreate when no target is resolved.
    await (wrapper.vm as any).$.setupState.handleCreate();
    expect(routerPush).not.toHaveBeenCalled();
    wrapper.unmount();
  });
});
