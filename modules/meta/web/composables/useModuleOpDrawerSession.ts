// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, ref, type Ref } from 'vue';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { ChoyMessage } from '@/web';
import { createPlanDialogSessionGate } from '../views/module_catalog_chrome';
import { createModuleOpProgressSession, type ModuleOpStatusSnapshot } from './useModuleOpProgress';
import {
  createModuleOpDrawerProgressHooks,
  mapOperationRowToSnapshot,
} from './moduleOpDrawerProgress';

export type ModuleOpDrawerStep = 'plan' | 'progress' | 'result';
export type ModuleOpDrawerPlanAction = 'install' | 'upgrade' | 'uninstall';

export type ModuleOpDrawerPlanView = {
  Id: string;
  baseRevision?: string;
  affectedModules: Array<{
    moduleName: string;
    reason?: string;
    currentVersion?: string;
    targetVersion?: string;
    origin?: string;
  }>;
  affectedApps: string[];
  risks: Array<{ code: string; level: string; message?: string }>;
  blockers: Array<{ code: string; level: string; message?: string }>;
};

export type ModuleOpDrawerSessionMsgs = {
  failedToLoadPlan: string;
  applyFailed: string;
  jobStillRunning: string;
  serviceRestarting: string;
  failedToGetStatus: string;
  completed: string;
  operationFailed: string;
  succeededReloadFailed: string;
  operationSucceeded: string;
};

const OP_STATUS_FIELDS = [
  'Status',
  'Phase',
  'ErrorDomain',
  'ErrorCode',
  'ErrorMessage',
  'ReloadTriggered',
  'ReloadFailed',
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

function mapPlanResponse(nextPlan: any): ModuleOpDrawerPlanView {
  return {
    Id: String(nextPlan?.Id || '').trim(),
    baseRevision: nextPlan?.BaseRevision != null ? String(nextPlan.BaseRevision) : undefined,
    affectedModules: (nextPlan?.ModuleOrder || []).map((entry: any) => ({
      moduleName: String(entry?.ModuleName || '').trim(),
      reason: entry?.Reason != null ? String(entry.Reason) : undefined,
      currentVersion: entry?.CurrentVersion != null ? String(entry.CurrentVersion) : undefined,
      targetVersion: entry?.TargetVersion != null ? String(entry.TargetVersion) : undefined,
      origin: entry?.Origin != null ? String(entry.Origin) : undefined,
    })),
    affectedApps: (nextPlan?.AffectedApps || []).map((name: unknown) => String(name || '').trim()).filter(Boolean),
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
export function useModuleOpDrawerSession(opts: {
  operationStore: WebModelStore<any> | Ref<WebModelStore<any> | null | undefined>;
  isOpen: () => boolean;
  msgs: ModuleOpDrawerSessionMsgs;
  onSettled?: () => void;
}) {
  const step = ref<ModuleOpDrawerStep>('plan');
  const planLoading = ref(false);
  const executeLoading = ref(false);
  const plan = ref<ModuleOpDrawerPlanView | null>(null);
  const opStatus = ref<ModuleOpStatusSnapshot | null>(null);
  const activeModuleName = ref('');
  const activeAction = ref<ModuleOpDrawerPlanAction | 'progress' | ''>('');
  const useRegistryOrigin = ref(false);
  const planSession = createPlanDialogSessionGate();

  function resolveStore(): WebModelStore<any> | null {
    const raw = opts.operationStore as any;
    const store = raw && typeof raw === 'object' && 'value' in raw ? raw.value : raw;
    return store || null;
  }

  async function fetchOperationStatus(operationId: string): Promise<ModuleOpStatusSnapshot> {
    const store = resolveStore();
    if (!store) throw new Error('operation store missing');
    return readOperationStatusSnapshot(store, operationId);
  }

  const progress = createModuleOpProgressSession(
    createModuleOpDrawerProgressHooks({
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
    return opStatus.value.ReloadFailed ? opts.msgs.succeededReloadFailed : opts.msgs.operationSucceeded;
  });

  function reset() {
    planSession.invalidate();
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
    progress.stop();
  }

  async function runPlan(input: {
    moduleName: string;
    action: ModuleOpDrawerPlanAction;
    origin?: 'local' | 'registry';
  }) {
    const moduleName = String(input.moduleName || '').trim();
    const action = input.action;
    if (!moduleName || !action) return;
    const store = resolveStore();
    if (!store) {
      ChoyMessage.error(opts.msgs.failedToLoadPlan);
      return;
    }

    activeModuleName.value = moduleName;
    activeAction.value = action;
    step.value = 'plan';
    opStatus.value = null;
    plan.value = null;
    const requestSeq = planSession.begin();
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
      if (planSession.isCurrent(requestSeq)) planLoading.value = false;
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

  async function watchProgress(operationId: string) {
    const id = String(operationId || '').trim();
    if (!id) return;
    const store = resolveStore();
    const canRead =
      !!store && (typeof (store as any).Browse === 'function' || typeof (store as any).Search === 'function');
    if (!canRead) {
      // Page-mount stubs may omit Browse/Search; keep the drawer open without crashing.
      activeAction.value = activeAction.value || 'progress';
      step.value = 'progress';
      opStatus.value = { status: 'queued' };
      return;
    }
    activeAction.value = activeAction.value || 'progress';
    step.value = 'progress';
    opStatus.value = null;
    await progress.watch(id);
    if (opts.isOpen()) opts.onSettled?.();
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
    step.value = 'progress';
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
      return;
    } finally {
      executeLoading.value = false;
    }
    // progress.watch swallows status-read errors; do not remap them as Apply failures.
    await progress.watch(operationId);
    if (opts.isOpen()) opts.onSettled?.();
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
    replanWithRegistry,
    watchProgress,
    confirmApply,
  };
}
