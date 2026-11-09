// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import ModuleCatalogOpDialog from './ModuleCatalogOpDialog.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

test('ModuleCatalogOpDialog.vue mounts closed shell for each action title branch', async () => {
  const actions = ['install', 'upgrade', 'uninstall', 'progress', 'sync', 'load_demo', ''] as const;
  for (const action of actions) {
    const wrapper = mount(ModuleCatalogOpDialog as any, {
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

test('ModuleCatalogOpDialog.vue opens load_demo via PlanDemo', async () => {
  const plans: any[] = [];
  const store = {
    PlanDemo: async (req: any) => {
      plans.push(req);
      return {
        Id: 'op-demo-1',
        Action: 'load-demo',
        ModuleOrder: [{ ModuleName: 'base', Reason: 'pending-demo' }],
        AffectedApps: [],
        Risks: [],
        Blockers: [],
      };
    },
    Apply: async () => ({ Id: 'op-demo-1' }),
    Browse: async () => ({ Status: 'succeeded', ReloadTriggered: false, ReloadFailed: false }),
  };
  const wrapper = mount(ModuleCatalogOpDialog as any, {
    props: {
      open: true,
      action: 'load_demo',
      moduleName: '',
      operationStore: store,
    },
    global: buildPageMountGlobal({ route: { path: '/meta/modules', fullPath: '/meta/modules' } }),
  });
  await flushPromises();
  await flushPromises();
  expect(plans.length).toBe(1);
  expect(wrapper.text()).toContain('Load Demo Data');
  wrapper.unmount();
});

test('ModuleCatalogOpDialog.vue plans and confirms then emits applied', async () => {
  const plans: any[] = [];
  const applies: any[] = [];
  const applied: string[] = [];
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

  const wrapper = mount(ModuleCatalogOpDialog as any, {
    props: {
      open: true,
      action: 'install',
      moduleName: 'core',
      operationStore: store,
      onApplied: (id: string) => applied.push(id),
    },
    global: buildPageMountGlobal({ route: { path: '/meta/modules', fullPath: '/meta/modules' } }),
  });
  await flushPromises();
  await flushPromises();
  expect(plans.length).toBe(1);
  expect(plans[0].ModuleName).toBe('core');
  expect(plans[0].Action).toBe('install');
  expect(wrapper.find('[data-testid="module-op-dialog"]').exists()).toBe(true);

  const confirm = wrapper.find('[data-testid="module-op-confirm"]');
  expect(confirm.exists()).toBe(true);
  await confirm.trigger('click');
  await flushPromises();
  await flushPromises();
  expect(applies.length).toBe(1);
  expect(applied).toEqual(['op-plan-1']);
  // Confirm dialog stays on plan until host closes; no locked progress step.
  expect(wrapper.find('[data-testid="module-op-progress-headline"]').exists()).toBe(false);
  wrapper.unmount();
});

test('ModuleCatalogOpDialog.vue progress action hands off via applied and closes', async () => {
  const applied: string[] = [];
  const openUpdates: boolean[] = [];
  const wrapper = mount(ModuleCatalogOpDialog as any, {
    props: {
      open: true,
      action: 'progress',
      moduleName: 'core',
      operationId: 'op-lock',
      operationStore: {
        Browse: async () => ({ Status: 'queued', Phase: 'queued' }),
      },
      onApplied: (id: string) => applied.push(id),
      'onUpdate:open': (next: boolean) => openUpdates.push(next),
    },
    global: buildPageMountGlobal({ route: { path: '/meta/modules', fullPath: '/meta/modules' } }),
  });
  await flushPromises();
  await flushPromises();
  expect(applied).toEqual(['op-lock']);
  expect(openUpdates.includes(false)).toBe(true);
  wrapper.unmount();
});

test('ModuleCatalogOpDialog.vue covers apply failure, registry replan, and close', async () => {
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
    Apply: async () => {
      throw new Error('boom');
    },
  };
  const wrapper = mount(ModuleCatalogOpDialog as any, {
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
  expect(wrapper.find('[data-testid="module-op-result"]').exists()).toBe(true);
  expect(wrapper.find('[data-testid="module-op-done"]').exists()).toBe(true);
  await wrapper.find('[data-testid="module-op-done"]').trigger('click');
  await flushPromises();
  wrapper.unmount();
});
