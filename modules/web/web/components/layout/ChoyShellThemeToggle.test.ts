// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createI18n } from 'vue-i18n';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import {
  CHOY_THEME_STORAGE_KEY,
  type ChoyThemeMode,
} from '../../composables/applyChoyThemePreference';
import source from '../../i18n/source';
import ChoyShellThemeToggle from './ChoyShellThemeToggle.vue';

function themeButton(mounted: ReturnType<typeof mountApp>): HTMLButtonElement | null {
  return mounted.q('[data-testid=choy-shell-theme]') as HTMLButtonElement | null;
}

function writeTheme(theme: ChoyThemeMode) {
  localStorage.setItem(CHOY_THEME_STORAGE_KEY, JSON.stringify({ theme }));
}

describe('ChoyShellThemeToggle', () => {
  beforeEach(() => {
    localStorage.removeItem(CHOY_THEME_STORAGE_KEY);
  });

  test('falls back to i18n keys and cycles light, dark, and auto', async () => {
    writeTheme('light');
    const mounted = mountApp(ChoyShellThemeToggle as any);
    await flushPromises();
    const btn = themeButton(mounted);
    expect(btn).not.toBeNull();
    expect(btn?.getAttribute('aria-label')).toBe('layout.header.lightMode');
    const cycleTheme = mounted.setupState()?.cycleTheme as (() => void) | undefined;
    expect(typeof cycleTheme).toBe('function');
    cycleTheme?.();
    await flushPromises();
    expect(themeButton(mounted)?.getAttribute('aria-label')).toBe('layout.header.darkMode');
    cycleTheme?.();
    await flushPromises();
    expect(themeButton(mounted)?.getAttribute('aria-label')).toBe('layout.header.autoMode');
    mounted.unmount();
  });

  test('uses vue-i18n layout strings when the plugin is installed', async () => {
    writeTheme('dark');
    const i18n = createI18n({
      legacy: false,
      locale: 'en',
      missingWarn: false,
      fallbackWarn: false,
      messages: { en: source },
    });
    const mounted = mountApp(ChoyShellThemeToggle as any, { plugins: [i18n] });
    await flushPromises();
    expect(themeButton(mounted)?.getAttribute('aria-label')).toBe('Dark Mode');
    mounted.unmount();
  });
});
