// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, getActivePinia, setActivePinia } from 'pinia';
import { buildPageMountGlobal } from '@choysum/page-mount';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import { h } from 'vue';
import ChoyButton from './ChoyButton.vue';
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

function mountLoadErrorWithGo() {
  const pinia = createPinia();
  setActivePinia(pinia);
  let goCalls = 0;
  const { plugins, router } = buildPageMountGlobal({
    route: { name: 'CountryList', path: '/web/base/countries', fullPath: '/web/base/countries' },
    router: {
      go: () => {
        goCalls += 1;
      },
      replace: async () => {
        throw new Error('replace must not run for same-path recovery retry');
      },
    },
  });
  const mounted = mountApp(ChoyViewLoadError as any, { plugins: [pinia, ...plugins] });
  return { mounted, goCalls: () => goCalls, router, store: useConnectivityStore() };
}

test('ChoyViewLoadError: soft-retries after connectivity recovery while error surface is shown', async () => {
  setViewLoadError({
    message: 'Failed to fetch dynamically imported module',
    fullPath: '/web/base/countries',
  });
  const { mounted, goCalls, store } = mountLoadErrorWithGo();
  await flushPromises();
  store.markUnreachable();
  await flushPromises();
  store.markReachable();
  await new Promise((r) => setTimeout(r, VIEW_LOAD_RETRY_AFTER_RECOVERY_MS + 50));
  await flushPromises();
  expect(viewLoadError.value).toBeNull();
  expect(goCalls()).toBe(1);
  mounted.unmount();
});

test('ChoyViewLoadError: clears pending recovery timer on unmount', async () => {
  setViewLoadError({
    message: 'Failed to fetch dynamically imported module',
    fullPath: '/web/base/countries',
  });
  const { mounted, goCalls, store } = mountLoadErrorWithGo();
  await flushPromises();
  store.markUnreachable();
  store.markReachable();
  await flushPromises(); // recovery watch must schedule the timer before unmount
  // Unmount before settle elapses so clearRecoveryRetryTimer cancels a live timer.
  mounted.unmount();
  await new Promise((r) => setTimeout(r, VIEW_LOAD_RETRY_AFTER_RECOVERY_MS + 50));
  expect(goCalls()).toBe(0);
  expect(viewLoadError.value).not.toBeNull();
  clearViewLoadError();
});

test('ChoyViewLoadError: flapping recovery before settle soft-retries once', async () => {
  setViewLoadError({
    message: 'Failed to fetch dynamically imported module',
    fullPath: '/web/base/countries',
  });
  const { mounted, goCalls, store } = mountLoadErrorWithGo();
  await flushPromises();
  store.markUnreachable();
  store.markReachable();
  await flushPromises(); // first timer live
  store.markUnreachable();
  store.markReachable();
  await flushPromises(); // cancels first timer, reschedules once
  await new Promise((r) => setTimeout(r, VIEW_LOAD_RETRY_AFTER_RECOVERY_MS + 50));
  await flushPromises();
  expect(goCalls()).toBe(1);
  expect(viewLoadError.value).toBeNull();
  mounted.unmount();
});

test('ChoyViewLoadError: does not auto-retry again when soft-retry re-sets the same path', async () => {
  setViewLoadError({
    message: 'Failed to fetch dynamically imported module',
    fullPath: '/web/base/countries',
  });
  const { mounted, goCalls, store } = mountLoadErrorWithGo();
  await flushPromises();
  store.markUnreachable();
  store.markReachable();
  await flushPromises();
  await new Promise((r) => setTimeout(r, VIEW_LOAD_RETRY_AFTER_RECOVERY_MS + 50));
  await flushPromises();
  expect(goCalls()).toBe(1);
  // Soft-retry cleared then the load failed again on the same path.
  setViewLoadError({
    message: 'Failed to fetch dynamically imported module',
    fullPath: '/web/base/countries',
  });
  store.markUnreachable();
  store.markReachable();
  await flushPromises();
  await new Promise((r) => setTimeout(r, VIEW_LOAD_RETRY_AFTER_RECOVERY_MS + 50));
  await flushPromises();
  expect(goCalls()).toBe(1);
  mounted.unmount();
  clearViewLoadError();
});

test('ChoyViewLoadError: allows another auto-retry after a clear sticks', async () => {
  setViewLoadError({
    message: 'Failed to fetch dynamically imported module',
    fullPath: '/web/base/countries',
  });
  const { mounted, goCalls, store } = mountLoadErrorWithGo();
  await flushPromises();
  store.markUnreachable();
  store.markReachable();
  await flushPromises();
  await new Promise((r) => setTimeout(r, VIEW_LOAD_RETRY_AFTER_RECOVERY_MS + 50));
  await flushPromises();
  expect(goCalls()).toBe(1);
  expect(viewLoadError.value).toBeNull();
  // One-shot flag resets only after the clear settles.
  await new Promise((r) => setTimeout(r, VIEW_LOAD_RETRY_AFTER_RECOVERY_MS + 150));
  await flushPromises();
  setViewLoadError({
    message: 'Failed to fetch dynamically imported module',
    fullPath: '/web/base/countries',
  });
  store.markUnreachable();
  store.markReachable();
  await flushPromises();
  await new Promise((r) => setTimeout(r, VIEW_LOAD_RETRY_AFTER_RECOVERY_MS + 50));
  await flushPromises();
  expect(goCalls()).toBe(2);
  mounted.unmount();
});

test('ChoyViewLoadError: skips auto-retry when navigating away from the failed view', async () => {
  setViewLoadError({
    message: 'Failed to fetch dynamically imported module',
    fullPath: '/web/base/countries',
  });
  const pinia = createPinia();
  setActivePinia(pinia);
  let goCalls = 0;
  const guards: Array<(to: { fullPath: string }) => void> = [];
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
        guards.push(fn);
        return () => {
          const i = guards.indexOf(fn);
          if (i >= 0) guards.splice(i, 1);
        };
      },
    },
  });
  const mounted = mountApp(ChoyViewLoadError as any, { plugins: [pinia, ...plugins] });
  await flushPromises();
  const store = useConnectivityStore();
  store.markUnreachable();
  store.markReachable();
  await flushPromises(); // timer scheduled
  // Leave for another route before settle — cancel timer and skip soft-retry.
  for (const g of guards) g({ fullPath: '/web/base/banks' });
  await new Promise((r) => setTimeout(r, VIEW_LOAD_RETRY_AFTER_RECOVERY_MS + 50));
  await flushPromises();
  expect(goCalls).toBe(0);
  expect(viewLoadError.value).not.toBeNull();
  mounted.unmount();
  clearViewLoadError();
});
