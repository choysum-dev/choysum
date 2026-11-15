// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { watch, type WatchStopHandle } from 'vue';
import NProgress from 'nprogress';
import {
  useLoadingBarVisible,
  useLoadingSlowHintVisible,
} from '@/core/web/client/loadingRegistry';

let bound = false;
let stopBar: WatchStopHandle | null = null;
let stopHint: WatchStopHandle | null = null;

/**
 * Drive NProgress from the loading registry bar channel (200ms delayed).
 * Call once from App setup. Router no longer starts/stops NProgress directly.
 */
export function bindChromeLoadingBar(nprogress: typeof NProgress = NProgress): () => void {
  if (bound) return unbindChromeLoadingBar;
  bound = true;
  nprogress.configure({ showSpinner: false });
  const barVisible = useLoadingBarVisible();
  stopBar = watch(
    barVisible,
    show => {
      if (show) nprogress.start();
      else nprogress.done();
    },
    { immediate: true },
  );
  return unbindChromeLoadingBar;
}

export function unbindChromeLoadingBar(): void {
  stopBar?.();
  stopBar = null;
  stopHint?.();
  stopHint = null;
  bound = false;
  try {
    NProgress.done();
  } catch {
    // ignore
  }
}

/** Subscribe to slow-hint visibility (for hosts that need a JS side-effect). */
export function watchLoadingSlowHint(onChange: (show: boolean) => void): WatchStopHandle {
  const hint = useLoadingSlowHintVisible();
  stopHint = watch(hint, onChange, { immediate: true });
  return stopHint;
}
