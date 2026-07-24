// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Pure helpers for Preferences dialog defaults when User fields are empty.
 * Dialog open does not silently persist browser TZ; login/register may (D20).
 */

export function detectBrowserTimezone(): string | null {
  try {
    const tz = String(Intl.DateTimeFormat().resolvedOptions().timeZone || '').trim();
    return tz || null;
  } catch {
    return null;
  }
}

/**
 * Resolve the Language select value: saved User.Language, else current session
 * terminology lang, else en_US. Marks whether the value came from the session
 * (not a persisted user preference).
 */
export function resolvePreferenceLanguage(
  userLanguage: string | null | undefined,
  sessionTerminologyLang: string | null | undefined
): { code: string; fromSession: boolean } {
  const saved = String(userLanguage || '').trim();
  if (saved) {
    return { code: saved, fromSession: false };
  }
  const session = String(sessionTerminologyLang || '').trim();
  return { code: session || 'en_US', fromSession: true };
}

/**
 * When User.Timezone is empty, suggest the browser IANA zone if it appears in
 * the allowed selection. Otherwise leave empty (caller shows placeholder).
 */
export function resolvePreferenceTimezone(
  userTimezone: string | null | undefined,
  browserTimezone: string | null | undefined,
  allowedValues: readonly string[]
): { timezone: string | null; fromBrowser: boolean } {
  const saved = String(userTimezone || '').trim();
  if (saved) {
    return { timezone: saved, fromBrowser: false };
  }
  const suggested = String(browserTimezone || '').trim();
  if (!suggested) {
    return { timezone: null, fromBrowser: false };
  }
  if (allowedValues.length > 0 && !allowedValues.includes(suggested)) {
    return { timezone: null, fromBrowser: false };
  }
  return { timezone: suggested, fromBrowser: true };
}
