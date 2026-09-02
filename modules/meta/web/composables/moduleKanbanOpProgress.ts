// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModuleOpProgressHooks, ModuleOpStatusSnapshot } from './useModuleOpProgress';

export type ModuleKanbanOpProgressBindings = {
  fetchStatus: (jobId: string) => Promise<ModuleOpStatusSnapshot>;
  isDialogOpen: () => boolean;
  setOpStatus: (status: ModuleOpStatusSnapshot | null) => void;
  setDialogStep: (step: 'plan' | 'progress' | 'result') => void;
  warn: (message: string) => void;
  error: (message: string) => void;
  t: (message: string) => string;
};

/**
 * Builds C1 progress hooks for ModuleKanbanView dialog wiring.
 */
export function createModuleKanbanOpProgressHooks(
  bindings: ModuleKanbanOpProgressBindings
): ModuleOpProgressHooks {
  return {
    fetchStatus: bindings.fetchStatus,
    isActive: () => bindings.isDialogOpen(),
    onStatus: (status) => {
      bindings.setOpStatus(status);
    },
    onTerminal: () => {
      bindings.setDialogStep('result');
    },
    onTimeout: () => {
      bindings.setDialogStep('result');
      bindings.setOpStatus({
        status: 'dispatching',
        resultStatus: undefined,
      });
      bindings.warn(bindings.t('Job is still running in the background; refresh later'));
    },
    onTransientNetworkError: () => {
      bindings.warn(bindings.t('Service is restarting; status will retry automatically'));
    },
    onHardError: (message) => {
      bindings.error(message || bindings.t('Failed to get status'));
    },
  };
}
