// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** Theme mode aligned with auth User.Preferences.theme. */
export type ChoyThemeMode = 'light' | 'dark' | 'auto';

/** Density aligned with auth User.Preferences.density (`standard` → comfortable). */
export type ChoyDensityPreference = 'comfortable' | 'compact' | 'standard';

export type ChoyThemePreference = {
  theme?: ChoyThemeMode;
  density?: ChoyDensityPreference;
};

export type ResolvedChoyThemePreference = {
  theme: ChoyThemeMode;
  density: 'comfortable' | 'compact';
  /** Effective dark class after resolving `auto`. */
  dark: boolean;
};

export const CHOY_THEME_STORAGE_KEY = 'choy.ui.theme';

export type ApplyChoyThemePreferenceOptions = {
  root?: ParentNode & {
    classList: DOMTokenList;
    setAttribute: (name: string, value: string) => void;
    removeAttribute: (name: string) => void;
  };
  storage?: Pick<Storage, 'getItem' | 'setItem'> | null;
  persist?: boolean;
  storageKey?: string;
  /** Override for `auto` theme resolution (defaults to matchMedia prefers-color-scheme). */
  prefersDark?: boolean;
};

/**
 * Normalizes prefs (including auth `standard` density) into a concrete UI state.
 */
export function resolveChoyThemePreference(
  prefs: ChoyThemePreference | null | undefined,
  opts: { prefersDark?: boolean } = {},
): ResolvedChoyThemePreference {
  const theme: ChoyThemeMode =
    prefs?.theme === 'dark' || prefs?.theme === 'auto' || prefs?.theme === 'light'
      ? prefs.theme
      : 'light';
  const densityRaw = prefs?.density;
  const density: 'comfortable' | 'compact' =
    densityRaw === 'compact' ? 'compact' : 'comfortable';
  const prefersDark =
    opts.prefersDark ??
    (typeof matchMedia === 'function' ? matchMedia('(prefers-color-scheme: dark)').matches : false);
  const dark = theme === 'dark' || (theme === 'auto' && prefersDark);
  return { theme, density, dark };
}

/**
 * Reads persisted theme preference from storage (localStorage by default).
 */
export function readChoyThemePreference(
  storage?: Pick<Storage, 'getItem'> | null,
  storageKey: string = CHOY_THEME_STORAGE_KEY,
): ChoyThemePreference {
  try {
    // Resolve default store inside try: some browsers throw on localStorage access.
    const store =
      storage === undefined
        ? typeof localStorage !== 'undefined'
          ? localStorage
          : null
        : storage;
    if (!store) return {};
    const raw = store.getItem(storageKey);
    if (!raw) return {};
    const parsed = JSON.parse(raw) as ChoyThemePreference;
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return {};
    return {
      theme:
        parsed.theme === 'dark' || parsed.theme === 'auto' || parsed.theme === 'light'
          ? parsed.theme
          : undefined,
      density:
        parsed.density === 'compact' ||
        parsed.density === 'comfortable' ||
        parsed.density === 'standard'
          ? parsed.density
          : undefined,
    };
  } catch {
    return {};
  }
}

/**
 * Persists theme preference JSON to storage.
 */
export function persistChoyThemePreference(
  prefs: ChoyThemePreference,
  storage?: Pick<Storage, 'setItem'> | null,
  storageKey: string = CHOY_THEME_STORAGE_KEY,
): void {
  try {
    // Resolve default store inside try: denied storage must not break apply.
    const store =
      storage === undefined
        ? typeof localStorage !== 'undefined'
          ? localStorage
          : null
        : storage;
    if (!store) return;
    store.setItem(storageKey, JSON.stringify(prefs));
  } catch {
    // Quota / private mode / SecurityError — ignore.
  }
}

/**
 * Applies dark / density to the document root and optionally persists prefs.
 * Compatible with auth User.Preferences `{ theme, density }` shape.
 */
export function applyChoyThemePreference(
  prefs: ChoyThemePreference | null | undefined,
  opts: ApplyChoyThemePreferenceOptions = {},
): ResolvedChoyThemePreference {
  const resolved = resolveChoyThemePreference(prefs, { prefersDark: opts.prefersDark });
  const root =
    opts.root ??
    (typeof document !== 'undefined' ? document.documentElement : null);
  if (root) {
    root.classList.toggle('dark', resolved.dark);
    if (resolved.density === 'compact') {
      root.setAttribute('data-density', 'compact');
    } else {
      root.removeAttribute('data-density');
    }
  }
  if (opts.persist !== false) {
    // Persist normalized theme; keep caller density (e.g. auth `standard`) for round-trip.
    persistChoyThemePreference(
      {
        theme: resolved.theme,
        density: prefs?.density ?? resolved.density,
      },
      opts.storage,
      opts.storageKey,
    );
  }
  return resolved;
}
