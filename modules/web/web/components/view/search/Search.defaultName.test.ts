// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, nextTick, reactive, toRef } from 'vue';

import { createTermReference } from '@/core/service/i18n';
import { SearchNavContextKey } from '@/web/web/composables/search/searchNavContext';
import { UseUserFiltersKey } from '@/web/web/composables/search/useUserFilters';
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import Search from './Search.vue';
import SearchFilter from './SearchFilter.vue';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import Dialog from '@/web/web/components/vendor/ui/dialog/Dialog.vue';
import DialogContent from '@/web/web/components/vendor/ui/dialog/DialogContent.vue';
import DialogTitle from '@/web/web/components/vendor/ui/dialog/DialogTitle.vue';
import Popover from '@/web/web/components/vendor/ui/popover/Popover.vue';
import PopoverContent from '@/web/web/components/vendor/ui/popover/PopoverContent.vue';
import PopoverTrigger from '@/web/web/components/vendor/ui/popover/PopoverTrigger.vue';

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

const stubbed = [
  ChoyButton,
  Dialog,
  DialogContent,
  DialogTitle,
  Popover,
  PopoverContent,
  PopoverTrigger,
  SearchFilter,
];

function installEpStubs() {
  stubSfc(ChoyButton as any, {
    name: 'ChoyButton',
    inheritAttrs: false,
    props: {
      variant: { type: String, default: 'default' },
      size: { type: String, default: 'default' },
      disabled: { type: Boolean, default: false },
      type: { type: String, default: 'button' },
    },
    emits: ['click'],
    setup(props: any, { slots, emit, attrs }: any) {
      return () =>
        h(
          'button',
          {
            type: props.type || 'button',
            ...attrs,
            class: ['choy-btn', attrs.class],
            disabled: props.disabled || undefined,
            onClick: (e: any) => emit('click', e),
          },
          slots.default?.(),
        );
    },
  });
  stubSfc(Dialog as any, {
    name: 'Dialog',
    props: { open: { type: Boolean, default: false } },
    emits: ['update:open'],
    setup(props: any, { slots }: any) {
      return () => (props.open ? h('div', { class: 'choy-dialog', role: 'dialog' }, slots.default?.()) : null);
    },
  });
  stubSfc(DialogContent as any, {
    name: 'DialogContent',
    setup(_: any, { slots }: any) {
      return () => h('div', { class: 'dialog-content' }, slots.default?.());
    },
  });
  stubSfc(DialogTitle as any, {
    name: 'DialogTitle',
    setup(_: any, { slots }: any) {
      return () => h('div', { class: 'dialog-title' }, slots.default?.());
    },
  });
  stubSfc(Popover as any, {
    name: 'Popover',
    setup(_: any, { slots }: any) {
      return () => h('div', { class: 'el-popover' }, slots.default?.());
    },
  });
  stubSfc(PopoverTrigger as any, {
    name: 'PopoverTrigger',
    setup(_: any, { slots }: any) {
      return () => h('div', { class: 'popover-trigger' }, slots.default?.());
    },
  });
  stubSfc(PopoverContent as any, {
    name: 'PopoverContent',
    setup(_: any, { slots }: any) {
      return () => h('div', { class: 'pop' }, slots.default?.());
    },
  });
  stubSfc(SearchFilter as any, {
    name: 'SearchFilter',
    setup: () => () => h('div', { 'data-stub': 'SearchFilter' }),
  });
}

function restoreEpStubs() {
  for (const Comp of stubbed) restoreSfc(Comp as any);
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
  return mountApp(Search as any, {
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
      [SearchNavContextKey as symbol]: {
        breadcrumbStore: breadcrumbState,
        menuStore: menuState,
        route: routeState,
      },
      [UseUserFiltersKey as symbol]: makeUserFiltersFactory(),
    },
  });
}

async function openSaveDialog(m: ReturnType<typeof mountSearch>) {
  const saveOpen = m.qa('button').find(b => (b.textContent || '').includes('Save current filters'));
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
    const nameInput = (m.q('[role="dialog"]') as HTMLElement | null)?.querySelector('input') as HTMLInputElement | null;
    expect(nameInput?.value).toBe('Partners');
    m.unmount();
  });

  test('falls back to menu title when breadcrumb/route empty', async () => {
    menuState.activeMenu = { titleText: createTermReference('web', 'Menu Label', { scope: 'web/menu' }) };
    const m = mountSearch();
    await flushPromises();
    await openSaveDialog(m);
    const nameInput = (m.q('[role="dialog"]') as HTMLElement | null)?.querySelector('input') as HTMLInputElement | null;
    expect(nameInput?.value).toBe('Menu Label');
    m.unmount();
  });

  test('uses route meta pageTitle when higher sources are empty', async () => {
    routeState.meta = { pageTitle: 'Route Title' };
    const m = mountSearch();
    await flushPromises();
    await openSaveDialog(m);
    const nameInput = (m.q('[role="dialog"]') as HTMLElement | null)?.querySelector('input') as HTMLInputElement | null;
    expect(nameInput?.value).toBe('Route Title');
    m.unmount();
  });

  test('falls back to model identity when all titles empty', async () => {
    const m = mountSearch();
    await flushPromises();
    await openSaveDialog(m);
    const nameInput = (m.q('[role="dialog"]') as HTMLElement | null)?.querySelector('input') as HTMLInputElement | null;
    expect(nameInput?.value).toBe('demo.Widget');
    m.unmount();
  });
});
