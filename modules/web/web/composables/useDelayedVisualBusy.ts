// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  onBeforeUnmount,
  onScopeDispose,
  ref,
  toValue,
  watch,
  type MaybeRefOrGetter,
  type Ref,
} from 'vue';
import { loadingTiming } from '@/core/web/client/loadingRegistry';

/**
 * Delay showing a busy overlay until `busy` has stayed true for `delayMs`.
 * Clears immediately when busy becomes false (so fast work never flashes).
 * Does not affect aria-busy / inert — callers keep those on the logical busy flag.
 */
export function useDelayedVisualBusy(
  busy: MaybeRefOrGetter<boolean>,
  delayMs: MaybeRefOrGetter<number> = () => loadingTiming.SURFACE_BUSY_DELAY_MS,
): Ref<boolean> {
  const visualBusy = ref(false);
  let timer: ReturnType<typeof setTimeout> | null = null;

  function clearTimer(): void {
    if (timer == null) return;
    clearTimeout(timer);
    timer = null;
  }

  // Separate sources so a busy getter that reallocates does not reset the timer
  // when the normalized busy/delay values are unchanged.
  watch(
    [() => !!toValue(busy), () => Math.max(0, Number(toValue(delayMs)) || 0)],
    ([on, delay]) => {
      clearTimer();
      if (!on) {
        visualBusy.value = false;
        return;
      }
      if (delay === 0) {
        visualBusy.value = true;
        return;
      }
      timer = setTimeout(() => {
        timer = null;
        visualBusy.value = true;
      }, delay);
    },
    { immediate: true },
  );

  // Component instances + detached effectScope (tests / rare hosts).
  onBeforeUnmount(clearTimer);
  onScopeDispose(clearTimer);

  return visualBusy;
}
