// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Preferences dialog open helpers: seed Language/Timezone without extra RPCs
 * when the auth profile already has values.
 */

import {
  detectBrowserTimezone,
  resolvePreferenceLanguage,
  resolvePreferenceTimezone,
} from './preferences_defaults';

function languageRefId(value: unknown): string {
  if (value == null || Array.isArray(value)) return '';
  if (typeof value !== 'object') return String(value).trim();
  if (!('Id' in value)) return '';
  return String((value as { Id?: unknown }).Id ?? '').trim();
}

/** True when open path should Browse User (no in-memory profile yet). */
export function needsUserLoadOnPreferencesOpen(currentUser: unknown): boolean {
  return !languageRefId((currentUser as { Id?: unknown } | null | undefined)?.Id);
}

/**
 * Seed LanguageId from saved User.LanguageId, else Search once by session code.
 * Skips Search when a saved id is already present.
 */
export async function resolvePreferencesSeedLanguageId(opts: {
  languageId: unknown;
  sessionTerminologyLang: string | null | undefined;
  searchLanguageByCode: (code: string) => Promise<string>;
}): Promise<{ seedLanguageId: string; languageFromSession: boolean }> {
  const savedId = languageRefId(opts.languageId);
  if (savedId) {
    return { seedLanguageId: savedId, languageFromSession: false };
  }
  const session = resolvePreferenceLanguage(null, opts.sessionTerminologyLang);
  const code = String(session.code || '').trim();
  if (!code) {
    return { seedLanguageId: '', languageFromSession: session.fromSession };
  }
  try {
    const id = String((await opts.searchLanguageByCode(code)) || '').trim();
    return { seedLanguageId: id, languageFromSession: session.fromSession };
  } catch {
    return { seedLanguageId: '', languageFromSession: session.fromSession };
  }
}

/**
 * Seed Timezone from saved User.Timezone, else browser IANA hint.
 * Does not call FieldsGet; SelectionField FieldsGet validates selection later.
 */
export function resolvePreferencesSeedTimezone(userTimezone: unknown): {
  seedTimezone: string;
  timezoneFromBrowser: boolean;
} {
  const saved =
    userTimezone == null || Array.isArray(userTimezone) ? null : String(userTimezone).trim() || null;
  const resolved = resolvePreferenceTimezone(saved, detectBrowserTimezone(), []);
  return {
    seedTimezone: resolved.timezone ?? '',
    timezoneFromBrowser: resolved.fromBrowser,
  };
}
