// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, getCurrentScope, onScopeDispose, ref, unref, type Ref } from 'vue';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { ChoyMessage } from '@/web';
import {
  formatModuleOpProgressHeadline,
  formatModuleOpProgressValue,
} from './moduleCatalogOpDrawerProgress';
import { createModuleOpProgressSession, type ModuleOpStatusSnapshot } from './useModuleOpProgress';
import { readOperationStatusSnapshot } from './useModuleCatalogOpDrawerSession';

export type ModuleCatalogOpProgressMsgs = {
  jobStillRunning: string;
  serviceRestarting: string;
  failedToGetStatus: string;
  settledOk: string;
  settledFailed: (message: string) => string;
  reloadingService: () => string;
  executing: () => string;
  queued: () => string;
  finishing: () => string;
  inProgress: () => string;
  /** Overlay title: e.g. "Installing partner". */
  titleFor: (action: string, moduleName: string) => string;
};

/**
 * Lightweight catalog op monitor: tip/poll + locked progress strip (no modal).
 * Used after Confirm Apply and for card/deep-link "View Progress".
 */
export function useModuleCatalogOpProgress(
  operationStore: WebModelStore<any> | Ref<WebModelStore<any> | null | undefined> | null | undefined,
  msgs: ModuleCatalogOpProgressMsgs,
  opts?: {
    refreshCatalog?: () => void | Promise<void>;
  },
): {
  progressBusy: Ref<boolean>;
  progressTitle: Ref<string>;
  progressHeadline: Ref<string>;
  progressValue: Ref<number>;
  progressModuleName: Ref<string>;
  progressAction: Ref<string>;
  startOpProgress: (
    operationId: string,
    meta?: { moduleName?: string; action?: string },
  ) => Promise<void>;
  stop: () => void;
} {
  const watching = ref(false);
  const progressBusy = watching;
  const progressModuleName = ref('');
  const progressAction = ref('');
  const opStatus = ref<ModuleOpStatusSnapshot | null>(null);
  let lastTerminal: ModuleOpStatusSnapshot | null = null;
  let watchGeneration = 0;

  const progress = createModuleOpProgressSession({
    fetchStatus: async operationId => {
      const store = resolveOperationStore();
      if (!store) throw new Error('operation store missing');
      return readOperationStatusSnapshot(store, operationId);
    },
    isActive: () => watching.value,
    onStatus: status => {
      opStatus.value = status;
    },
    onTerminal: status => {
      lastTerminal = status;
      opStatus.value = status;
    },
    onTimeout: () => {
      ChoyMessage.warning(msgs.jobStillRunning);
    },
    onTransientNetworkError: () => {
      ChoyMessage.warning(msgs.serviceRestarting);
    },
    onHardError: message => {
      if (!message || message === 'Failed to get status') {
        ChoyMessage.error(msgs.failedToGetStatus);
        return;
      }
      ChoyMessage.error(message);
    },
  });

  const progressHeadline = computed(() =>
    formatModuleOpProgressHeadline(opStatus.value, progressModuleName.value, {
      reloadingService: msgs.reloadingService,
      executing: msgs.executing,
      queued: msgs.queued,
      finishing: msgs.finishing,
      inProgress: msgs.inProgress,
    }),
  );

  const progressTitle = computed(() =>
    msgs.titleFor(progressAction.value, progressModuleName.value),
  );

  const progressValue = computed(() => formatModuleOpProgressValue(opStatus.value));

  function resolveOperationStore(): WebModelStore<any> | null {
    let store: any = operationStore;
    for (let i = 0; i < 3; i += 1) {
      const next = unref(store);
      if (next === store) break;
      store = next;
    }
    return store || null;
  }

  async function settleOp(status: ModuleOpStatusSnapshot) {
    try {
      await opts?.refreshCatalog?.();
    } catch {
      // Refresh is best-effort.
    }
    if (status.resultStatus === 'FAILED') {
      const err =
        typeof status.summary === 'string' && status.summary.trim()
          ? status.summary.trim()
          : status.errorCode || 'failed';
      ChoyMessage.error(msgs.settledFailed(err));
      return;
    }
    if (status.resultStatus === 'SUCCEEDED') {
      ChoyMessage.success(msgs.settledOk);
    }
  }

  async function startOpProgress(
    operationId: string,
    meta?: { moduleName?: string; action?: string },
  ) {
    const id = String(operationId || '').trim();
    if (!id) return;

    // Replace any in-flight watch (card peek / deep-link while another op runs).
    progress.stop();
    watchGeneration += 1;
    const generation = watchGeneration;

    progressModuleName.value = String(meta?.moduleName || '').trim();
    progressAction.value = String(meta?.action || '').trim();
    opStatus.value = { status: 'queued', phase: 'queued' };
    lastTerminal = null;
    watching.value = true;

    const store = resolveOperationStore();
    if (!store) {
      watching.value = false;
      try {
        await opts?.refreshCatalog?.();
      } catch {
        // ignore
      }
      return;
    }

    try {
      await progress.watch(id);
      if (generation !== watchGeneration) return;
      if (lastTerminal) await settleOp(lastTerminal);
    } finally {
      if (generation === watchGeneration) {
        watching.value = false;
        lastTerminal = null;
      }
    }
  }

  function stop() {
    watchGeneration += 1;
    progress.stop();
    watching.value = false;
    lastTerminal = null;
    opStatus.value = null;
    progressModuleName.value = '';
    progressAction.value = '';
  }

  if (getCurrentScope()) onScopeDispose(stop);

  return {
    progressBusy,
    progressTitle,
    progressHeadline,
    progressValue,
    progressModuleName,
    progressAction,
    startOpProgress,
    stop,
  };
}
