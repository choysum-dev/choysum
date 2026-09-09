// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { inject, onScopeDispose, watch, type InjectionKey, type Ref } from 'vue';
import { onTips as defaultOnTips, subscribeThread as defaultSubscribeThread } from '@/core/web/tip';

const POLL_FALLBACK_MS = 30_000;

export type UseChatterThreadTipsDeps = {
  onTips?: typeof defaultOnTips;
  subscribeThread?: typeof defaultSubscribeThread;
  /** Override poll interval (default 30s); tests pass a short value with real timers. */
  pollFallbackMs?: number;
};

/** Optional override for `useChatterThreadTips` (unit harness). */
export type UseChatterThreadTipsFn = typeof useChatterThreadTips;
export const UseChatterThreadTipsKey: InjectionKey<UseChatterThreadTipsFn> = Symbol('UseChatterThreadTips');

/** Resolve injected tips factory, else the product default. */
export function useInjectedChatterThreadTips(
  model: Ref<string>,
  resId: Ref<string | undefined>,
  refresh: () => Promise<void>,
  deps?: UseChatterThreadTipsDeps
): void {
  const override = inject(UseChatterThreadTipsKey, null);
  (override ?? useChatterThreadTips)(model, resId, refresh, deps);
}

export function useChatterThreadTips(
  model: Ref<string>,
  resId: Ref<string | undefined>,
  refresh: () => Promise<void>,
  deps?: UseChatterThreadTipsDeps
): void {
  const onTips = deps?.onTips ?? defaultOnTips;
  const subscribeThread = deps?.subscribeThread ?? defaultSubscribeThread;
  const pollFallbackMs = deps?.pollFallbackMs ?? POLL_FALLBACK_MS;

  let tipController: AbortController | null = null;
  let pollTimer: ReturnType<typeof setInterval> | undefined;

  function stopPollFallback(): void {
    if (pollTimer != null) {
      clearInterval(pollTimer);
    }
    pollTimer = undefined;
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

  // onScopeDispose works in component setup and standalone effectScope (unit tests).
  onScopeDispose(() => {
    stopTips();
  });
}
