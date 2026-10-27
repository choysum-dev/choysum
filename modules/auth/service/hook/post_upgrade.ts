// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { HookPostUpgrade, HookPreUpgrade } from '@/core/service/api/model';

type ChoysumDb = {
  dialectName?: string;
  execute?: (sql: string, params: string) => Promise<unknown>;
};

function choysumDb(): ChoysumDb | undefined {
  return (globalThis as { $choysum?: { db?: ChoysumDb } }).$choysum?.db;
}

/** True only for missing auth_role_inheritance table or deleted_at column — not generic "does not exist". */
function isMissingRelationError(err: unknown): boolean {
  const msg = String(err instanceof Error ? err.message : err).toLowerCase();

  // Table missing (sqlite / mysql / mssql / postgres relation name).
  if (
    msg.includes('no such table') ||
    msg.includes('unknown table') ||
    msg.includes('invalid object name') ||
    (msg.includes('auth_role_inheritance') &&
      (msg.includes('does not exist') || msg.includes("doesn't exist")))
  ) {
    return true;
  }

  // deleted_at column missing — require the column name so unrelated "does not exist" is not swallowed.
  if (!msg.includes('deleted_at')) return false;
  return (
    msg.includes('no such column') ||
    msg.includes('unknown column') ||
    msg.includes('invalid column name') ||
    msg.includes('does not exist') ||
    msg.includes("doesn't exist")
  );
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
 *
 * Runs on pre_upgrade (required phase) so failures abort the upgrade. Missing
 * table/column is ignored for installs that never had this relation.
 */
export class AuthRoleInheritanceHooks {
  @HookPreUpgrade()
  static async purgeSoftDeletedRoleInheritance(): Promise<void> {
    const db = choysumDb();
    if (!db?.execute) return;
    // Portable DELETE — no dialect-specific SQL. Covers sqlite/postgres/mysql/mssql.
    const sql = `DELETE FROM auth_role_inheritance WHERE deleted_at IS NOT NULL`;
    try {
      await db.execute(sql, '[]');
    } catch (err) {
      if (isMissingRelationError(err)) return;
      throw err;
    }
  }
}
