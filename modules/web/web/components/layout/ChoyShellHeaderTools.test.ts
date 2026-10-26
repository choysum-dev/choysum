// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import {
  CHOY_THEME_STORAGE_KEY,
  type ChoyThemeMode,
} from '../../composables/applyChoyThemePreference';
import ChoyShellHeaderTools from './ChoyShellHeaderTools.vue';

function writeTheme(theme: ChoyThemeMode) {
  localStorage.setItem(CHOY_THEME_STORAGE_KEY, JSON.stringify({ theme }));
}

describe('ChoyShellHeaderTools', () => {
  beforeEach(() => {
    localStorage.removeItem(CHOY_THEME_STORAGE_KEY);
  });

  test('composes theme toggle and locale menu with a tools separator', async () => {
    writeTheme('light');
    const mounted = mountApp(ChoyShellHeaderTools as any);
    await flushPromises();
    expect(mounted.q('[data-testid=choy-shell-header-tools]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-shell-theme]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-shell-header-sep-tools]')).not.toBeNull();
    expect(mounted.q('[data-testid=choy-shell-locale]')).not.toBeNull();
    mounted.unmount();
  });
});
