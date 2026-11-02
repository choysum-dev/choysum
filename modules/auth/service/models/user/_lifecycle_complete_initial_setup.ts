// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createServiceByModel } from '@/core/service/rpc';
import { pool, type AppSettingModelCtor } from '@/core/service';
import { withRecordRuleAndFieldRuleBypass } from '@/core/service/api/authz_bypass';
import type MetaModelData from '@/meta/service/models/model_data';
import { newAuthError, AuthErrCode, GrpcCode } from '../../error';
import { _t } from '../../i18n';
import AppSetting from '../app_setting';
import Session from '../session';
import Token from '../token';
import { hashPassword, withPermissionGraphBypass } from './_authz_shared';
import { SETUP_COMPLETED_AT_KEY } from './setup_completed_at_key';
import type User from './user';

export { SETUP_COMPLETED_AT_KEY } from './setup_completed_at_key';

const MetaModelDataService = createServiceByModel<typeof MetaModelData>('meta.MetaModelData');

/** Matches auth.User Username varchar size. */
const SETUP_USERNAME_MAX_LEN = 100;

const LEGACY_SYSTEM_INIT_DONE_KEY = 'system.init.done';

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

function alreadyCompleteError() {
  return newAuthError({
    code: AuthErrCode.INITIAL_SETUP_ALREADY_COMPLETE,
    message: _t('Initial setup has already been completed', {
      scope: 'service/models/user/_lifecycle_complete_initial_setup',
    }),
  }).withGrpcCode(GrpcCode.FailedPrecondition);
}

function isUniqueConstraintError(err: unknown): boolean {
  const msg =
    err instanceof Error
      ? err.message
      : typeof err === 'object' && err !== null && 'message' in err
        ? String((err as { message: unknown }).message)
        : String(err);
  return /unique constraint|unique index|duplicate key|UNIQUE constraint failed|duplicate entry/i.test(msg);
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
  if (username.length > SETUP_USERNAME_MAX_LEN) {
    throw newAuthError({
      code: AuthErrCode.VALIDATION_FAILED,
      message: _t('Username is too long', {
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

/** True when a pre-PC-3 workspace left system.init.done in meta_setting. */
async function readLegacySystemInitDone(): Promise<boolean> {
  const root = globalThis as { $choysum?: { db?: { dialectName?: string; query?: (sql: string, parameters: string) => Promise<string> } } };
  const db = root.$choysum?.db;
  if (!db?.query) return false;
  const keyCol = db.dialectName === 'mysql' ? '`key`' : '"key"';
  try {
    const raw = await db.query(
      `SELECT value FROM meta_setting WHERE ${keyCol} = ? LIMIT 1`,
      JSON.stringify([LEGACY_SYSTEM_INIT_DONE_KEY])
    );
    const rows = JSON.parse(raw || '[]') as Array<{ value?: string }>;
    const value = String(rows?.[0]?.value ?? '').trim().toLowerCase();
    return value === 'true' || value === '1' || value === 'yes';
  } catch {
    // Table missing or dialect mismatch: treat as no legacy marker.
    return false;
  }
}

/**
 * Reject when SetupComplete is already recorded (new marker or legacy init sentinel).
 * Backfills setup.completed_at when only the legacy marker exists.
 */
async function assertSetupStillPending(): Promise<void> {
  if (String((await readSetupCompletedAt()) || '').trim()) {
    throw alreadyCompleteError();
  }
  if (await readLegacySystemInitDone()) {
    await writeSetupCompletedAt(new Date().toISOString());
    throw alreadyCompleteError();
  }
}

/**
 * Create-only claim of setup.completed_at. Concurrent losers map unique hits to ALREADY_COMPLETE.
 */
async function claimSetupCompletedAt(iso: string): Promise<void> {
  await withRecordRuleAndFieldRuleBypass(async () => {
    const store = pool<AppSettingModelCtor>('auth', 'AppSetting');
    if (String((await store.Get(SETUP_COMPLETED_AT_KEY, null)) || '').trim()) {
      throw alreadyCompleteError();
    }
    try {
      // Create-only (not Set upsert) so a concurrent loser hits the unique key and maps to ALREADY_COMPLETE.
      await AppSetting.Create({ Key: SETUP_COMPLETED_AT_KEY, Value: iso } as any);
    } catch (err) {
      if (isUniqueConstraintError(err)) {
        throw alreadyCompleteError();
      }
      throw err;
    }
    // Create does not clear Get memo; Set same value invalidates request-scoped cache.
    await store.Set(SETUP_COMPLETED_AT_KEY, iso);
  });
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
    /** Create-only marker claim; throws INITIAL_SETUP_ALREADY_COMPLETE when lost. */
    claimSetupComplete?: (iso: string) => Promise<void>;
    withSavepoint?: <T>(fn: () => Promise<T>) => Promise<T>;
    revokeAdminSessions?: (adminId: string) => Promise<void>;
  }
): Promise<CompleteInitialSetupResult> {
  const username = normalizeSetupUsername(params.Username);
  const password = typeof params.Password === 'string' ? params.Password : '';
  const passwordHash = validateAndHashSetupPassword(username, password);

  const alreadyComplete = deps.isSetupComplete
    ? await deps.isSetupComplete()
    : false;
  if (alreadyComplete) {
    throw alreadyCompleteError();
  }
  if (!deps.isSetupComplete) {
    await assertSetupStillPending();
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
    const run = async () => {
      // Re-check inside the write window so concurrent callers cannot both update admin.
      if (deps.isSetupComplete) {
        if (await deps.isSetupComplete()) {
          throw alreadyCompleteError();
        }
      } else {
        await assertSetupStillPending();
      }

      const iso = deps.nowIso ? deps.nowIso() : new Date().toISOString();
      if (deps.claimSetupComplete) {
        await deps.claimSetupComplete(iso);
      } else {
        await claimSetupCompletedAt(iso);
      }

      try {
        await deps.updateAdmin(resolvedAdminId, { Username: username, PasswordHash: passwordHash });
      } catch (err) {
        if (isUniqueConstraintError(err)) {
          throw newAuthError({
            code: AuthErrCode.USERNAME_TAKEN,
            message: _t('Username is already in use', {
              scope: 'service/models/user/_lifecycle_complete_initial_setup',
            }),
          })
            .withGrpcCode(GrpcCode.AlreadyExists)
            .withMetadata({ username });
        }
        throw err;
      }
    };

    if (deps.withSavepoint) {
      await deps.withSavepoint(run);
    } else {
      await run();
    }
  });

  // After the marker+credential commit: drop any pre-setup seed-admin sessions.
  if (deps.revokeAdminSessions) {
    await deps.revokeAdminSessions(resolvedAdminId);
  }

  return { UserId: resolvedAdminId };
}

/** Wire CompleteInitialSetup onto auth.User (browse/update/search via model). */
export async function completeInitialSetupForUser(
  UserModel: typeof User,
  params: CompleteInitialSetupParams
): Promise<CompleteInitialSetupResult> {
  return await runCompleteInitialSetup(params, {
    searchByUsername: async username =>
      (await UserModel.Search(['Username', '=', username], {
        fields: ['Id'],
        limit: 8,
        withDeleted: true,
      } as any)) as Array<{
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
      // PasswordHash is field-rule denied for normal roles; bypass for this first-run write.
      await withRecordRuleAndFieldRuleBypass(async () => {
        await UserModel.UpdateById(adminId, values as any, ['Id']);
      });
    },
    withSavepoint: fn => UserModel.withSavepoint(fn),
    revokeAdminSessions: async adminId => {
      await Token.revokeAllUserTokens(adminId, undefined, 'CompleteInitialSetup rotated seed admin credentials');
      await withRecordRuleAndFieldRuleBypass(async () => {
        await Session.RevokeAllForUser(adminId);
      });
    },
  });
}
