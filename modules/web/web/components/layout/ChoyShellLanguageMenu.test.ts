// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createI18n } from 'vue-i18n';
import { createPinia, setActivePinia } from 'pinia';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import source from '../../i18n/source';
import { useI18nStore } from '../../stores/i18nStore';
import ChoyShellLanguageMenu from './ChoyShellLanguageMenu.vue';

describe('ChoyShellLanguageMenu', () => {
  test('falls back to default languages when pinia is missing', async () => {
    setActivePinia(undefined as any);
    const mounted = mountApp(ChoyShellLanguageMenu as any);
    await flushPromises();
    const trigger = mounted.q('[data-testid=choy-shell-locale]');
    expect(trigger).not.toBeNull();
    expect(trigger?.getAttribute('aria-label')).toBe('Languages');
    expect(mounted.q('[data-lucide=Languages]')).not.toBeNull();
    mounted.unmount();
  });

  test('selects a language through the i18n store', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    const i18n = createI18n({
      legacy: false,
      locale: 'en_US',
      missingWarn: false,
      fallbackWarn: false,
      messages: { en_US: source },
    });
    const store = useI18nStore();
    const mounted = mountApp(ChoyShellLanguageMenu as any, { plugins: [pinia, i18n] });
    await flushPromises();
    expect(mounted.q('[data-testid=choy-shell-locale]')?.getAttribute('aria-label')).toBe('Languages');
    store.activeLanguages = [
      { Code: 'en_US', Name: 'English' },
      { Code: 'zh_CN', Name: 'Chinese' },
    ];
    await flushPromises();
    expect(mounted.q('[data-testid=choy-shell-locale]')).not.toBeNull();
    mounted.unmount();
  });

});