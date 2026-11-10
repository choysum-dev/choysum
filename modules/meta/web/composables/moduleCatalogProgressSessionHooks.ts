// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoyMessage } from '@/web';
import { MODULE_OP_STATUS_FETCH_FAILED } from './useModuleOpProgress';

export type ModuleCatalogProgressNotifyMsgs = {
  jobStillRunning: string;
  serviceRestarting: string;
  failedToGetStatus: string;
};

/** Shared tip/poll progress toasts for catalog sync and op strip sessions. */
export function moduleCatalogProgressNotifyHooks(msgs: ModuleCatalogProgressNotifyMsgs) {
  return {
    onTimeout: () => {
      ChoyMessage.warning(msgs.jobStillRunning);
    },
    onTransientNetworkError: () => {
      ChoyMessage.warning(msgs.serviceRestarting);
    },
    onHardError: (message: string) => {
      if (!message || message === MODULE_OP_STATUS_FETCH_FAILED) {
        ChoyMessage.error(msgs.failedToGetStatus);
        return;
      }
      ChoyMessage.error(message);
    },
  };
}
