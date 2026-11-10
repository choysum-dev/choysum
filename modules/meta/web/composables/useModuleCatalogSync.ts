// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, getCurrentScope, onScopeDispose, ref, unref, type Ref } from 'vue';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { ChoyMessage } from '@/web';
import {
  formatModuleOpSummary,
  type CatalogSyncResultSummary,
} from '../views/module_catalog_chrome';
import {
  createModuleOpProgressSession,
  type ModuleOpProgressDeps,
  type ModuleOpStatusSnapshot,
} from './useModuleOpProgress';
import { moduleCatalogProgressNotifyHooks } from './moduleCatalogProgressSessionHooks';
import { readOperationStatusSnapshot } from './useModuleCatalogOpDrawerSession';

export type ModuleCatalogSyncMsgs = {
  failed: (message: string) => string;
  /** Toast when sync succeeds (summary may be empty). */
  settledOk: (summary: string) => string;
  /** Toast when catalog refreshed but some sources failed. */
  settledPartial: (summary: string) => string;
  /** Toast when the sync operation failed. */
  settledFailed: (message: string) => string;
  jobStillRunning: string;
  serviceRestarting: string;
  failedToGetStatus: string;
  /** Optional i18n for syncResult lines; falls back to formatModuleOpSummary. */
  formatSummaryOk?: (sync: CatalogSyncResultSummary) => string;
  formatSummaryPartial?: (sync: CatalogSyncResultSummary) => string;
};

/**
 * Manual "Update Module List": RequestSync + background tip/poll + toast (no drawer).
 */
export function useModuleCatalogSync(
  catalogStore: WebModelStore<any>,
  operationStore: WebModelStore<any> | Ref<WebModelStore<any> | null | undefined> | null | undefined,
  msgs: ModuleCatalogSyncMsgs,
  opts?: {
    refreshCatalog?: () => void | Promise<void>;
    /** Inject timers/tip for FE unit tests only. */
    progressDeps?: ModuleOpProgressDeps;
  },
): {
  syncBusy: Ref<boolean>;
  /** Alias of syncBusy (enqueue or watch in flight). */
  syncLoading: Ref<boolean>;
  onSync: () => Promise<void>;
  stop: () => void;
} {
  const enqueueLoading = ref(false);
  const watching = ref(false);
  const syncBusy = computed(() => enqueueLoading.value || watching.value);
  const syncLoading = syncBusy;

  let lastTerminal: ModuleOpStatusSnapshot | null = null;

  const progress = createModuleOpProgressSession({
    fetchStatus: async operationId => {
      const store = resolveOperationStore();
      if (!store) throw new Error('operation store missing');
      return readOperationStatusSnapshot(store, operationId);
    },
    isActive: () => watching.value,
    onStatus: () => {},
    onTerminal: status => {
      lastTerminal = status;
    },
    ...moduleCatalogProgressNotifyHooks({
      jobStillRunning: msgs.jobStillRunning,
      serviceRestarting: msgs.serviceRestarting,
      failedToGetStatus: msgs.failedToGetStatus,
    }),
  }, opts?.progressDeps);

  function resolveOperationStore(): WebModelStore<any> | null {
    let store: any = operationStore;
    for (let i = 0; i < 3; i += 1) {
      const next = unref(store);
      if (next === store) break;
      store = next;
    }
    return store || null;
  }

  function formatSyncSummary(status: ModuleOpStatusSnapshot): string {
    const summary = status.summary;
    const sync =
      summary && typeof summary === 'object'
        ? ((summary as { syncResult?: CatalogSyncResultSummary }).syncResult ?? null)
        : null;
    if (sync) {
      if (sync.failed > 0 && msgs.formatSummaryPartial) return msgs.formatSummaryPartial(sync);
      if (sync.failed === 0 && msgs.formatSummaryOk) return msgs.formatSummaryOk(sync);
    }
    return formatModuleOpSummary(summary);
  }

  async function settleSync(status: ModuleOpStatusSnapshot) {
    try {
      await opts?.refreshCatalog?.();
    } catch {
      // Refresh is best-effort.
    }
    const detail = formatSyncSummary(status).trim();
    if (status.resultStatus === 'FAILED') {
      const err =
        typeof status.summary === 'string' && status.summary.trim()
          ? status.summary.trim()
          : detail || status.errorCode || 'failed';
      ChoyMessage.error(msgs.settledFailed(err));
      return;
    }
    if (status.resultStatus !== 'SUCCEEDED') return;
    if (status.partialFailure) {
      ChoyMessage.warning(msgs.settledPartial(detail));
      return;
    }
    ChoyMessage.success(msgs.settledOk(detail));
  }

  async function onSync() {
    if (syncBusy.value) return;
    enqueueLoading.value = true;
    let operationId = '';
    try {
      const resp = (await (catalogStore as any).RequestSync({ Force: true })) as
        | { Id?: string }
        | string
        | null;
      operationId = typeof resp === 'string' ? resp : String(resp?.Id || '').trim();
    } catch (error: any) {
      const detail =
        typeof error === 'string'
          ? error
          : String(error?.message || error?.code || error || 'request failed');
      ChoyMessage.warning(msgs.failed(detail));
      return;
    } finally {
      enqueueLoading.value = false;
    }
    if (!operationId) {
      ChoyMessage.warning(msgs.jobStillRunning);
      return;
    }

    const opStore = resolveOperationStore();
    if (!opStore) {
      ChoyMessage.warning(msgs.jobStillRunning);
      return;
    }

    lastTerminal = null;
    watching.value = true;
    try {
      await progress.watch(operationId);
      if (lastTerminal) await settleSync(lastTerminal);
    } finally {
      watching.value = false;
      lastTerminal = null;
    }
  }

  function stop() {
    progress.stop();
    watching.value = false;
    enqueueLoading.value = false;
    lastTerminal = null;
  }

  if (getCurrentScope()) onScopeDispose(stop);

  return { syncBusy, syncLoading, onSync, stop };
}
