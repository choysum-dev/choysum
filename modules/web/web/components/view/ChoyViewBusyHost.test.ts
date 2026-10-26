// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyViewBusyHost from './ChoyViewBusyHost.vue';

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

  test('busy host overlays spinner, sets inert, and teleports a live status', async () => {
    const mounted = mountApp(ChoyViewBusyHost as any, {
      props: { busy: true, busyLabel: 'Saving…', class: 'min-h-40' },
      slots: {
        default: () => h('div', { 'data-testid': 'body' }, 'ok'),
      },
    });
    await flushPromises();
    const host = mounted.q('[data-slot=view-body-host]') as HTMLElement | null;
    expect(host?.getAttribute('aria-busy')).toBe('true');
    expect(host?.hasAttribute('inert')).toBe(true);
    expect(host?.className || '').toContain('pointer-events-none');
    expect(host?.className || '').toContain('min-h-40');

    const overlay = mounted.q('[data-slot=view-busy-overlay]') as HTMLElement | null;
    expect(overlay).not.toBeNull();
    expect(overlay?.querySelector('[data-testid=choy-spinner]')).not.toBeNull();

    // Teleport lands on document.body; query via host tree first, then body for the live region.
    const status =
      mounted.q('[data-slot=view-busy-status]') ||
      (document.body.querySelector('[data-slot=view-busy-status]') as Element | null);
    expect(status).not.toBeNull();
    expect(status?.getAttribute('role')).toBe('status');
    expect((status?.textContent || '').trim()).toBe('Saving…');
    mounted.unmount();
    expect(document.body.querySelector('[data-slot=view-busy-status]')).toBeNull();
  });
});
