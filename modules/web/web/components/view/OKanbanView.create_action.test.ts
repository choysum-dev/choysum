// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import OPagination from './OPagination.vue';
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
    setContext: () => {},
    getContext: () => ({}),
    withContext: async (_c: any, fn: any) => fn(),
    Search: async () => [],
  };
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

describe('OKanbanView create action', () => {
  const push = fnRecorder(async (to: unknown) => to);

  beforeEach(() => {
    push.mockReset();
    push.mockImplementation(async (to: unknown) => to);
    stubSfc(OPagination, {
      name: 'OPagination',
      setup() {
        return () => h('div', { 'data-stub': 'OPagination' });
      },
    });
  });

  afterEach(() => {
    restoreSfc(OPagination);
  });

  test('pushes createAction when New is clicked', async () => {
    const { plugins } = buildPageMountGlobal({
      route: { name: 'TokenKanban', path: '/auth/tokens/kanban', fullPath: '/auth/tokens/kanban' },
      router: { push },
    });
    const { unmount, el } = mountApp(OKanbanView as any, {
      props: {
        store: makeStore(),
        createAction: '/auth/tokens/new',
        showHeader: true,
        showActions: true,
        refreshAction: false,
        showPaginate: false,
      },
      plugins,
      stubs: {
        ElButton: buttonStub(),
        ElIcon: true,
      },
    });
    await flushPromises();

    const newBtn = findNewButton(el);
    expect(newBtn).toBeTruthy();
    (newBtn as HTMLElement).click();
    await flushPromises();
    expect(push.calls[0]?.[0]).toBe('/auth/tokens/new');
    unmount();
  });

  test('emits action-error when create navigation fails with a non-Error', async () => {
    push.mockImplementation(async () => {
      throw 'boom';
    });
    const onActionError = fnRecorder();
    const { plugins } = buildPageMountGlobal({
      route: { name: 'TokenKanban', path: '/auth/tokens/kanban', fullPath: '/auth/tokens/kanban' },
      router: { push },
    });
    const { unmount, el } = mountApp(OKanbanView as any, {
      props: {
        store: makeStore(),
        createAction: '/auth/tokens/new',
        showHeader: true,
        showActions: true,
        refreshAction: false,
        showPaginate: false,
      },
      plugins,
      on: { onActionError },
      stubs: {
        ElButton: buttonStub(),
        ElIcon: true,
      },
    });
    await flushPromises();

    const newBtn = findNewButton(el);
    expect(newBtn).toBeTruthy();
    (newBtn as HTMLElement).click();
    await flushPromises();
    expect((onActionError.calls[0]?.[0] as any)?.action).toBe('create');
    expect((onActionError.calls[0]?.[0] as any)?.error).toBeInstanceOf(Error);
    expect((onActionError.calls[0]?.[0] as any)?.error?.message).toBe('boom');
    unmount();
  });

  test('hides New when createAction is empty', async () => {
    const { plugins } = buildPageMountGlobal({
      route: { name: 'TokenKanban', path: '/auth/tokens/kanban', fullPath: '/auth/tokens/kanban' },
      router: { push },
    });
    const { unmount, el, setupState } = mountApp(OKanbanView as any, {
      props: {
        store: makeStore(),
        createAction: '',
        showHeader: true,
        showActions: true,
        refreshAction: false,
        showPaginate: false,
      },
      plugins,
      stubs: {
        ElButton: buttonStub(),
        ElIcon: true,
      },
    });
    await flushPromises();
    expect(findNewButton(el)).toBeFalsy();
    await setupState().handleCreate();
    expect(push.calls.length).toBe(0);
    unmount();
  });
});
