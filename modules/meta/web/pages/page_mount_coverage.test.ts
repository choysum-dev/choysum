// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import { buildPageMountGlobal } from '@choysum/page-mount';
import ModuleCatalog from './ModuleCatalog.vue';
import ModuleShow from './ModuleShow.vue';
import { metaMenus } from '../menu/menus';

const pages: Array<[string, any, string]> = [
  ['ModuleCatalog', ModuleCatalog, '/meta/modules'],
  ['ModuleShow', ModuleShow, '/meta/modules/core'],
];

test('meta page mount: every ChoyPage host mounts under choysumMount', async () => {
  const failures: string[] = [];
  for (const [name, Comp, path] of pages) {
    let wrapper: ReturnType<typeof mount> | null = null;
    try {
      wrapper = mount(Comp as any, {
        global: buildPageMountGlobal({
          route: {
            path,
            fullPath: path,
            name: name === 'ModuleShow' ? 'MetaModuleShow' : 'MetaModuleCatalog',
            params: name === 'ModuleShow' ? { name: 'core' } : {},
          },
        }),
      });
      await flushPromises();
      const ok =
        wrapper.find('[data-anchor="choy.page"]').exists() ||
        wrapper.find('[data-testid="fe-stub-choy-page"]').exists() ||
        wrapper.find('[data-testid="fe-stub-child-view"]').exists();
      if (!ok) failures.push(name);
    } catch (error) {
      failures.push(`${name}: ${(error as Error).message || String(error)}`);
    } finally {
      wrapper?.unmount();
    }
  }
  if (failures.length) throw new Error(`meta page mount failed: ${failures.join(', ')}`);
});

test('meta menus: Modules leaf declares an icon and has no children', () => {
  expect(metaMenus.length).toBe(1);
  expect(metaMenus[0]!.icon).toBeTruthy();
  expect(metaMenus[0]!.path).toBe('/meta/modules');
  expect(metaMenus[0]!.children == null || metaMenus[0]!.children.length === 0).toBe(true);
});
