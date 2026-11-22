// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import * as VueRouter from 'vue-router';
import { flushPromises, mountApp, stubSfc, restoreSfc } from '@/web/web/__tests__/mountApp';
import SandboxRecordListView from './SandboxRecordListView.vue';
import { ChoyListView } from '@/web';

const createFeStubRouter = (VueRouter as any).createFeStubRouter as (overrides?: object) => {
  router: { push: (to: unknown) => Promise<unknown>; currentRoute: { value: { path: string } } };
};

test('SandboxRecordListView wires detailPath onto ChoyListView', async () => {
  const pinia = createPinia();
  setActivePinia(pinia);
  const { router } = createFeStubRouter({
    route: {
      path: '/sandbox/records',
      fullPath: '/sandbox/records',
      meta: {
        viewKind: 'list',
        viewPaths: {
          list: '/sandbox/records',
          kanban: '/sandbox/records/kanban',
          chart: '/sandbox/records/chart',
        },
      },
    },
  });
  const seen: { detailPath?: unknown } = {};
  stubSfc(ChoyListView as any, {
    name: 'ChoyListView',
    props: ['store', 'detailPath', 'searchView', 'actionIds', 'hasAction', 'showHeader'],
    setup(props: any) {
      seen.detailPath = props.detailPath;
      return () => h('div', { 'data-stub': 'ChoyListView' });
    },
  });

  const store = {
    modelName: 'sandbox.Record',
    storeId: 'sandbox-list-detail',
    state: { queryState: {}, result: { rows: [], total: 0 }, selection: [] },
    metadata: { fields: {} },
  };
  const mounted = mountApp(SandboxRecordListView as any, {
    props: { store, showHeader: false },
    plugins: [pinia, router as any],
  });
  await flushPromises();

  expect(seen.detailPath).toBe('/sandbox/records/:id');
  mounted.unmount();
  restoreSfc(ChoyListView as any);
});
