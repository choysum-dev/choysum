// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import { useAuthStore } from '@/auth/web/stores/auth';
import LanguageFormView from './LanguageFormView.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

function seedLanguageWritePermission(actions: string[] = ['*']) {
  const auth = useAuthStore();
  auth.identity = {
    metadata: { activeCompanyId: 'c1', enabledCompanyIds: ['c1'] },
  } as any;
  auth.permissionState = {
    permStateVersion: 1,
    byCompany: {
      '*': { ui: { routes: [], menus: [], actions } },
    },
  } as any;
}

function buildStore(row: { Id: string; Code: string; IsActive: boolean; Name?: string }, opts?: {
  recordsEntry?: unknown;
  rows?: unknown[];
  useEmptyRecords?: boolean;
  browse?: () => Promise<unknown>;
  activate?: (req: unknown) => Promise<unknown>;
  deactivate?: (req: unknown) => Promise<unknown>;
}) {
  const activateCalls: unknown[] = [];
  const deactivateCalls: unknown[] = [];
  const updateByIdCalls: unknown[] = [];
  const browseCalls: unknown[] = [];
  const store = {
    $id: 'fe-stub-language-activate-store',
    state: {
      records: opts?.useEmptyRecords
        ? {}
        : opts?.recordsEntry !== undefined
          ? { [row.Id]: opts.recordsEntry }
          : { [row.Id]: { ...row } },
      result: { rows: opts?.rows ?? [] },
    },
    records: {},
    Activate: async (req: unknown) => {
      activateCalls.push(req);
      if (opts?.activate) return opts.activate(req);
      return { Id: row.Id, Code: row.Code, IsActive: true };
    },
    Deactivate: async (req: unknown) => {
      deactivateCalls.push(req);
      if (opts?.deactivate) return opts.deactivate(req);
      return { Id: row.Id, Code: row.Code, IsActive: false };
    },
    UpdateById: async (...args: unknown[]) => {
      updateByIdCalls.push(args);
      return row;
    },
    Browse: async (...args: unknown[]) => {
      browseCalls.push(args);
      if (opts?.browse) return opts.browse();
      return { ...row, IsActive: !row.IsActive };
    },
  };
  return { store, activateCalls, deactivateCalls, updateByIdCalls, browseCalls };
}

function mountLanguageForm(store: unknown, recordId: string) {
  const wrapper = mount(LanguageFormView as any, {
    props: { store, recordId, viewMode: 'display' },
    global: buildPageMountGlobal({
      route: { path: `/base/languages/${recordId}`, fullPath: `/base/languages/${recordId}` },
    }),
  });
  // Pinia auth store is ready after mount (same pattern as ModuleCatalogFormView.mount.test).
  seedLanguageWritePermission();
  return wrapper;
}

test('LanguageFormView: display shows Activate when inactive', async () => {
  const { store, activateCalls, updateByIdCalls, browseCalls } = buildStore({
    Id: 'lang-inactive',
    Code: 'zh_HK',
    IsActive: false,
    Name: 'Chinese (HK)',
  });
  const wrapper = mountLanguageForm(store, 'lang-inactive');
  await flushPromises();
  const btn = wrapper.find('[data-testid="language-activate"]');
  expect(btn.exists()).toBe(true);
  expect(wrapper.find('[data-testid="language-deactivate"]').exists()).toBe(false);
  await btn.trigger('click');
  await flushPromises();
  expect(activateCalls).toEqual([{ Id: 'lang-inactive' }]);
  expect(updateByIdCalls.length).toBe(0);
  expect(browseCalls.length).toBe(1);
  expect(wrapper.find('[data-testid="language-deactivate"]').exists()).toBe(true);
  wrapper.unmount();
});

test('LanguageFormView: resolves Activate from route :id when prop omitted', async () => {
  seedLanguageWritePermission();
  const row = { Id: 'lang-route', Code: 'ca_ES', IsActive: false, Name: 'Catalan' };
  const { store } = buildStore(row);
  const wrapper = mount(LanguageFormView as any, {
    props: { store, viewMode: 'display' },
    global: buildPageMountGlobal({
      route: {
        path: '/base/languages/lang-route',
        fullPath: '/base/languages/lang-route',
        params: { id: 'lang-route' },
      },
    }),
  });
  seedLanguageWritePermission();
  await flushPromises();
  expect(wrapper.find('[data-testid="language-activate"]').exists()).toBe(true);
  wrapper.unmount();
});

test('LanguageFormView: display shows Deactivate when active', async () => {
  const { store, deactivateCalls, updateByIdCalls } = buildStore({
    Id: 'lang-active',
    Code: 'zh_CN',
    IsActive: true,
    Name: 'Chinese',
  });
  const wrapper = mountLanguageForm(store, 'lang-active');
  await flushPromises();
  const btn = wrapper.find('[data-testid="language-deactivate"]');
  expect(btn.exists()).toBe(true);
  expect(wrapper.find('[data-testid="language-activate"]').exists()).toBe(false);
  await btn.trigger('click');
  await flushPromises();
  expect(deactivateCalls).toEqual([{ Id: 'lang-active' }]);
  expect(updateByIdCalls.length).toBe(0);
  wrapper.unmount();
});

test('LanguageFormView: unwraps RecordRow payload in store.records', async () => {
  const row = { Id: 'lang-payload', Code: 'fr_FR', IsActive: false, Name: 'French' };
  const { store } = buildStore(row, {
    recordsEntry: { kind: 'record', key: 'lang-payload', payload: row },
  });
  const wrapper = mountLanguageForm(store, 'lang-payload');
  await flushPromises();
  expect(wrapper.find('[data-testid="language-activate"]').exists()).toBe(true);
  wrapper.unmount();
});

test('LanguageFormView: resolves row from result.rows when records empty', async () => {
  const row = { Id: 'lang-rows', Code: 'de_DE', IsActive: true, Name: 'German' };
  const { store } = buildStore(row, {
    useEmptyRecords: true,
    rows: [{ kind: 'record', key: 'lang-rows', payload: row }],
  });
  const wrapper = mountLanguageForm(store, 'lang-rows');
  await flushPromises();
  expect(wrapper.find('[data-testid="language-deactivate"]').exists()).toBe(true);
  wrapper.unmount();
});

test('LanguageFormView: hides activate actions without write permission', async () => {
  const { store } = buildStore({ Id: 'lang-noperm', Code: 'zh_HK', IsActive: false });
  const wrapper = mount(LanguageFormView as any, {
    props: { store, recordId: 'lang-noperm', viewMode: 'display' },
    global: buildPageMountGlobal({
      route: { path: '/base/languages/lang-noperm', fullPath: '/base/languages/lang-noperm' },
    }),
  });
  seedLanguageWritePermission([]);
  await flushPromises();
  expect(wrapper.find('[data-testid="language-activate"]').exists()).toBe(false);
  expect(wrapper.find('[data-testid="language-deactivate"]').exists()).toBe(false);
  wrapper.unmount();
});

test('LanguageFormView: ignores empty Browse payload after Activate', async () => {
  const { store, activateCalls } = buildStore(
    { Id: 'lang-browse-empty', Code: 'zh_HK', IsActive: false },
    {
      browse: async () => ({}),
    }
  );
  const wrapper = mountLanguageForm(store, 'lang-browse-empty');
  await flushPromises();
  await wrapper.find('[data-testid="language-activate"]').trigger('click');
  await flushPromises();
  expect(activateCalls).toEqual([{ Id: 'lang-browse-empty' }]);
  // Empty Browse payload does not override; Activate button remains.
  expect(wrapper.find('[data-testid="language-activate"]').exists()).toBe(true);
  wrapper.unmount();
});

test('LanguageFormView: missing Activate method is handled without UpdateById', async () => {
  const { store, updateByIdCalls } = buildStore({ Id: 'lang-missing', Code: 'zh_HK', IsActive: false });
  delete (store as any).Activate;
  const wrapper = mountLanguageForm(store, 'lang-missing');
  await flushPromises();
  await wrapper.find('[data-testid="language-activate"]').trigger('click');
  await flushPromises();
  expect(updateByIdCalls.length).toBe(0);
  expect(wrapper.find('[data-testid="language-activate"]').exists()).toBe(true);
  wrapper.unmount();
});

test('LanguageFormView: missing Deactivate method is handled', async () => {
  const { store, updateByIdCalls } = buildStore({ Id: 'lang-missing-deact', Code: 'zh_CN', IsActive: true });
  delete (store as any).Deactivate;
  const wrapper = mountLanguageForm(store, 'lang-missing-deact');
  await flushPromises();
  await wrapper.find('[data-testid="language-deactivate"]').trigger('click');
  await flushPromises();
  expect(updateByIdCalls.length).toBe(0);
  expect(wrapper.find('[data-testid="language-deactivate"]').exists()).toBe(true);
  wrapper.unmount();
});

test('LanguageFormView: Activate without Browse still succeeds', async () => {
  const { store, activateCalls } = buildStore({ Id: 'lang-nobrowse', Code: 'zh_HK', IsActive: false });
  delete (store as any).Browse;
  const wrapper = mountLanguageForm(store, 'lang-nobrowse');
  await flushPromises();
  await wrapper.find('[data-testid="language-activate"]').trigger('click');
  await flushPromises();
  expect(activateCalls).toEqual([{ Id: 'lang-nobrowse' }]);
  wrapper.unmount();
});
