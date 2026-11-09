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

export type ModuleOpProgressHeadlineMessages = {
  reloadingService: () => string;
  executing: () => string;
  queued: () => string;
  finishing: () => string;
  inProgress: () => string;
};

/** Coarse bar value from phase/status (ops have stages, not a true percent). */
export function formatModuleOpProgressValue(
  snap: ModuleOpStatusSnapshot | null | undefined,
): number {
  const phase = String(snap?.phase || '').toLowerCase();
  const status = String(snap?.status || '').toLowerCase();
  if (phase === 'reloading' || snap?.ReloadTriggered) return 82;
  if (phase === 'executing' || status === 'running') return 55;
  if (phase === 'done' || status === 'succeeded' || status === 'failed' || status === 'cancelled') {
    return 100;
  }
  if (phase === 'queued' || status === 'queued' || status === 'dispatching' || !status) return 18;
  return 35;
}

/** Human progress line from snapshot phase/status (prefer phase). */
export function formatModuleOpProgressHeadline(
  snap: ModuleOpStatusSnapshot | null | undefined,
  _moduleName: string,
  msgs: ModuleOpProgressHeadlineMessages,
): string {
  const phase = String(snap?.phase || '').toLowerCase();
  const status = String(snap?.status || '').toLowerCase();
  if (phase === 'reloading' || snap?.ReloadTriggered) {
    return msgs.reloadingService();
  }
  if (phase === 'executing' || status === 'running') {
    return msgs.executing();
  }
  if (phase === 'queued' || status === 'queued' || status === 'dispatching' || !status) {
    return msgs.queued();
  }
  if (phase === 'done') {
    return msgs.finishing();
  }
  return msgs.inProgress();
}

/** Map a MetaModuleOperation row into the C1 progress snapshot. */
export function mapOperationRowToSnapshot(
  row: Record<string, unknown> | null | undefined,
): ModuleOpStatusSnapshot {
  const status = String(row?.Status || '').toLowerCase();
  const reloadTriggered = !!row?.ReloadTriggered;
  const reloadFailed = !!row?.ReloadFailed;
  const errorMessage = row?.ErrorMessage != null ? String(row.ErrorMessage) : undefined;
  const errorCode = row?.ErrorCode != null ? String(row.ErrorCode) : undefined;
  const phaseRaw = row?.Phase != null ? String(row.Phase).trim() : '';
  const phase = phaseRaw || undefined;
  const action = String(row?.Action || '').toLowerCase();
  const syncResult = action === 'sync' ? parseCatalogSyncResult(row?.PlanJson) : null;
  const partialFailure = !!syncResult && syncResult.failed > 0 && status === 'succeeded';
  // Prefer error text or syncResult; do not mirror phase into summary (UI uses phase).
  let summary: unknown = errorMessage || undefined;
  if (syncResult) {
    summary = { syncResult };
  }
  return {
    status,
    phase,
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
