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
import ChoyButton from '@/web/web/components/button/ChoyButton.vue';
import ChoyActionButton from '@/web/web/components/button/ChoyActionButton.vue';
import ChoyListView from './ChoyListView.vue';
import ChoyPagination from './ChoyPagination.vue';
import ChoyTableHost from '@/web/web/components/internal/ChoyTableHost.vue';
import ChoyTableColumn from '@/web/web/components/table/ChoyTableColumn.vue';
import ListInlineEditScope from '@/web/web/components/view/ListInlineEditScope.vue';

function makeStore() {
  return {
    fullModelName: 'partner.Partner',
    storeId: 'list-create-' + Math.random().toString(36).slice(2),
    metadata: { fields: { Name: { type: 'varchar' } } },
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
    Search: fnRecorder(async () => []),
    Count: fnRecorder(async () => 0),
  } as any;
}

function findNewButton(el: HTMLElement) {
  return Array.from(el.querySelectorAll('button')).find(b => (b.textContent || '').includes('New'));
}

function stubListChrome() {
  // fe-qjs: nested vendor Button often drops native click; stub chrome buttons.
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
  stubSfc(ChoyActionButton as any, {
    name: 'ChoyActionButton',
    inheritAttrs: false,
    props: {
      variant: { type: String, default: 'default' },
      size: { type: String, default: 'default' },
      disabled: { type: Boolean, default: false },
      busy: { type: Boolean, default: false },
      type: { type: String, default: 'button' },
      actionId: { type: String, default: undefined },
      hasAction: { type: Function, default: undefined },
      requiresConnectivity: { type: Boolean, default: true },
    },
    emits: ['execute'],
    setup(props: any, { slots, emit, attrs }: any) {
      return () =>
        h(
          'button',
          {
            type: props.type || 'button',
            ...attrs,
            class: ['choy-action-btn', attrs.class],
            disabled: props.disabled || props.busy || undefined,
            onClick: () => emit('execute'),
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
  restoreSfc(ChoyActionButton as any);
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
        actionIds: { create: 'partner.action.partner_create' },
        hasAction: () => true,
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
        actionIds: { create: 'partner.action.partner_create' },
        hasAction: () => true,
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

  test('shows empty-state New only when header is hidden', async () => {
    restoreSfc(ChoyTableHost);
    stubSfc(ChoyTableHost, {
      name: 'ChoyTableHost',
      setup(_p: any, { slots }: any) {
        return () =>
          h('div', { 'data-stub': 'ChoyTableHost' }, [slots.default?.(), h('div', { class: 'empty-slot' }, slots.empty?.())]);
      },
    });
    const { plugins } = buildPageMountGlobal({
      route: { name: 'PartnerList', path: '/partner/partners', fullPath: '/partner/partners' },
      router: { push },
    });
    const { unmount, el, root } = mountApp(ChoyListView as any, {
      props: {
        store: makeStore(),
        createAction: '/partner/partners/new',
        actionIds: { create: 'partner.action.partner_create' },
        hasAction: () => true,
        showHeader: false,
        showActions: true,
        refreshAction: false,
        deleteAction: false,
        showPaginate: false,
      },
      plugins,
    });
    await flushPromises();
    expect(findNewButton(el)).toBeTruthy();
    expect(root).toBeTruthy();
    unmount();
  });

  test('hides New when createAction is set but actionIds.create is omitted', async () => {
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
    expect(findNewButton(el)).toBeFalsy();
    expect(root).toBeTruthy();
    unmount();
  });

  test('hides toolbar New when header is hidden but createAction is empty', async () => {
    restoreSfc(ChoyTableHost);
    stubSfc(ChoyTableHost, {
      name: 'ChoyTableHost',
      setup(_p: any, { slots }: any) {
        return () =>
          h('div', { 'data-stub': 'ChoyTableHost' }, [slots.default?.(), h('div', { class: 'empty-slot' }, slots.empty?.())]);
      },
    });
    const { plugins } = buildPageMountGlobal({
      route: { name: 'PartnerList', path: '/partner/partners', fullPath: '/partner/partners' },
      router: { push },
    });
    const { unmount, el, root } = mountApp(ChoyListView as any, {
      props: {
        store: makeStore(),
        createAction: '',
        showHeader: false,
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

  test('retrySurface reloads via loadData with forcedCondition', async () => {
    const { plugins } = buildPageMountGlobal({
      route: { name: 'PartnerList', path: '/partner/partners', fullPath: '/partner/partners' },
      router: { push },
    });
    const store = makeStore();
    const forcedCondition = ['ParentId', '=', 'p1'] as any;
    const { unmount, root } = mountApp(ChoyListView as any, {
      props: {
        store,
        createAction: '',
        showHeader: true,
        showActions: true,
        refreshAction: false,
        deleteAction: false,
        showPaginate: false,
        forcedCondition,
      },
      plugins,
    });
    await flushPromises();
    const searchesBefore = store.Search.calls.length;
    const exposed = root as { retrySurface?: () => Promise<void> };
    expect(typeof exposed.retrySurface).toBe('function');
    await exposed.retrySurface?.();
    await flushPromises();
    expect(store.Search.calls.length).toBeGreaterThan(searchesBefore);
    const lastArgs = store.Search.calls[store.Search.calls.length - 1] || [];
    const blob = JSON.stringify(lastArgs);
    expect(blob).toContain('ParentId');
    expect(blob).toContain('p1');
    unmount();
  });

  test('soft-fail retryPending and Cancel restore committed search payload', async () => {
    const { plugins } = buildPageMountGlobal({
      route: { name: 'PartnerList', path: '/partner/partners', fullPath: '/partner/partners' },
      router: { push },
    });
    const store = makeStore();
    const { unmount, root, setupState } = mountApp(ChoyListView as any, {
      props: {
        store,
        createAction: '',
        showHeader: false,
        showActions: false,
        refreshAction: false,
        deleteAction: false,
        showPaginate: false,
      },
      plugins,
    });
    await flushPromises();

    const state = setupState() as {
      onSurfaceRetry: () => Promise<void>;
      onSurfaceCancel: () => void;
      onSearch: (payload: { keyword?: string; appliedFilters: unknown[] }) => void;
      lastSearchPayload: { keyword?: string } | null;
      controller: {
        vm: { error: unknown; hasAppliedOnce?: boolean; result?: unknown; visibleNodes?: unknown[] };
        retryPending: () => Promise<unknown>;
        restoreCommitted: () => void;
      };
    };
    state.controller.vm.hasAppliedOnce = true;
    state.controller.vm.result = { kind: 'search', rows: [{ Id: '1' }] } as any;
    state.controller.vm.visibleNodes = [{ Id: '1' }] as any;
    store.state.queryState.keyword = 'committed';
    // Commit search via handler, then overwrite with a failed pending payload.
    state.onSearch({ keyword: 'committed', appliedFilters: [] });
    state.onSearch({ keyword: 'pending-fail', appliedFilters: [] });
    state.controller.vm.error = new Error('offline page');
    state.onSurfaceCancel();
    expect(state.controller.vm.error).toBeNull();
    expect(state.lastSearchPayload?.keyword).toBe('committed');

    // Soft-fail Retry uses retryPending when error is set.
    let retried = 0;
    state.controller.vm.error = new Error('still offline');
    state.controller.retryPending = async () => {
      retried += 1;
      state.controller.vm.error = null;
    };
    await state.onSurfaceRetry();
    expect(retried).toBe(1);

    // Fallback Cancel when restoreCommitted is missing.
    delete (state.controller as any).restoreCommitted;
    state.controller.vm.error = new Error('fallback');
    state.onSurfaceCancel();
    expect(state.controller.vm.error).toBeNull();
    expect(root).toBeTruthy();
    unmount();
  });
});
