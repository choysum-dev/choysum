// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { nextTick, ref } from 'vue';
import { readOperationStatusSnapshot, useModuleCatalogOpDrawerSession } from './useModuleCatalogOpDrawerSession';

const msgs = {
  failedToLoadPlan: 'plan-fail',
  applyFailed: 'apply-fail',
  jobStillRunning: 'still',
  serviceRestarting: 'restart',
  failedToGetStatus: 'no-status',
  completed: 'done',
  operationFailed: 'failed',
  succeededReloadFailed: 'reload-fail',
  operationSucceeded: 'ok',
};

function makeStore(overrides: Record<string, unknown> = {}) {
  return {
    Plan: async () => ({
      Id: 'op-1',
      ModuleOrder: [{ ModuleName: 'core', Reason: 'root', CurrentVersion: '', TargetVersion: '1.0', Origin: 'local' }],
      AffectedApps: ['core'],
      Risks: [],
      Blockers: [],
    }),
    Apply: async () => ({ Id: 'op-1' }),
    Browse: async () => ({
      Status: 'succeeded',
      ReloadTriggered: false,
      ReloadFailed: false,
    }),
    ...overrides,
  } as any;
}

test('useModuleCatalogOpDrawerSession: runPlan maps plan payload and gates confirm on blockers', async () => {
  const open = ref(true);
  const store = makeStore({
    Plan: async () => ({
      Id: 'op-block',
      ModuleOrder: [{ ModuleName: 'x' }],
      AffectedApps: [],
      Risks: [{ Code: 'R1', Level: 'warn', Message: 'careful' }],
      Blockers: [{ Code: 'B1', Level: 'error', Message: 'stop' }],
    }),
  });
  const api = useModuleCatalogOpDrawerSession({
    operationStore: store,
    isOpen: () => open.value,
    msgs,
  });

  await api.runPlan({ moduleName: 'core', action: 'install' });
  expect(api.plan.value?.Id).toBe('op-block');
  expect(api.plan.value?.blockers).toHaveLength(1);
  expect(api.plan.value?.risks[0]?.code).toBe('R1');
  expect(api.plan.value?.changingModules?.map(m => m.moduleName)).toEqual(['x']);
  expect(api.canConfirm.value).toBe(false);
});

test('useModuleCatalogOpDrawerSession: maps Role/Effect into changing vs satisfied', async () => {
  const store = makeStore({
    Plan: async () => ({
      Id: 'op-part',
      ModuleOrder: [
        {
          ModuleName: 'core',
          Application: 'core',
          Role: 'dependency',
          Effect: 'already_satisfied',
        },
        {
          ModuleName: 'partner',
          Application: 'partner',
          Role: 'target',
          Effect: 'install',
        },
      ],
      EnsureModules: [],
      AffectedApps: ['core', 'partner'],
      NeedsWebBuild: true,
      Risks: [],
      Blockers: [],
    }),
  });
  const api = useModuleCatalogOpDrawerSession({
    operationStore: store,
    isOpen: () => true,
    msgs,
  });
  await api.runPlan({ moduleName: 'partner', action: 'install' });
  expect(api.plan.value?.changingModules.map(m => m.moduleName)).toEqual(['partner']);
  expect(api.plan.value?.satisfiedModules.map(m => m.moduleName)).toEqual(['core']);
  expect(api.plan.value?.needsWebBuild).toBe(true);
  expect(api.plan.value?.showAffectedApplications).toBe(false);
  expect(api.plan.value?.changedApplications).toEqual(['partner']);
});

test('useModuleCatalogOpDrawerSession: confirmApply emits onApplied without watching', async () => {
  const applied: string[] = [];
  const settled: unknown[] = [];
  let browseCalls = 0;
  const store = {
    Plan: async () => ({
      Id: 'op-1',
      ModuleOrder: [{ ModuleName: 'core' }],
      AffectedApps: [],
      Risks: [],
      Blockers: [],
    }),
    Apply: async () => ({ Id: 'op-1' }),
    Browse: async () => {
      browseCalls += 1;
      return { Status: 'succeeded' };
    },
  } as any;
  const api = useModuleCatalogOpDrawerSession({
    operationStore: store,
    isOpen: () => true,
    msgs,
    onApplied: id => applied.push(id),
    onSettled: status => settled.push(status),
  });
  await api.runPlan({ moduleName: 'core', action: 'install' });
  await api.confirmApply();
  expect(applied).toEqual(['op-1']);
  expect(settled.length).toBe(0);
  expect(browseCalls).toBe(0);
  expect(api.step.value).toBe('plan');
});

test('useModuleCatalogOpDrawerSession: confirmApply Apply failure settles result', async () => {
  const open = ref(true);
  const settled: unknown[] = [];
  const store = makeStore({
    Apply: async () => {
      throw new Error('apply boom');
    },
  });
  const api = useModuleCatalogOpDrawerSession({
    operationStore: store,
    isOpen: () => open.value,
    msgs,
    onSettled: status => settled.push(status),
  });

  await api.runPlan({ moduleName: 'core', action: 'upgrade' });
  expect(api.canConfirm.value).toBe(true);
  await api.confirmApply();
  await nextTick();
  expect(api.step.value).toBe('result');
  expect(settled.length).toBe(1);
  expect(api.opStatus.value?.resultStatus).toBe('FAILED');
});

test('useModuleCatalogOpDrawerSession: resultTitle uses partial success copy', () => {
  const api = useModuleCatalogOpDrawerSession({
    operationStore: makeStore(),
    isOpen: () => true,
    msgs: { ...msgs, operationSucceededPartial: 'partial-ok' },
  });
  api.opStatus.value = { status: 'succeeded', resultStatus: 'SUCCEEDED', partialFailure: true };
  expect(api.resultTitle.value).toBe('partial-ok');
  api.opStatus.value = {
    status: 'succeeded',
    resultStatus: 'SUCCEEDED',
    partialFailure: false,
    ReloadFailed: true,
  };
  expect(api.resultTitle.value).toBe('reload-fail');
});

test('useModuleCatalogOpDrawerSession: resultTitle stays neutral without SUCCEEDED', async () => {
  const open = ref(true);
  const api = useModuleCatalogOpDrawerSession({
    operationStore: makeStore({
      Browse: async () => ({ Status: 'cancelled' }),
    }),
    isOpen: () => open.value,
    msgs,
  });
  await api.watchProgress('op-cancelled');
  expect(api.step.value).toBe('result');
  expect(api.resultTitle.value).toBe('done');
});

test('useModuleCatalogOpDrawerSession: replanWithRegistry passes Origin registry', async () => {
  const open = ref(true);
  const plans: any[] = [];
  const store = makeStore({
    Plan: async (req: any) => {
      plans.push(req);
      return {
        Id: `op-${plans.length}`,
        ModuleOrder: [{ ModuleName: 'core', Origin: req.Origin || 'local' }],
        AffectedApps: [],
        Risks: [],
        Blockers: [],
      };
    },
  });
  const api = useModuleCatalogOpDrawerSession({
    operationStore: store,
    isOpen: () => open.value,
    msgs,
  });
  await api.runPlan({ moduleName: 'core', action: 'upgrade' });
  await api.replanWithRegistry();
  expect(plans[0].Origin).toBeUndefined();
  expect(plans[1].Origin).toBe('registry');
  expect(api.useRegistryOrigin.value).toBe(true);
});

test('useModuleCatalogOpDrawerSession: superseded plan response is ignored', async () => {
  const open = ref(true);
  let resolvePlan: ((v: any) => void) | undefined;
  const store = makeStore({
    Plan: () =>
      new Promise(resolve => {
        resolvePlan = resolve;
      }),
  });
  const api = useModuleCatalogOpDrawerSession({
    operationStore: store,
    isOpen: () => open.value,
    msgs,
  });
  const first = api.runPlan({ moduleName: 'core', action: 'install' });
  api.reset();
  resolvePlan?.({
    Id: 'stale',
    ModuleOrder: [],
    AffectedApps: [],
    Risks: [],
    Blockers: [],
  });
  await first;
  expect(api.plan.value).toBeNull();
  expect(api.planLoading.value).toBe(false);
});

test('useModuleCatalogOpDrawerSession: stop clears planLoading for an in-flight plan', async () => {
  const open = ref(true);
  let resolvePlan: ((v: any) => void) | undefined;
  const store = makeStore({
    Plan: () =>
      new Promise(resolve => {
        resolvePlan = resolve;
      }),
  });
  const api = useModuleCatalogOpDrawerSession({
    operationStore: store,
    isOpen: () => open.value,
    msgs,
  });
  const first = api.runPlan({ moduleName: 'core', action: 'install' });
  expect(api.planLoading.value).toBe(true);
  api.stop();
  expect(api.planLoading.value).toBe(false);
  resolvePlan?.({
    Id: 'stale',
    ModuleOrder: [],
    AffectedApps: [],
    Risks: [],
    Blockers: [],
  });
  await first;
  expect(api.plan.value).toBeNull();
  expect(api.planLoading.value).toBe(false);
});

test('useModuleCatalogOpDrawerSession: runPlanDemo maps PlanDemo payload', async () => {
  const open = ref(true);
  const store = makeStore({
    PlanDemo: async () => ({
      Id: 'op-demo',
      Action: 'load-demo',
      ModuleOrder: [{ ModuleName: 'base', Reason: 'pending-demo' }],
      AffectedApps: [],
      Risks: [],
      Blockers: [],
    }),
  });
  const api = useModuleCatalogOpDrawerSession({
    operationStore: store,
    isOpen: () => open.value,
    msgs,
  });
  await api.runPlanDemo();
  expect(api.activeAction.value).toBe('load_demo');
  expect(api.plan.value?.Id).toBe('op-demo');
  expect(api.plan.value?.affectedModules).toEqual([
    {
      moduleName: 'base',
      application: undefined,
      role: undefined,
      effect: undefined,
      reason: 'pending-demo',
      currentVersion: undefined,
      targetVersion: undefined,
      origin: undefined,
    },
  ]);
  expect(api.plan.value?.changingModules.map(m => m.moduleName)).toEqual(['base']);
  expect(api.canConfirm.value).toBe(true);
});

test('useModuleCatalogOpDrawerSession: runPlanDemo guards missing PlanDemo and resets on throw', async () => {
  const open = ref(true);
  const missing = useModuleCatalogOpDrawerSession({
    operationStore: null as any,
    isOpen: () => open.value,
    msgs,
  });
  await missing.runPlanDemo();
  expect(missing.plan.value).toBeNull();
  expect(missing.activeAction.value).toBe('');

  const noMethod = useModuleCatalogOpDrawerSession({
    operationStore: makeStore(),
    isOpen: () => open.value,
    msgs,
  });
  // Seed a stale load_demo step; missing PlanDemo must clear it.
  (noMethod.activeAction as { value: string }).value = 'load_demo';
  noMethod.planLoading.value = true;
  await noMethod.runPlanDemo();
  expect(noMethod.plan.value).toBeNull();
  expect(noMethod.activeAction.value).toBe('' as typeof noMethod.activeAction.value);
  expect(noMethod.planLoading.value).toBe(false);
  expect(noMethod.step.value).toBe('plan');

  // Missing PlanDemo invalidates an in-flight plan so a late response cannot land.
  let resolveStale: ((v: any) => void) | undefined;
  const storeRef = ref<any>(
    makeStore({
      PlanDemo: () =>
        new Promise(resolve => {
          resolveStale = resolve;
        }),
    }),
  );
  const api = useModuleCatalogOpDrawerSession({
    operationStore: storeRef,
    isOpen: () => open.value,
    msgs,
  });
  const first = api.runPlanDemo();
  storeRef.value = makeStore(); // no PlanDemo
  await api.runPlanDemo();
  resolveStale?.({
    Id: 'should-drop',
    ModuleOrder: [{ ModuleName: 'base' }],
    AffectedApps: [],
    Risks: [],
    Blockers: [],
  });
  await first;
  expect(api.plan.value).toBeNull();

  const throws = useModuleCatalogOpDrawerSession({
    operationStore: makeStore({
      PlanDemo: async () => {
        throw new Error('demo plan boom');
      },
    }),
    isOpen: () => open.value,
    msgs,
  });
  await throws.runPlanDemo();
  expect(throws.plan.value).toBeNull();
  expect(throws.activeAction.value).toBe('');
  expect(throws.planLoading.value).toBe(false);
});

test('useModuleCatalogOpDrawerSession: superseded PlanDemo response is ignored', async () => {
  const open = ref(true);
  let resolvePlan: ((v: any) => void) | undefined;
  const store = makeStore({
    PlanDemo: () =>
      new Promise(resolve => {
        resolvePlan = resolve;
      }),
  });
  const api = useModuleCatalogOpDrawerSession({
    operationStore: store,
    isOpen: () => open.value,
    msgs,
  });
  const first = api.runPlanDemo();
  api.reset();
  resolvePlan?.({
    Id: 'stale-demo',
    ModuleOrder: [],
    AffectedApps: [],
    Risks: [],
    Blockers: [],
  });
  await first;
  expect(api.plan.value).toBeNull();
});

test('useModuleCatalogOpDrawerSession: superseded PlanDemo error is ignored', async () => {
  const open = ref(true);
  let rejectPlan: ((e: Error) => void) | undefined;
  const store = makeStore({
    PlanDemo: () =>
      new Promise((_resolve, reject) => {
        rejectPlan = reject;
      }),
  });
  const api = useModuleCatalogOpDrawerSession({
    operationStore: store,
    isOpen: () => open.value,
    msgs,
  });
  const first = api.runPlanDemo();
  api.reset();
  rejectPlan?.(new Error('stale boom'));
  await first;
  expect(api.plan.value).toBeNull();
  expect(api.activeAction.value).toBe('');
});

test('useModuleCatalogOpDrawerSession: runPlan errors without store or when Plan throws', async () => {
  const open = ref(true);
  const missing = useModuleCatalogOpDrawerSession({
    operationStore: null as any,
    isOpen: () => open.value,
    msgs,
  });
  await missing.runPlan({ moduleName: 'core', action: 'install' });
  expect(missing.plan.value).toBeNull();

  const store = makeStore({
    Plan: async () => {
      throw new Error('plan boom');
    },
  });
  const api = useModuleCatalogOpDrawerSession({
    operationStore: store,
    isOpen: () => open.value,
    msgs,
  });
  await api.runPlan({ moduleName: 'core', action: 'install' });
  expect(api.plan.value).toBeNull();
  expect(api.planLoading.value).toBe(false);
});

test('useModuleCatalogOpDrawerSession: confirmApply reports Apply failure and clears loading', async () => {
  const open = ref(true);
  const store = makeStore({
    Apply: async () => {
      throw new Error('apply boom');
    },
  });
  const api = useModuleCatalogOpDrawerSession({
    operationStore: store,
    isOpen: () => open.value,
    msgs,
  });
  await api.runPlan({ moduleName: 'core', action: 'install' });
  await api.confirmApply();
  expect(api.step.value).toBe('result');
  expect(api.opStatus.value?.errorCode).toBe('REQUEST_FAILED');
  expect(api.executeLoading.value).toBe(false);
});

test('useModuleCatalogOpDrawerSession: confirmApply without store is a no-op error', async () => {
  const open = ref(true);
  const api = useModuleCatalogOpDrawerSession({
    operationStore: makeStore(),
    isOpen: () => open.value,
    msgs,
  });
  await api.runPlan({ moduleName: 'core', action: 'install' });
  const bare = useModuleCatalogOpDrawerSession({
    operationStore: null as any,
    isOpen: () => open.value,
    msgs,
  });
  bare.plan.value = {
    Id: 'op-x',
    affectedModules: [],
    ensureModules: [],
    changingModules: [],
    satisfiedModules: [],
    affectedApps: [],
    changedApplications: [],
    showAffectedApplications: false,
    needsWebBuild: false,
    risks: [],
    blockers: [],
  };
  bare.step.value = 'plan';
  await bare.confirmApply();
  expect(bare.step.value).toBe('plan');
});

test('readOperationStatusSnapshot: Browse, Search, and empty-store paths', async () => {
  const viaBrowse = await readOperationStatusSnapshot(
    {
      Browse: async () => ({ Status: 'queued' }),
    } as any,
    'op-1',
  );
  expect(viaBrowse.status).toBe('queued');

  const viaSearch = await readOperationStatusSnapshot(
    {
      Search: async () => [{ Status: 'failed', ErrorCode: 'E' }],
    } as any,
    'op-2',
  );
  expect(viaSearch.status).toBe('failed');
  expect(viaSearch.errorCode).toBe('E');

  const empty = await readOperationStatusSnapshot({} as any, 'op-3');
  expect(empty.status).toBe('');
});

test('useModuleCatalogOpDrawerSession: watchProgress uses Search fallback and handles missing readers', async () => {
  const open = ref(true);
  const searchStore = {
    Search: async () => [{ Status: 'succeeded', ReloadTriggered: false, ReloadFailed: false }],
  } as any;
  const viaSearch = useModuleCatalogOpDrawerSession({
    operationStore: searchStore,
    isOpen: () => open.value,
    msgs,
  });
  await viaSearch.watchProgress('op-search');
  expect(viaSearch.step.value).toBe('result');
  expect(viaSearch.opStatus.value?.status).toBe('succeeded');

  const stub = useModuleCatalogOpDrawerSession({
    operationStore: {} as any,
    isOpen: () => open.value,
    msgs,
  });
  await stub.watchProgress('op-stub');
  expect(stub.step.value).toBe('progress');
  expect(stub.opStatus.value?.status).toBe('queued');
});
