// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import LanguageFormView from './LanguageFormView.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

function buildStore(row: { Id: string; Code: string; IsActive: boolean; Name?: string }) {
  const activateCalls: unknown[] = [];
  const deactivateCalls: unknown[] = [];
  const updateByIdCalls: unknown[] = [];
  const store = {
    $id: 'fe-stub-language-activate-store',
    state: {
      records: {
        [row.Id]: { ...row },
      },
      result: { rows: [] },
    },
    records: {},
    Activate: async (req: unknown) => {
      activateCalls.push(req);
      return { Id: row.Id, Code: row.Code, IsActive: true };
    },
    Deactivate: async (req: unknown) => {
      deactivateCalls.push(req);
      return { Id: row.Id, Code: row.Code, IsActive: false };
    },
    UpdateById: async (...args: unknown[]) => {
      updateByIdCalls.push(args);
      return row;
    },
    Browse: async () => row,
  };
  return { store, activateCalls, deactivateCalls, updateByIdCalls };
}

test('LanguageFormView: display shows Activate when inactive', async () => {
  const { store, activateCalls, updateByIdCalls } = buildStore({
    Id: 'lang-inactive',
    Code: 'zh_HK',
    IsActive: false,
    Name: 'Chinese (HK)',
  });
  const wrapper = mount(LanguageFormView as any, {
    props: { store, recordId: 'lang-inactive', viewMode: 'display' },
    global: buildPageMountGlobal({
      route: { path: '/base/languages/lang-inactive', fullPath: '/base/languages/lang-inactive' },
    }),
  });
  await flushPromises();
  const btn = wrapper.find('[data-testid="language-activate"]');
  expect(btn.exists()).toBe(true);
  expect(wrapper.find('[data-testid="language-deactivate"]').exists()).toBe(false);
  await btn.trigger('click');
  await flushPromises();
  expect(activateCalls).toEqual([{ Id: 'lang-inactive' }]);
  expect(updateByIdCalls.length).toBe(0);
  wrapper.unmount();
});

test('LanguageFormView: display shows Deactivate when active', async () => {
  const { store, deactivateCalls, updateByIdCalls } = buildStore({
    Id: 'lang-active',
    Code: 'zh_CN',
    IsActive: true,
    Name: 'Chinese',
  });
  const wrapper = mount(LanguageFormView as any, {
    props: { store, recordId: 'lang-active', viewMode: 'display' },
    global: buildPageMountGlobal({
      route: { path: '/base/languages/lang-active', fullPath: '/base/languages/lang-active' },
    }),
  });
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
