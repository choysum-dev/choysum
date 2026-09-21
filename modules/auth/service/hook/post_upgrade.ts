// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { HookPostUpgrade } from '@/core/service/api/model';

/**
 * Copy leftover auth_user.language POSIX codes into language_id.
 * The old varchar column is not dropped by schema sync; missing columns are ignored.
 */
export class AuthUserLanguageHooks {
  @HookPostUpgrade()
  static async backfillUserLanguageId(): Promise<void> {
    const db = (globalThis as { $choysum?: { db?: { dialectName?: string; execute?: (sql: string, params: string) => Promise<unknown> } } }).$choysum?.db;
    if (!db?.execute || !db.dialectName) return;
    const dialect = String(db.dialectName);
    let sql = '';
    if (dialect === 'sqlite') {
      sql = `UPDATE auth_user
        SET language_id = (
          SELECT id FROM base_language WHERE code = auth_user.language LIMIT 1
        )
        WHERE (language_id IS NULL OR language_id = '')
          AND language IS NOT NULL
          AND trim(language) <> ''`;
    } else if (dialect === 'postgres' || dialect === 'mysql') {
      sql = `UPDATE auth_user
        SET language_id = (
          SELECT id FROM base_language WHERE code = auth_user.language LIMIT 1
        )
        WHERE COALESCE(language_id, '') = ''
          AND COALESCE(language, '') <> ''`;
    }
    if (!sql) return;
    try {
      await db.execute(sql, '[]');
    } catch {
      // Old language column may already be absent; users can set LanguageId again.
    }
  }
}
