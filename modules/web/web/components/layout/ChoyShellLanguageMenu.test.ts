// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createI18n } from 'vue-i18n';
import { createPinia, setActivePinia } from 'pinia';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { shellLanguageSelect } from './shellLanguageSelect';
import source from '../../i18n/source';
import { setI18nCreateStoreForTests, useI18nStore } from '../../stores/i18nStore';
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

  test('exposed onSelect resolves store after mount', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    const store = useI18nStore();
    store.isInitialized = true;
    store.lang = 'en_US';
    store.activeLanguages = [{ Code: 'en_US' }, { Code: 'zh_CN' }];
    setI18nCreateStoreForTests(
      () =>
        ({
          GetActiveLanguages: async () => [{ Code: 'en_US' }, { Code: 'zh_CN' }],
        }) as any,
    );
    const prevFetch = globalThis.fetch;
    globalThis.fetch = (async () =>
      ({
        ok: true,
        status: 200,
        json: async () => ({
          lang: 'zh_CN',
          bcp47: 'zh-CN',
          hash: 'zh',
          unchanged: true,
          messages: null,
        }),
      }) as Response) as typeof fetch;
    const mounted = mountApp(ChoyShellLanguageMenu as any, { plugins: [pinia] });
    await flushPromises();
    const vm = mounted.root as { onSelect?: (c: string) => Promise<void>; resolveI18nStore?: () => unknown };
    expect(vm.resolveI18nStore?.()).not.toBeNull();
    expect(typeof vm.onSelect).toBe('function');
    await vm.onSelect?.('zh_CN');
    mounted.unmount();
    globalThis.fetch = prevFetch;
    setI18nCreateStoreForTests(null);
  });

  test('exposed onSelect without pinia is a no-op', async () => {
    setActivePinia(undefined as any);
    const mounted = mountApp(ChoyShellLanguageMenu as any);
    await flushPromises();
    const vm = mounted.root as { onSelect?: (c: string) => Promise<void>; resolveI18nStore?: () => unknown };
    expect(vm.resolveI18nStore?.()).toBeNull();
    await vm.onSelect?.('zh_CN');
    mounted.unmount();
  });
});