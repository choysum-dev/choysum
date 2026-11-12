// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  applyChoyThemePreference,
  readChoyThemePreference,
  type ApplyChoyThemePreferenceOptions,
  type ChoyDensityPreference,
  type ChoyThemeMode,
} from '@/web/web/composables/applyChoyThemePreference';
import { normalizePreferencesDensity, type PreferencesDensityUi } from './preferences_bag';

/** Theme modes offered on auth.User.Theme / Preferences.theme. */
export const PREFERENCES_THEME_MODES: readonly ChoyThemeMode[] = ['light', 'dark', 'auto'];

/** Normalize stored preference to a concrete Theme selection value. */
export function resolvePreferencesThemeMode(theme: unknown): ChoyThemeMode {
  return theme === 'dark' || theme === 'auto' || theme === 'light' ? theme : 'light';
}

/** Density offered in the Preferences dialog UI. */
export function resolvePreferencesDensity(density: unknown): PreferencesDensityUi {
  return normalizePreferencesDensity(density);
}

/**
 * Resolve Theme for Preferences seeding: User.Theme, then Preferences.theme,
 * then local shell storage (guest / pre-migration).
 */
export function resolveUserThemeMode(
  user: { Theme?: unknown; Preferences?: { theme?: unknown } | null } | null | undefined,
  fallback: ChoyThemeMode = 'light',
): ChoyThemeMode {
  const fromTheme = resolvePreferencesThemeMode(user?.Theme);
  if (user?.Theme === 'light' || user?.Theme === 'dark' || user?.Theme === 'auto') {
    return fromTheme;
  }
  const fromPrefs = user?.Preferences?.theme;
  if (fromPrefs === 'light' || fromPrefs === 'dark' || fromPrefs === 'auto') {
    return fromPrefs;
  }
  return resolvePreferencesThemeMode(fallback);
}

/** Read the current shell theme from local storage. */
export function readPreferencesThemeMode(
  storage?: ApplyChoyThemePreferenceOptions['storage'],
): ChoyThemeMode {
  return resolvePreferencesThemeMode(readChoyThemePreference(storage).theme);
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
