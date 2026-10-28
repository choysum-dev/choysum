// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import { buildPageMountGlobal } from '@choysum/page-mount';
import SandboxRecord from './SandboxRecord.vue';
import SandboxRecordList from './SandboxRecordList.vue';
import SandboxRecordKanban from './SandboxRecordKanban.vue';
import SandboxRecordChart from './SandboxRecordChart.vue';
import SandboxTag from './SandboxTag.vue';
import SandboxRecordProperties from './SandboxRecordProperties.vue';
import { sandboxMenus } from '../menu/menus';

const pages: Array<[string, any, string]> = [
  ['SandboxRecordList', SandboxRecordList, '/sandbox/records'],
  ['SandboxRecordKanban', SandboxRecordKanban, '/sandbox/records/kanban'],
  ['SandboxRecordChart', SandboxRecordChart, '/sandbox/records/chart'],
  ['SandboxRecord', SandboxRecord, '/sandbox/records/1'],
  ['SandboxTag', SandboxTag, '/sandbox/tags/new'],
  ['SandboxRecordProperties', SandboxRecordProperties, '/sandbox/configuration/record-properties'],
];

test('sandbox page mount: every ChoyPage host mounts under choysumMount', async () => {
  const failures: string[] = [];
  for (const [name, Comp, path] of pages) {
    let wrapper: ReturnType<typeof mount> | null = null;
    try {
      wrapper = mount(Comp as any, {
        global: buildPageMountGlobal({ route: { path, fullPath: path } }),
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
  expect(failures).toEqual([]);
});

test('sandbox menus: root declares an icon and leaves stay text-only', () => {
  expect(sandboxMenus.length).toBeGreaterThan(0);
  expect(sandboxMenus[0]!.icon).toBeTruthy();
  expect(sandboxMenus[0]!.children?.every(child => child.icon == null)).toBe(true);
  const configuration = sandboxMenus[0]!.children?.find(child => child.id === 'sandbox.menu.configuration');
  expect(configuration?.path).toBeFalsy();
  expect(configuration?.children?.some(child => child.path === '/sandbox/configuration/record-properties')).toBe(
    true,
  );
});
