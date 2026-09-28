// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ChoyNotificationBell from './ChoyNotificationBell.vue';
import NotificationBell from './NotificationBell.vue';

describe('ChoyNotificationBell mount', () => {
  afterEach(() => {
    restoreSfc(NotificationBell as any);
  });

  function stubInboxEngine() {
    stubSfc(NotificationBell as any, {
      setup(_props: unknown, ctx: { attrs: Record<string, unknown> }) {
        return () =>
          h('div', {
            'data-testid': 'inbox-engine',
            class: ctx.attrs.class,
          });
      },
    });
  }

  test('omitted count hosts the inbox engine', async () => {
    stubInboxEngine();
    const wrapper = mountApp(ChoyNotificationBell as any, {
      props: { class: 'ms-2' },
    });
    await flushPromises();
    const inbox = wrapper.q('[data-testid="inbox-engine"]');
    expect(inbox).not.toBeNull();
    expect(String(inbox?.getAttribute('class') || '').includes('ms-2')).toBe(true);
    expect(wrapper.q('[data-anchor="choy.notification-bell"]')).toBeNull();
    wrapper.unmount();
  });

  test('explicit count renders chrome badge and aria label', async () => {
    stubInboxEngine();
    const wrapper = mountApp(ChoyNotificationBell as any, {
      props: { count: 3, label: 'Alerts' },
    });
    await flushPromises();
    expect(wrapper.q('[data-testid="inbox-engine"]')).toBeNull();
    expect(wrapper.q('[data-anchor="choy.notification-bell"]')).not.toBeNull();
    expect(wrapper.text().includes('3')).toBe(true);
    const btn = wrapper.q('button') as HTMLButtonElement | null;
    expect(btn).not.toBeNull();
    const aria = String(btn?.getAttribute('aria-label') || '');
    expect(aria.includes('Alerts')).toBe(true);
    expect(aria.includes('3 unread')).toBe(true);
    wrapper.unmount();
  });

  test('count over 99 shows 99+ badge text', async () => {
    stubInboxEngine();
    const wrapper = mountApp(ChoyNotificationBell as any, { props: { count: 120 } });
    await flushPromises();
    expect(wrapper.text().includes('99+')).toBe(true);
    wrapper.unmount();
  });

  test('count 0 keeps chrome without a badge numeral', async () => {
    stubInboxEngine();
    const wrapper = mountApp(ChoyNotificationBell as any, { props: { count: 0 } });
    await flushPromises();
    expect(wrapper.q('[data-anchor="choy.notification-bell"]')).not.toBeNull();
    expect(wrapper.text().includes('99+')).toBe(false);
    const btn = wrapper.q('button') as HTMLButtonElement | null;
    expect(String(btn?.getAttribute('aria-label') || '')).toBe('Notifications');
    wrapper.unmount();
  });
});
