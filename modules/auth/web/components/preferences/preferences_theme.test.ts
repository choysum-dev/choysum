// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  applyPreferencesThemeMode,
  PREFERENCES_THEME_MODES,
  readPreferencesThemeMode,
  resolvePreferencesThemeMode,
  resolveUserThemeMode,
} from './preferences_theme';
import { CHOY_THEME_STORAGE_KEY } from '@/web/web/composables/applyChoyThemePreference';

test('resolvePreferencesThemeMode: known modes pass through; others become light', () => {
  expect(resolvePreferencesThemeMode('light')).toBe('light');
  expect(resolvePreferencesThemeMode('dark')).toBe('dark');
  expect(resolvePreferencesThemeMode('auto')).toBe('auto');
  expect(resolvePreferencesThemeMode('')).toBe('light');
  expect(resolvePreferencesThemeMode(undefined)).toBe('light');
  expect(resolvePreferencesThemeMode('nope')).toBe('light');
});

test('resolveUserThemeMode: Theme wins, then Preferences.theme, then fallback', () => {
  expect(resolveUserThemeMode({ Theme: 'dark', Preferences: { theme: 'light' } }, 'auto')).toBe(
    'dark',
  );
  expect(resolveUserThemeMode({ Theme: null, Preferences: { theme: 'auto' } }, 'light')).toBe(
    'auto',
  );
  expect(resolveUserThemeMode({ Theme: null, Preferences: {} }, 'dark')).toBe('dark');
  expect(resolveUserThemeMode(null, 'auto')).toBe('auto');
});

test('PREFERENCES_THEME_MODES lists light, dark, auto', () => {
  expect([...PREFERENCES_THEME_MODES]).toEqual(['light', 'dark', 'auto']);
});

test('read/applyPreferencesThemeMode round-trip via storage', () => {
  const bag = new Map<string, string>();
  const storage = {
    getItem: (key: string) => bag.get(key) ?? null,
    setItem: (key: string, value: string) => {
      bag.set(key, value);
    },
  };
  const root = {
    classList: {
      add() {},
      remove() {},
      contains() {
        return false;
      },
      toggle() {
        return false;
      },
    },
    setAttribute() {},
    removeAttribute() {},
  } as unknown as NonNullable<Parameters<typeof applyPreferencesThemeMode>[1]>['root'];

  expect(readPreferencesThemeMode(storage)).toBe('light');

  expect(applyPreferencesThemeMode('dark', { storage, root, persist: true })).toBe('dark');
  expect(readPreferencesThemeMode(storage)).toBe('dark');
  expect(bag.get(CHOY_THEME_STORAGE_KEY)).toBeTruthy();

  expect(applyPreferencesThemeMode('auto', { storage, root, persist: true })).toBe('auto');
  expect(readPreferencesThemeMode(storage)).toBe('auto');
});
