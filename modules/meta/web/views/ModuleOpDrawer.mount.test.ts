// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import ModuleOpDrawer from './ModuleOpDrawer.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

test('ModuleOpDrawer.vue mounts closed shell for each action title branch', async () => {
  const actions = ['install', 'upgrade', 'uninstall', 'progress', 'sync', 'load_demo', ''] as const;
  for (const action of actions) {
    const wrapper = mount(ModuleOpDrawer as any, {
      props: {
        open: false,
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

test('ModuleOpDrawer.vue plans and confirms through operation store', async () => {
  const plans: any[] = [];
  const applies: any[] = [];
  const store = {
    Plan: async (req: any) => {
      plans.push(req);
      return {
        Id: 'op-plan-1',
        ModuleOrder: [{ ModuleName: 'core', TargetVersion: '1.0', Origin: 'local' }],
        AffectedApps: ['core'],
        Risks: [],
        Blockers: [],
      };
    },
    Apply: async (req: any) => {
      applies.push(req);
      return { Id: 'op-plan-1' };
    },
    Browse: async () => ({
      Status: 'succeeded',
      ReloadTriggered: false,
      ReloadFailed: false,
    }),
  };

  const wrapper = mount(ModuleOpDrawer as any, {
    props: {
      open: true,
      action: 'install',
      moduleName: 'core',
      operationStore: store,
    },
    global: buildPageMountGlobal({ route: { path: '/meta/modules', fullPath: '/meta/modules' } }),
  });
  await flushPromises();
  await flushPromises();
  expect(plans.length).toBe(1);
  expect(plans[0].ModuleName).toBe('core');
  expect(plans[0].Action).toBe('install');

  const confirm = wrapper.find('[data-testid="module-op-confirm"]');
  expect(confirm.exists()).toBe(true);
  await confirm.trigger('click');
  await flushPromises();
  await flushPromises();
  expect(applies.length).toBe(1);
  expect(wrapper.find('[data-testid="module-op-status"]').exists()).toBe(true);
  wrapper.unmount();
});
