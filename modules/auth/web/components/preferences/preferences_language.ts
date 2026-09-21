// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { terminologyCodeFromLanguageId } from '@/auth/web/stores/auth/language_preference';

/**
 * Resolve a language Code from User.LanguageId for the preferences dialog.
 * Browse failures yield '' so the dialog can fall back to session terminology.
 */
export async function resolveLanguageCodeFromId(
  languageId: unknown,
  browseLanguage: (id: string, fields: string[]) => Promise<{ Code?: string } | null | undefined>
): Promise<string> {
  try {
    return await terminologyCodeFromLanguageId(languageId, browseLanguage);
  } catch {
    return '';
  }
}
