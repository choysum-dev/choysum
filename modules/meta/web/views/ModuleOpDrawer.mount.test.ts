// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import ModuleOpDrawer from './ModuleOpDrawer.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

test('ModuleOpDrawer.vue mounts for each action title branch', async () => {
  const actions = ['install', 'upgrade', 'uninstall', 'progress', 'sync', 'load_demo', ''] as const;
  for (const action of actions) {
    const wrapper = mount(ModuleOpDrawer as any, {
      props: {
        open: true,
        action,
        operationId: action === 'progress' ? 'op-1' : '',
        moduleName: action && action !== 'progress' ? 'core' : '',
      },
      global: buildPageMountGlobal({ route: { path: '/meta/modules', fullPath: '/meta/modules' } }),
    });
    await flushPromises();
    expect(wrapper.element != null).toBe(true);
    wrapper.unmount();
  }
});
