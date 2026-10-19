// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import AuthPanel from './AuthPanel.vue';

describe('AuthPanel', () => {
  test('renders slot content without a nested brand or footer', async () => {
    const Host = defineComponent({
      setup() {
        return () =>
          h(AuthPanel as any, null, {
            default: () => h('div', { 'data-testid': 'auth-slot' }, 'form'),
          });
      },
    });
    const mounted = mountApp(Host as any);
    await flushPromises();
    expect(mounted.q('[data-testid=auth-slot]')?.textContent).toBe('form');
    expect(mounted.q('[data-testid=auth-panel-brand]')).toBeNull();
    expect(mounted.q('[data-testid=auth-panel-attrib]')).toBeNull();
    mounted.unmount();
  });
});
