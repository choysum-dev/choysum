// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, nextTick } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import { useConnectivityStore } from '../../stores/connectivityStore';
import ChoyButton from '../button/ChoyButton.vue';
import ChoyConnectivityPill from './ChoyConnectivityPill.vue';

beforeEach(() => {
  setActivePinia(createPinia());
  useConnectivityStore().resetForTests();
  stubSfc(ChoyButton as any, {
    name: 'ChoyButton',
    inheritAttrs: false,
    props: {
      type: { type: String, default: 'button' },
      size: { type: String, default: 'default' },
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
});

test('ChoyConnectivityPill: hidden while reachable', () => {
  const mounted = mountApp(ChoyConnectivityPill as any);
  expect(mounted.q('[data-testid=choy-connectivity-pill]')).toBeNull();
  mounted.unmount();
});

test('ChoyConnectivityPill: shows when unreachable and hides after recover', async () => {
  const store = useConnectivityStore();
  store.markUnreachable();
  const mounted = mountApp(ChoyConnectivityPill as any);
  await nextTick();
  const pill = mounted.q('[data-testid=choy-connectivity-pill]') as HTMLElement | null;
  expect(pill).toBeTruthy();
  // Icon-only: accessible name via aria-label; no panel / visible text.
  const aria = pill?.getAttribute('aria-label') || '';
  expect(aria).toContain('Connection lost');
  expect(aria).toContain('Retry connection');
  expect((pill?.textContent || '').trim()).toBe('');
  expect(pill?.querySelector('svg')).toBeTruthy();
  expect(mounted.q('[data-testid=choy-connectivity-pill-panel]')).toBeNull();
  store.markReachable();
  await nextTick();
  expect(mounted.q('[data-testid=choy-connectivity-pill]')).toBeNull();
  mounted.unmount();
});

test('ChoyConnectivityPill: click probes connectivity directly', async () => {
  const store = useConnectivityStore();
  store.markUnreachable();
  const mounted = mountApp(ChoyConnectivityPill as any);
  await nextTick();

  let probed = 0;
  const orig = store.probeConnectivity.bind(store);
  store.probeConnectivity = (async () => {
    probed += 1;
    store.markReachable();
    return true;
  }) as typeof store.probeConnectivity;

  try {
    mounted.click('[data-testid=choy-connectivity-pill]');
    await flushPromises();
    expect(probed).toBe(1);
    expect(store.status).toBe('reachable');
    expect(mounted.q('[data-testid=choy-connectivity-pill]')).toBeNull();
  } finally {
    store.probeConnectivity = orig;
  }
  mounted.unmount();
});
