// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { nextTick, onActivated, onUnmounted } from 'vue';
import {
  hasActiveSurfaceBusy,
  settleNavigationIfActive,
} from '@/core/web/client/loadingRegistry';

/**
 * Drop leftover navigation chrome when no surface owns the ticket.
 * Used after KeepAlive reactivation paints cached content without a new surface.
 */
export function settleNavChromeIfNoSurface(): void {
  if (!hasActiveSurfaceBusy()) settleNavigationIfActive();
}

/**
 * KeepAlive reactivation often paints cached content without opening a new
 * surface. Settle leftover navigation chrome on activate when idle so the top
 * bar / handoff do not sit for NAV_IDLE_SETTLE_MS.
 */
export function useSettleNavChromeOnActivate(): void {
  let disposed = false;
  onUnmounted(() => {
    disposed = true;
  });
  onActivated(() => {
    void nextTick(() => {
      if (disposed) return;
      settleNavChromeIfNoSurface();
    });
  });
}
