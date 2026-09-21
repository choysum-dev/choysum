// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Resolve a terminology POSIX code from a base.Language id via Browse.
 * Returns '' when the id is empty or Browse yields no Code.
 */
export async function terminologyCodeFromLanguageId(
  languageId: unknown,
  browseLanguage: (id: string, fields: string[]) => Promise<{ Code?: string } | null | undefined>
): Promise<string> {
  const id = String(languageId || '').trim();
  if (!id) return '';
  const row = await browseLanguage(id, ['Code']);
  return String(row?.Code || '').trim();
}

/**
 * Apply a User.LanguageId (and optional display overrides) to the FE i18n store.
 * Best-effort: callers wrap this when login/auth must continue on i18n failure.
 */
export async function applyUserLanguagePreference(opts: {
  languageId: unknown;
  displayOverrides?: unknown;
  browseLanguage: (id: string, fields: string[]) => Promise<{ Code?: string } | null | undefined>;
  setUiKey: (uiKey: string) => Promise<unknown> | unknown;
  setDisplayOverrides: (overrides: unknown) => void;
  langToUiKey: (terminologyLang: string) => string;
}): Promise<void> {
  const preferredLang = await terminologyCodeFromLanguageId(opts.languageId, opts.browseLanguage);
  if (preferredLang) {
    await opts.setUiKey(opts.langToUiKey(preferredLang));
  }
  opts.setDisplayOverrides(opts.displayOverrides ?? null);
}
