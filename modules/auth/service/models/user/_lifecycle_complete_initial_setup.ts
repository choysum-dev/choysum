// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createServiceByModel } from '@/core/service/rpc';
import { pool, type AppSettingModelCtor } from '@/core/service';
import { withRecordRuleAndFieldRuleBypass } from '@/core/service/api/authz_bypass';
import type MetaModelData from '@/meta/service/models/model_data';
import { newAuthError, AuthErrCode, GrpcCode } from '../../error';
import { _t } from '../../i18n';
import { hashPassword, withPermissionGraphBypass } from './_authz_shared';
import { SETUP_COMPLETED_AT_KEY } from './setup_completed_at_key';
import type User from './user';

export { SETUP_COMPLETED_AT_KEY } from './setup_completed_at_key';

const MetaModelDataService = createServiceByModel<typeof MetaModelData>('meta.MetaModelData');

export type CompleteInitialSetupParams = {
  Username: string;
  Password: string;
};

export type CompleteInitialSetupResult = {
  UserId: string;
};

function normalizeSetupUsername(raw: string): string {
  return String(raw || '').trim();
}

/**
 * Validate CompleteInitialSetup username/password and return a store-ready hash.
 */
export function validateAndHashSetupPassword(username: string, password: string): string {
  if (!username || !password) {
    throw newAuthError({
      code: AuthErrCode.VALIDATION_FAILED,
      message: _t('Username and password are required', {
        scope: 'service/models/user/_lifecycle_complete_initial_setup',
      }),
    }).withGrpcCode(GrpcCode.InvalidArgument);
  }
  return hashPassword(password);
}

async function readSetupCompletedAt(): Promise<string | null> {
  return await withRecordRuleAndFieldRuleBypass(() =>
    pool<AppSettingModelCtor>('auth', 'AppSetting').Get(SETUP_COMPLETED_AT_KEY, null)
  );
}

async function writeSetupCompletedAt(iso: string): Promise<void> {
  await withRecordRuleAndFieldRuleBypass(() =>
    pool<AppSettingModelCtor>('auth', 'AppSetting').Set(SETUP_COMPLETED_AT_KEY, iso)
  );
}

/**
 * Complete first-run setup: update seed admin credentials and mark SetupComplete.
 */
export async function runCompleteInitialSetup(
  params: CompleteInitialSetupParams,
  deps: {
    searchByUsername: (username: string) => Promise<Array<{ Id?: string }>>;
    browseAdmin: (adminId: string) => Promise<{ Id?: string; Username?: string } | null | undefined>;
    updateAdmin: (
      adminId: string,
      values: { Username: string; PasswordHash: string }
    ) => Promise<void>;
    nowIso?: () => string;
    resolveAdminId?: () => Promise<string | null>;
    isSetupComplete?: () => Promise<boolean>;
    markSetupComplete?: (iso: string) => Promise<void>;
  }
): Promise<CompleteInitialSetupResult> {
  const username = normalizeSetupUsername(params.Username);
  const password = typeof params.Password === 'string' ? params.Password : '';
  const passwordHash = validateAndHashSetupPassword(username, password);

  const alreadyComplete = deps.isSetupComplete
    ? await deps.isSetupComplete()
    : Boolean(String((await readSetupCompletedAt()) || '').trim());
  if (alreadyComplete) {
    throw newAuthError({
      code: AuthErrCode.INITIAL_SETUP_ALREADY_COMPLETE,
      message: _t('Initial setup has already been completed', {
        scope: 'service/models/user/_lifecycle_complete_initial_setup',
      }),
    }).withGrpcCode(GrpcCode.FailedPrecondition);
  }

  const adminId = deps.resolveAdminId
    ? await deps.resolveAdminId()
    : await withPermissionGraphBypass(async () => MetaModelDataService.RefOrNull('auth.user_admin'));
  const resolvedAdminId = String(adminId || '').trim();
  if (!resolvedAdminId) {
    throw newAuthError({
      code: AuthErrCode.INITIAL_ADMIN_NOT_FOUND,
      message: _t('Seed administrator account was not found', {
        scope: 'service/models/user/_lifecycle_complete_initial_setup',
      }),
    }).withGrpcCode(GrpcCode.FailedPrecondition);
  }

  const admin = await deps.browseAdmin(resolvedAdminId);
  if (!admin || !String(admin.Id || '').trim()) {
    throw newAuthError({
      code: AuthErrCode.INITIAL_ADMIN_NOT_FOUND,
      message: _t('Seed administrator account was not found', {
        scope: 'service/models/user/_lifecycle_complete_initial_setup',
      }),
    }).withGrpcCode(GrpcCode.FailedPrecondition);
  }

  const currentUsername = String(admin.Username || '').trim();
  if (username !== currentUsername) {
    const existing = await deps.searchByUsername(username);
    const taken = existing.some(row => String(row?.Id || '').trim() !== resolvedAdminId);
    if (taken) {
      throw newAuthError({
        code: AuthErrCode.USERNAME_TAKEN,
        message: _t('Username is already in use', {
          scope: 'service/models/user/_lifecycle_complete_initial_setup',
        }),
      })
        .withGrpcCode(GrpcCode.AlreadyExists)
        .withMetadata({ username });
    }
  }

  await withPermissionGraphBypass(async () => {
    await deps.updateAdmin(resolvedAdminId, { Username: username, PasswordHash: passwordHash });
    const iso = deps.nowIso ? deps.nowIso() : new Date().toISOString();
    if (deps.markSetupComplete) {
      await deps.markSetupComplete(iso);
    } else {
      await writeSetupCompletedAt(iso);
    }
  });

  return { UserId: resolvedAdminId };
}

/** Wire CompleteInitialSetup onto auth.User (browse/update/search via model). */
export async function completeInitialSetupForUser(
  UserModel: typeof User,
  params: CompleteInitialSetupParams
): Promise<CompleteInitialSetupResult> {
  return await runCompleteInitialSetup(params, {
    searchByUsername: async username =>
      (await UserModel.Search(['Username', '=', username], { fields: ['Id'], limit: 8 })) as Array<{
        Id?: string;
      }>,
    browseAdmin: async adminId => {
      const row = (await UserModel.Browse(adminId, ['Id', 'Username'] as any)) as {
        Id?: string;
        Username?: string;
      } | null;
      return row;
    },
    updateAdmin: async (adminId, values) => {
      await UserModel.UpdateById(adminId, values as any, ['Id']);
    },
  });
}
