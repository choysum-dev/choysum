// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyStatusbarButton from './ChoyStatusbarButton.vue';

describe('ChoyStatusbarButton', () => {
  test('defaults to neutral outline sm with brand hover classes', async () => {
    const w = mountApp(ChoyStatusbarButton as any, {
      props: { 'data-testid': 'sb' },
      slots: { default: () => 'Go' },
    });
    try {
      await flushPromises();
      const btn = w.q('[data-testid=sb]') as HTMLElement | null;
      expect(btn).not.toBeNull();
      const cls = btn!.className || '';
      expect(cls).toContain('border-input');
      expect(cls).toContain('h-control-sm');
      expect(cls).toContain('text-foreground');
      expect(cls).toContain('hover:bg-primary/10');
      expect(cls).not.toContain('border-primary');
    } finally {
      w.unmount();
    }
  });

  test('allows outline-destructive override', async () => {
    const w = mountApp(ChoyStatusbarButton as any, {
      props: { 'data-testid': 'sb-danger', variant: 'outline-destructive' },
      slots: { default: () => 'Stop' },
    });
    try {
      await flushPromises();
      const btn = w.q('[data-testid=sb-danger]') as HTMLElement | null;
      expect(btn).not.toBeNull();
      const cls = btn!.className || '';
      expect(cls).toContain('text-destructive');
      expect(cls).toContain('h-control-sm');
    } finally {
      w.unmount();
    }
  });
});
