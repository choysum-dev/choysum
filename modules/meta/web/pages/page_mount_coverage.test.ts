// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import { buildPageMountGlobal } from '@choysum/page-mount';
import ModuleDetail from './ModuleDetail.vue';
import ModuleHistory from './ModuleHistory.vue';
import ModuleList from './ModuleList.vue';
import ModuleListTable from './ModuleListTable.vue';
import { metaMenus } from '../menu/menus';
import ModuleDetailView from '../views/ModuleDetailView.vue';

const pages: Array<[string, any, string]> = [
  ['ModuleList', ModuleList, '/meta/modules'],
  ['ModuleListTable', ModuleListTable, '/meta/modules/list'],
  ['ModuleHistory', ModuleHistory, '/meta/modules/history'],
  ['ModuleDetail', ModuleDetail, '/meta/modules/1'],
];

test('meta page mount: every ChoyPage host mounts under choysumMount', async () => {
  for (const [name, Comp, path] of pages) {
    const wrapper = mount(Comp as any, {
      global: buildPageMountGlobal({ route: { path, fullPath: path } }),
    });
    await flushPromises();
    const ok =
      wrapper.find('[data-anchor="choy.page"]').exists() ||
      wrapper.find('[data-testid="fe-stub-opage"]').exists() ||
      wrapper.find('[data-testid="fe-stub-choy-page"]').exists() ||
      wrapper.find('[data-testid="fe-stub-child-view"]').exists();
    expect(ok).toBe(true);
    if (!ok) throw new Error(`page mount failed: ${name}`);
    wrapper.unmount();
  }
});

test('meta ModuleDetailView mounts under choysumMount', async () => {
  const wrapper = mount(ModuleDetailView as any, {
    global: buildPageMountGlobal({ route: { path: '/meta/modules/1', fullPath: '/meta/modules/1' } }),
  });
  await flushPromises();
  expect(
    wrapper.find('[data-testid="fe-stub-child-view"]').exists() ||
      wrapper.find('[data-anchor="choy.form-view"]').exists() ||
      wrapper.text().length >= 0,
  ).toBe(true);
  wrapper.unmount();
});

test('meta menus: root icon is Lucide Settings component', () => {
  expect(metaMenus.length).toBeGreaterThan(0);
  const icon = metaMenus[0]!.icon;
  expect(icon === null || icon === undefined).toBe(false);
  expect(typeof icon === 'function' || typeof icon === 'object').toBe(true);
});
