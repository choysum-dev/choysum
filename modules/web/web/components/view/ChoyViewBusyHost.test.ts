// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import {
  flushPromises,
  fnRecorder,
  mountApp,
  restoreSfc,
  stubSfc,
} from '@/web/web/__tests__/mountApp';
import { loadingTiming } from '@/core/web/client/loadingRegistry';
import ChoyButton from '../layout/ChoyButton.vue';
import { useConnectivityStore } from '../../stores/connectivityStore';
import ChoyViewBusyHost from './ChoyViewBusyHost.vue';

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function waitForOverlayDelay(): Promise<void> {
  await flushPromises();
  await sleep(loadingTiming.SURFACE_BUSY_DELAY_MS + 30);
  await flushPromises();
}

describe('ChoyViewBusyHost', () => {
  test('idle host renders slot content without overlay or live region', async () => {
    const mounted = mountApp(ChoyViewBusyHost as any, {
      slots: {
        default: () => h('div', { 'data-testid': 'body' }, 'ok'),
      },
    });
    await flushPromises();
    const host = mounted.q('[data-slot=view-body-host]') as HTMLElement | null;
    expect(host).not.toBeNull();
    expect(host?.getAttribute('aria-busy')).toBeNull();
    expect(host?.hasAttribute('inert')).toBe(false);
    expect(mounted.q('[data-testid=body]')?.textContent).toBe('ok');
    expect(mounted.q('[data-slot=view-busy-overlay]')).toBeNull();
    expect(mounted.q('[data-slot=view-busy-status]')).toBeNull();
    mounted.unmount();
  });

  test('busy host with hasFrame overlays spinner after delay, sets inert immediately, and teleports a live status', async () => {
    // Clear teleports left by earlier view mounts in the same FE bundle run.
    for (const node of Array.from(
      document.body.querySelectorAll('[data-slot=view-busy-status]'),
    )) {
      node.parentNode?.removeChild(node);
    }
    const mounted = mountApp(ChoyViewBusyHost as any, {
      // hasFrame: delay still applies; no-frame busy paints overlay immediately.
      props: { busy: true, hasFrame: true, busyLabel: 'Saving…', class: 'min-h-40' },
      slots: {
        default: () => h('div', { 'data-testid': 'body' }, 'ok'),
      },
    });
    await flushPromises();
    const host = mounted.q('[data-slot=view-body-host]') as HTMLElement | null;
    expect(host?.getAttribute('aria-busy')).toBe('true');
    expect(host?.hasAttribute('inert')).toBe(true);
    expect(host?.getAttribute('data-busy-mode')).toBe('block');
    expect(host?.className || '').toContain('pointer-events-none');
    expect(host?.className || '').toContain('min-h-40');
    expect(host?.className || '').toContain('choy-view-busy-host');
    // Overlay is delayed; inert / live region are immediate.
    expect(mounted.q('[data-slot=view-busy-overlay]')).toBeNull();

    const statusesEarly = Array.from(
      document.body.querySelectorAll('[data-slot=view-busy-status]'),
    );
    expect(statusesEarly.length).toBe(1);

    await waitForOverlayDelay();
    const overlay = mounted.q('[data-slot=view-busy-overlay]') as HTMLElement | null;
    expect(overlay).not.toBeNull();
    expect(overlay?.getAttribute('data-mode')).toBe('block');
    expect(overlay?.querySelector('[data-testid=choy-spinner]')).not.toBeNull();

    // Teleport lands on document.body; require exactly one live region (no leaked stubs).
    const statuses = Array.from(
      document.body.querySelectorAll('[data-slot=view-busy-status]'),
    );
    expect(statuses.length).toBe(1);
    const status = statuses[0]!;
    expect(status.getAttribute('role')).toBe('status');
    expect((status.textContent || '').trim()).toBe('Saving…');
    mounted.unmount();
    expect(
      Array.from(document.body.querySelectorAll('[data-slot=view-busy-status]'))
        .length,
    ).toBe(0);
  });

  test('busy host never shows overlay when cleared before delay (hasFrame)', async () => {
    const mounted = mountApp(ChoyViewBusyHost as any, {
      reactiveProps: true,
      props: { busy: true, hasFrame: true },
      slots: {
        default: () => h('div', { 'data-testid': 'body' }, 'ok'),
      },
    });
    await flushPromises();
    expect(mounted.q('[data-slot=view-body-host]')?.getAttribute('aria-busy')).toBe('true');
    expect(mounted.q('[data-slot=view-busy-overlay]')).toBeNull();

    mounted.props.busy = false;
    await flushPromises();
    await sleep(loadingTiming.SURFACE_BUSY_DELAY_MS + 30);
    await flushPromises();
    expect(mounted.q('[data-slot=view-busy-overlay]')).toBeNull();
    expect(mounted.q('[data-slot=view-body-host]')?.getAttribute('aria-busy')).toBeNull();
    mounted.unmount();
  });

  test('no-frame busy shows overlay immediately and keeps slot', async () => {
    const mounted = mountApp(ChoyViewBusyHost as any, {
      props: { busy: true, hasFrame: false },
      slots: {
        default: () => h('div', { 'data-testid': 'body' }, 'table chrome'),
      },
    });
    await flushPromises();
    expect(mounted.q('[data-slot=view-busy-overlay]')).not.toBeNull();
    expect(mounted.q('[data-testid=body]')?.textContent).toBe('table chrome');
    mounted.unmount();
  });

  test('stale mode fades host without inert or blocking overlay', async () => {
    const mounted = mountApp(ChoyViewBusyHost as any, {
      props: { busy: true, mode: 'stale' },
      slots: {
        default: () => h('div', { 'data-testid': 'body' }, 'ok'),
      },
    });
    await flushPromises();
    const host = mounted.q('[data-slot=view-body-host]') as HTMLElement | null;
    expect(host?.getAttribute('aria-busy')).toBe('true');
    expect(host?.getAttribute('data-busy-mode')).toBe('stale');
    expect(host?.hasAttribute('inert')).toBe(false);
    expect(host?.className || '').toContain('opacity-70');
    expect(host?.className || '').toContain('choy-view-busy-host');
    expect(host?.className || '').not.toContain('pointer-events-none');
    expect(mounted.q('[data-slot=view-busy-overlay]')).toBeNull();
    await waitForOverlayDelay();
    expect(mounted.q('[data-slot=view-busy-overlay]')).toBeNull();
    mounted.unmount();
  });

  test('settled error without frame overlays surface error and keeps slot chrome', async () => {
    setActivePinia(createPinia());
    useConnectivityStore().markUnreachable();
    // fe-qjs: nested vendor Button often drops native click; stub like List chrome tests.
    stubSfc(ChoyButton as any, {
      name: 'ChoyButton',
      inheritAttrs: false,
      props: {
        type: { type: String, default: 'button' },
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
    const onRetry = fnRecorder(() => {});
    try {
      const mounted = mountApp(ChoyViewBusyHost as any, {
        props: {
          busy: false,
          error: new Error('Failed to fetch'),
          hasFrame: false,
          hasSettled: true,
        },
        on: { onRetry: () => onRetry() },
        slots: {
          default: () => h('div', { 'data-testid': 'body' }, 'table chrome'),
        },
      });
      await flushPromises();
      expect(mounted.q('[data-testid=choy-view-surface-error]')).not.toBeNull();
      expect(mounted.text()).toContain('Connection lost');
      expect(mounted.text()).toContain('Could not load this data');
      // Slot stays mounted under the overlay (no table-header flicker on Retry).
      expect(mounted.q('[data-testid=body]')?.textContent).toBe('table chrome');
      mounted.click('[data-testid=choy-view-surface-error-retry]');
      await flushPromises();
      expect(onRetry.calls.length).toBe(1);
      mounted.unmount();
    } finally {
      restoreSfc(ChoyButton as any);
      useConnectivityStore().resetForTests();
    }
  });

  test('error with hasFrame shows stale banner and keeps slot content', async () => {
    setActivePinia(createPinia());
    stubSfc(ChoyButton as any, {
      name: 'ChoyButton',
      inheritAttrs: false,
      props: {
        type: { type: String, default: 'button' },
        size: { type: String, default: 'default' },
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
    const onRetry = fnRecorder(() => {});
    try {
      const mounted = mountApp(ChoyViewBusyHost as any, {
        props: {
          busy: false,
          error: new Error('Failed to fetch'),
          hasFrame: true,
          hasSettled: true,
        },
        on: { onRetry: () => onRetry() },
        slots: {
          default: () => h('div', { 'data-testid': 'body' }, 'cached rows'),
        },
      });
      await flushPromises();
      expect(mounted.q('[data-testid=choy-view-surface-error]')).toBeNull();
      expect(mounted.q('[data-testid=choy-view-surface-stale-banner]')).not.toBeNull();
      expect(mounted.text()).toContain('Could not refresh.');
      expect(mounted.q('[data-testid=body]')?.textContent).toBe('cached rows');
      mounted.click('[data-testid=choy-view-surface-stale-retry]');
      await flushPromises();
      expect(onRetry.calls.length).toBe(1);
      mounted.unmount();
    } finally {
      restoreSfc(ChoyButton as any);
    }
  });
});

