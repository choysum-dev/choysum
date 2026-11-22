// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, getActivePinia, setActivePinia } from 'pinia';
import { buildPageMountGlobal } from '@choysum/page-mount';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import { h } from 'vue';
import ChoyButton from './ChoyButton.vue';
import ChoyViewLoadError from './ChoyViewLoadError.vue';
import { clearViewLoadError, setViewLoadError } from '../../router/viewLoadError';
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

test('ChoyViewLoadError: Asset primary is Retry loading; unreachable copy kept', async () => {
  setViewLoadError({
    message: 'Failed to fetch dynamically imported module',
    fullPath: '/web/base/countries',
  });
  const pinia = createPinia();
  setActivePinia(pinia);
  const { plugins } = buildPageMountGlobal({
    route: { name: 'CountryList', path: '/web/base/countries', fullPath: '/web/base/countries' },
  });
  // Prefer our Pinia so markUnreachable matches the store LoadError reads.
  const mounted = mountApp(ChoyViewLoadError as any, { plugins: [pinia, ...plugins] });
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
