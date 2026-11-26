// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import { h } from 'vue';
import * as VueRouter from 'vue-router';
import { useAuthStore } from '@/auth/web/stores/auth';
import { ChoyActionButton, ChoyListView } from '@/web';
import { flushPromises, fnRecorder, mountApp, stubSfc, restoreSfc } from '@/web/web/__tests__/mountApp';
import SandboxRecordListView from './SandboxRecordListView.vue';
import { sandboxRecordActions } from './sandbox_record_actions';

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

test('SandboxRecordListView contextual workflow marks draft rows', async () => {
  const pinia = createPinia();
  setActivePinia(pinia);
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

  const softRefresh = fnRecorder(async () => undefined);
  let cleared = 0;
  stubSfc(ChoyActionButton as any, {
    name: 'ChoyActionButton',
    inheritAttrs: false,
    emits: ['execute'],
    setup(_: unknown, { slots, attrs, emit }: any) {
      return () =>
        h('button', { type: 'button', ...attrs, onClick: () => emit('execute') }, slots.default?.());
    },
  });
  stubSfc(ChoyListView as any, {
    name: 'ChoyListView',
    props: ['store', 'detailPath', 'searchView', 'actionIds', 'hasAction', 'showHeader'],
    setup(_: unknown, { slots, expose }: any) {
      expose({
        softRefresh,
        clearSelection: () => {
          cleared += 1;
        },
      });
      const selectedItems = [{ Id: 'd1', State: 'draft' }];
      return () =>
        h('div', { 'data-stub': 'ChoyListView' }, [
          slots['contextual-actions']?.({ selectedItems }),
        ]);
    },
  });

  const updateById = fnRecorder(async () => undefined);
  const store = {
    modelName: 'sandbox.Record',
    storeId: 'sandbox-list-workflow',
    state: { queryState: {}, result: { rows: [], total: 0 }, selection: [] },
    metadata: { fields: {} },
    UpdateById: updateById,
  };

  const { router } = createFeStubRouter({
    route: {
      path: '/sandbox/records',
      fullPath: '/sandbox/records',
      meta: {
        viewKind: 'list',
        viewPaths: { list: '/sandbox/records' },
      },
    },
  });

  const mounted = mountApp(SandboxRecordListView as any, {
    props: { store, showHeader: false },
    plugins: [pinia, router as any],
  });
  await flushPromises();

  expect(mounted.q('[data-testid=sandbox-list-mark-open]')).toBeTruthy();
  mounted.click('[data-testid=sandbox-list-mark-open]');
  await flushPromises();
  expect(updateById.calls.length).toBe(1);
  expect(softRefresh.calls.length).toBe(1);
  expect(cleared).toBe(1);

  mounted.unmount();
  restoreSfc(ChoyListView as any);
  restoreSfc(ChoyActionButton as any);
});

async function mountListWithSelection(opts: {
  selectedItems: Array<{ Id: string; State: string }>;
  UpdateById: (...args: unknown[]) => Promise<unknown>;
}) {
  const pinia = createPinia();
  setActivePinia(pinia);
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

  const softRefresh = fnRecorder(async () => undefined);
  stubSfc(ChoyActionButton as any, {
    name: 'ChoyActionButton',
    inheritAttrs: false,
    emits: ['execute'],
    setup(_: unknown, { slots, attrs, emit }: any) {
      return () =>
        h('button', { type: 'button', ...attrs, onClick: () => emit('execute') }, slots.default?.());
    },
  });
  stubSfc(ChoyListView as any, {
    name: 'ChoyListView',
    props: ['store', 'detailPath', 'searchView', 'actionIds', 'hasAction', 'showHeader'],
    setup(_: unknown, { slots, expose }: any) {
      expose({
        softRefresh,
        clearSelection: () => undefined,
      });
      return () =>
        h('div', { 'data-stub': 'ChoyListView' }, [
          slots['contextual-actions']?.({ selectedItems: opts.selectedItems }),
        ]);
    },
  });

  const store = {
    modelName: 'sandbox.Record',
    storeId: 'sandbox-list-workflow',
    state: { queryState: {}, result: { rows: [], total: 0 }, selection: [] },
    metadata: { fields: {} },
    UpdateById: opts.UpdateById,
  };
  const { router } = createFeStubRouter({
    route: {
      path: '/sandbox/records',
      fullPath: '/sandbox/records',
      meta: { viewKind: 'list', viewPaths: { list: '/sandbox/records' } },
    },
  });
  const mounted = mountApp(SandboxRecordListView as any, {
    props: { store, showHeader: false },
    plugins: [pinia, router as any],
  });
  await flushPromises();
  return { mounted, softRefresh };
}

test('SandboxRecordListView partial bulk update still soft-refreshes', async () => {
  let calls = 0;
  const updateById = fnRecorder(async (id: string) => {
    calls += 1;
    if (id === 'd2') throw new Error('second failed');
    return undefined;
  });
  const { mounted, softRefresh } = await mountListWithSelection({
    selectedItems: [
      { Id: 'd1', State: 'draft' },
      { Id: 'd2', State: 'draft' },
    ],
    UpdateById: updateById as any,
  });

  mounted.click('[data-testid=sandbox-list-mark-open]');
  await flushPromises();
  expect(calls).toBe(2);
  // Partial progress still applies softRefresh via ViewWorkflowPartialError.
  expect(softRefresh.calls.length).toBe(1);

  mounted.unmount();
  restoreSfc(ChoyListView as any);
  restoreSfc(ChoyActionButton as any);
});

test('SandboxRecordListView total bulk failure skips softRefresh', async () => {
  const updateById = fnRecorder(async () => {
    throw new Error('all failed');
  });
  const { mounted, softRefresh } = await mountListWithSelection({
    selectedItems: [{ Id: 'd1', State: 'draft' }],
    UpdateById: updateById as any,
  });

  mounted.click('[data-testid=sandbox-list-mark-open]');
  await flushPromises();
  expect(updateById.calls.length).toBe(1);
  expect(softRefresh.calls.length).toBe(0);

  mounted.unmount();
  restoreSfc(ChoyListView as any);
  restoreSfc(ChoyActionButton as any);
});
