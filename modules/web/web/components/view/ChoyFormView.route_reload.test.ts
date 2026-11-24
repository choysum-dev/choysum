// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Dense QJS subset of FormView route-reload behavior. Uses the real form
 * controller with Browse + a reactive stub route (createFeStubRouter).
 */

import { createPinia, setActivePinia } from 'pinia';
import * as VueRouter from 'vue-router';
const createFeStubRouter = (VueRouter as any).createFeStubRouter;
import type { App } from 'vue';

import { flushPromises, fnRecorder, mountApp } from '@/web/web/__tests__/mountApp';
import FormView from './ChoyFormView.vue';

const NoopLoading = {
  install(app: App) {
    app.directive('loading', {
      mounted() {},
      updated() {},
      unmounted() {},
    });
  },
};

function fakeStore(browseImpl?: (id: string) => Promise<any>) {
  const Browse = fnRecorder(async (id: string) => {
    if (browseImpl) return browseImpl(id);
    return { Id: id, Name: 'row-' + id };
  });
  return {
    fullModelName: 'demo.Widget',
    storeId: 'demo.Widget-' + Math.random().toString(36).slice(2),
    metadata: { fields: { Name: { type: 'varchar' } } },
    state: { queryState: {}, result: undefined, selection: [], planCache: new Map() },
    setContext: () => {},
    getContext: () => ({}),
    withContext: async (_ctx: any, fn: any) => fn(),
    DefaultGet: async (seed: any) => ({ ...(seed || {}), Code: 'server' }),
    Browse,
  } as any;
}

describe('FormView reloads when route identity changes', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
  });

  function mountForm(opts?: {
    route?: Record<string, unknown>;
    props?: Record<string, unknown>;
    store?: any;
    on?: Record<string, (...args: any[]) => void>;
  }) {
    const { router, route } = createFeStubRouter({
      route: {
        name: 'WidgetDetail',
        path: '/demo/widgets/1',
        fullPath: '/demo/widgets/1',
        params: { id: '1' },
        query: {},
        meta: {},
        ...(opts?.route || {}),
      },
    });
    const store = opts?.store || fakeStore();
    const wrapper = mountApp(FormView as any, {
      props: {
        store,
        showHeader: true,
        showActions: true,
        showMessages: false,
        ...(opts?.props || {}),
      },
      plugins: [createPinia(), router, NoopLoading],
      on: opts?.on,
      stubs: {
        Breadcrumb: true,
        ChoyViewContainer: true,
      },
    });
    return { wrapper, route, store };
  }

  test('reloads display when route params.id changes', async () => {
    const { wrapper, route, store } = mountForm();
    await flushPromises();
    expect(store.Browse.calls.map((c: any[]) => c[0])).toContain('1');

    route.params = { id: '2' };
    route.path = '/demo/widgets/2';
    route.fullPath = '/demo/widgets/2';
    await flushPromises();

    const ids = store.Browse.calls.map((c: any[]) => c[0]);
    expect(ids).toContain('1');
    expect(ids).toContain('2');
    expect(wrapper.setupState().controller.vm.draft).toMatchObject({ Id: '2' });
    wrapper.unmount();
  });

  test('retrySurface reloads display for the current record', async () => {
    const { wrapper, store } = mountForm();
    await flushPromises();
    const before = store.Browse.calls.length;
    const exposed = wrapper.root as { retrySurface?: () => Promise<void> };
    expect(typeof exposed.retrySurface).toBe('function');
    await exposed.retrySurface?.();
    await flushPromises();
    expect(store.Browse.calls.length).toBeGreaterThan(before);
    wrapper.unmount();
  });

  test('surface retry restores edit after failed reload; cancel clears error and emits', async () => {
    const store = fakeStore();
    const surfaceCancel = fnRecorder(() => {});
    const { wrapper } = mountForm({
      store,
      props: {
        recordId: '1',
        viewMode: 'edit',
        resolveRecordIdFromRoute: false,
        embedded: true,
        initialValues: { Id: '1', Name: 'seed' },
      },
      // Vue 3 listener prop for emit('surface-cancel').
      on: { onSurfaceCancel: surfaceCancel },
    });
    await flushPromises();

    const exposed = wrapper.root as {
      retrySurface?: () => Promise<void>;
      cancelSurface?: () => void;
    };
    const state = wrapper.setupState() as {
      controller: {
        vm: { mode: string; error: unknown; original: any };
        beginDisplay: (id: string) => Promise<void>;
        beginEdit: () => void;
      };
      isEmbedded: boolean;
    };
    const vm = state.controller.vm;
    expect(state.isEmbedded).toBe(true);

    // Soft-fail without rejecting out of the test host (QuickJS treats unhandled rejects as fatal).
    state.controller.beginDisplay = async () => {
      vm.error = new Error('offline browse');
      vm.mode = 'display';
    };
    await exposed.retrySurface?.();
    await flushPromises();
    expect(vm.error instanceof Error && vm.error.message).toBe('offline browse');
    expect(vm.mode).toBe('edit');
    expect(vm.original).toBeTruthy();

    exposed.cancelSurface?.();
    expect(vm.error).toBeNull();
    expect(surfaceCancel.calls.length).toBe(1);
    wrapper.unmount();
  });

  test('does not follow route when resolveRecordIdFromRoute is false', async () => {
    const store = fakeStore();
    const { wrapper, route } = mountForm({
      props: { recordId: 'fixed-1', resolveRecordIdFromRoute: false },
      store,
    });
    await flushPromises();
    expect(store.Browse.calls.map((c: any[]) => c[0])).toContain('fixed-1');
    const afterMount = store.Browse.calls.length;

    route.params = { id: '9' };
    route.fullPath = '/demo/widgets/9';
    await flushPromises();

    expect(store.Browse.calls.length).toBe(afterMount);
    wrapper.unmount();
  });

  test('switches to create when navigating to a Create route without id', async () => {
    const { wrapper, route } = mountForm();
    await flushPromises();

    route.name = 'WidgetCreate';
    route.params = {};
    route.path = '/demo/widgets/new';
    route.fullPath = '/demo/widgets/new';
    await flushPromises();

    expect(wrapper.setupState().controller.vm.mode).toBe('create');
    wrapper.unmount();
  });

  test('resolves effective id from route.params.recordId', async () => {
    const store = fakeStore();
    const { wrapper } = mountForm({
      route: {
        name: 'WidgetDetail',
        path: '/demo/widgets/r1',
        fullPath: '/demo/widgets/r1',
        params: { recordId: 'rec-1' },
        query: {},
      },
      store,
    });
    await flushPromises();
    expect(store.Browse.calls.map((c: any[]) => c[0])).toContain('rec-1');
    wrapper.unmount();
  });

  test('keeps display mode for initialValues-only preview', async () => {
    const { wrapper } = mountForm({
      props: {
        recordId: undefined,
        resolveRecordIdFromRoute: false,
        viewMode: 'display',
        initialValues: { Name: 'preview' },
      },
      route: {
        name: 'WidgetDetail',
        path: '/demo/widgets/preview',
        fullPath: '/demo/widgets/preview',
        params: {},
        query: {},
      },
    });
    await flushPromises();
    expect(wrapper.setupState().controller.vm.mode).toBe('display');
    wrapper.unmount();
  });

  test('reloadDisplay no-ops while editing so unsaved drafts are kept', async () => {
    const store = fakeStore();
    const { wrapper } = mountForm({
      props: { recordId: 'edit-1', resolveRecordIdFromRoute: false, viewMode: 'edit' },
      store,
    });
    await flushPromises();
    const ss = wrapper.setupState();
    ss.controller.vm.mode = 'edit';
    ss.controller.vm.draft = { Id: 'edit-1', Name: 'dirty' };
    const before = store.Browse.calls.length;
    await wrapper.root.reloadDisplay();
    await flushPromises();
    expect(store.Browse.calls.length).toBe(before);
    expect(ss.controller.vm.draft).toMatchObject({ Name: 'dirty' });
    wrapper.unmount();
  });
});
