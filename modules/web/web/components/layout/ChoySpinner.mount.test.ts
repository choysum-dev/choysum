// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoySpinner from './ChoySpinner.vue';

describe('ChoySpinner mount', () => {
  test('defaults aria-label to Loading and accepts an override', async () => {
    const def = mountApp(ChoySpinner as any, {
      props: { class: 'spin-extra' },
    });
    await flushPromises();
    const el = def.q('[data-testid=choy-spinner]');
    expect(el).not.toBeNull();
    expect(el?.getAttribute('aria-label')).toBe('Loading');
    def.unmount();

    const custom = mountApp(ChoySpinner as any, {
      props: { label: 'Saving' },
    });
    await flushPromises();
    expect(custom.q('[data-testid=choy-spinner]')?.getAttribute('aria-label')).toBe('Saving');
    custom.unmount();
  });
});
