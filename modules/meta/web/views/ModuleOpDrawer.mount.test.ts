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

test('ModuleOpDrawer.vue covers result branches, registry replan, and close', async () => {
  const plans: any[] = [];
  const store = {
    Plan: async (req: any) => {
      plans.push(req);
      return {
        Id: 'op-up',
        ModuleOrder: [{ ModuleName: 'core', Origin: req.Origin || 'local' }],
        AffectedApps: [],
        Risks: [],
        Blockers: [],
      };
    },
    Apply: async () => ({ Id: 'op-up' }),
    Browse: async () => ({
      // Succeeded with reload failure hits the amber resultAlertBoxClass branch.
      Status: 'succeeded',
      ReloadTriggered: true,
      ReloadFailed: true,
      ErrorDomain: 'OPS',
      ErrorCode: 'E1',
      ErrorMessage: 'boom',
    }),
  };
  const wrapper = mount(ModuleOpDrawer as any, {
    props: {
      open: true,
      action: 'upgrade',
      moduleName: 'core',
      registryNewerThanLocal: true,
      operationStore: store,
    },
    global: buildPageMountGlobal({ route: { path: '/meta/modules', fullPath: '/meta/modules' } }),
  });
  await flushPromises();
  await flushPromises();
  const registry = wrapper.find('[data-testid="module-op-use-registry"]');
  expect(registry.exists()).toBe(true);
  await registry.trigger('click');
  await flushPromises();
  expect(plans.some((p: any) => p.Origin === 'registry')).toBe(true);

  await wrapper.find('[data-testid="module-op-confirm"]').trigger('click');
  await flushPromises();
  await flushPromises();
  expect(wrapper.find('[data-testid="module-op-reload"]').exists()).toBe(true);
  const reloadStub = () => undefined;
  const previousReload = (window.location as { reload: () => void }).reload;
  Object.defineProperty(window.location, 'reload', {
    configurable: true,
    writable: true,
    value: reloadStub,
  });
  try {
    const retry = wrapper.find('[data-testid="module-op-reload-retry"]');
    expect(retry.exists()).toBe(true);
    await retry.trigger('click');
    const done = wrapper.find('[data-testid="module-op-done"]');
    expect(done.exists()).toBe(true);
    await done.trigger('click');
    await flushPromises();
  } finally {
    Object.defineProperty(window.location, 'reload', {
      configurable: true,
      writable: true,
      value: previousReload,
    });
  }
  wrapper.unmount();

  // Cover the FAILED resultAlertBoxClass branch (distinct from ReloadFailed amber).
  const failedStore = {
    Plan: async () => ({
      Id: 'op-fail',
      ModuleOrder: [],
      AffectedApps: [],
      Risks: [],
      Blockers: [],
    }),
    Apply: async () => ({ Id: 'op-fail' }),
    Browse: async () => ({
      Status: 'failed',
      ErrorDomain: 'OPS',
      ErrorCode: 'E2',
      ErrorMessage: 'nope',
    }),
  };
  const failed = mount(ModuleOpDrawer as any, {
    props: {
      open: true,
      action: 'install',
      moduleName: 'core',
      operationStore: failedStore,
    },
    global: buildPageMountGlobal({ route: { path: '/meta/modules', fullPath: '/meta/modules' } }),
  });
  await flushPromises();
  await flushPromises();
  await failed.find('[data-testid="module-op-confirm"]').trigger('click');
  await flushPromises();
  await flushPromises();
  expect(failed.find('[data-testid="module-op-result"]').exists()).toBe(true);
  failed.unmount();

  // Cancelled terminal has no resultStatus — neutral resultAlertBoxClass (not success green).
  const cancelled = mount(ModuleOpDrawer as any, {
    props: {
      open: true,
      action: 'progress',
      moduleName: 'core',
      operationId: 'op-cancel',
      operationStore: {
        Browse: async () => ({ Status: 'cancelled' }),
      },
    },
    global: buildPageMountGlobal({ route: { path: '/meta/modules', fullPath: '/meta/modules' } }),
  });
  await flushPromises();
  await flushPromises();
  const alert = cancelled.find('[data-testid="module-op-alert"]');
  expect(alert.exists()).toBe(true);
  const alertClass = String((alert.element as any)?.getAttribute?.('class') || '');
  expect(alertClass).toContain('border-border');
  expect(alertClass).not.toContain('border-emerald');
  cancelled.unmount();
});
