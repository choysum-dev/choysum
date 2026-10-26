// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * ListView create-action under QJS.
 * Prefer New-button clicks when painted; otherwise keep progressive density on
 * empty createAction + mounted expose surface.
 */

import { h } from 'vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import ChoyListView from './ChoyListView.vue';
import ChoyPagination from './ChoyPagination.vue';
import ChoyTableHost from '@/web/web/components/internal/ChoyTableHost.vue';
import ChoyTableColumn from '@/web/web/components/table/ChoyTableColumn.vue';
import ListInlineEditScope from '@/web/web/components/view/ListInlineEditScope.vue';

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

function findNewButton(el: HTMLElement) {
  return Array.from(el.querySelectorAll('button')).find(b => (b.textContent || '').includes('New'));
}

function stubListChrome() {
  // fe-qjs: nested vendor Button often drops native click; stub ChoyButton like Search tests.
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
            onClick: (e: Event) => emit('click', e),
          },
          slots.default?.(),
        );
    },
  });
  stubSfc(ChoyTableHost, {
    name: 'ChoyTableHost',
    setup(_p: any, { slots }: any) {
      return () => h('div', { 'data-stub': 'ChoyTableHost' }, slots.default?.());
    },
  });
  stubSfc(ChoyTableColumn, {
    name: 'ChoyTableColumn',
    setup: () => () => null,
  });
  stubSfc(ChoyPagination, {
    name: 'ChoyPagination',
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
  restoreSfc(ChoyButton as any);
  restoreSfc(ChoyTableHost);
  restoreSfc(ChoyTableColumn);
  restoreSfc(ChoyPagination);
  restoreSfc(ListInlineEditScope);
}

describe('ListView create action', () => {
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
    const { unmount, el, root } = mountApp(ChoyListView as any, {
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
    });
    await flushPromises();

    const newBtn = findNewButton(el);
    expect(newBtn).toBeTruthy();
    (newBtn as HTMLElement).click();
    await flushPromises();
    expect(push.calls[0]?.[0]).toBe('/partner/partners/new');
    expect(root).toBeTruthy();
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
    const { unmount, el, root } = mountApp(ChoyListView as any, {
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
    });
    await flushPromises();

    const newBtn = findNewButton(el);
    expect(newBtn).toBeTruthy();
    (newBtn as HTMLElement).click();
    await flushPromises();
    expect((onActionError.calls[0]?.[0] as any)?.action).toBe('create');
    expect((onActionError.calls[0]?.[0] as any)?.error?.message).toBe('nav failed');
    expect(root).toBeTruthy();
    unmount();
  });

  test('hides New when createAction resolves to empty', async () => {
    const { plugins } = buildPageMountGlobal({
      route: { name: 'PartnerList', path: '/partner/partners', fullPath: '/partner/partners' },
      router: { push },
    });
    const { unmount, el, root } = mountApp(ChoyListView as any, {
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
    });
    await flushPromises();
    expect(findNewButton(el)).toBeFalsy();
    expect(root).toBeTruthy();
    expect(push.calls.length).toBe(0);
    unmount();
  });
});
