// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** Normalize ManyToOne `{ Id }` / scalar LanguageId to a trimmed id or null. */
export function normalizeLanguageId(value: unknown): string | null {
  if (value == null || Array.isArray(value)) return null;
  if (typeof value === 'object') {
    if (!('Id' in value)) return null;
    const id = String((value as { Id?: unknown }).Id ?? '').trim();
    return id || null;
  }
  const id = String(value).trim();
  return id || null;
}

/**
 * Resolve a terminology POSIX code from a base.Language id via Browse.
 * Accepts a raw id or a ManyToOne `{ Id }` reference.
 * Returns '' when the id is empty or Browse yields no Code.
 */
export async function terminologyCodeFromLanguageId(
  languageId: unknown,
  browseLanguage: (id: string, fields: string[]) => Promise<{ Code?: string; IsActive?: boolean } | null | undefined>
): Promise<string> {
  const id = normalizeLanguageId(languageId);
  if (!id) return '';
  const row = await browseLanguage(id, ['Code', 'IsActive']);
  if (row?.IsActive === false) return '';
  return String(row?.Code || '').trim();
}

/**
 * Apply a User.LanguageId to the FE i18n store.
 * When getCurrentUiKey returns the same uiKey as the preferred language, setUiKey is skipped
 * (avoids re-fetching translations / active languages on Preferences open / loadUser).
 * Best-effort: callers wrap this when login/auth must continue on i18n failure.
 */
export async function applyUserLanguagePreference(opts: {
  languageId: unknown;
  browseLanguage: (id: string, fields: string[]) => Promise<{ Code?: string; IsActive?: boolean } | null | undefined>;
  setUiKey: (uiKey: string) => Promise<unknown> | unknown;
  langToUiKey: (terminologyLang: string) => string;
  getCurrentUiKey?: () => string | null | undefined;
}): Promise<void> {
  const preferredLang = await terminologyCodeFromLanguageId(opts.languageId, opts.browseLanguage);
  if (!preferredLang) return;
  const uiKey = opts.langToUiKey(preferredLang);
  const current = opts.getCurrentUiKey?.();
  if (current != null && String(current) === uiKey) {
    return;
  }
  await opts.setUiKey(uiKey);
}
