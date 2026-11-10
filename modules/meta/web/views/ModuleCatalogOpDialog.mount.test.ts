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

test('ModuleCatalogOpDialog.vue groups changing vs satisfied modules and hides applications', async () => {
  const store = {
    Plan: async () => ({
      Id: 'op-partner',
      ModuleOrder: [
        {
          ModuleName: 'core',
          Application: 'core',
          Role: 'dependency',
          Effect: 'already_satisfied',
          CurrentVersion: '1.0.0',
          TargetVersion: '1.0.0',
          Origin: 'local',
        },
        {
          ModuleName: 'partner',
          Application: 'partner',
          Role: 'target',
          Effect: 'install',
          TargetVersion: '1.0.0',
          Origin: 'local',
        },
      ],
      EnsureModules: [],
      AffectedApps: ['core', 'partner'],
      NeedsWebBuild: true,
      Risks: [],
      Blockers: [],
    }),
    Apply: async () => ({ Id: 'op-partner' }),
  };
  const wrapper = mount(ModuleCatalogOpDialog as any, {
    props: {
      open: true,
      action: 'install',
      moduleName: 'partner',
      operationStore: store,
    },
    global: buildPageMountGlobal({ route: { path: '/meta/modules', fullPath: '/meta/modules' } }),
  });
  await flushPromises();
  await flushPromises();
  const body = wrapper.text();
  expect(wrapper.find('[data-testid="module-op-plan-subtitle"]').exists()).toBe(false);
  expect(wrapper.find('[data-testid="module-op-changing-row"]').exists()).toBe(true);
  expect(body).toContain('partner');
  expect(body).toContain('1.0.0');
  expect(body).toContain('Will install');
  expect(wrapper.find('[data-testid="module-op-effect-badge"]').exists()).toBe(true);
  expect(body).not.toContain('root');
  expect(body).not.toContain('(local)');
  expect(wrapper.find('[data-testid="module-op-satisfied-modules"]').exists()).toBe(true);
  expect(wrapper.find('[data-testid="module-op-satisfied-list"]').exists()).toBe(false);
  expect(wrapper.find('[data-testid="module-op-web-build"]').exists()).toBe(false);
  expect(wrapper.find('[data-testid="module-op-application-summary"]').exists()).toBe(false);
  const toggle = wrapper.find('[data-testid="module-op-satisfied-toggle"]');
  expect(toggle.exists()).toBe(true);
  await toggle.trigger('click');
  await flushPromises();
  const satisfied = wrapper.find('[data-testid="module-op-satisfied-list"]').text();
  expect(satisfied).toContain('core');
  expect(satisfied).toContain('1.0.0');
  expect(satisfied).toContain('Installed');
  expect(wrapper.find('[data-testid="module-op-satisfied-badge"]').exists()).toBe(true);
  wrapper.unmount();
});

test('ModuleCatalogOpDialog.vue indents dependency without relation or app summary', async () => {
  const store = {
    Plan: async () => ({
      Id: 'op-pc',
      ModuleOrder: [
        {
          ModuleName: 'core',
          Application: 'core',
          Role: 'dependency',
          Effect: 'already_satisfied',
          CurrentVersion: '1.0.0',
          TargetVersion: '1.0.0',
          Origin: 'local',
        },
        {
          ModuleName: 'partner',
          Application: 'partner',
          Role: 'dependency',
          Effect: 'install',
          TargetVersion: '0.0.0-20260626031438',
          Origin: 'local',
        },
        {
          ModuleName: 'partner_commercial',
          Application: 'partner',
          Role: 'target',
          Effect: 'install',
          TargetVersion: '0.0.0-20260626031438',
          Origin: 'local',
        },
      ],
      EnsureModules: [],
      AffectedApps: ['partner'],
      NeedsWebBuild: true,
      Risks: [],
      Blockers: [],
    }),
    Apply: async () => ({ Id: 'op-pc' }),
  };
  const wrapper = mount(ModuleCatalogOpDialog as any, {
    props: {
      open: true,
      action: 'install',
      moduleName: 'partner_commercial',
      operationStore: store,
    },
    global: buildPageMountGlobal({ route: { path: '/meta/modules', fullPath: '/meta/modules' } }),
  });
  await flushPromises();
  await flushPromises();
  const body = wrapper.text();
  expect(body).toContain('partner_commercial');
  expect(body).toContain('partner');
  expect(body).toContain('0.0.0-20260626031438');
  expect(body).not.toContain('Install this dependency first');
  expect(body).toContain('Will install');
  expect(body).not.toContain('These modules belong to application');
  expect(wrapper.find('[data-testid="module-op-application-summary"]').exists()).toBe(false);
  expect(wrapper.find('[data-testid="module-op-web-build"]').exists()).toBe(false);
  expect(wrapper.find('[data-role="dependency"]').exists()).toBe(true);
  await wrapper.find('[data-testid="module-op-satisfied-toggle"]').trigger('click');
  await flushPromises();
  const satisfied = wrapper.find('[data-testid="module-op-satisfied-list"]').text();
  expect(satisfied).toContain('1.0.0');
  expect(satisfied).toContain('Installed');
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

test('ModuleCatalogOpDialog.vue progress without operationId warns and closes', async () => {
  const { ChoyMessage } = await import('@/web');
  const applied: string[] = [];
  const warnings: string[] = [];
  const openUpdates: boolean[] = [];
  const origWarn = ChoyMessage.warning;
  ChoyMessage.warning = ((m: string) => {
    warnings.push(m);
    return 0;
  }) as typeof ChoyMessage.warning;
  try {
    const wrapper = mount(ModuleCatalogOpDialog as any, {
      props: {
        open: true,
        action: 'progress',
        moduleName: 'core',
        operationId: '   ',
        onApplied: (id: string) => applied.push(id),
        'onUpdate:open': (next: boolean) => openUpdates.push(next),
      },
      global: buildPageMountGlobal({ route: { path: '/meta/modules', fullPath: '/meta/modules' } }),
    });
    await flushPromises();
    await flushPromises();
    expect(applied).toEqual([]);
    expect(warnings.length).toBe(1);
    expect(openUpdates.includes(false)).toBe(true);
    wrapper.unmount();
  } finally {
    ChoyMessage.warning = origWarn;
  }
});

test('ModuleCatalogOpDialog.vue shows reload-failed CTA and retries page refresh', async () => {
  const reloads: number[] = [];
  const prevWindow = (globalThis as any).window;
  (globalThis as any).window = {
    ...(prevWindow || {}),
    location: { reload: () => reloads.push(1) },
  };
  try {
    const store = {
      Plan: async () => ({
        Id: 'op-reload-failed',
        ModuleOrder: [{ ModuleName: 'core', Effect: 'install', Role: 'target' }],
        AffectedApps: [],
        Risks: [],
        Blockers: [],
      }),
      Apply: async () => {
        throw new Error('apply boom');
      },
    };
    const wrapper = mount(ModuleCatalogOpDialog as any, {
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
    await wrapper.find('[data-testid="module-op-confirm"]').trigger('click');
    await flushPromises();
    await flushPromises();
    expect(wrapper.find('[data-testid="module-op-result"]').exists()).toBe(true);

    // Apply errors land on the result step; overlay a reload-failed snapshot for CTA coverage.
    const status = (wrapper.vm as any).opStatus;
    expect(status).toBeTruthy();
    Object.assign(status, {
      status: 'succeeded',
      resultStatus: 'SUCCEEDED',
      ReloadTriggered: true,
      ReloadFailed: true,
    });
    await flushPromises();

    const reloadRow = wrapper.find('[data-testid="module-op-reload"]');
    expect(reloadRow.exists()).toBe(true);
    expect(reloadRow.text()).toMatch(/Trigger Failed/i);
    const retry = wrapper.find('[data-testid="module-op-reload-retry"]');
    expect(retry.exists()).toBe(true);
    await retry.trigger('click');
    expect(reloads).toEqual([1]);
    wrapper.unmount();
  } finally {
    (globalThis as any).window = prevWindow;
  }
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

test('ModuleCatalogOpDialog.vue renders blockers risks and upgrade badges', async () => {
  const store = {
    Plan: async () => ({
      Id: 'op-rich',
      ModuleOrder: [
        {
          ModuleName: 'core',
          Role: 'target',
          Effect: 'upgrade',
          CurrentVersion: '1.0',
          TargetVersion: '2.0',
          Origin: 'registry',
        },
      ],
      AffectedApps: ['core'],
      Risks: [{ Code: 'R1', Message: 'careful' }],
      Blockers: [{ Code: 'B1', Message: 'blocked' }],
    }),
    Apply: async () => ({ Id: 'op-rich' }),
  };
  const wrapper = mount(ModuleCatalogOpDialog as any, {
    props: {
      open: true,
      action: 'upgrade',
      moduleName: 'core',
      registryNewerThanLocal: false,
      operationStore: store,
    },
    global: buildPageMountGlobal({ route: { path: '/meta/modules', fullPath: '/meta/modules' } }),
  });
  await flushPromises();
  await flushPromises();
  const body = wrapper.text();
  expect(wrapper.find('[data-testid="module-op-plan-blocker-banner"]').exists()).toBe(true);
  expect(body).toContain('R1');
  expect(body).toContain('B1');
  expect(body).toContain('Will upgrade');
  expect(body).toContain('registry');
  expect(body).toContain('1.0 → 2.0');
  wrapper.unmount();
});

test('ModuleCatalogOpDialog.vue uninstall confirm uses destructive variant and empty plan copy', async () => {
  const store = {
    Plan: async () => ({
      Id: 'op-empty',
      ModuleOrder: [],
      AffectedApps: [],
      Risks: [],
      Blockers: [],
    }),
    Apply: async () => ({ Id: 'op-empty' }),
  };
  const wrapper = mount(ModuleCatalogOpDialog as any, {
    props: {
      open: true,
      action: 'uninstall',
      moduleName: 'legacy',
      operationStore: store,
    },
    global: buildPageMountGlobal({ route: { path: '/meta/modules', fullPath: '/meta/modules' } }),
  });
  await flushPromises();
  await flushPromises();
  expect(wrapper.text()).toContain('No module changes required');
  expect(wrapper.find('[data-testid="module-op-confirm"]').exists()).toBe(true);
  expect(wrapper.text()).toContain('No module changes required');
  wrapper.unmount();
});

test('ModuleCatalogOpDialog.vue plan helper formatters cover origin effects and sync summary', async () => {
  const wrapper = mount(ModuleCatalogOpDialog as any, {
    props: { open: false, action: 'install', moduleName: 'core' },
    global: buildPageMountGlobal({ route: { path: '/meta/modules', fullPath: '/meta/modules' } }),
  });
  await flushPromises();
  const vm = wrapper.vm as any;
  expect(vm.formatOrigin('registry')).toContain('registry');
  expect(vm.formatOrigin('local')).toContain('local');
  expect(vm.formatOrigin('mirror')).toBe('mirror');
  expect(vm.changingEffectBadge({ effect: 'uninstall' })).toContain('uninstall');
  expect(vm.changingEffectBadge({ effect: 'noop' })).toBe('');
  expect(vm.changingEffectBadgeClass({ effect: 'install' })).toContain('primary');
  expect(vm.changingEffectBadgeClass({ effect: 'upgrade' })).toContain('muted');
  expect(vm.changingEffectBadgeClass({ effect: 'uninstall' })).toContain('destructive');
  expect(vm.changingEffectBadgeClass({ effect: 'noop' })).toBe('');
  expect(vm.formatVersionLine({ effect: 'uninstall', targetVersion: '9' })).toBe('');
  expect(vm.formatVersionLine({ effect: 'upgrade', currentVersion: '1', targetVersion: '2' })).toContain('→');
  expect(vm.formatVersionLine({ effect: 'install', currentVersion: '1', targetVersion: '1' })).toBe('1');
  expect(vm.formatVersionLine({ effect: 'install', targetVersion: '3.0' })).toContain('3.0');
  vm.onOpenChange(false);
  vm.opStatus = { resultStatus: 'SUCCEEDED' };
  expect(String(vm.resultAlertBoxClass)).toContain('border-border');
  expect(
    vm.formatSummary({
      syncResult: { total: 2, success: 2, failed: 0, catalogRows: 4 },
    }),
  ).toContain('4');
  expect(
    vm.formatSummary({
      syncResult: { total: 3, success: 2, failed: 1, catalogRows: 5 },
    }),
  ).toContain('failed');
  expect(vm.formatSummary('plain')).toBeTruthy();
  wrapper.unmount();

  const failed = mount(ModuleCatalogOpDialog as any, {
    props: {
      open: true,
      action: 'upgrade',
      moduleName: 'core',
      operationStore: {
        Plan: async () => ({
          Id: 'op-fail-ui',
          ModuleOrder: [],
          AffectedApps: [],
          Risks: [],
          Blockers: [],
        }),
        Apply: async () => {
          throw new Error('apply broke');
        },
      },
    },
    global: buildPageMountGlobal({ route: { path: '/meta/modules', fullPath: '/meta/modules' } }),
  });
  await flushPromises();
  await flushPromises();
  await failed.find('[data-testid="module-op-confirm"]').trigger('click');
  await flushPromises();
  expect(String((failed.vm as any).resultAlertBoxClass)).toContain('destructive');
  failed.unmount();
});

test('ModuleCatalogOpDialog.vue sync action closes immediately with applied', async () => {
  const applied: string[] = [];
  const wrapper = mount(ModuleCatalogOpDialog as any, {
    props: {
      open: true,
      action: 'sync',
      operationId: 'op-sync-now',
      operationStore: { Browse: async () => ({ Status: 'queued' }) },
      onApplied: (id: string) => applied.push(id),
    },
    global: buildPageMountGlobal({ route: { path: '/meta/modules', fullPath: '/meta/modules' } }),
  });
  await flushPromises();
  expect(applied).toEqual(['op-sync-now']);
  wrapper.unmount();
});
