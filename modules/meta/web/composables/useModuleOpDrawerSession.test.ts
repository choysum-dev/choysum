// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { nextTick, ref } from 'vue';
import { useModuleOpDrawerSession } from './useModuleOpDrawerSession';

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
