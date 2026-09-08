// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, nextTick, reactive, toRef } from 'vue';
import {
  ElButton,
  ElCheckbox,
  ElDialog,
  ElDivider,
  ElForm,
  ElFormItem,
  ElIcon,
  ElInput,
  ElPopover,
  ElTag,
  ElTooltip,
  ElTreeSelect,
} from 'element-plus';

import { createTermReference } from '@/core/service/i18n';
import { OSearchNavContextKey } from '@/web/web/composables/search/oSearchNavContext';
import { UseUserFiltersKey } from '@/web/web/composables/search/useUserFilters';
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import OSearch from './OSearch.vue';
import OSearchFilter from './OSearchFilter.vue';

const savedFiltersApi = {
  state: null as null | {
    favoriteMenuItems: any[];
    loading: boolean;
    loadError: string | null;
    defaultsForOpen: any[];
  },
  load: fnRecorder(async () => {}),
  apply: fnRecorder(),
  saveCurrent: fnRecorder(async () => ({ Id: '1' })),
  remove: fnRecorder(async () => {}),
  lastScopeKey: '' as string,
};

const breadcrumbState = {
  breadcrumbStack: [] as Array<{ title?: string; titleText?: any }>,
};
const menuState = {
  activeMenu: null as null | { title?: string; titleText?: any },
};
const routeState = {
  path: '/web/partners/42',
  meta: {} as Record<string, unknown>,
};

const epSfcs = [
  ElButton,
  ElTag,
  ElTooltip,
  ElDialog,
  ElDivider,
  ElIcon,
  ElPopover,
  ElTreeSelect,
  ElForm,
  ElFormItem,
  ElInput,
  ElCheckbox,
];

function installEpStubs() {
  stubSfc(ElButton as any, {
    name: 'ElButton',
    emits: ['click'],
    setup(_, { slots, emit }: any) {
      return () =>
        h('button', { type: 'button', class: 'el-btn', onClick: (e: any) => emit('click', e) }, slots.default?.());
    },
  });
  stubSfc(ElTag as any, { name: 'ElTag', setup: () => () => h('span', { class: 'el-tag' }) });
  stubSfc(ElTooltip as any, {
    name: 'ElTooltip',
    setup(_, { slots }: any) {
      return () => h('div', {}, slots.default?.());
    },
  });
  stubSfc(ElPopover as any, {
    name: 'ElPopover',
    setup(_, { slots }: any) {
      return () =>
        h('div', { class: 'el-popover' }, [slots.reference?.(), h('div', { class: 'pop' }, slots.default?.())]);
    },
  });
  stubSfc(ElDialog as any, {
    name: 'ElDialog',
    props: ['modelValue', 'title'],
    setup(props: any, { slots }: any) {
      return () =>
        props.modelValue
          ? h('div', { class: 'el-dialog' }, [slots.default?.(), slots.footer?.()])
          : null;
    },
  });
  stubSfc(ElForm as any, {
    name: 'ElForm',
    setup(_, { slots }: any) {
      return () => h('form', {}, slots.default?.());
    },
  });
  stubSfc(ElFormItem as any, {
    name: 'ElFormItem',
    setup(_, { slots }: any) {
      return () => h('div', {}, slots.default?.());
    },
  });
  stubSfc(ElInput as any, {
    name: 'ElInput',
    props: ['modelValue'],
    emits: ['update:modelValue'],
    setup(props: any, { emit }: any) {
      return () =>
        h('input', {
          class: 'fav-name',
          value: props.modelValue,
          onInput: (e: any) => emit('update:modelValue', e.target.value),
        });
    },
  });
  stubSfc(ElCheckbox as any, { name: 'ElCheckbox', setup: () => () => h('label', { class: 'fav-check' }) });
  stubSfc(ElDivider as any, { name: 'ElDivider', setup: () => () => h('hr') });
  stubSfc(ElIcon as any, {
    name: 'ElIcon',
    setup(_, { slots }: any) {
      return () => h('i', {}, slots.default?.());
    },
  });
  stubSfc(ElTreeSelect as any, { name: 'ElTreeSelect', setup: () => () => h('div', { class: 'tree' }) });
  stubSfc(OSearchFilter as any, {
    name: 'OSearchFilter',
    setup: () => () => h('div', { 'data-stub': 'OSearchFilter' }),
  });
}

function restoreEpStubs() {
  for (const Comp of epSfcs) restoreSfc(Comp as any);
  restoreSfc(OSearchFilter as any);
}

function makeUserFiltersFactory() {
  savedFiltersApi.state = reactive({
    favoriteMenuItems: [] as any[],
    loading: false,
    loadError: null as string | null,
    defaultsForOpen: [] as any[],
  });
  return (params: { scopeKey?: () => string }) => {
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
      updateMeta: fnRecorder(async () => {}),
    };
  };
}

function mountSearch() {
  installEpStubs();
  return mountApp(OSearch as any, {
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
    provide: {
      [OSearchNavContextKey as symbol]: {
        breadcrumbStore: breadcrumbState,
        menuStore: menuState,
        route: routeState,
      },
      [UseUserFiltersKey as symbol]: makeUserFiltersFactory(),
    },
  });
}

async function openSaveDialog(m: ReturnType<typeof mountSearch>) {
  const saveOpen = m.qa('.el-btn').find(b => (b.textContent || '').includes('Save current filters'));
  expect(saveOpen).toBeTruthy();
  (saveOpen as HTMLElement).click();
  await nextTick();
}

describe('OSearch default favorite name + scopeKey', () => {
  afterEach(() => {
    restoreEpStubs();
  });

  beforeEach(() => {
    breadcrumbState.breadcrumbStack = [];
    menuState.activeMenu = null;
    routeState.path = '/web/partners/42';
    routeState.meta = {};
    savedFiltersApi.lastScopeKey = '';
    savedFiltersApi.load.mockClear();
  });

  test('passes route path as scopeKey and prefills Name from breadcrumb src', async () => {
    breadcrumbState.breadcrumbStack = [
      { title: 'ignored', titleText: createTermReference('web', 'Partners', { scope: 'web/pages' }) },
    ];
    const m = mountSearch();
    await flushPromises();
    expect(savedFiltersApi.lastScopeKey).toBe('/web/partners/42');
    await openSaveDialog(m);
    expect((m.q('input.fav-name') as HTMLInputElement).value).toBe('Partners');
    m.unmount();
  });

  test('falls back to menu title when breadcrumb/route empty', async () => {
    menuState.activeMenu = { titleText: createTermReference('web', 'Menu Label', { scope: 'web/menu' }) };
    const m = mountSearch();
    await flushPromises();
    await openSaveDialog(m);
    expect((m.q('input.fav-name') as HTMLInputElement).value).toBe('Menu Label');
    m.unmount();
  });

  test('uses route meta pageTitle when higher sources are empty', async () => {
    routeState.meta = { pageTitle: 'Route Title' };
    const m = mountSearch();
    await flushPromises();
    await openSaveDialog(m);
    expect((m.q('input.fav-name') as HTMLInputElement).value).toBe('Route Title');
    m.unmount();
  });

  test('falls back to model identity when all titles empty', async () => {
    const m = mountSearch();
    await flushPromises();
    await openSaveDialog(m);
    expect((m.q('input.fav-name') as HTMLInputElement).value).toBe('demo.Widget');
    m.unmount();
  });
});
