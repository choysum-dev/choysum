// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';

import { flushPromises, fnRecorder, mountApp } from '@/web/web/__tests__/mountApp';
import { useConnectivityStore } from '@/web/web/stores/connectivityStore';
import ChoyViewOverflowRefreshItem from './ChoyViewOverflowRefreshItem.vue';

describe('ChoyViewOverflowRefreshItem', () => {
  afterEach(() => {
    try {
      useConnectivityStore().resetForTests();
    } catch {
      /* no pinia */
    }
  });

  test('renders Refresh label, default test id, and emits select', async () => {
    const onSelect = fnRecorder();
    const mounted = mountApp(ChoyViewOverflowRefreshItem as any, {
      props: {},
      on: { onSelect },
    });
    await flushPromises();
    const item = mounted.q('[data-testid=view-overflow-refresh]');
    expect(item).not.toBeNull();
    expect(item!.textContent || '').toMatch(/Refresh/i);
    mounted.setupState().onExecute();
    await flushPromises();
    expect(onSelect.calls.length).toBe(1);
    mounted.unmount();
  });

  test('honors custom testId, label, and disabled', async () => {
    const mounted = mountApp(ChoyViewOverflowRefreshItem as any, {
      props: {
        testId: 'custom-refresh',
        label: 'Reload board',
        disabled: true,
      },
    });
    await flushPromises();
    const item = mounted.q('[data-testid=custom-refresh]') as HTMLElement | null;
    expect(item).not.toBeNull();
    expect(item!.textContent || '').toContain('Reload board');
    expect(
      item!.getAttribute('data-disabled') != null ||
        item!.hasAttribute('disabled') ||
        (item as HTMLButtonElement).disabled === true,
    ).toBe(true);
    mounted.unmount();
  });

  test('blocks select emit while session unreachable', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    useConnectivityStore().markUnreachable();
    const onSelect = fnRecorder();
    const mounted = mountApp(ChoyViewOverflowRefreshItem as any, {
      props: {},
      plugins: [pinia],
      on: { onSelect },
    });
    await flushPromises();
    const item = mounted.q('[data-testid=view-overflow-refresh]') as HTMLElement | null;
    expect(item).not.toBeNull();
    expect(
      item!.getAttribute('data-disabled') != null ||
        item!.hasAttribute('disabled') ||
        (item as HTMLButtonElement).disabled === true,
    ).toBe(true);
    mounted.unmount();
  });
});
