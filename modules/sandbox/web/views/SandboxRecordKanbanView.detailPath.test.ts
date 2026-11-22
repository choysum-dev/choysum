// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import * as VueRouter from 'vue-router';
import { flushPromises, mountApp, stubSfc, restoreSfc } from '@/web/web/__tests__/mountApp';
import SandboxRecordKanbanView from './SandboxRecordKanbanView.vue';
import { ChoyKanbanView } from '@/web';

const createFeStubRouter = (VueRouter as any).createFeStubRouter as (overrides?: object) => {
  router: { push: (to: unknown) => Promise<unknown>; currentRoute: { value: { path: string } } };
};

test('SandboxRecordKanbanView wires detailPath onto ChoyKanbanView', async () => {
  const pinia = createPinia();
  setActivePinia(pinia);
  const { router } = createFeStubRouter({
    route: {
      path: '/sandbox/records/kanban',
      fullPath: '/sandbox/records/kanban',
      meta: {
        viewKind: 'kanban',
        viewPaths: {
          list: '/sandbox/records',
          kanban: '/sandbox/records/kanban',
          chart: '/sandbox/records/chart',
        },
      },
    },
  });
  const seen: { detailPath?: unknown } = {};
  stubSfc(ChoyKanbanView as any, {
    name: 'ChoyKanbanView',
    props: ['store', 'detailPath', 'showHeader', 'createLabel'],
    setup(props: any) {
      seen.detailPath = props.detailPath;
      return () => h('div', { 'data-stub': 'ChoyKanbanView' });
    },
  });

  const store = {
    modelName: 'sandbox.Record',
    storeId: 'sandbox-kanban-detail',
    state: { queryState: {}, result: { rows: [], total: 0 } },
    metadata: { fields: {} },
  };
  const mounted = mountApp(SandboxRecordKanbanView as any, {
    props: { store, showHeader: false },
    plugins: [pinia, router as any],
  });
  await flushPromises();
  expect(seen.detailPath).toBe('/sandbox/records/:id');
  mounted.unmount();
  restoreSfc(ChoyKanbanView as any);
});
