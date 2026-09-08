// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, inject, onScopeDispose, ref, type InjectionKey } from 'vue';
import { onTips as defaultOnTips, subscribeNotifications as defaultSubscribeNotifications } from '@/core/web/tip';
import { getNotificationStore as defaultGetNotificationStore } from './chatterStores';
import type { InboxNotificationRow } from './chatterTypes';

export type { InboxNotificationRow } from './chatterTypes';

const INBOX_FIELDS = ['Id', 'MessageId', 'Model', 'ResId', 'AuthorUid', 'IsRead', 'CreatedAt'] as const;
const POLL_FALLBACK_MS = 30_000;

export type UseNotificationInboxDeps = {
  getNotificationStore?: typeof defaultGetNotificationStore;
  onTips?: typeof defaultOnTips;
  subscribeNotifications?: typeof defaultSubscribeNotifications;
  /** Override poll interval (default 30s); tests pass a short value with real timers. */
  pollFallbackMs?: number;
};

/** Optional override for `useNotificationInbox` (unit harness). */
export type UseNotificationInboxFn = typeof useNotificationInbox;
export const UseNotificationInboxKey: InjectionKey<UseNotificationInboxFn> = Symbol('UseNotificationInbox');

/** Resolve injected inbox factory, else the product default. */
export function useInjectedNotificationInbox(
  enabled: () => boolean,
  deps?: UseNotificationInboxDeps
): ReturnType<typeof useNotificationInbox> {
  const override = inject(UseNotificationInboxKey, null);
  return (override ?? useNotificationInbox)(enabled, deps);
}

export function useNotificationInbox(enabled: () => boolean, deps?: UseNotificationInboxDeps) {
  const rows = ref<InboxNotificationRow[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);
  const resolveNotificationStore = deps?.getNotificationStore ?? defaultGetNotificationStore;
  const onTips = deps?.onTips ?? defaultOnTips;
  const subscribeNotifications = deps?.subscribeNotifications ?? defaultSubscribeNotifications;
  const pollFallbackMs = deps?.pollFallbackMs ?? POLL_FALLBACK_MS;
  const notificationStore = resolveNotificationStore();
  let tipController: AbortController | null = null;
  let pollTimer: ReturnType<typeof setInterval> | undefined;
  let refreshGeneration = 0;
  let sessionGeneration = 0;

  const unreadCount = computed(() => rows.value.filter(row => row?.IsRead !== true).length);

  async function refresh(): Promise<void> {
    const generation = ++refreshGeneration;
    if (!enabled()) {
      if (generation !== refreshGeneration) return;
      rows.value = [];
      error.value = null;
      loading.value = false;
      return;
    }
    loading.value = true;
    error.value = null;
    try {
      const nextRows = await notificationStore.SearchInbox({ fields: [...INBOX_FIELDS], limit: 20 });
      if (generation !== refreshGeneration) return;
      rows.value = nextRows;
    } catch (err) {
      if (generation !== refreshGeneration) return;
      rows.value = [];
      error.value = err instanceof Error ? err.message : String(err);
    } finally {
      if (generation === refreshGeneration) loading.value = false;
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
    }, pollFallbackMs);
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
    const session = sessionGeneration;
    await refresh();
    if (session !== sessionGeneration || !enabled()) return;
    void startTips();
  }

  function deactivate(): void {
    sessionGeneration += 1;
    refreshGeneration += 1;
    stopTips();
    rows.value = [];
    error.value = null;
    loading.value = false;
  }

  // onScopeDispose works in component setup and standalone effectScope (unit tests).
  onScopeDispose(() => {
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
