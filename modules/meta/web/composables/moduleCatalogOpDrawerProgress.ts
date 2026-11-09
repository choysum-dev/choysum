// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModuleOpProgressHooks, ModuleOpStatusSnapshot } from './useModuleOpProgress';
import { parseCatalogSyncResult } from '../views/module_catalog_chrome';

export type ModuleCatalogOpDrawerProgressMessages = {
  jobStillRunning: () => string;
  serviceRestarting: () => string;
  failedToGetStatus: () => string;
};

export type ModuleCatalogOpDrawerProgressBindings = {
  fetchStatus: (operationId: string) => Promise<ModuleOpStatusSnapshot>;
  isDrawerOpen: () => boolean;
  setOpStatus: (status: ModuleOpStatusSnapshot | null) => void;
  setDrawerStep: (step: 'plan' | 'progress' | 'result') => void;
  warn: (message: string) => void;
  error: (message: string) => void;
  messages: ModuleCatalogOpDrawerProgressMessages;
};

/** Map a MetaModuleOperation row into the C1 progress snapshot. */
export function mapOperationRowToSnapshot(
  row: Record<string, unknown> | null | undefined,
): ModuleOpStatusSnapshot {
  const status = String(row?.Status || '').toLowerCase();
  const reloadTriggered = !!row?.ReloadTriggered;
  const reloadFailed = !!row?.ReloadFailed;
  const errorMessage = row?.ErrorMessage != null ? String(row.ErrorMessage) : undefined;
  const errorCode = row?.ErrorCode != null ? String(row.ErrorCode) : undefined;
  const phase = row?.Phase != null ? String(row.Phase) : undefined;
  const action = String(row?.Action || '').toLowerCase();
  const syncResult = action === 'sync' ? parseCatalogSyncResult(row?.PlanJson) : null;
  const partialFailure = !!syncResult && syncResult.failed > 0 && status === 'succeeded';
  let summary: unknown = errorMessage || (phase ? { message: phase } : undefined);
  if (syncResult) {
    summary = { syncResult };
  }
  return {
    status,
    summary,
    resultStatus: status === 'succeeded' ? 'SUCCEEDED' : status === 'failed' ? 'FAILED' : undefined,
    partialFailure,
    ReloadTriggered: reloadTriggered,
    ReloadFailed: reloadFailed,
    // Reload succeeded when triggered and not failed — web client should hard-refresh.
    ReloadWeb: reloadTriggered && !reloadFailed,
    errorDomain: row?.ErrorDomain != null ? String(row.ErrorDomain) : undefined,
    errorCode,
  };
}

/**
 * Builds C1 progress hooks for ModuleCatalogOpDrawer wiring (tip + poll read the operation row).
 */
export function createModuleCatalogOpDrawerProgressHooks(
  bindings: ModuleCatalogOpDrawerProgressBindings,
): ModuleOpProgressHooks {
  return {
    fetchStatus: bindings.fetchStatus,
    isActive: () => bindings.isDrawerOpen(),
    onStatus: status => {
      bindings.setOpStatus(status);
    },
    onTerminal: () => {
      bindings.setDrawerStep('result');
    },
    onTimeout: () => {
      bindings.setDrawerStep('result');
      bindings.setOpStatus({
        status: 'dispatching',
        resultStatus: undefined,
      });
      bindings.warn(bindings.messages.jobStillRunning());
    },
    onTransientNetworkError: () => {
      bindings.warn(bindings.messages.serviceRestarting());
    },
    onHardError: message => {
      if (!message || message === 'Failed to get status') {
        bindings.error(bindings.messages.failedToGetStatus());
        return;
      }
      bindings.error(message);
    },
  };
}
