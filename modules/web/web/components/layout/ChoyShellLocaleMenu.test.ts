// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createI18n } from 'vue-i18n';
import { createPinia, setActivePinia } from 'pinia';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import source from '../../i18n/source';
import { useI18nStore } from '../../stores/i18nStore';
import ChoyShellLocaleMenu from './ChoyShellLocaleMenu.vue';

describe('ChoyShellLocaleMenu', () => {
  test('falls back to default UI keys when pinia is missing', async () => {
    setActivePinia(undefined as any);
    const mounted = mountApp(ChoyShellLocaleMenu as any);
    await flushPromises();
    const trigger = mounted.q('[data-testid=choy-shell-locale]');
    expect(trigger).not.toBeNull();
    expect(trigger?.getAttribute('aria-label')).toBe('layout.header.languages');
    expect(mounted.q('[data-lucide=Languages]')).not.toBeNull();
    const ss = mounted.setupState() as {
      localeName?: (code: string) => string;
      onSelect?: (code: string) => void;
    };
    expect(ss.localeName?.('xx')).toBe('xx');
    expect(ss.localeName?.('en')).toBe('English');
    ss.onSelect?.('zh-CN');
    mounted.unmount();
  });

  test('selects a UI key through the i18n store', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    const i18n = createI18n({
      legacy: false,
      locale: 'en',
      missingWarn: false,
      fallbackWarn: false,
      messages: { en: source },
    });
    const store = useI18nStore();
    const mounted = mountApp(ChoyShellLocaleMenu as any, { plugins: [pinia, i18n] });
    await flushPromises();
    expect(mounted.q('[data-testid=choy-shell-locale]')?.getAttribute('aria-label')).toBe('Languages');
    const onSelect = mounted.setupState()?.onSelect as ((code: string) => void) | undefined;
    expect(typeof onSelect).toBe('function');
    onSelect?.('zh-CN');
    await flushPromises();
    store.setActiveUiKeys(['en', 'zh-CN', 'ja']);
    await flushPromises();
    expect(mounted.q('[data-testid=choy-shell-locale]')).not.toBeNull();
    mounted.unmount();
  });
});
