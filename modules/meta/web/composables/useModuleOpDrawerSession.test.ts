// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { nextTick, ref } from 'vue';
import { readOperationStatusSnapshot, useModuleOpDrawerSession } from './useModuleOpDrawerSession';

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

test('useModuleOpDrawerSession: runPlan maps plan payload and gates confirm on blockers', async () => {
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
  const api = useModuleOpDrawerSession({
    operationStore: store,
    isOpen: () => open.value,
    msgs,
  });

  await api.runPlan({ moduleName: 'core', action: 'install' });
  expect(api.plan.value?.Id).toBe('op-block');
  expect(api.plan.value?.blockers).toHaveLength(1);
  expect(api.plan.value?.risks[0]?.code).toBe('R1');
  expect(api.canConfirm.value).toBe(false);
});

test('useModuleOpDrawerSession: confirmApply watches progress to terminal', async () => {
  const open = ref(true);
  let browseCalls = 0;
  const store = makeStore({
    Browse: async () => {
      browseCalls += 1;
      return { Status: 'succeeded', ReloadTriggered: false, ReloadFailed: false };
    },
  });
  const settled: number[] = [];
  const api = useModuleOpDrawerSession({
    operationStore: store,
    isOpen: () => open.value,
    msgs,
    onSettled: () => settled.push(1),
  });

  await api.runPlan({ moduleName: 'core', action: 'upgrade' });
  expect(api.canConfirm.value).toBe(true);
  await api.confirmApply();
  await nextTick();
  expect(api.step.value).toBe('result');
  expect(browseCalls).toBeGreaterThanOrEqual(1);
  expect(settled.length).toBe(1);
  expect(api.resultTitle.value).toBe('ok');
});

test('useModuleOpDrawerSession: resultTitle stays neutral without SUCCEEDED', async () => {
  const open = ref(true);
  const api = useModuleOpDrawerSession({
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

test('useModuleOpDrawerSession: replanWithRegistry passes Origin registry', async () => {
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
  const api = useModuleOpDrawerSession({
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

test('useModuleOpDrawerSession: superseded plan response is ignored', async () => {
  const open = ref(true);
  let resolvePlan: ((v: any) => void) | undefined;
  const store = makeStore({
    Plan: () =>
      new Promise(resolve => {
        resolvePlan = resolve;
      }),
  });
  const api = useModuleOpDrawerSession({
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
});

test('useModuleOpDrawerSession: runPlanDemo maps PlanDemo payload', async () => {
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
  const api = useModuleOpDrawerSession({
    operationStore: store,
    isOpen: () => open.value,
    msgs,
  });
  await api.runPlanDemo();
  expect(api.activeAction.value).toBe('load_demo');
  expect(api.plan.value?.Id).toBe('op-demo');
  expect(api.plan.value?.affectedModules).toEqual([
    { moduleName: 'base', reason: 'pending-demo', currentVersion: undefined, targetVersion: undefined, origin: undefined },
  ]);
  expect(api.canConfirm.value).toBe(true);
});

test('useModuleOpDrawerSession: runPlan errors without store or when Plan throws', async () => {
  const open = ref(true);
  const missing = useModuleOpDrawerSession({
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
  const api = useModuleOpDrawerSession({
    operationStore: store,
    isOpen: () => open.value,
    msgs,
  });
  await api.runPlan({ moduleName: 'core', action: 'install' });
  expect(api.plan.value).toBeNull();
  expect(api.planLoading.value).toBe(false);
});

test('useModuleOpDrawerSession: confirmApply reports Apply failure and clears loading', async () => {
  const open = ref(true);
  const store = makeStore({
    Apply: async () => {
      throw new Error('apply boom');
    },
  });
  const api = useModuleOpDrawerSession({
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

test('useModuleOpDrawerSession: confirmApply without store is a no-op error', async () => {
  const open = ref(true);
  const api = useModuleOpDrawerSession({
    operationStore: makeStore(),
    isOpen: () => open.value,
    msgs,
  });
  await api.runPlan({ moduleName: 'core', action: 'install' });
  const bare = useModuleOpDrawerSession({
    operationStore: null as any,
    isOpen: () => open.value,
    msgs,
  });
  bare.plan.value = { Id: 'op-x', affectedModules: [], affectedApps: [], risks: [], blockers: [] };
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

test('useModuleOpDrawerSession: watchProgress uses Search fallback and handles missing readers', async () => {
  const open = ref(true);
  const searchStore = {
    Search: async () => [{ Status: 'succeeded', ReloadTriggered: false, ReloadFailed: false }],
  } as any;
  const viaSearch = useModuleOpDrawerSession({
    operationStore: searchStore,
    isOpen: () => open.value,
    msgs,
  });
  await viaSearch.watchProgress('op-search');
  expect(viaSearch.step.value).toBe('result');
  expect(viaSearch.opStatus.value?.status).toBe('succeeded');

  const stub = useModuleOpDrawerSession({
    operationStore: {} as any,
    isOpen: () => open.value,
    msgs,
  });
  await stub.watchProgress('op-stub');
  expect(stub.step.value).toBe('progress');
  expect(stub.opStatus.value?.status).toBe('queued');
});
