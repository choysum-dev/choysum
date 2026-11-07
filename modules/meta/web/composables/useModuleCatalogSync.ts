// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref, watch, type Ref } from 'vue';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { ChoyMessage } from '@/web';

export type ModuleCatalogSyncMsgs = {
  triggered: (operationId: string) => string;
  triggeredEmpty: string;
  failed: (message: string) => string;
};

/**
 * Catalog sync helpers: Force/IfStale RequestSync plus one-shot empty-list auto sync.
 * Auto-sync only runs after the store has a concrete result payload (not during initial undefined).
 */
export function useModuleCatalogSync(
  store: WebModelStore<any>,
  msgs: ModuleCatalogSyncMsgs,
): {
  syncLoading: Ref<boolean>;
  requestSync: (opts?: { Force?: boolean; IfStale?: boolean }) => Promise<string>;
  watchEmptyCatalogAutoSync: () => void;
} {
  const syncLoading = ref(false);
  const autoSyncTriggered = ref(false);

  async function requestSync(opts: { Force?: boolean; IfStale?: boolean } = {}): Promise<string> {
    if (syncLoading.value) return '';
    syncLoading.value = true;
    try {
      const resp = (await (store as any).RequestSync({
        Force: opts.Force !== undefined ? !!opts.Force : true,
        IfStale: !!opts.IfStale,
      })) as { Id?: string } | string | null;
      const operationId = typeof resp === 'string' ? resp : String(resp?.Id || '').trim();
      ChoyMessage.success(operationId ? msgs.triggered(operationId) : msgs.triggeredEmpty);
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

  function watchEmptyCatalogAutoSync() {
    watch(
      () => (store as any)?.state?.result,
      result => {
        if (autoSyncTriggered.value) return;
        // Ignore unset results (pre-fetch). Only the first settled rows payload may auto-sync.
        if (result == null || typeof result !== 'object') return;
        if ((store as any)?.state?.loading) return;
        const rows = (result as any).rows as any[] | undefined;
        if (!Array.isArray(rows)) return;
        // Consume the first settled result so a later empty search cannot force sync.
        autoSyncTriggered.value = true;
        if (rows.length > 0) return;
        void requestSync({ Force: true, IfStale: false });
      },
      // Cover stores that already hold a settled empty result before the watcher mounts.
      { immediate: true },
    );
  }

  return { syncLoading, requestSync, watchEmptyCatalogAutoSync };
}
