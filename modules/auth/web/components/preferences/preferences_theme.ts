// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  applyChoyThemePreference,
  readChoyThemePreference,
  type ApplyChoyThemePreferenceOptions,
  type ChoyDensityPreference,
  type ChoyThemeMode,
} from '@/web/web/composables/applyChoyThemePreference';

/** Theme modes offered on auth.User.Theme. */
export const PREFERENCES_THEME_MODES: readonly ChoyThemeMode[] = ['light', 'dark', 'auto'];

export type PreferencesDensityUi = 'comfortable' | 'compact';

/** Normalize stored preference to a concrete Theme selection value. */
export function resolvePreferencesThemeMode(theme: unknown): ChoyThemeMode {
  return theme === 'dark' || theme === 'auto' || theme === 'light' ? theme : 'light';
}

/** Normalize Density; legacy `standard` maps to comfortable. */
export function resolvePreferencesDensity(density: unknown): PreferencesDensityUi {
  return density === 'compact' ? 'compact' : 'comfortable';
}

/**
 * Resolve Theme for Preferences seeding: User.Theme, then local shell storage.
 */
export function resolveUserThemeMode(
  user: { Theme?: unknown } | null | undefined,
  fallback: ChoyThemeMode = 'light',
): ChoyThemeMode {
  if (user?.Theme === 'light' || user?.Theme === 'dark' || user?.Theme === 'auto') {
    return user.Theme;
  }
  return resolvePreferencesThemeMode(fallback);
}

/** Resolve Density from User.Density with localStorage-friendly fallback. */
export function resolveUserDensity(
  user: { Density?: unknown } | null | undefined,
  fallback: PreferencesDensityUi = 'comfortable',
): PreferencesDensityUi {
  if (user?.Density === 'compact' || user?.Density === 'comfortable') {
    return user.Density;
  }
  return resolvePreferencesDensity(fallback);
}

/** Read the current shell theme from local storage. */
export function readPreferencesThemeMode(
  storage?: ApplyChoyThemePreferenceOptions['storage'],
): ChoyThemeMode {
  return resolvePreferencesThemeMode(readChoyThemePreference(storage).theme);
}

/** Read shell density from local storage. */
export function readPreferencesDensityMode(
  storage?: ApplyChoyThemePreferenceOptions['storage'],
): PreferencesDensityUi {
  return resolvePreferencesDensity(readChoyThemePreference(storage).density);
}

/** Apply a theme to the shell document (and optionally persist locally). */
export function applyPreferencesThemeMode(
  mode: ChoyThemeMode,
  opts?: ApplyChoyThemePreferenceOptions,
): ChoyThemeMode {
  const next = resolvePreferencesThemeMode(mode);
  applyChoyThemePreference({ theme: next }, opts);
  return next;
}

/** Apply theme + density to the shell (and optionally persist locally). */
export function applyPreferencesAppearance(
  theme: unknown,
  density: unknown,
  opts?: ApplyChoyThemePreferenceOptions,
): { theme: ChoyThemeMode; density: PreferencesDensityUi } {
  const nextTheme = resolvePreferencesThemeMode(theme);
  const nextDensity = resolvePreferencesDensity(density);
  applyChoyThemePreference(
    { theme: nextTheme, density: nextDensity as ChoyDensityPreference },
    opts,
  );
  return { theme: nextTheme, density: nextDensity };
}
