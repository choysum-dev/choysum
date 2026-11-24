// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, getActivePinia, setActivePinia } from 'pinia';
import { buildPageMountGlobal } from '@choysum/page-mount';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import { h } from 'vue';
import ChoyButton from '../button/ChoyButton.vue';
import ChoyViewLoadError from './ChoyViewLoadError.vue';
import {
  clearViewLoadError,
  setViewLoadError,
  VIEW_LOAD_RETRY_AFTER_RECOVERY_MS,
  viewLoadError,
} from '../../router/viewLoadError';
import { useConnectivityStore } from '../../stores/connectivityStore';

beforeEach(() => {
  setActivePinia(createPinia());
  useConnectivityStore().resetForTests();
  clearViewLoadError();
  stubSfc(ChoyButton as any, {
    name: 'ChoyButton',
    inheritAttrs: false,
    props: {
      type: { type: String, default: 'button' },
      variant: { type: String, default: 'default' },
      disabled: { type: Boolean, default: false },
    },
    emits: ['click'],
    setup(props: any, { slots, emit, attrs }: any) {
      return () =>
        h(
          'button',
          {
            type: props.type || 'button',
            ...attrs,
            disabled: props.disabled || undefined,
            onClick: (e: Event) => emit('click', e),
          },
          slots.default?.(),
        );
    },
  });
});

afterEach(() => {
  restoreSfc(ChoyButton as any);
  clearViewLoadError();
  try {
    if (getActivePinia()) useConnectivityStore().resetForTests();
  } catch {
    // Pinia may be torn down by page-mount plugins; next beforeEach recreates it.
  }
});

function mountLoadError() {
  const pinia = createPinia();
  setActivePinia(pinia);
  const { plugins } = buildPageMountGlobal({
    route: { name: 'CountryList', path: '/web/base/countries', fullPath: '/web/base/countries' },
  });
  return mountApp(ChoyViewLoadError as any, { plugins: [pinia, ...plugins] });
}

test('ChoyViewLoadError: Asset primary is Retry loading; unreachable copy kept', async () => {
  setViewLoadError({
    message: 'Failed to fetch dynamically imported module',
    fullPath: '/web/base/countries',
  });
  const mounted = mountLoadError();
  useConnectivityStore().markUnreachable();
  await flushPromises();
  expect(mounted.q('[data-testid=choy-view-load-error]')).not.toBeNull();
  expect(mounted.text()).toContain('Connection lost');
  expect(mounted.text()).toContain('Cannot reach the server');
  expect(mounted.text()).toContain('Retry loading');
  expect(mounted.text()).toContain('Reload page');
  expect(mounted.text()).not.toContain('Could not load this data');
  mounted.unmount();
});

test('ChoyViewLoadError: reachable session uses Failed to load this view copy', async () => {
  setViewLoadError({
    message: 'Loading chunk failed',
    fullPath: '/web/base/banks',
  });
  const mounted = mountLoadError();
  await flushPromises();
  expect(mounted.text()).toContain('Failed to load this view');
  expect(mounted.text()).toContain('page resources could not be loaded');
  mounted.unmount();
});

test('ChoyViewLoadError: Retry loading clears error and replace-navigates', async () => {
  setViewLoadError({
    message: 'Failed to fetch dynamically imported module',
    fullPath: '/web/base/countries?retry=1',
  });
  const mounted = mountLoadError();
  await flushPromises();
  mounted.click('[data-testid=choy-view-load-error-retry]');
  await flushPromises();
  expect(viewLoadError.value).toBeNull();
  mounted.unmount();
});

test('ChoyViewLoadError: same-path Retry uses history go(0) instead of replace', async () => {
  setViewLoadError({
    message: 'Failed to fetch dynamically imported module',
    fullPath: '/web/base/countries',
  });
  const pinia = createPinia();
  setActivePinia(pinia);
  let goCalls = 0;
  const { plugins } = buildPageMountGlobal({
    route: { name: 'CountryList', path: '/web/base/countries', fullPath: '/web/base/countries' },
    router: {
      go: () => {
        goCalls += 1;
      },
      replace: async () => {
        throw new Error('replace must not run for same-path retry');
      },
    },
  });
  const mounted = mountApp(ChoyViewLoadError as any, { plugins: [pinia, ...plugins] });
  await flushPromises();
  mounted.click('[data-testid=choy-view-load-error-retry]');
  await flushPromises();
  expect(viewLoadError.value).toBeNull();
  expect(goCalls).toBe(1);
  mounted.unmount();
});

test('ChoyViewLoadError: Reload page calls location.reload', async () => {
  setViewLoadError({
    message: 'Failed to fetch dynamically imported module',
    fullPath: '/web/base/countries',
  });
  const mounted = mountLoadError();
  await flushPromises();
  const g = globalThis as { window?: { location?: { reload?: () => void } } };
  const loc = g.window?.location;
  let reloads = 0;
  if (loc) {
    const orig = loc.reload;
    loc.reload = () => {
      reloads += 1;
    };
    try {
      mounted.click('[data-testid=choy-view-load-error-reload]');
      await flushPromises();
      expect(reloads).toBe(1);
    } finally {
      loc.reload = orig;
    }
  }
  mounted.unmount();
});

type FakeTimer = { id: number; due: number; fn: () => void };

function installFakeTimers() {
  const g = globalThis as {
    setTimeout: typeof setTimeout;
    clearTimeout: typeof clearTimeout;
  };
  const realSetTimeout = g.setTimeout.bind(globalThis);
  const realClearTimeout = g.clearTimeout.bind(globalThis);
  let now = 0;
  let nextId = 1;
  const timers: FakeTimer[] = [];
  g.setTimeout = ((fn: unknown, ms?: number, ...args: unknown[]) => {
    const id = nextId++;
    const delay = Number(ms) || 0;
    timers.push({
      id,
      due: now + delay,
      fn: () => {
        if (typeof fn === 'function') (fn as (...a: unknown[]) => void)(...args);
      },
    });
    return id as unknown as ReturnType<typeof setTimeout>;
  }) as unknown as typeof setTimeout;
  g.clearTimeout = ((id: ReturnType<typeof setTimeout>) => {
    const i = timers.findIndex((t) => t.id === (id as unknown as number));
    if (i >= 0) timers.splice(i, 1);
  }) as unknown as typeof clearTimeout;

  async function flushMicrotasks() {
    await Promise.resolve();
    await Promise.resolve();
  }

  async function advance(ms: number) {
    const target = now + ms;
    for (;;) {
      timers.sort((a, b) => a.due - b.due || a.id - b.id);
      const next = timers[0];
      if (!next || next.due > target) break;
      now = next.due;
      timers.shift();
      next.fn();
      await flushMicrotasks();
    }
    now = target;
    await flushMicrotasks();
  }

  async function flush() {
    await flushMicrotasks();
    await advance(0);
  }

  function restore() {
    g.setTimeout = realSetTimeout;
    g.clearTimeout = realClearTimeout;
    timers.length = 0;
  }

  return { advance, flush, restore, realSetTimeout };
}

function mountLoadErrorWithGo(opts?: {
  routeFullPath?: string;
  replace?: () => Promise<void>;
}) {
  const pinia = createPinia();
  setActivePinia(pinia);
  let goCalls = 0;
  let replaceCalls = 0;
  const routeFullPath = opts?.routeFullPath || '/web/base/countries';
  const { plugins } = buildPageMountGlobal({
    route: { name: 'CountryList', path: routeFullPath, fullPath: routeFullPath },
    router: {
      go: () => {
        goCalls += 1;
      },
      replace: async () => {
        replaceCalls += 1;
        if (opts?.replace) {
          await opts.replace();
          return;
        }
        throw new Error('replace must not run for same-path recovery retry');
      },
    },
  });
  const mounted = mountApp(ChoyViewLoadError as any, { plugins: [pinia, ...plugins] });
  return {
    mounted,
    goCalls: () => goCalls,
    replaceCalls: () => replaceCalls,
    store: useConnectivityStore(),
  };
}

test('ChoyViewLoadError: soft-retries after connectivity recovery while error surface is shown', async () => {
  const clock = installFakeTimers();
  try {
    setViewLoadError({
      message: 'Failed to fetch dynamically imported module',
      fullPath: '/web/base/countries',
    });
    const { mounted, goCalls, store } = mountLoadErrorWithGo();
    await clock.flush();
    store.markUnreachable();
    await clock.flush();
    store.markReachable();
    await clock.flush();
    await clock.advance(VIEW_LOAD_RETRY_AFTER_RECOVERY_MS);
    expect(viewLoadError.value).toBeNull();
    expect(goCalls()).toBe(1);
    mounted.unmount();
  } finally {
    clock.restore();
  }
});

test('ChoyViewLoadError: soft-retries failed destination while currentRoute is still previous page', async () => {
  const clock = installFakeTimers();
  try {
    // Mid-navigation chunk failure: error records destination, router has not committed yet.
    setViewLoadError({
      message: 'Failed to fetch dynamically imported module',
      fullPath: '/web/base/countries',
    });
    const { mounted, goCalls, replaceCalls, store } = mountLoadErrorWithGo({
      routeFullPath: '/web/home',
      replace: async () => {},
    });
    await clock.flush();
    store.markUnreachable();
    store.markReachable();
    await clock.flush();
    await clock.advance(VIEW_LOAD_RETRY_AFTER_RECOVERY_MS);
    expect(goCalls()).toBe(0);
    expect(replaceCalls()).toBe(1);
    expect(viewLoadError.value).toBeNull();
    mounted.unmount();
  } finally {
    clock.restore();
  }
});

test('ChoyViewLoadError: clears pending recovery timer on unmount', async () => {
  const clock = installFakeTimers();
  try {
    setViewLoadError({
      message: 'Failed to fetch dynamically imported module',
      fullPath: '/web/base/countries',
    });
    const { mounted, goCalls, store } = mountLoadErrorWithGo();
    await clock.flush();
    store.markUnreachable();
    store.markReachable();
    await clock.flush();
    mounted.unmount();
    await clock.advance(VIEW_LOAD_RETRY_AFTER_RECOVERY_MS + 50);
    expect(goCalls()).toBe(0);
    expect(viewLoadError.value).not.toBeNull();
    clearViewLoadError();
  } finally {
    clock.restore();
  }
});

test('ChoyViewLoadError: flapping recovery before settle soft-retries once', async () => {
  const clock = installFakeTimers();
  try {
    setViewLoadError({
      message: 'Failed to fetch dynamically imported module',
      fullPath: '/web/base/countries',
    });
    const { mounted, goCalls, store } = mountLoadErrorWithGo();
    await clock.flush();
    store.markUnreachable();
    store.markReachable();
    await clock.flush();
    store.markUnreachable();
    store.markReachable();
    await clock.flush();
    await clock.advance(VIEW_LOAD_RETRY_AFTER_RECOVERY_MS);
    expect(goCalls()).toBe(1);
    expect(viewLoadError.value).toBeNull();
    mounted.unmount();
  } finally {
    clock.restore();
  }
});

test('ChoyViewLoadError: does not auto-retry again when soft-retry re-sets the same path', async () => {
  const clock = installFakeTimers();
  try {
    setViewLoadError({
      message: 'Failed to fetch dynamically imported module',
      fullPath: '/web/base/countries',
    });
    const { mounted, goCalls, store } = mountLoadErrorWithGo();
    await clock.flush();
    store.markUnreachable();
    store.markReachable();
    await clock.flush();
    await clock.advance(VIEW_LOAD_RETRY_AFTER_RECOVERY_MS);
    expect(goCalls()).toBe(1);
    // Re-set before the one-shot reset window (settle + 100ms) elapses.
    setViewLoadError({
      message: 'Failed to fetch dynamically imported module',
      fullPath: '/web/base/countries',
    });
    store.markUnreachable();
    store.markReachable();
    await clock.flush();
    await clock.advance(VIEW_LOAD_RETRY_AFTER_RECOVERY_MS);
    expect(goCalls()).toBe(1);
    mounted.unmount();
    clearViewLoadError();
  } finally {
    clock.restore();
  }
});

test('ChoyViewLoadError: allows another auto-retry after a clear sticks', async () => {
  const clock = installFakeTimers();
  try {
    setViewLoadError({
      message: 'Failed to fetch dynamically imported module',
      fullPath: '/web/base/countries',
    });
    const { mounted, goCalls, store } = mountLoadErrorWithGo();
    await clock.flush();
    store.markUnreachable();
    store.markReachable();
    await clock.flush();
    await clock.advance(VIEW_LOAD_RETRY_AFTER_RECOVERY_MS);
    expect(goCalls()).toBe(1);
    expect(viewLoadError.value).toBeNull();
    await clock.advance(VIEW_LOAD_RETRY_AFTER_RECOVERY_MS + 100);
    setViewLoadError({
      message: 'Failed to fetch dynamically imported module',
      fullPath: '/web/base/countries',
    });
    store.markUnreachable();
    store.markReachable();
    await clock.flush();
    await clock.advance(VIEW_LOAD_RETRY_AFTER_RECOVERY_MS);
    expect(goCalls()).toBe(2);
    mounted.unmount();
  } finally {
    clock.restore();
  }
});

test('ChoyViewLoadError: skips auto-retry when navigating away from the failed view', async () => {
  const clock = installFakeTimers();
  try {
    setViewLoadError({
      message: 'Failed to fetch dynamically imported module',
      fullPath: '/web/base/countries',
    });
    const pinia = createPinia();
    setActivePinia(pinia);
    let goCalls = 0;
    const beforeGuards: Array<(to: { fullPath: string }) => void> = [];
    const afterHooks: Array<() => void> = [];
    const { plugins } = buildPageMountGlobal({
      route: { name: 'CountryList', path: '/web/base/countries', fullPath: '/web/base/countries' },
      router: {
        go: () => {
          goCalls += 1;
        },
        replace: async () => {
          throw new Error('replace must not run');
        },
        beforeEach: (fn: (to: { fullPath: string }) => void) => {
          beforeGuards.push(fn);
          return () => {
            const i = beforeGuards.indexOf(fn);
            if (i >= 0) beforeGuards.splice(i, 1);
          };
        },
        afterEach: (fn: () => void) => {
          afterHooks.push(fn);
          return () => {
            const i = afterHooks.indexOf(fn);
            if (i >= 0) afterHooks.splice(i, 1);
          };
        },
      },
    });
    const mounted = mountApp(ChoyViewLoadError as any, { plugins: [pinia, ...plugins] });
    await clock.flush();
    const store = useConnectivityStore();
    store.markUnreachable();
    store.markReachable();
    await clock.flush();
    for (const g of beforeGuards) g({ fullPath: '/web/base/banks' });
    await clock.advance(VIEW_LOAD_RETRY_AFTER_RECOVERY_MS);
    expect(goCalls).toBe(0);
    expect(viewLoadError.value).not.toBeNull();
    // Return to the failed path clears navigatingAwayFromError.
    for (const g of beforeGuards) g({ fullPath: '/web/base/countries' });
    store.markUnreachable();
    store.markReachable();
    await clock.flush();
    await clock.advance(VIEW_LOAD_RETRY_AFTER_RECOVERY_MS);
    expect(goCalls).toBe(1);
    expect(viewLoadError.value).toBeNull();
    // Let the one-shot flag reset after a clear sticks, then re-arm the surface.
    await clock.advance(VIEW_LOAD_RETRY_AFTER_RECOVERY_MS + 100);
    setViewLoadError({
      message: 'Failed to fetch dynamically imported module',
      fullPath: '/web/base/countries',
    });
    store.markUnreachable();
    store.markReachable();
    await clock.flush();
    for (const g of beforeGuards) g({ fullPath: '/web/base/banks' });
    await clock.advance(VIEW_LOAD_RETRY_AFTER_RECOVERY_MS);
    expect(goCalls).toBe(1);
    for (const hook of afterHooks) hook();
    store.markUnreachable();
    store.markReachable();
    await clock.flush();
    await clock.advance(VIEW_LOAD_RETRY_AFTER_RECOVERY_MS);
    expect(goCalls).toBe(2);
    mounted.unmount();
    clearViewLoadError();
  } finally {
    clock.restore();
  }
});
