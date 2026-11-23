// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, ref } from 'vue';
import { defineStore, getActivePinia } from 'pinia';
import {
  resetConnectivityHooksForTests,
  setConnectivityHooks,
} from '@/core/web/client/connectivityHooks';

export type ConnectivityStatus = 'reachable' | 'unreachable';

/** Consecutive transport outages required before flipping to unreachable. */
export const TRANSPORT_OUTAGE_THRESHOLD = 2;

/** Bound hung /healthz probes so the pill Retry cannot stick on probing. */
export const HEALTHZ_PROBE_TIMEOUT_MS = 8000;

/**
 * Server-root liveness path (see internal/server/server_base_routes.go).
 * Not joined with SPA BASE_URL — `/web/` mounts still probe `/healthz`.
 */
const HEALTHZ_PATH = '/healthz';

/**
 * Session reachability for honest disconnect feedback.
 * reachable | unreachable only; Tip stream death is not authoritative here.
 */
export const useConnectivityStore = defineStore('connectivity', () => {
  const status = ref<ConnectivityStatus>('reachable');
  const transportFailureStreak = ref(0);
  const probing = ref(false);
  /** Fires once per unreachable → reachable edge (for toast consumers). */
  const recoveryGeneration = ref(0);
  /** Bumped on reachable→unreachable so in-flight probes cannot stale-recover. */
  const offlineEpoch = ref(0);

  let navigatorBound = false;
  let onOnline: (() => void) | undefined;
  let onOffline: (() => void) | undefined;

  const isUnreachable = computed(() => status.value === 'unreachable');

  function markReachable(): void {
    transportFailureStreak.value = 0;
    if (status.value === 'unreachable') {
      status.value = 'reachable';
      recoveryGeneration.value += 1;
    } else {
      status.value = 'reachable';
    }
  }

  function markUnreachable(): void {
    if (status.value !== 'unreachable') {
      offlineEpoch.value += 1;
    }
    status.value = 'unreachable';
  }

  function noteTransportSuccess(): void {
    markReachable();
  }

  function noteTransportOutage(): void {
    transportFailureStreak.value += 1;
    if (transportFailureStreak.value >= TRANSPORT_OUTAGE_THRESHOLD) {
      markUnreachable();
    }
  }

  function isBrowserOffline(target: Window | undefined): boolean {
    try {
      return target?.navigator?.onLine === false;
    } catch {
      return false;
    }
  }

  /**
   * Probe server liveness via HTTP /healthz (no business Ping RPC).
   * 2xx → reachable; otherwise stay unreachable.
   * Stale probes (epoch advanced while in flight) return without changing status.
   */
  async function probeConnectivity(
    fetchImpl: typeof fetch = fetch,
    timeoutMs: number = HEALTHZ_PROBE_TIMEOUT_MS,
  ): Promise<boolean> {
    probing.value = true;
    const epochAtStart = offlineEpoch.value;
    const abort =
      typeof AbortController !== 'undefined' ? new AbortController() : undefined;
    const delay = Math.max(1, Number(timeoutMs) || HEALTHZ_PROBE_TIMEOUT_MS);
    const timer =
      abort != null
        ? setTimeout(() => {
            try {
              abort.abort();
            } catch {
              // AbortController.abort must never escape the timer.
            }
          }, delay)
        : undefined;
    try {
      const res = await fetchImpl(HEALTHZ_PATH, {
        method: 'GET',
        cache: 'no-store',
        signal: abort?.signal,
      });
      if (epochAtStart !== offlineEpoch.value) {
        // Superseded by a newer offline edge — leave whatever recovery already did.
        return status.value === 'reachable';
      }
      if (res.ok && !isBrowserOffline(typeof window !== 'undefined' ? window : undefined)) {
        markReachable();
        return true;
      }
      markUnreachable();
      return false;
    } catch {
      if (epochAtStart !== offlineEpoch.value) {
        return status.value === 'reachable';
      }
      markUnreachable();
      return false;
    } finally {
      if (timer != null) clearTimeout(timer);
      probing.value = false;
    }
  }

  function bindNavigator(target: Window | undefined = typeof window !== 'undefined' ? window : undefined): void {
    if (!target || navigatorBound) return;
    onOffline = () => {
      markUnreachable();
    };
    onOnline = () => {
      // Online is unproven — clear streak and re-probe; do not mark reachable yet.
      transportFailureStreak.value = 0;
      void probeConnectivity();
    };
    target.addEventListener('offline', onOffline);
    target.addEventListener('online', onOnline);
    navigatorBound = true;
    if (typeof target.navigator?.onLine === 'boolean' && !target.navigator.onLine) {
      markUnreachable();
    }
  }

  function unbindNavigator(target: Window | undefined = typeof window !== 'undefined' ? window : undefined): void {
    if (!target || !navigatorBound) return;
    if (onOffline) target.removeEventListener('offline', onOffline);
    if (onOnline) target.removeEventListener('online', onOnline);
    onOffline = undefined;
    onOnline = undefined;
    navigatorBound = false;
  }

  /** Test helper: reset streak/status without touching hooks. */
  function resetForTests(): void {
    status.value = 'reachable';
    transportFailureStreak.value = 0;
    probing.value = false;
    recoveryGeneration.value = 0;
    offlineEpoch.value = 0;
  }

  return {
    status,
    isUnreachable,
    transportFailureStreak,
    probing,
    recoveryGeneration,
    offlineEpoch,
    markReachable,
    markUnreachable,
    noteTransportSuccess,
    noteTransportOutage,
    probeConnectivity,
    bindNavigator,
    unbindNavigator,
    resetForTests,
  };
});

/**
 * Safe read for shell/router code that may run before Pinia is installed.
 * Returns false when the store is unavailable (treat as reachable).
 */
export function isConnectivityUnreachable(): boolean {
  try {
    if (!getActivePinia()) return false;
    return useConnectivityStore().status === 'unreachable';
  } catch {
    return false;
  }
}

/**
 * Wire RPC lifecycle transport hooks to this Pinia store and bind navigator.
 * Call once after Pinia is installed (appSetup). Idempotent for HMR / re-setup.
 */
export function bindConnectivityLifecycle(): void {
  if (!getActivePinia()) return;
  const store = useConnectivityStore();
  setConnectivityHooks({
    onTransportSuccess: () => store.noteTransportSuccess(),
    onTransportOutage: () => store.noteTransportOutage(),
  });
  store.unbindNavigator();
  store.bindNavigator();
  // Lazy import avoids circular init with useChoyMessage ↔ store.
  void import('./recoveryToast')
    .then(mod => {
      if (!getActivePinia()) return;
      mod.bindConnectivityRecoveryToast();
    })
    .catch(() => {
      // Recovery toast is best-effort; never leave an unhandled rejection.
    });
}

/** Test helper: drop hooks registered by bindConnectivityLifecycle. */
export function unbindConnectivityLifecycleForTests(): void {
  resetConnectivityHooksForTests();
  try {
    if (getActivePinia()) {
      useConnectivityStore().unbindNavigator();
      useConnectivityStore().resetForTests();
    }
  } catch {
    // Pinia may be absent in isolated unit harnesses.
  }
}
