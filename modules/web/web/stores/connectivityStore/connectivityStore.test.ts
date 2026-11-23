// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import { getConnectivityHooks, resetConnectivityHooksForTests } from '@/core/web/client/connectivityHooks';
import {
  TRANSPORT_OUTAGE_THRESHOLD,
  bindConnectivityLifecycle,
  isConnectivityUnreachable,
  unbindConnectivityLifecycleForTests,
  useConnectivityStore,
} from './index';

beforeEach(() => {
  setActivePinia(createPinia());
  unbindConnectivityLifecycleForTests();
  useConnectivityStore().resetForTests();
});

afterEach(() => {
  unbindConnectivityLifecycleForTests();
  resetConnectivityHooksForTests();
});

test('connectivityStore: reaches unreachable after threshold transport outages', () => {
  const store = useConnectivityStore();
  expect(store.status).toBe('reachable');
  for (let i = 0; i < TRANSPORT_OUTAGE_THRESHOLD - 1; i++) {
    store.noteTransportOutage();
    expect(store.status).toBe('reachable');
  }
  store.noteTransportOutage();
  expect(store.status).toBe('unreachable');
  expect(store.isUnreachable).toBe(true);
});

test('isConnectivityUnreachable mirrors store status and is false without Pinia', () => {
  expect(isConnectivityUnreachable()).toBe(false);
  useConnectivityStore().markUnreachable();
  expect(isConnectivityUnreachable()).toBe(true);
  setActivePinia(undefined as any);
  expect(isConnectivityUnreachable()).toBe(false);
  // Restore for describe afterEach (unbind / reset need an active Pinia).
  setActivePinia(createPinia());
});

test('connectivityStore: transport success clears streak and recovers', () => {
  const store = useConnectivityStore();
  store.noteTransportOutage();
  store.noteTransportOutage();
  expect(store.status).toBe('unreachable');
  const gen = store.recoveryGeneration;
  store.noteTransportSuccess();
  expect(store.status).toBe('reachable');
  expect(store.transportFailureStreak).toBe(0);
  expect(store.recoveryGeneration).toBe(gen + 1);
});

test('connectivityStore: probeConnectivity marks reachable on 2xx', async () => {
  const store = useConnectivityStore();
  store.markUnreachable();
  const ok = await store.probeConnectivity(async () => ({ ok: true } as Response));
  expect(ok).toBe(true);
  expect(store.status).toBe('reachable');
});

test('connectivityStore: probeConnectivity stays unreachable on failure', async () => {
  const store = useConnectivityStore();
  store.markUnreachable();
  const ok = await store.probeConnectivity(async () => {
    throw new Error('Failed to fetch');
  });
  expect(ok).toBe(false);
  expect(store.status).toBe('unreachable');
});

test('connectivityStore: probeConnectivity ignores OK after a newer offline epoch', async () => {
  const store = useConnectivityStore();
  let resolveFetch!: (v: Response) => void;
  const pending = new Promise<Response>(resolve => {
    resolveFetch = resolve;
  });
  const probe = store.probeConnectivity(() => pending);
  store.markUnreachable();
  resolveFetch({ ok: true } as Response);
  const ok = await probe;
  expect(ok).toBe(false);
  expect(store.status).toBe('unreachable');
});

test('connectivityStore: stale probe OK does not reverse a newer recovery', async () => {
  const store = useConnectivityStore();
  let resolveA!: (v: Response) => void;
  const pendingA = new Promise<Response>(resolve => {
    resolveA = resolve;
  });
  const probeA = store.probeConnectivity(() => pendingA);
  store.markUnreachable();
  const okB = await store.probeConnectivity(async () => ({ ok: true } as Response));
  expect(okB).toBe(true);
  expect(store.status).toBe('reachable');
  resolveA({ ok: true } as Response);
  await probeA;
  expect(store.status).toBe('reachable');
});

test('connectivityStore: repeated markUnreachable does not bump offline epoch', () => {
  const store = useConnectivityStore();
  store.markUnreachable();
  const epoch = store.offlineEpoch;
  store.markUnreachable();
  expect(store.offlineEpoch).toBe(epoch);
});

test('connectivityStore: probeConnectivity non-OK response marks unreachable', async () => {
  const store = useConnectivityStore();
  const ok = await store.probeConnectivity(async () => ({ ok: false } as Response));
  expect(ok).toBe(false);
  expect(store.status).toBe('unreachable');
});

test('connectivityStore: probeConnectivity timeout aborts hung fetch', async () => {
  const store = useConnectivityStore();
  const ok = await store.probeConnectivity(async (_url, init) => {
    return await new Promise<Response>((_resolve, reject) => {
      const signal = init && (init as RequestInit).signal;
      if (!signal) {
        reject(new Error('missing signal'));
        return;
      }
      signal.addEventListener('abort', () => {
        reject(Object.assign(new Error('Aborted'), { name: 'AbortError' }));
      });
    });
  }, 20);
  expect(ok).toBe(false);
  expect(store.status).toBe('unreachable');
  expect(store.probing).toBe(false);
});

test('connectivityStore: navigator offline marks unreachable', () => {
  const store = useConnectivityStore();
  const listeners: Record<string, Array<() => void>> = {};
  const fakeWindow = {
    navigator: { onLine: true },
    addEventListener(type: string, fn: () => void) {
      (listeners[type] ||= []).push(fn);
    },
    removeEventListener(type: string, fn: () => void) {
      listeners[type] = (listeners[type] || []).filter(x => x !== fn);
    },
  } as unknown as Window;
  store.bindNavigator(fakeWindow);
  listeners.offline?.[0]?.();
  expect(store.status).toBe('unreachable');
  store.unbindNavigator(fakeWindow);
});

test('connectivityStore: navigator online re-probes', async () => {
  const store = useConnectivityStore();
  const listeners: Record<string, Array<() => void>> = {};
  const fakeWindow = {
    navigator: { onLine: true },
    addEventListener(type: string, fn: () => void) {
      (listeners[type] ||= []).push(fn);
    },
    removeEventListener(type: string, fn: () => void) {
      listeners[type] = (listeners[type] || []).filter(x => x !== fn);
    },
  } as unknown as Window;
  store.bindNavigator(fakeWindow);
  store.markUnreachable();
  const g = globalThis as { fetch?: typeof fetch };
  const origFetch = g.fetch;
  let probed = 0;
  g.fetch = (async () => {
    probed += 1;
    return { ok: true } as Response;
  }) as typeof fetch;
  try {
    listeners.online?.[0]?.();
    await new Promise<void>(resolve => setTimeout(resolve, 0));
    expect(probed).toBe(1);
    expect(store.status).toBe('reachable');
  } finally {
    g.fetch = origFetch;
    store.unbindNavigator(fakeWindow);
  }
});

test('connectivityStore: bindNavigator starts unreachable when already offline', () => {
  const store = useConnectivityStore();
  const fakeWindow = {
    navigator: { onLine: false },
    addEventListener() {},
    removeEventListener() {},
  } as unknown as Window;
  store.bindNavigator(fakeWindow);
  expect(store.status).toBe('unreachable');
  store.unbindNavigator(fakeWindow);
});

test('bindConnectivityLifecycle: hooks drive the store', () => {
  bindConnectivityLifecycle();
  const hooks = getConnectivityHooks();
  expect(typeof hooks.onTransportSuccess).toBe('function');
  expect(typeof hooks.onTransportOutage).toBe('function');
  const store = useConnectivityStore();
  hooks.onTransportOutage?.();
  hooks.onTransportOutage?.();
  expect(store.status).toBe('unreachable');
  hooks.onTransportSuccess?.();
  expect(store.status).toBe('reachable');
});

test('bindConnectivityLifecycle: rebind is idempotent', () => {
  bindConnectivityLifecycle();
  bindConnectivityLifecycle();
  const hooks = getConnectivityHooks();
  expect(typeof hooks.onTransportOutage).toBe('function');
});

test('isConnectivityUnreachable: broken Pinia is treated as reachable', () => {
  setActivePinia({} as any);
  expect(isConnectivityUnreachable()).toBe(false);
  setActivePinia(createPinia());
});

test('unbindConnectivityLifecycleForTests: tolerates broken Pinia', () => {
  setActivePinia({} as any);
  unbindConnectivityLifecycleForTests();
  setActivePinia(createPinia());
});

test('connectivityStore: stale probe catch does not reverse a newer recovery', async () => {
  const store = useConnectivityStore();
  let rejectA!: (e: Error) => void;
  const pendingA = new Promise<Response>((_resolve, reject) => {
    rejectA = reject;
  });
  const probeA = store.probeConnectivity(() => pendingA);
  store.markUnreachable();
  const okB = await store.probeConnectivity(async () => ({ ok: true } as Response));
  expect(okB).toBe(true);
  expect(store.status).toBe('reachable');
  rejectA(new Error('Failed to fetch'));
  await probeA;
  expect(store.status).toBe('reachable');
});

test('connectivityStore: probeConnectivity abort timer swallows abort errors', async () => {
  const store = useConnectivityStore();
  const RealAC = globalThis.AbortController;
  class ThrowingAbortController {
    signal = { aborted: false, addEventListener() {}, removeEventListener() {} };
    abort() {
      throw new Error('abort boom');
    }
  }
  (globalThis as { AbortController?: typeof AbortController }).AbortController =
    ThrowingAbortController as unknown as typeof AbortController;
  try {
    const ok = await store.probeConnectivity(async () => {
      await new Promise<void>(resolve => setTimeout(resolve, 40));
      return { ok: false } as Response;
    }, 5);
    expect(ok).toBe(false);
    expect(store.probing).toBe(false);
  } finally {
    (globalThis as { AbortController?: typeof AbortController }).AbortController = RealAC;
  }
});

test('connectivityStore: superseded probe finally does not clear newer probing', async () => {
  const store = useConnectivityStore();
  let resolveA!: (v: Response) => void;
  const pendingA = new Promise<Response>(resolve => {
    resolveA = resolve;
  });
  const probeA = store.probeConnectivity(() => pendingA);
  expect(store.probing).toBe(true);
  let resolveB!: (v: Response) => void;
  const pendingB = new Promise<Response>(resolve => {
    resolveB = resolve;
  });
  const probeB = store.probeConnectivity(() => pendingB);
  expect(store.probing).toBe(true);
  resolveA({ ok: false } as Response);
  await probeA;
  // A finished first but B is still in flight — probing must stay true.
  expect(store.probing).toBe(true);
  resolveB({ ok: true } as Response);
  await probeB;
  expect(store.probing).toBe(false);
});

test('connectivityStore: browser offline check tolerates navigator access errors', async () => {
  const store = useConnectivityStore();
  store.markUnreachable();
  const desc = Object.getOwnPropertyDescriptor(window, 'navigator');
  Object.defineProperty(window, 'navigator', {
    configurable: true,
    get() {
      throw new Error('navigator boom');
    },
  });
  try {
    const ok = await store.probeConnectivity(async () => ({ ok: true } as Response));
    // Throwing navigator is treated as not-offline, so OK probe recovers.
    expect(ok).toBe(true);
    expect(store.status).toBe('reachable');
  } finally {
    if (desc) Object.defineProperty(window, 'navigator', desc);
    else delete (window as { navigator?: Navigator }).navigator;
  }
});
