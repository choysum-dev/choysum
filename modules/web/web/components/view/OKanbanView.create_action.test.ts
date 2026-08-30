// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach } from 'vitest';
import { flushPromises, mount } from '@vue/test-utils';
import { defineComponent, h, reactive, ref } from 'vue';

const routerPush = vi.fn(async () => undefined);

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: routerPush,
    resolve: vi.fn((loc: { name?: string }) => ({
      name: loc?.name,
      matched: loc?.name ? [{ path: '/x' }] : [],
    })),
  }),
  useRoute: () => ({ name: 'TokenKanban', fullPath: '/auth/tokens/kanban', params: {}, query: {} }),
}));

vi.mock('@/web/web/controllers/kanbanController', () => ({
  createKanbanController: vi.fn(() => ({
    vm: reactive({
      result: { kind: 'search', total: 0, rows: [] },
    }),
    lanes: ref([]),
    laneRecords: ref({}),
    apply: vi.fn(async () => {}),
    paginate: vi.fn(async () => {}),
    getLaneField: () => null,
    getLaneRemain: () => 0,
    preloadLane: vi.fn(async () => {}),
    loadMoreLane: vi.fn(async () => {}),
  })),
}));

vi.mock('@/web/web/components/view/kanbanFirstFrame', () => ({
  shouldDeferViewFirstFrame: () => false,
  shouldDeferKanbanFirstFrame: () => false,
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
  };
});

vi.mock('vuedraggable', () => ({
  default: defineComponent({
    name: 'DraggableStub',
    setup(_, { slots }) {
      return () => h('div', { class: 'draggable-stub' }, slots.default?.());
    },
  }),
}));

import OKanbanView from './OKanbanView.vue';

function makeStore() {
  return {
    fullModelName: 'auth.Token',
    storeId: 's',
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
    },
    setContext: vi.fn(),
    getContext: vi.fn(() => ({})),
    withContext: vi.fn(async (_c: any, fn: any) => fn()),
  };
}

const stubs = {
  OViewContainer: { template: '<div><slot name="header" /><slot /></div>' },
  OPagination: true,
  'el-button': { template: '<button type="button" v-bind="$attrs"><slot /></button>' },
  'el-icon': true,
};

describe('OKanbanView create action', () => {
  beforeEach(() => {
    routerPush.mockReset();
    routerPush.mockImplementation(async () => undefined);
  });

  it('pushes createAction when New is clicked', async () => {
    const wrapper = mount(OKanbanView as any, {
      props: {
        store: makeStore(),
        createAction: '/auth/tokens/new',
        showHeader: true,
        showActions: true,
        refreshAction: false,
        showPaginate: false,
      },
      global: { stubs },
    });
    await flushPromises();

    const newBtn = wrapper.findAll('button').find(b => b.text().includes('New'));
    expect(newBtn).toBeTruthy();
    await newBtn!.trigger('click');
    await flushPromises();
    expect(routerPush).toHaveBeenCalledWith('/auth/tokens/new');
    wrapper.unmount();
  });

  it('emits action-error when create navigation fails with a non-Error', async () => {
    routerPush.mockRejectedValueOnce('boom');
    const wrapper = mount(OKanbanView as any, {
      props: {
        store: makeStore(),
        createAction: '/auth/tokens/new',
        showHeader: true,
        showActions: true,
        refreshAction: false,
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
      action: 'paginate',
      error: expect.objectContaining({ message: 'boom' }),
    });
    wrapper.unmount();
  });

  it('hides New when createAction is empty', async () => {
    const wrapper = mount(OKanbanView as any, {
      props: {
        store: makeStore(),
        createAction: '',
        showHeader: true,
        showActions: true,
        refreshAction: false,
        showPaginate: false,
      },
      global: { stubs },
    });
    await flushPromises();
    expect(wrapper.findAll('button').some(b => b.text().includes('New'))).toBe(false);
    await (wrapper.vm as any).$.setupState.handleCreate();
    expect(routerPush).not.toHaveBeenCalled();
    wrapper.unmount();
  });
});
