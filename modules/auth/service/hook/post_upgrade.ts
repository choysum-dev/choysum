// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { HookPostUpgrade } from '@/core/service/api/model';

type ChoysumDb = {
  execute?: (sql: string, params: string) => Promise<unknown>;
};

function resolveDb(): ChoysumDb | undefined {
  return (globalThis as { $choysum?: { db?: ChoysumDb } }).$choysum?.db;
}

/**
 * Scheme B removed TranslationTerm grants from terminology.editor in bootstrap.
 * Init-data does not delete omitted xml_ids; purge leftover rows on upgrade.
 */
export class AuthTerminologyGrantCleanupHooks {
  @HookPostUpgrade()
  static async revokeObsoleteTerminologyEditorGrants(): Promise<void> {
    const db = resolveDb();
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

    let grantDeletesOk = true;
    for (const [index, sql] of statements.entries()) {
      const isMappingCleanup = index === statements.length - 1;
      if (isMappingCleanup && !grantDeletesOk) break;
      try {
        await db.execute(sql, '[]');
      } catch (err) {
        if (!isMappingCleanup) grantDeletesOk = false;
        console.warn('[auth] obsolete terminology.editor grant cleanup failed', err);
      }
    }
  }
}
