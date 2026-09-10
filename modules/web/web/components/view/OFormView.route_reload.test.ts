// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Dense QJS subset of OFormView route-reload behavior. Uses the real form
 * controller with Browse + a reactive stub route (createFeStubRouter).
 */

import { createPinia, setActivePinia } from 'pinia';
import * as VueRouter from 'vue-router';
const createFeStubRouter = (VueRouter as any).createFeStubRouter;
import type { App } from 'vue';

import { flushPromises, fnRecorder, mountApp } from '@/web/web/__tests__/mountApp';
import OFormView from './OFormView.vue';

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
    fieldsMetadata: { Name: { type: 'varchar' } },
    state: { queryState: {}, result: undefined, selection: [], planCache: new Map() },
    setContext: () => {},
    getContext: () => ({}),
    withContext: async (_ctx: any, fn: any) => fn(),
    DefaultGet: async (seed: any) => ({ ...(seed || {}), Code: 'server' }),
    Browse,
  } as any;
}

describe('OFormView reloads when route identity changes', () => {
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
    const wrapper = mountApp(OFormView as any, {
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
        ElButton: true,
        ElIcon: true,
        ElForm: true,
        OBreadcrumb: true,
        OViewContainer: true,
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
});
