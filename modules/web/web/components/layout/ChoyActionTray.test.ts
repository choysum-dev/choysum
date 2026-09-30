// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { h } from 'vue';
import ChoyActionTray from './ChoyActionTray.vue';
import ChoyBadge from './ChoyBadge.vue';

describe('ChoyActionTray', () => {
  test('renders toolbar with optional class', async () => {
    const mounted = mountApp(ChoyActionTray as any, {
      props: { class: 'extra-tray' },
      slots: {
        default: () => h('button', { type: 'button' }, 'Go'),
      },
    });
    await flushPromises();
    const el = mounted.q('[data-testid=choy-action-tray]') as HTMLElement | null;
    expect(el).not.toBeNull();
    expect(el?.getAttribute('role')).toBe('toolbar');
    expect(el?.className || '').toContain('extra-tray');
    mounted.unmount();
  });
});

describe('ChoyBadge', () => {
  test('renders default and outline variants', async () => {
    const a = mountApp(ChoyBadge as any, {
      slots: { default: () => '3' },
    });
    await flushPromises();
    expect(a.text()).toContain('3');
    a.unmount();

    const b = mountApp(ChoyBadge as any, {
      props: { variant: 'outline', class: 'ms-1' },
      slots: { default: () => 'New' },
    });
    await flushPromises();
    expect(b.text()).toContain('New');
    b.unmount();
  });
});
