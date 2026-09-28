// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyButton from './ChoyButton.vue';

describe('ChoyButton mount', () => {
  test('forwards data-testid and aria-label to the native button', async () => {
    const w = mountApp(ChoyButton as any, {
      props: {
        'data-testid': 'company-switch-trigger',
        'aria-label': 'Switch company',
        'aria-expanded': 'false',
      },
      slots: { default: () => 'Acme' },
    });
    try {
      await flushPromises();

      const btn = w.q('[data-testid="company-switch-trigger"]') as HTMLButtonElement | null;
      expect(btn).not.toBeNull();
      expect(btn?.tagName.toLowerCase()).toBe('button');
      expect(btn?.getAttribute('aria-label')).toBe('Switch company');
      expect(btn?.getAttribute('aria-expanded')).toBe('false');
      expect(btn?.getAttribute('data-anchor')).toBe('choy.button');
      expect((btn?.textContent || '').trim()).toBe('Acme');
    } finally {
      w.unmount();
    }
  });
});
