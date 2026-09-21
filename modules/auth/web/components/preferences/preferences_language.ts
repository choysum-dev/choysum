// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Resolve a language Code from User.LanguageId for the preferences dialog.
 * Browse failures yield '' so the dialog can fall back to session terminology.
 */
export async function resolveLanguageCodeFromId(
  languageId: unknown,
  browseLanguage: (id: string, fields: string[]) => Promise<{ Code?: string } | null | undefined>
): Promise<string> {
  const id = String(languageId || '').trim();
  if (!id) return '';
  try {
    const row = await browseLanguage(id, ['Code']);
    return String(row?.Code || '').trim();
  } catch {
    return '';
  }
}
