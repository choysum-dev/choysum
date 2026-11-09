// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref, type Ref } from 'vue';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { ChoyMessage } from '@/web';

export type ModuleCatalogSyncMsgs = {
  triggered: (operationId: string) => string;
  failed: (message: string) => string;
};

/**
 * Catalog sync helper for the manual "Update Module List" action.
 */
export function useModuleCatalogSync(
  store: WebModelStore<any>,
  msgs: ModuleCatalogSyncMsgs,
): {
  syncLoading: Ref<boolean>;
  requestSync: () => Promise<string>;
} {
  const syncLoading = ref(false);

  async function requestSync(): Promise<string> {
    if (syncLoading.value) return '';
    syncLoading.value = true;
    try {
      const resp = (await (store as any).RequestSync({ Force: true })) as { Id?: string } | string | null;
      const operationId = typeof resp === 'string' ? resp : String(resp?.Id || '').trim();
      // Skip toast when the server reused nothing / returned empty (should be rare for Force).
      if (operationId) {
        ChoyMessage.success(msgs.triggered(operationId));
      }
      return operationId;
    } catch (error: any) {
      const detail =
        typeof error === 'string'
          ? error
          : String(error?.message || error?.code || error || 'request failed');
      ChoyMessage.warning(msgs.failed(detail));
      return '';
    } finally {
      syncLoading.value = false;
    }
  }

  return { syncLoading, requestSync };
}
