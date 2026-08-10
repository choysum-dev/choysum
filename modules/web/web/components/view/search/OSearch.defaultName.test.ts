// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@vue/test-utils';
import { nextTick } from 'vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createTermReference } from '@/core/service/i18n';
import OSearch from './OSearch.vue';

const { savedFiltersApi, breadcrumbState, menuState, routeState } = vi.hoisted(() => ({
  savedFiltersApi: {
    state: null as null | {
      favoriteMenuItems: any[];
      loading: boolean;
      loadError: string | null;
      defaultsForOpen: any[];
    },
    load: vi.fn(async () => {}),
    apply: vi.fn(),
    saveCurrent: vi.fn(async () => ({ Id: '1' })),
    remove: vi.fn(async () => {}),
    lastScopeKey: '' as string,
  },
  breadcrumbState: {
    breadcrumbStack: [] as Array<{ title?: string; titleText?: any }>,
  },
  menuState: {
    activeMenu: null as null | { title?: string; titleText?: any },
  },
  routeState: {
    path: '/web/partners/42',
    meta: {} as Record<string, unknown>,
  },
}));

vi.mock('@/web/web/i18n', async () => {
  const actual = await vi.importActual<typeof import('@/web/web/i18n')>('@/web/web/i18n');
  return {
    ...actual,
    createTranslate: () => ({
      _t: (msg: string) => msg,
      _lt: (msg: string) => msg,
    }),
  };
});

vi.mock('@/web/web/stores/breadcrumbStore', () => ({
  useBreadcrumbStore: () => breadcrumbState,
}));

vi.mock('@/web/web/stores/menuStore', () => ({
  useMenuStore: () => menuState,
}));

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<any>('vue-router');
  return {
    ...actual,
    useRoute: () => routeState,
  };
});

vi.mock('@/web/web/composables/search/useUserFilters', async () => {
  const { reactive, toRef } = await import('vue');
  savedFiltersApi.state = reactive({
    favoriteMenuItems: [] as any[],
    loading: false,
    loadError: null as string | null,
    defaultsForOpen: [] as any[],
  });
  return {
    useUserFilters: (params: { scopeKey?: () => string }) => {
      savedFiltersApi.lastScopeKey = String(params.scopeKey?.() ?? '');
      return {
        favoriteMenuItems: toRef(savedFiltersApi.state!, 'favoriteMenuItems'),
        loading: toRef(savedFiltersApi.state!, 'loading'),
        loadError: toRef(savedFiltersApi.state!, 'loadError'),
        defaultsForOpen: toRef(savedFiltersApi.state!, 'defaultsForOpen'),
        load: savedFiltersApi.load,
        apply: savedFiltersApi.apply,
        saveCurrent: savedFiltersApi.saveCurrent,
        remove: savedFiltersApi.remove,
      };
    },
  };
});

vi.mock('element-plus', async () => {
  const actual = await vi.importActual<any>('element-plus');
  return {
    ...actual,
    ElMessage: { warning: vi.fn(), success: vi.fn(), error: vi.fn() },
    ElMessageBox: { confirm: vi.fn(async () => true) },
  };
});

const elementStubs = {
  'el-tooltip': { template: `<div><slot /></div>` },
  'el-button': {
    emits: ['click'],
    template: `<button type="button" class="el-btn" @click="$emit('click', $event)"><slot /></button>`,
  },
  'el-tag': true,
  'el-popover': {
    template: `<div class="el-popover"><slot name="reference" /><div class="pop"><slot /></div></div>`,
  },
  'el-dialog': {
    props: ['modelValue', 'title'],
    template: `<div v-if="modelValue" class="el-dialog"><slot /><slot name="footer" /></div>`,
  },
  'el-form': { template: `<form><slot /></form>` },
  'el-form-item': { template: `<div><slot /></div>` },
  'el-input': {
    props: ['modelValue'],
    emits: ['update:modelValue'],
    template: `<input class="fav-name" :value="modelValue" @input="$emit('update:modelValue', $event.target.value)" />`,
  },
  'el-checkbox': true,
  'el-divider': true,
  'el-icon': true,
  OSearchFilter: true,
  OSearchGroup: true,
};

function mountSearch() {
  return mount(OSearch as any, {
    props: {
      store: {
        storeId: 'demo.Widget',
        application: 'demo',
        modelName: 'Widget',
        fieldsMetadata: {},
        state: { queryState: {} },
      },
      placeholder: 'Find…',
    },
    global: { stubs: elementStubs },
  });
}

async function openSaveDialog(wrapper: ReturnType<typeof mountSearch>) {
  const saveOpen = wrapper.findAll('.el-btn').find(b => b.text().includes('Save current filters'));
  expect(saveOpen).toBeTruthy();
  await saveOpen!.trigger('click');
  await nextTick();
}

describe('OSearch default favorite name + scopeKey', () => {
  beforeEach(() => {
    breadcrumbState.breadcrumbStack = [];
    menuState.activeMenu = null;
    routeState.path = '/web/partners/42';
    routeState.meta = {};
    savedFiltersApi.lastScopeKey = '';
    savedFiltersApi.load.mockClear();
  });

  it('passes route path as scopeKey and prefills Name from breadcrumb src', async () => {
    breadcrumbState.breadcrumbStack = [
      { title: 'ignored', titleText: createTermReference('web', 'Partners', { scope: 'web/pages' }) },
    ];
    const wrapper = mountSearch();
    await flushPromises();
    expect(savedFiltersApi.lastScopeKey).toBe('/web/partners/42');
    await openSaveDialog(wrapper);
    expect((wrapper.find('input.fav-name').element as HTMLInputElement).value).toBe('Partners');
  });

  it('falls back to menu title when breadcrumb/route empty', async () => {
    menuState.activeMenu = { titleText: createTermReference('web', 'Menu Label', { scope: 'web/menu' }) };
    const wrapper = mountSearch();
    await flushPromises();
    await openSaveDialog(wrapper);
    expect((wrapper.find('input.fav-name').element as HTMLInputElement).value).toBe('Menu Label');
  });

  it('uses route meta pageTitle when higher sources are empty', async () => {
    routeState.meta = { pageTitle: 'Route Title' };
    const wrapper = mountSearch();
    await flushPromises();
    await openSaveDialog(wrapper);
    expect((wrapper.find('input.fav-name').element as HTMLInputElement).value).toBe('Route Title');
  });

  it('falls back to model identity when all titles empty', async () => {
    const wrapper = mountSearch();
    await flushPromises();
    await openSaveDialog(wrapper);
    expect((wrapper.find('input.fav-name').element as HTMLInputElement).value).toBe('demo.Widget');
  });
});
