// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import type { App } from 'vue';
import * as VueRouter from 'vue-router';

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { useConnectivityStore } from '@/web/web/stores/connectivityStore';
import FormView from './ChoyFormView.vue';

const createFeStubRouter = (VueRouter as any).createFeStubRouter as (overrides?: object) => {
  router: {
    push: (to: unknown) => Promise<unknown>;
    currentRoute: { value: { path: string; params?: Record<string, string> } };
  };
};

const NoopLoading = {
  install(app: App) {
    app.directive('loading', {
      mounted() {},
      updated() {},
      unmounted() {},
    });
  },
};

describe('ChoyFormView offline save gates', () => {
  afterEach(() => {
    try {
      useConnectivityStore().resetForTests();
    } catch {
      /* no pinia */
    }
  });

  test('save execute and native submit no-op while unreachable', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    useConnectivityStore().markUnreachable();

    const { router } = createFeStubRouter({
      route: { path: '/demo/widget/b', params: { id: 'b' }, query: {}, meta: {} },
    });
    let submits = 0;
    const store = {
      modelName: 'demo.Widget',
      fullModelName: 'demo.Widget',
      storeId: 'demo.Widget-offline-gate',
      metadata: { fields: { Name: { type: 'varchar' } } },
      state: { queryState: {}, result: undefined, selection: [], planCache: new Map() },
      setContext: () => {},
      getContext: () => ({}),
      withContext: async (_ctx: any, fn: any) => fn(),
      Read: async () => ({ Id: 'b', Name: 'B' }),
      Create: async () => {
        submits += 1;
        return { Id: 'n1' };
      },
      Write: async () => {
        submits += 1;
        return { Id: 'b' };
      },
    } as any;

    const mounted = mountApp(FormView as any, {
      props: {
        store,
        recordId: 'b',
        showHeader: true,
        showActions: true,
        showMessages: false,
        resolveRecordIdFromRoute: false,
      },
      plugins: [pinia, router as any, NoopLoading],
      stubs: { Breadcrumb: true },
    });
    await flushPromises();

    const ss = mounted.setupState() as {
      sessionUnreachable: boolean;
      onSaveExecute: () => unknown;
      onNativeSubmitGated: () => void;
      onSettingsSaveGated: () => unknown;
    };
    expect(ss.sessionUnreachable).toBe(true);
    await ss.onSaveExecute();
    ss.onNativeSubmitGated();
    await ss.onSettingsSaveGated();
    await flushPromises();
    expect(submits).toBe(0);

    mounted.unmount();
  });
});
