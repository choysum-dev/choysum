// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, getActivePinia, setActivePinia } from 'pinia';
import { buildPageMountGlobal } from '@choysum/page-mount';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import { h } from 'vue';
import ChoyButton from './ChoyButton.vue';
import ChoyViewLoadError from './ChoyViewLoadError.vue';
import { clearViewLoadError, setViewLoadError, viewLoadError } from '../../router/viewLoadError';
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
