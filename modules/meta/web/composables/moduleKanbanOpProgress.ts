// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModuleOpProgressHooks, ModuleOpStatusSnapshot } from './useModuleOpProgress';

export type ModuleKanbanOpProgressMessages = {
  jobStillRunning: () => string;
  serviceRestarting: () => string;
  failedToGetStatus: () => string;
};

export type ModuleKanbanOpProgressBindings = {
  fetchStatus: (jobId: string) => Promise<ModuleOpStatusSnapshot>;
  isDialogOpen: () => boolean;
  setOpStatus: (status: ModuleOpStatusSnapshot | null) => void;
  setDialogStep: (step: 'plan' | 'progress' | 'result') => void;
  warn: (message: string) => void;
  error: (message: string) => void;
  messages: ModuleKanbanOpProgressMessages;
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
      bindings.warn(bindings.messages.jobStillRunning());
    },
    onTransientNetworkError: () => {
      bindings.warn(bindings.messages.serviceRestarting());
    },
    onHardError: (message) => {
      if (!message || message === 'Failed to get status') {
        bindings.error(bindings.messages.failedToGetStatus());
        return;
      }
      bindings.error(message);
    },
  };
}
