// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { HookPostUpgrade } from '@/core/service/api/model';
import { migratePreferencesCompanyScopeToColumns } from './preferences_company_scope_migrate';
import { revokeObsoleteTerminologyEditorGrants } from './terminology_grant_cleanup';

export class AuthTerminologyGrantCleanupHooks {
  @HookPostUpgrade()
  static async cleanupObsoleteTerminologyEditorGrants(): Promise<void> {
    await revokeObsoleteTerminologyEditorGrants();
  }

  @HookPostUpgrade()
  static async migratePreferencesCompanyScope(): Promise<void> {
    await migratePreferencesCompanyScopeToColumns();
  }
}
