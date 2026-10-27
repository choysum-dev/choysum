// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { HookPostUpgrade } from '@/core/service/api/model';

type ChoysumDb = {
  dialectName?: string;
  execute?: (sql: string, params: string) => Promise<unknown>;
};

function choysumDb(): ChoysumDb | undefined {
  return (globalThis as { $choysum?: { db?: ChoysumDb } }).$choysum?.db;
}

/**
 * Copy leftover auth_user.language POSIX codes into language_id.
 * The old varchar column is not dropped by schema sync; missing columns are ignored.
 */
export class AuthUserLanguageHooks {
  @HookPostUpgrade()
  static async backfillUserLanguageId(): Promise<void> {
    const db = choysumDb();
    if (!db?.execute || !db.dialectName) return;
    const dialect = String(db.dialectName).trim().toLowerCase();
    let sql = '';
    if (dialect === 'sqlite') {
      sql = `UPDATE auth_user
        SET language_id = (
          SELECT id FROM base_language
          WHERE code = trim(auth_user.language) AND is_active = 1
          LIMIT 1
        )
        WHERE (language_id IS NULL OR language_id = '')
          AND language IS NOT NULL
          AND trim(language) <> ''`;
    } else if (dialect === 'postgres' || dialect === 'mysql') {
      sql = `UPDATE auth_user
        SET language_id = (
          SELECT id FROM base_language
          WHERE code = trim(auth_user.language) AND is_active = true
          LIMIT 1
        )
        WHERE COALESCE(language_id, '') = ''
          AND COALESCE(trim(language), '') <> ''`;
    }
    if (!sql) return;
    try {
      await db.execute(sql, '[]');
    } catch (err) {
      // Old language column may already be absent; users can set LanguageId again.
      console.warn('[auth] User.LanguageId backfill failed', err);
    }
  }
}

/**
 * RoleInheritance is softDelete: false (pure FK link). Physically remove any
 * historical soft-deleted edges so they cannot reappear as live M2M links.
 */
export class AuthRoleInheritanceHooks {
  @HookPostUpgrade()
  static async purgeSoftDeletedRoleInheritance(): Promise<void> {
    const db = choysumDb();
    if (!db?.execute || !db.dialectName) return;
    const dialect = String(db.dialectName).trim().toLowerCase();
    if (dialect !== 'sqlite' && dialect !== 'postgres' && dialect !== 'mysql') return;
    const sql = `DELETE FROM auth_role_inheritance WHERE deleted_at IS NOT NULL`;
    try {
      await db.execute(sql, '[]');
    } catch (err) {
      // Table or column may be absent on fresh installs before schema ensure.
      console.warn('[auth] RoleInheritance soft-delete purge failed', err);
    }
  }
}
