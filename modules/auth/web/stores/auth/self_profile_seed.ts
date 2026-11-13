// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Self-profile dialog open helpers: seed empty LanguageId / Timezone without
 * extra RPCs when the auth profile already has values. Does not silently
 * persist browser TZ (login/register may).
 */

import { detectBrowserTimezone } from '@/web/web/utils/requestTimezone';
import { normalizeLanguageId } from './language_preference';

/**
 * Resolve terminology code: saved user language code, else session terminology
 * lang, else en_US. Marks whether the value came from the session.
 */
export function resolveSessionLanguageCode(
  userLanguage: string | null | undefined,
  sessionTerminologyLang: string | null | undefined,
): { code: string; fromSession: boolean } {
  const saved = String(userLanguage || '').trim();
  if (saved) {
    return { code: saved, fromSession: false };
  }
  const session = String(sessionTerminologyLang || '').trim();
  return { code: session || 'en_US', fromSession: true };
}

/** True when open path should Browse User (no in-memory profile yet). */
export function needsSelfProfileLoad(currentUser: unknown): boolean {
  return !normalizeLanguageId((currentUser as { Id?: unknown } | null | undefined)?.Id);
}

/**
 * Seed LanguageId from saved User.LanguageId, else Search once by session code.
 * Skips Search when a saved id is already present.
 */
export async function resolveSelfProfileSeedLanguageId(opts: {
  languageId: unknown;
  sessionTerminologyLang: string | null | undefined;
  searchLanguageByCode: (code: string) => Promise<string>;
}): Promise<{ seedLanguageId: string; languageFromSession: boolean }> {
  const savedId = normalizeLanguageId(opts.languageId);
  if (savedId) {
    return { seedLanguageId: savedId, languageFromSession: false };
  }
  const session = resolveSessionLanguageCode(null, opts.sessionTerminologyLang);
  const code = String(session.code || '').trim();
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
export function resolveSelfProfileSeedTimezone(
  userTimezone: unknown,
  detectBrowser: () => string = detectBrowserTimezone,
): {
  seedTimezone: string;
  timezoneFromBrowser: boolean;
} {
  const saved =
    userTimezone == null || Array.isArray(userTimezone)
      ? ''
      : String(userTimezone).trim();
  if (saved) {
    return { seedTimezone: saved, timezoneFromBrowser: false };
  }
  const suggested = String(detectBrowser() || '').trim();
  if (!suggested) {
    return { seedTimezone: '', timezoneFromBrowser: false };
  }
  return { seedTimezone: suggested, timezoneFromBrowser: true };
}
