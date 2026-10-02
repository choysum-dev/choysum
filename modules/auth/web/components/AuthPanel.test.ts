// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h } from 'vue';
import { createI18n } from 'vue-i18n';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import AuthPanel from './AuthPanel.vue';

describe('AuthPanel', () => {
  test('renders slot content and attribution fallback without i18n', async () => {
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
    const attrib = mounted.q('[data-testid=auth-panel-attrib]');
    expect(attrib).toBeTruthy();
    // Without vue-i18n, useI18n throws and keys are returned as-is.
    expect(attrib!.textContent || '').toContain('layout.footer');
    mounted.unmount();
  });

  test('uses vue-i18n layout.footer keys when i18n is installed', async () => {
    const i18n = createI18n({
      legacy: false,
      locale: 'en',
      messages: {
        en: {
          layout: {
            footer: {
              copyright: '© {year} Choysum. All rights reserved.',
              powered: 'Powered by Choysum',
              version: 'Version {version}',
            },
          },
        },
      },
    });
    const Host = defineComponent({
      setup() {
        return () =>
          h(AuthPanel as any, null, {
            default: () => h('div', { 'data-testid': 'auth-slot' }, 'form'),
          });
      },
    });
    const mounted = mountApp(Host as any, { plugins: [i18n] });
    await flushPromises();
    const attrib = mounted.q('[data-testid=auth-panel-attrib]')?.textContent || '';
    expect(attrib).toContain('Choysum');
    expect(attrib).toContain('Powered by Choysum');
    expect(attrib).toContain('Version');
    mounted.unmount();
  });
});
