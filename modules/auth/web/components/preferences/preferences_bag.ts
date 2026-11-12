// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Helpers for User.Preferences bag keys edited in the Preferences dialog
 * (density / notifications / display). Preserves unrelated bag keys on merge.
 */

export type PreferencesDensityUi = 'comfortable' | 'compact';

export type PreferencesNotificationsDraft = {
  email: boolean;
  inApp: boolean;
  push: boolean;
};

export type PreferencesDisplayDraft = {
  dateFormat: string;
  timeFormat: string;
  currency: string;
};

export type PreferencesBagDraft = {
  density: PreferencesDensityUi;
  notifications: PreferencesNotificationsDraft;
  display: PreferencesDisplayDraft;
};

export type PreferencesBagShape = {
  density?: string;
  theme?: string;
  notifications?: {
    email?: boolean;
    push?: boolean;
    inApp?: boolean;
  };
  display?: {
    dateFormat?: string;
    timeFormat?: string;
    currency?: string;
  };
  [key: string]: unknown;
};

/** Normalize bag density for UI selects (`standard` → comfortable). */
export function normalizePreferencesDensity(value: unknown): PreferencesDensityUi {
  return value === 'compact' ? 'compact' : 'comfortable';
}

/** Notification toggles: missing keys default to true. */
export function readPreferencesNotifications(raw: unknown): PreferencesNotificationsDraft {
  const src =
    raw && typeof raw === 'object' && !Array.isArray(raw)
      ? (raw as Record<string, unknown>)
      : {};
  return {
    email: src.email === false ? false : true,
    inApp: src.inApp === false ? false : true,
    push: src.push === false ? false : true,
  };
}

/** Sparse display overrides; empty string means follow Language. */
export function readPreferencesDisplay(raw: unknown): PreferencesDisplayDraft {
  const src =
    raw && typeof raw === 'object' && !Array.isArray(raw)
      ? (raw as Record<string, unknown>)
      : {};
  return {
    dateFormat: String(src.dateFormat ?? '').trim(),
    timeFormat: String(src.timeFormat ?? '').trim(),
    currency: String(src.currency ?? '').trim(),
  };
}

/** Read editable bag slice from User.Preferences (or null). */
export function readPreferencesBag(preferences: unknown): PreferencesBagDraft {
  const bag =
    preferences && typeof preferences === 'object' && !Array.isArray(preferences)
      ? (preferences as PreferencesBagShape)
      : {};
  return {
    density: normalizePreferencesDensity(bag.density),
    notifications: readPreferencesNotifications(bag.notifications),
    display: readPreferencesDisplay(bag.display),
  };
}

function sparseDisplay(display: PreferencesDisplayDraft): PreferencesBagShape['display'] | undefined {
  const dateFormat = String(display.dateFormat || '').trim();
  const timeFormat = String(display.timeFormat || '').trim();
  const currency = String(display.currency || '').trim();
  if (!dateFormat && !timeFormat && !currency) return undefined;
  const out: NonNullable<PreferencesBagShape['display']> = {};
  if (dateFormat) out.dateFormat = dateFormat;
  if (timeFormat) out.timeFormat = timeFormat;
  if (currency) out.currency = currency;
  return out;
}

/**
 * Merge dialog bag draft into existing Preferences, preserving company-scope
 * and other unknown keys. Theme is not written here (User.Theme + constraint).
 */
export function mergePreferencesBag(
  existing: unknown,
  draft: PreferencesBagDraft,
): PreferencesBagShape {
  const base =
    existing && typeof existing === 'object' && !Array.isArray(existing)
      ? { ...(existing as PreferencesBagShape) }
      : ({} as PreferencesBagShape);

  base.density = draft.density;
  base.notifications = {
    email: !!draft.notifications.email,
    inApp: !!draft.notifications.inApp,
    push: !!draft.notifications.push,
  };

  const display = sparseDisplay(draft.display);
  if (display) {
    base.display = display;
  } else {
    delete base.display;
  }

  return base;
}
