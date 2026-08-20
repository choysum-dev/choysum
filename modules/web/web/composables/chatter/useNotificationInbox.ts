// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, onBeforeUnmount, ref } from 'vue';
import { onTips, subscribeNotifications } from '@/core/web/tip';
import { getNotificationStore } from './chatterStores';
import type { InboxNotificationRow } from './chatterTypes';

export type { InboxNotificationRow } from './chatterTypes';

const INBOX_FIELDS = ['Id', 'MessageId', 'Model', 'ResId', 'AuthorUid', 'IsRead', 'CreatedAt'] as const;

export function useNotificationInbox(enabled: () => boolean) {
  const rows = ref<InboxNotificationRow[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);
  const notificationStore = getNotificationStore();
  let tipController: AbortController | null = null;

  const unreadCount = computed(() => rows.value.filter(row => row?.IsRead !== true).length);

  async function refresh(): Promise<void> {
    if (!enabled()) {
      rows.value = [];
      error.value = null;
      return;
    }
    loading.value = true;
    error.value = null;
    try {
      rows.value = await notificationStore.SearchInbox({ fields: [...INBOX_FIELDS], limit: 20 });
    } catch (err) {
      rows.value = [];
      error.value = err instanceof Error ? err.message : String(err);
    } finally {
      loading.value = false;
    }
  }

  async function markRead(notificationId: string): Promise<void> {
    const id = String(notificationId || '').trim();
    if (!id) return;
    await notificationStore.MarkRead([id]);
    await refresh();
  }

  async function markAllRead(): Promise<void> {
    await notificationStore.MarkAllRead();
    await refresh();
  }

  function stopTips(): void {
    tipController?.abort();
    tipController = null;
  }

  async function startTips(): Promise<void> {
    stopTips();
    if (!enabled()) return;
    tipController = new AbortController();
    const signal = tipController.signal;
    try {
      await onTips(subscribeNotifications(signal), async () => {
        await refresh();
      }, signal);
    } catch {
      // Best-effort; inbox can still be opened manually.
    }
  }

  async function activate(): Promise<void> {
    await refresh();
    void startTips();
  }

  function deactivate(): void {
    stopTips();
    rows.value = [];
    error.value = null;
  }

  onBeforeUnmount(() => {
    deactivate();
  });

  return {
    rows,
    loading,
    error,
    unreadCount,
    refresh,
    markRead,
    markAllRead,
    activate,
    deactivate,
  };
}
