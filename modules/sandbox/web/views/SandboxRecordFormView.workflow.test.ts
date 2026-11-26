// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import { h } from 'vue';
import { useAuthStore } from '@/auth/web/stores/auth';
import { ChoyFormView, ChoyStatusbarButton } from '@/web';
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import { buildPageMountGlobal } from '@choysum/page-mount';
import SandboxRecordFormView from './SandboxRecordFormView.vue';
import { sandboxRecordActions } from './sandbox_record_actions';

function seedSandboxWrite() {
  const auth = useAuthStore();
  auth.identity = {
    metadata: { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] },
  } as any;
  auth.permissionState = {
    permStateVersion: 1,
    byCompany: {
      c1: { ui: { routes: [], menus: [], actions: [sandboxRecordActions.edit, '*'] } },
    },
  } as any;
}

function makeStore(state: string) {
  return {
    fullModelName: 'sandbox.SandboxRecord',
    storeId: 'sandbox-form-workflow-' + Math.random().toString(36).slice(2),
    metadata: { fields: { State: { type: 'selection' }, Progress: { type: 'selection' } } },
    state: {
      current: { Id: 'rec-1', State: state, Progress: state, NameChar: 'Sample' },
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
    Browse: async () => null,
    FieldsGet: async () => ({}),
    UpdateById: fnRecorder(async () => ({ Id: 'rec-1', State: 'open', Progress: 'open' })),
  } as any;
}

function stubFormChrome() {
  stubSfc(ChoyStatusbarButton as any, {
    name: 'ChoyStatusbarButton',
    inheritAttrs: false,
    emits: ['execute'],
    setup(_: unknown, { slots, attrs, emit }: any) {
      return () =>
        h('button', { type: 'button', ...attrs, onClick: () => emit('execute') }, slots.default?.());
    },
  });
  stubSfc(ChoyFormView as any, {
    name: 'ChoyFormView',
    props: ['store', 'mode', 'showHeader'],
    setup(props: any, { slots, expose }: any) {
      expose({
        getViewMode: () => 'display',
        reloadDisplay: async () => undefined,
        refresh: async () => undefined,
      });
      const formData = props.store?.state?.current || {};
      return () =>
        h('div', { 'data-stub': 'ChoyFormView' }, [
          slots['statusbar-buttons']?.({
            formData,
            recordId: formData.Id,
            loading: false,
          }),
          slots['statusbar-status']?.(),
        ]);
    },
  });
}

describe('SandboxRecordFormView workflow statusbar', () => {
  afterEach(() => {
    restoreSfc(ChoyFormView as any);
    restoreSfc(ChoyStatusbarButton as any);
  });

  test('Mark Open updates State/Progress and reloads display', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    seedSandboxWrite();
    stubFormChrome();
    const store = makeStore('draft');
    const { plugins } = buildPageMountGlobal({
      route: { name: 'SandboxForm', path: '/sandbox/records/rec-1', fullPath: '/sandbox/records/rec-1' },
    });
    const mounted = mountApp(SandboxRecordFormView as any, {
      props: { store, showHeader: false },
      plugins: [...plugins, pinia],
    });
    await flushPromises();
    expect(mounted.q('[data-testid=sandbox-mark-open]')).toBeTruthy();
    expect(mounted.q('[data-testid=sandbox-mark-done]')).toBeFalsy();
    mounted.click('[data-testid=sandbox-mark-open]');
    await flushPromises();
    expect(store.UpdateById.calls.length).toBe(1);
    expect(store.UpdateById.calls[0]?.[0]).toBe('rec-1');
    expect(store.UpdateById.calls[0]?.[1]).toEqual({ State: 'open', Progress: 'open' });
    mounted.unmount();
  });

  test('Mark Done and Reset Draft buttons follow workflow state', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    seedSandboxWrite();
    stubFormChrome();
    const { plugins } = buildPageMountGlobal({
      route: { name: 'SandboxForm', path: '/sandbox/records/rec-1', fullPath: '/sandbox/records/rec-1' },
    });

    const openStore = makeStore('open');
    const openMounted = mountApp(SandboxRecordFormView as any, {
      props: { store: openStore, showHeader: false },
      plugins: [...plugins, pinia],
    });
    await flushPromises();
    expect(openMounted.q('[data-testid=sandbox-mark-done]')).toBeTruthy();
    openMounted.click('[data-testid=sandbox-mark-done]');
    await flushPromises();
    expect(openStore.UpdateById.calls[0]?.[1]).toEqual({ State: 'done', Progress: 'done' });
    openMounted.unmount();

    const doneStore = makeStore('done');
    const doneMounted = mountApp(SandboxRecordFormView as any, {
      props: { store: doneStore, showHeader: false },
      plugins: [...plugins, pinia],
    });
    await flushPromises();
    expect(doneMounted.q('[data-testid=sandbox-reset-draft]')).toBeTruthy();
    doneMounted.click('[data-testid=sandbox-reset-draft]');
    await flushPromises();
    expect(doneStore.UpdateById.calls[0]?.[1]).toEqual({ State: 'draft', Progress: 'draft' });
    doneMounted.unmount();
  });

  test('markState no-ops when record id is empty', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    seedSandboxWrite();
    stubSfc(ChoyStatusbarButton as any, {
      name: 'ChoyStatusbarButton',
      inheritAttrs: false,
      emits: ['execute'],
      setup(_: unknown, { slots, attrs, emit }: any) {
        return () =>
          h('button', { type: 'button', ...attrs, onClick: () => emit('execute') }, slots.default?.());
      },
    });
    stubSfc(ChoyFormView as any, {
      name: 'ChoyFormView',
      props: ['store', 'mode', 'showHeader'],
      setup(props: any, { slots, expose }: any) {
        expose({
          getViewMode: () => 'display',
          reloadDisplay: async () => undefined,
        });
        return () =>
          h('div', { 'data-stub': 'ChoyFormView' }, [
            slots['statusbar-buttons']?.({
              formData: { ...props.store?.state?.current, State: 'draft' },
              recordId: '',
              loading: false,
            }),
          ]);
      },
    });
    const store = makeStore('draft');
    store.state.current.Id = '';
    const { plugins } = buildPageMountGlobal({
      route: { name: 'SandboxForm', path: '/sandbox/records/new', fullPath: '/sandbox/records/new' },
    });
    const mounted = mountApp(SandboxRecordFormView as any, {
      props: { store, showHeader: false },
      plugins: [...plugins, pinia],
    });
    await flushPromises();
    expect(mounted.q('[data-testid=sandbox-mark-open]')).toBeFalsy();
    expect(store.UpdateById.calls.length).toBe(0);
    mounted.unmount();
  });
});
