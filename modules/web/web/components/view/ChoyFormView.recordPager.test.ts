// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import type { App } from 'vue';
import * as VueRouter from 'vue-router';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import FormView from './ChoyFormView.vue';
import { useRecordSetStore } from '@/web/web/stores/recordSetStore';

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

test('ChoyFormView shows record pager when a matching recordSet session is active', async () => {
  const pinia = createPinia();
  setActivePinia(pinia);
  useRecordSetStore().seed({
    key: 'demo.Widget|list',
    model: 'demo.Widget',
    condition: {},
    total: 3,
    index: 1,
    ids: ['a', 'b', 'c'],
    windowStart: 0,
  });

  const { router } = createFeStubRouter({
    route: { path: '/demo/widget/b', params: { id: 'b' }, query: {}, meta: {} },
  });

  const store = {
    modelName: 'demo.Widget',
    fullModelName: 'demo.Widget',
    storeId: 'demo.Widget',
    metadata: { fields: { Name: { type: 'varchar' } } },
    state: { queryState: {}, result: undefined, selection: [], planCache: new Map() },
    setContext: () => {},
    getContext: () => ({}),
    withContext: async (_ctx: any, fn: any) => fn(),
    Read: async () => ({ Id: 'b', Name: 'B' }),
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

  expect(mounted.q('[data-testid="choy-form-pager"]')).not.toBeNull();
  expect(mounted.q('[data-testid="choy-form-pager-current"]')?.textContent || '').toContain('2');
  mounted.unmount();
});
