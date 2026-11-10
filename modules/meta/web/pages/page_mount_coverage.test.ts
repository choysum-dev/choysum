// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import { buildPageMountGlobal } from '@choysum/page-mount';
import ModuleCatalogKanban from './ModuleCatalogKanban.vue';
import ModuleCatalogList from './ModuleCatalogList.vue';
import ModuleCatalog from './ModuleCatalog.vue';
import { metaMenus } from '../menu/menus';

const pages: Array<[string, any, string, string]> = [
  ['ModuleCatalogKanban', ModuleCatalogKanban, '/meta/modules', 'MetaModuleCatalog'],
  ['ModuleCatalogList', ModuleCatalogList, '/meta/modules/list', 'MetaModuleCatalogList'],
  ['ModuleCatalog', ModuleCatalog, '/meta/modules/core', 'MetaModuleShow'],
];

test('meta page mount: every ChoyPage host mounts under choysumMount', async () => {
  const failures: string[] = [];
  for (const [name, Comp, path, routeName] of pages) {
    let wrapper: ReturnType<typeof mount> | null = null;
    try {
      wrapper = mount(Comp as any, {
        global: buildPageMountGlobal({
          route: {
            path,
            fullPath: path,
            name: routeName,
            params: name === 'ModuleCatalog' ? { name: 'core' } : {},
            meta:
              name === 'ModuleCatalog'
                ? {}
                : {
                    viewKind: name === 'ModuleCatalogList' ? 'list' : 'kanban',
                    defaultGroups: [],
                    viewPaths: { kanban: '/meta/modules', list: '/meta/modules/list' },
                  },
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

test('meta menus: Modules app root has icon and Catalog leaf at /meta/modules', () => {
  expect(metaMenus.length).toBe(1);
  expect(metaMenus[0]!.icon).toBeTruthy();
  expect(metaMenus[0]!.path == null || metaMenus[0]!.path === '').toBe(true);
  expect(metaMenus[0]!.children).toHaveLength(1);
  expect(metaMenus[0]!.children![0]!.path).toBe('/meta/modules');
});
