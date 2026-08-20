// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { onBeforeUnmount, watch, type Ref } from 'vue';
import { onTips, subscribeThread } from '@/core/web/tip';

const POLL_FALLBACK_MS = 30_000;

export function useChatterThreadTips(
  model: Ref<string>,
  resId: Ref<string | undefined>,
  refresh: () => Promise<void>
): void {
  let tipController: AbortController | null = null;
  let pollTimer: ReturnType<typeof setInterval> | undefined;

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
    const threadModel = String(model.value || '').trim();
    const threadResId = String(resId.value || '').trim();
    if (!threadModel || !threadResId) return;

    tipController = new AbortController();
    const signal = tipController.signal;
    try {
      await onTips(
        subscribeThread(threadModel, threadResId, signal),
        async () => {
          await refresh();
        },
        signal
      );
    } catch {
      // Stream error; fall through to poll fallback when still subscribed.
    } finally {
      if (!signal.aborted) {
        startPollFallback();
      }
    }
  }

  watch([model, resId], () => {
    void startTips();
  }, { immediate: true });

  onBeforeUnmount(() => {
    stopTips();
  });
}
