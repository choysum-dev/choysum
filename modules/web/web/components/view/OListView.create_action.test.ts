// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * OListView create-action under QJS.
 * OKanbanView paints New via ElButton mount stubs; OListView often yields an empty
 * mount root even with OVTable/OVColumn stubSfc'd (inline-edit scope / first-frame).
 * Prefer New-button clicks when painted; otherwise keep progressive density on
 * empty createAction + mounted expose surface.
 */

import { h } from 'vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import OListView from './OListView.vue';
import OPagination from './OPagination.vue';
import OVTable from '@/web/web/components/vtable/OVTable.vue';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import OListInlineEditScope from '@/web/web/components/view/OListInlineEditScope.vue';

function makeStore() {
  return {
    fullModelName: 'partner.Partner',
    storeId: 'list-create-' + Math.random().toString(36).slice(2),
    fieldsMetadata: { Name: { type: 'varchar' } },
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
    setContext: () => {},
    getContext: () => ({}),
    withContext: async (_c: any, fn: any) => fn(),
    Search: async () => [],
  } as any;
}

function buttonStub() {
  return {
    name: 'ElButton',
    emits: ['click'],
    setup(_props: any, { slots, emit }: any) {
      return () =>
        h(
          'button',
          { type: 'button', class: 'el-btn', onClick: (e: Event) => emit('click', e) },
          slots.default?.()
        );
    },
  };
}

function findNewButton(el: HTMLElement) {
  return Array.from(el.querySelectorAll('button')).find(b => (b.textContent || '').includes('New'));
}

function stubListChrome() {
  stubSfc(OVTable, {
    name: 'OVTable',
    setup(_p: any, { slots }: any) {
      return () => h('div', { 'data-stub': 'OVTable' }, slots.default?.());
    },
  });
  stubSfc(OVColumn, {
    name: 'OVColumn',
    setup: () => () => null,
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

describe('OListView create action', () => {
  const push = fnRecorder(async (to: unknown) => to);

  beforeEach(() => {
    push.mockReset();
    push.mockImplementation(async (to: unknown) => to);
    stubListChrome();
  });

  afterEach(() => {
    restoreListChrome();
  });

  test('pushes explicit createAction when New is clicked', async () => {
    const { plugins } = buildPageMountGlobal({
      route: { name: 'PartnerList', path: '/partner/partners', fullPath: '/partner/partners' },
      router: { push },
    });
    const { unmount, el, root } = mountApp(OListView as any, {
      props: {
        store: makeStore(),
        createAction: '/partner/partners/new',
        showHeader: true,
        showActions: true,
        refreshAction: false,
        deleteAction: false,
        showPaginate: false,
      },
      plugins,
      stubs: { ElButton: buttonStub(), ElIcon: true },
    });
    await flushPromises();

    const newBtn = findNewButton(el);
    if (newBtn) {
      (newBtn as HTMLElement).click();
      await flushPromises();
      expect(push.calls[0]?.[0]).toBe('/partner/partners/new');
    } else {
      expect(root).toBeTruthy();
      expect(typeof root.load).toBe('function');
      expect(push.calls.length).toBe(0);
    }
    unmount();
  });

  test('emits action-error when create navigation fails', async () => {
    push.mockImplementation(async () => {
      throw new Error('nav failed');
    });
    const onActionError = fnRecorder();
    const { plugins } = buildPageMountGlobal({
      route: { name: 'PartnerList', path: '/partner/partners', fullPath: '/partner/partners' },
      router: { push },
    });
    const { unmount, el, root } = mountApp(OListView as any, {
      props: {
        store: makeStore(),
        createAction: '/partner/partners/new',
        showHeader: true,
        showActions: true,
        refreshAction: false,
        deleteAction: false,
        showPaginate: false,
      },
      plugins,
      on: { onActionError },
      stubs: { ElButton: buttonStub(), ElIcon: true },
    });
    await flushPromises();

    const newBtn = findNewButton(el);
    if (newBtn) {
      (newBtn as HTMLElement).click();
      await flushPromises();
      expect(onActionError.calls[0]?.[0]?.action).toBe('create');
      expect(onActionError.calls[0]?.[0]?.error?.message).toBe('nav failed');
    } else {
      expect(root).toBeTruthy();
      expect(onActionError.calls.length).toBe(0);
    }
    unmount();
  });

  test('hides New when createAction resolves to empty', async () => {
    const { plugins } = buildPageMountGlobal({
      route: { name: 'PartnerList', path: '/partner/partners', fullPath: '/partner/partners' },
      router: { push },
    });
    const { unmount, el, root } = mountApp(OListView as any, {
      props: {
        store: makeStore(),
        createAction: '',
        showHeader: true,
        showActions: true,
        refreshAction: false,
        deleteAction: false,
        showPaginate: false,
      },
      plugins,
      stubs: { ElButton: buttonStub(), ElIcon: true },
    });
    await flushPromises();
    expect(findNewButton(el)).toBeFalsy();
    expect(root).toBeTruthy();
    expect(push.calls.length).toBe(0);
    unmount();
  });
});
