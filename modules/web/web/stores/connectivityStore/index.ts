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

  /**
   * Probe server liveness via HTTP /healthz (no business Ping RPC).
   * 2xx → reachable; otherwise stay unreachable.
   */
  async function probeConnectivity(fetchImpl: typeof fetch = fetch): Promise<boolean> {
    probing.value = true;
    try {
      const res = await fetchImpl(HEALTHZ_PATH, { method: 'GET', cache: 'no-store' });
      if (res.ok) {
        markReachable();
        return true;
      }
      markUnreachable();
      return false;
    } catch {
      markUnreachable();
      return false;
    } finally {
      probing.value = false;
    }
  }

  function bindNavigator(target: Window | undefined = typeof window !== 'undefined' ? window : undefined): void {
    if (!target || navigatorBound) return;
    onOffline = () => {
      markUnreachable();
    };
    onOnline = () => {
      // Online is optimistic but unproven — wait for transport success or probe.
      transportFailureStreak.value = 0;
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
  }

  return {
    status,
    isUnreachable,
    transportFailureStreak,
    probing,
    recoveryGeneration,
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
 * Wire RPC lifecycle transport hooks to this Pinia store and bind navigator.
 * Call once after Pinia is installed (appSetup).
 */
export function bindConnectivityLifecycle(): void {
  if (!getActivePinia()) return;
  const store = useConnectivityStore();
  setConnectivityHooks({
    onTransportSuccess: () => store.noteTransportSuccess(),
    onTransportOutage: () => store.noteTransportOutage(),
  });
  store.bindNavigator();
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
