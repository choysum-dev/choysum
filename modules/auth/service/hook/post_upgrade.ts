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
 * Scheme B removed TranslationTerm grants from terminology.editor in bootstrap.
 * Init-data does not delete omitted xml_ids; purge leftover rows on upgrade.
 */
export class AuthTerminologyGrantCleanupHooks {
  @HookPostUpgrade()
  static async revokeObsoleteTerminologyEditorGrants(): Promise<void> {
    const db = (globalThis as { $choysum?: { db?: { execute?: (sql: string, params: string) => Promise<unknown> } } }).$choysum?.db;
    if (!db?.execute) return;

    const statements = [
      `DELETE FROM auth_role_method_access WHERE id IN (
        SELECT res_id FROM meta_model_data
        WHERE module = 'auth' AND name = 'rma_terminology_editor_tt_logical'
      )`,
      `DELETE FROM auth_role_field_rule WHERE id IN (
        SELECT res_id FROM meta_model_data
        WHERE module = 'auth' AND name = 'rfr_terminology_editor_tt_logical'
      )`,
      `DELETE FROM meta_model_data
       WHERE module = 'auth'
         AND name IN ('rma_terminology_editor_tt_logical', 'rfr_terminology_editor_tt_logical')`,
    ];

    for (const sql of statements) {
      try {
        await db.execute(sql, '[]');
      } catch (err) {
        console.warn('[auth] obsolete terminology.editor grant cleanup failed', err);
      }
    }
  }
}
