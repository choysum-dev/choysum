// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, onBeforeUnmount, ref } from 'vue';
import { onTips, subscribeNotifications } from '@/core/web/tip';
import { getNotificationStore } from './chatterStores';
import type { InboxNotificationRow } from './chatterTypes';

export type { InboxNotificationRow } from './chatterTypes';

const INBOX_FIELDS = ['Id', 'MessageId', 'Model', 'ResId', 'AuthorUid', 'IsRead', 'CreatedAt'] as const;
const POLL_FALLBACK_MS = 30_000;

export function useNotificationInbox(enabled: () => boolean) {
  const rows = ref<InboxNotificationRow[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);
  const notificationStore = getNotificationStore();
  let tipController: AbortController | null = null;
  let pollTimer: ReturnType<typeof setInterval> | undefined;

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
    try {
      await notificationStore.MarkRead([id]);
      await refresh();
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err);
    }
  }

  async function markAllRead(): Promise<void> {
    try {
      await notificationStore.MarkAllRead();
      await refresh();
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err);
    }
  }

  function stopPollFallback(): void {
    if (pollTimer != null) {
      clearInterval(pollTimer);
      pollTimer = undefined;
    }
  }

  function startPollFallback(): void {
    stopPollFallback();
    pollTimer = setInterval(() => {
      void refresh();
    }, POLL_FALLBACK_MS);
  }

  function stopTips(): void {
    tipController?.abort();
    tipController = null;
    stopPollFallback();
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
      // Stream error; fall through to poll fallback when still subscribed.
    } finally {
      if (!signal.aborted) {
        startPollFallback();
      }
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
