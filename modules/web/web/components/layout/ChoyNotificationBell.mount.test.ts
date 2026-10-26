// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyNotificationBell from './ChoyNotificationBell.vue';

describe('ChoyNotificationBell mount', () => {
  test('omitted count hosts the inbox engine', async () => {
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
    const wrapper = mountApp(ChoyNotificationBell as any, { props: { count: 120 } });
    await flushPromises();
    expect(wrapper.text().includes('99+')).toBe(true);
    wrapper.unmount();
  });

  test('count 0 keeps chrome without a badge numeral', async () => {
    const wrapper = mountApp(ChoyNotificationBell as any, { props: { count: 0 } });
    await flushPromises();
    expect(wrapper.q('[data-anchor="choy.notification-bell"]')).not.toBeNull();
    expect(wrapper.text().includes('99+')).toBe(false);
    const btn = wrapper.q('button') as HTMLButtonElement | null;
    expect(String(btn?.getAttribute('aria-label') || '')).toBe('Notifications');
    wrapper.unmount();
  });
});
