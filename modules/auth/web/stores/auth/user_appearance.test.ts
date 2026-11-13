// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  applyUserAppearance,
  resolveDensity,
  resolveThemeMode,
  resolveUserDensity,
  resolveUserThemeMode,
} from './user_appearance';
import {
  CHOY_THEME_STORAGE_KEY,
  readChoyThemePreference,
} from '@/web/web/composables/applyChoyThemePreference';

test('resolveThemeMode: known modes pass through; others become light', () => {
  expect(resolveThemeMode('light')).toBe('light');
  expect(resolveThemeMode('dark')).toBe('dark');
  expect(resolveThemeMode('auto')).toBe('auto');
  expect(resolveThemeMode('')).toBe('light');
  expect(resolveThemeMode(undefined)).toBe('light');
  expect(resolveThemeMode('nope')).toBe('light');
});

test('resolveDensity: compact vs comfortable', () => {
  expect(resolveDensity('compact')).toBe('compact');
  expect(resolveDensity('standard')).toBe('comfortable');
  expect(resolveDensity(undefined)).toBe('comfortable');
});

test('applyUserAppearance: returns normalized theme and density', () => {
  const root = {
    classList: { toggle: () => undefined },
    setAttribute: () => undefined,
    removeAttribute: () => undefined,
  } as any;
  const storage = { getItem: () => null, setItem: () => undefined };
  expect(applyUserAppearance('dark', 'compact', { root, storage, persist: false })).toEqual({
    theme: 'dark',
    density: 'compact',
  });
});

test('resolveUserThemeMode: User.Theme when set, else fallback', () => {
  expect(resolveUserThemeMode({ Theme: 'dark' }, 'auto')).toBe('dark');
  expect(resolveUserThemeMode({ Theme: 'auto' }, 'light')).toBe('auto');
  expect(resolveUserThemeMode({ Theme: null }, 'dark')).toBe('dark');
  expect(resolveUserThemeMode({}, 'auto')).toBe('auto');
  expect(resolveUserThemeMode(null, 'auto')).toBe('auto');
});

test('resolveUserDensity: User.Density when set, else fallback', () => {
  expect(resolveUserDensity({ Density: 'compact' }, 'comfortable')).toBe('compact');
  expect(resolveUserDensity({ Density: null }, 'compact')).toBe('compact');
  expect(resolveUserDensity({}, 'standard')).toBe('comfortable');
});

test('applyUserAppearance: persists theme for later readChoyThemePreference', () => {
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
  } as any;

  expect(resolveThemeMode(readChoyThemePreference(storage).theme)).toBe('light');

  applyUserAppearance('dark', 'comfortable', { storage, root, persist: true });
  expect(resolveThemeMode(readChoyThemePreference(storage).theme)).toBe('dark');
  expect(bag.get(CHOY_THEME_STORAGE_KEY)).toBeTruthy();

  applyUserAppearance('auto', 'compact', { storage, root, persist: true });
  expect(resolveThemeMode(readChoyThemePreference(storage).theme)).toBe('auto');
  expect(resolveDensity(readChoyThemePreference(storage).density)).toBe('compact');
});
