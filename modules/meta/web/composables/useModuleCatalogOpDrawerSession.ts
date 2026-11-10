// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, ref, unref, type Ref } from 'vue';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { ChoyMessage } from '@/web';
import { createPlanDialogSessionGate } from '../views/module_catalog_chrome';
import { createModuleOpProgressSession, type ModuleOpStatusSnapshot } from './useModuleOpProgress';
import {
  createModuleCatalogOpDrawerProgressHooks,
  mapOperationRowToSnapshot,
} from './moduleCatalogOpDrawerProgress';
import {
  changedApplicationsFromModules,
  mapPlanModuleEntry,
  partitionPlanModules,
  shouldShowAffectedApplications,
  type ModuleOpPlanModuleView,
} from '../views/module_op_plan_impact';

export type ModuleCatalogOpDrawerStep = 'plan' | 'progress' | 'result';
export type ModuleCatalogOpDrawerPlanAction = 'install' | 'upgrade' | 'uninstall' | 'load_demo';

export type ModuleCatalogOpDrawerPlanView = {
  Id: string;
  baseRevision?: string;
  /** Full ModuleOrder (+ Ensure) mapped entries; prefer changingModules / satisfiedModules for UI. */
  affectedModules: ModuleOpPlanModuleView[];
  ensureModules: ModuleOpPlanModuleView[];
  changingModules: ModuleOpPlanModuleView[];
  satisfiedModules: ModuleOpPlanModuleView[];
  /** Engine AffectedApps (full closure); not for direct dump in the confirm dialog. */
  affectedApps: string[];
  /** Applications of changing modules only. */
  changedApplications: string[];
  showAffectedApplications: boolean;
  needsWebBuild: boolean;
  risks: Array<{ code: string; level: string; message?: string }>;
  blockers: Array<{ code: string; level: string; message?: string }>;
};

export type ModuleCatalogOpDrawerSessionMsgs = {
  failedToLoadPlan: string;
  applyFailed: string;
  jobStillRunning: string;
  serviceRestarting: string;
  failedToGetStatus: string;
  completed: string;
  operationFailed: string;
  succeededReloadFailed: string;
  operationSucceeded: string;
  /** Sync completed with catalog refresh but some sources failed. */
  operationSucceededPartial?: string;
};

const OP_STATUS_FIELDS = [
  'Action',
  'Status',
  'Phase',
  'ErrorDomain',
  'ErrorCode',
  'ErrorMessage',
  'ReloadTriggered',
  'ReloadFailed',
  'PlanJson',
] as const;

/** Read one operation row via Browse, else Search-by-Id. */
export async function readOperationStatusSnapshot(
  store: WebModelStore<any>,
  operationId: string,
): Promise<ModuleOpStatusSnapshot> {
  let row: Record<string, unknown> | null | undefined;
  if (typeof (store as any).Browse === 'function') {
    row = (await (store as any).Browse(operationId, [...OP_STATUS_FIELDS])) as Record<string, unknown>;
  } else if (typeof (store as any).Search === 'function') {
    const rows = (await (store as any).Search(['Id', '=', operationId], {
      limit: 1,
      fields: [...OP_STATUS_FIELDS],
    })) as Array<Record<string, unknown>> | null;
    row = rows?.[0] || null;
  } else {
    row = null;
  }
  return mapOperationRowToSnapshot(row);
}

function mapPlanResponse(nextPlan: any): ModuleCatalogOpDrawerPlanView {
  const affectedModules = (nextPlan?.ModuleOrder || []).map(mapPlanModuleEntry);
  const ensureModules = (nextPlan?.EnsureModules || []).map(mapPlanModuleEntry);
  const { changing, satisfied } = partitionPlanModules(affectedModules, ensureModules);
  const changedApplications = changedApplicationsFromModules(changing);
  return {
    Id: String(nextPlan?.Id || '').trim(),
    baseRevision: nextPlan?.BaseRevision != null ? String(nextPlan.BaseRevision) : undefined,
    affectedModules,
    ensureModules,
    changingModules: changing,
    satisfiedModules: satisfied,
    affectedApps: (nextPlan?.AffectedApps || []).map((name: unknown) => String(name || '').trim()).filter(Boolean),
    changedApplications,
    showAffectedApplications: shouldShowAffectedApplications(changing, changedApplications),
    needsWebBuild: nextPlan?.NeedsWebBuild === true,
    risks: (nextPlan?.Risks || []).map((issue: any) => ({
      code: String(issue?.Code || '').trim(),
      level: String(issue?.Level || '').trim(),
      message: issue?.Message != null ? String(issue.Message) : undefined,
    })),
    blockers: (nextPlan?.Blockers || []).map((issue: any) => ({
      code: String(issue?.Code || '').trim(),
      level: String(issue?.Level || '').trim(),
      message: issue?.Message != null ? String(issue.Message) : undefined,
    })),
  };
}

/**
 * Drawer session: Plan → Confirm/Apply → C1 progress (operation-row Read + tip).
 */
export function useModuleCatalogOpDrawerSession(opts: {
  operationStore: WebModelStore<any> | Ref<WebModelStore<any> | null | undefined>;
  isOpen: () => boolean;
  msgs: ModuleCatalogOpDrawerSessionMsgs;
  onSettled?: (status?: ModuleOpStatusSnapshot | null) => void;
  /** Called after Apply succeeds, before progress watch (hosts refresh ActiveOperationId). */
  onApplied?: (operationId: string) => void;
}) {
  const step = ref<ModuleCatalogOpDrawerStep>('plan');
  const planLoading = ref(false);
  const executeLoading = ref(false);
  const plan = ref<ModuleCatalogOpDrawerPlanView | null>(null);
  const opStatus = ref<ModuleOpStatusSnapshot | null>(null);
  const activeModuleName = ref('');
  const activeAction = ref<ModuleCatalogOpDrawerPlanAction | 'progress' | 'sync' | ''>('');
  const useRegistryOrigin = ref(false);
  const planSession = createPlanDialogSessionGate();
  // Tracks which begin() seq last set planLoading, so finally can clear even after stop()/invalidate.
  let planLoadingOwnerSeq = 0;

  function resolveStore(): WebModelStore<any> | null {
    // Unwrap Ref/ComputedRef layers. Some hosts pass a ComputedRef as the prop
    // value; one unref is not always enough.
    let store: any = opts.operationStore;
    for (let i = 0; i < 3; i += 1) {
      const next = unref(store);
      if (next === store) break;
      store = next;
    }
    return store || null;
  }

  async function fetchOperationStatus(operationId: string): Promise<ModuleOpStatusSnapshot> {
    const store = resolveStore();
    if (!store) throw new Error('operation store missing');
    return readOperationStatusSnapshot(store, operationId);
  }

  const progress = createModuleOpProgressSession(
    createModuleCatalogOpDrawerProgressHooks({
      fetchStatus: fetchOperationStatus,
      isDrawerOpen: () => opts.isOpen(),
      setOpStatus: status => {
        opStatus.value = status;
      },
      setDrawerStep: next => {
        step.value = next;
      },
      warn: message => ChoyMessage.warning(message),
      error: message => ChoyMessage.error(message),
      messages: {
        jobStillRunning: () => opts.msgs.jobStillRunning,
        serviceRestarting: () => opts.msgs.serviceRestarting,
        failedToGetStatus: () => opts.msgs.failedToGetStatus,
      },
    }),
  );

  const canConfirm = computed(
    () =>
      step.value === 'plan' &&
      !!plan.value?.Id &&
      !(plan.value?.blockers?.length > 0) &&
      !planLoading.value &&
      !executeLoading.value,
  );

  const resultTitle = computed(() => {
    if (!opStatus.value) return opts.msgs.completed;
    if (opStatus.value.resultStatus === 'FAILED') return opts.msgs.operationFailed;
    if (opStatus.value.resultStatus !== 'SUCCEEDED') return opts.msgs.completed;
    if (opStatus.value.partialFailure) return opts.msgs.operationSucceededPartial ?? opts.msgs.operationSucceeded;
    return opStatus.value.ReloadFailed ? opts.msgs.succeededReloadFailed : opts.msgs.operationSucceeded;
  });

  function reset() {
    planSession.invalidate();
    planLoadingOwnerSeq = 0;
    plan.value = null;
    opStatus.value = null;
    step.value = 'plan';
    planLoading.value = false;
    executeLoading.value = false;
    useRegistryOrigin.value = false;
    activeModuleName.value = '';
    activeAction.value = '';
    progress.stop();
  }

  function stop() {
    planSession.invalidate();
    planLoadingOwnerSeq = 0;
    planLoading.value = false;
    executeLoading.value = false;
    progress.stop();
  }

  async function runPlan(input: {
    moduleName: string;
    action: Exclude<ModuleCatalogOpDrawerPlanAction, 'load_demo'>;
    origin?: 'local' | 'registry';
  }) {
    const moduleName = String(input.moduleName || '').trim();
    const action = input.action;
    if (!moduleName || !action) return;
    const store = resolveStore();
    if (!store || typeof (store as any).Plan !== 'function') {
      ChoyMessage.error(opts.msgs.failedToLoadPlan);
      return;
    }

    activeModuleName.value = moduleName;
    activeAction.value = action;
    step.value = 'plan';
    opStatus.value = null;
    plan.value = null;
    const requestSeq = planSession.begin();
    planLoadingOwnerSeq = requestSeq;
    planLoading.value = true;
    try {
      const req: Record<string, unknown> = { ModuleName: moduleName, Action: action };
      if (input.origin === 'registry' || useRegistryOrigin.value) {
        req.Origin = 'registry';
      }
      const nextPlan = await (store as any).Plan(req);
      if (!planSession.isCurrent(requestSeq)) return;
      plan.value = mapPlanResponse(nextPlan);
    } catch (error: any) {
      if (!planSession.isCurrent(requestSeq)) return;
      ChoyMessage.error(String(error?.message || opts.msgs.failedToLoadPlan));
      plan.value = null;
    } finally {
      // Clear even after stop()/invalidate when no newer plan owns the spinner.
      if (planLoadingOwnerSeq === requestSeq) planLoading.value = false;
    }
  }

  /** Plans a load-demo operation for all Installed+HasDemo+!DemoLoaded catalog rows. */
  async function runPlanDemo() {
    const store = resolveStore();
    if (!store || typeof (store as any).PlanDemo !== 'function') {
      // Drop any in-flight plan response and clear a stale load_demo plan step.
      planSession.invalidate();
      activeAction.value = '';
      step.value = 'plan';
      planLoading.value = false;
      plan.value = null;
      ChoyMessage.error(opts.msgs.failedToLoadPlan);
      return;
    }
    activeModuleName.value = '';
    activeAction.value = 'load_demo';
    step.value = 'plan';
    opStatus.value = null;
    plan.value = null;
    const requestSeq = planSession.begin();
    planLoadingOwnerSeq = requestSeq;
    planLoading.value = true;
    try {
      const nextPlan = await (store as any).PlanDemo({});
      if (!planSession.isCurrent(requestSeq)) return;
      plan.value = mapPlanResponse(nextPlan);
    } catch (error: any) {
      if (!planSession.isCurrent(requestSeq)) return;
      ChoyMessage.error(String(error?.message || opts.msgs.failedToLoadPlan));
      plan.value = null;
      // Leave the drawer without a half-open load_demo plan step.
      activeAction.value = '';
      step.value = 'plan';
    } finally {
      if (planLoadingOwnerSeq === requestSeq) planLoading.value = false;
    }
  }

  async function replanWithRegistry() {
    if (activeAction.value !== 'upgrade' || !activeModuleName.value) return;
    useRegistryOrigin.value = true;
    await runPlan({
      moduleName: activeModuleName.value,
      action: 'upgrade',
      origin: 'registry',
    });
  }

  async function watchProgress(operationId: string, preferredAction?: 'progress' | 'sync') {
    const id = String(operationId || '').trim();
    if (!id) return;
    const store = resolveStore();
    const canRead =
      !!store && (typeof (store as any).Browse === 'function' || typeof (store as any).Search === 'function');
    const nextAction = preferredAction || activeAction.value || 'progress';
    if (!canRead) {
      // Page-mount stubs may omit Browse/Search; keep the drawer open without crashing.
      activeAction.value = nextAction;
      step.value = 'progress';
      opStatus.value = { status: 'queued', phase: 'queued' };
      return;
    }
    activeAction.value = nextAction;
    step.value = 'progress';
    opStatus.value = null;
    await progress.watch(id);
    if (opts.isOpen()) opts.onSettled?.(opStatus.value);
  }

  async function confirmApply() {
    const operationId = String(plan.value?.Id || '').trim();
    if (!operationId || !canConfirm.value) return;
    const store = resolveStore();
    if (!store) {
      ChoyMessage.error(opts.msgs.applyFailed);
      return;
    }
    executeLoading.value = true;
    try {
      await (store as any).Apply({ Id: operationId });
    } catch (error: any) {
      step.value = 'result';
      opStatus.value = {
        status: 'failed',
        resultStatus: 'FAILED',
        errorDomain: 'CLIENT',
        errorCode: 'REQUEST_FAILED',
        summary: String(error?.message || opts.msgs.applyFailed),
      };
      ChoyMessage.error(String(error?.message || opts.msgs.applyFailed));
      if (opts.isOpen()) opts.onSettled?.(opStatus.value);
      return;
    } finally {
      executeLoading.value = false;
    }
    // Host closes the confirm dialog and watches progress on a locked strip.
    opts.onApplied?.(operationId);
  }

  return {
    step,
    planLoading,
    executeLoading,
    plan,
    opStatus,
    activeModuleName,
    activeAction,
    useRegistryOrigin,
    canConfirm,
    resultTitle,
    reset,
    stop,
    runPlan,
    runPlanDemo,
    replanWithRegistry,
    watchProgress,
    confirmApply,
  };
}
