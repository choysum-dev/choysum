// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Align shell theme/density with auth.User.Theme / User.Density.
 * Document + localStorage live in applyChoyThemePreference; this module only
 * prefers User fields then applies through that web primitive.
 */

import {
  applyChoyThemePreference,
  type ApplyChoyThemePreferenceOptions,
  type ChoyDensityPreference,
  type ChoyThemeMode,
} from '@/web/web/composables/applyChoyThemePreference';

export type UserDensityUi = 'comfortable' | 'compact';

/** Normalize unknown theme to a concrete Theme selection value. */
export function resolveThemeMode(theme: unknown): ChoyThemeMode {
  return theme === 'dark' || theme === 'auto' || theme === 'light' ? theme : 'light';
}

/** Normalize Density; legacy `standard` maps to comfortable. */
export function resolveDensity(density: unknown): UserDensityUi {
  return density === 'compact' ? 'compact' : 'comfortable';
}

/**
 * Resolve Theme: User.Theme when set, else normalized fallback (e.g. localStorage).
 */
export function resolveUserThemeMode(
  user: { Theme?: unknown } | null | undefined,
  fallback: unknown = 'light',
): ChoyThemeMode {
  if (user?.Theme === 'light' || user?.Theme === 'dark' || user?.Theme === 'auto') {
    return user.Theme;
  }
  return resolveThemeMode(fallback);
}

/** Resolve Density from User.Density with a normalized fallback. */
export function resolveUserDensity(
  user: { Density?: unknown } | null | undefined,
  fallback: unknown = 'comfortable',
): UserDensityUi {
  if (user?.Density === 'compact' || user?.Density === 'comfortable') {
    return user.Density;
  }
  return resolveDensity(fallback);
}

/** Apply theme + density to the shell (and optionally persist locally). */
export function applyUserAppearance(
  theme: unknown,
  density: unknown,
  opts?: ApplyChoyThemePreferenceOptions,
): { theme: ChoyThemeMode; density: UserDensityUi } {
  const nextTheme = resolveThemeMode(theme);
  const nextDensity = resolveDensity(density);
  applyChoyThemePreference(
    { theme: nextTheme, density: nextDensity as ChoyDensityPreference },
    opts,
  );
  return { theme: nextTheme, density: nextDensity };
}
