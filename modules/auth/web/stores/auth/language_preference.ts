// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Resolve a terminology POSIX code from a base.Language id via Browse.
 * Accepts a raw id or a ManyToOne `{ Id }` reference.
 * Returns '' when the id is empty or Browse yields no Code.
 */
export async function terminologyCodeFromLanguageId(
  languageId: unknown,
  browseLanguage: (id: string, fields: string[]) => Promise<{ Code?: string } | null | undefined>
): Promise<string> {
  const id = languageRefId(languageId);
  if (!id) return '';
  const row = await browseLanguage(id, ['Code']);
  return String(row?.Code || '').trim();
}

function languageRefId(value: unknown): string {
  if (value == null || Array.isArray(value)) return '';
  const raw =
    typeof value === 'object' && value !== null && 'Id' in value ? (value as { Id?: unknown }).Id : value;
  const id = String(raw ?? '').trim();
  return id && id !== '[object Object]' ? id : '';
}

/**
 * Apply a User.LanguageId (and optional display overrides) to the FE i18n store.
 * Display overrides are applied first so they stick even when language Browse/setUiKey rejects.
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
  opts.setDisplayOverrides(opts.displayOverrides ?? null);
  const preferredLang = await terminologyCodeFromLanguageId(opts.languageId, opts.browseLanguage);
  if (preferredLang) {
    await opts.setUiKey(opts.langToUiKey(preferredLang));
  }
}
