// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createI18n } from 'vue-i18n';
import { createPinia, setActivePinia } from 'pinia';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { shellLanguageSelect } from './shellLanguageSelect';
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
    store.isInitialized = true;
    store.lang = 'en_US';
    store.activeLanguages = [
      { Code: 'en_US', Name: 'English' },
      { Code: 'zh_CN', Name: 'Chinese' },
    ];
    const mounted = mountApp(ChoyShellLanguageMenu as any, { plugins: [pinia, i18n] });
    await flushPromises();
    expect(mounted.q('[data-testid=choy-shell-locale]')?.getAttribute('aria-label')).toBe('Languages');
    expect(mounted.q('[data-testid=choy-shell-locale-en_US]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-shell-locale-zh_CN]')).not.toBeNull();
    mounted.unmount();
  });

  test('onSelect path calls setLang through shellLanguageSelect', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    const store = useI18nStore();
    store.isInitialized = true;
    store.lang = 'en_US';
    store.activeLanguages = [{ Code: 'en_US' }, { Code: 'zh_CN' }];
    const setLang = async (code: string) => {
      store.lang = code;
      return true;
    };
    await shellLanguageSelect('zh_CN', setLang, async () => undefined);
    expect(store.lang).toBe('zh_CN');
  });
});