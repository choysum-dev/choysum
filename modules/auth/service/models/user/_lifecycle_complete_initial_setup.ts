// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createServiceByModel } from '@/core/service/rpc';
import { pool, type AppSettingModelCtor } from '@/core/service';
import { withRecordRuleAndFieldRuleBypass } from '@/core/service/api/authz_bypass';
import type MetaModelData from '@/meta/service/models/model_data';
import type Company from '@/base/service/models/company';
import type LanguageModel from '@/base/service/models/language';
import { newAuthError, AuthErrCode, GrpcCode } from '../../error';
import { _t } from '../../i18n';
import Session from '../session';
import Token from '../token';
import { hashPassword, withPermissionGraphBypass } from './_authz_shared';
import { SETUP_COMPLETED_AT_KEY } from './setup_completed_at_key';
import type User from './user';
import type { CompleteInitialSetupOrganization } from './_session_envelopes';

export { SETUP_COMPLETED_AT_KEY } from './setup_completed_at_key';

const MetaModelDataService = createServiceByModel<typeof MetaModelData>('meta.MetaModelData');
const CompanyService = createServiceByModel<typeof Company>('base.Company');
const LanguageService = createServiceByModel<typeof LanguageModel>('base.Language');

/** Matches auth.User Username varchar size. */
const SETUP_USERNAME_MAX_LEN = 100;
/** Matches auth.User Email varchar size. */
const SETUP_EMAIL_MAX_LEN = 100;
/** Same shape check as the Register page email rule. */
const SETUP_EMAIL_PATTERN = /^[\w-]+(\.[\w-]+)*@[\w-]+(\.[\w-]+)+$/;

const MAIN_COMPANY_XML_ID = 'base.company_main';

const ORGANIZATION_FIELDS: ReadonlySet<string> = new Set([
  'Name',
  'CountryCode',
  'CurrencyCode',
  'Timezone',
  'LanguageCode',
]);

export type CompleteInitialSetupParams = {
  Admin: {
    Username: string;
    Password: string;
    LanguageCode: string;
    Email?: string;
  };
  Organization: CompleteInitialSetupOrganization;
};

/** Values written onto the seed admin inside the claim savepoint. */
export type InitialSetupAdminValues = {
  Username: string;
  PasswordHash: string;
  LanguageId: string;
  Email: string | null;
};

export type CompleteInitialSetupResult = {
  UserId: string;
};

function normalizeSetupUsername(raw: unknown): string {
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

function errorMessageOf(err: unknown): string {
  return err instanceof Error
    ? err.message
    : typeof err === 'object' && err !== null && 'message' in err
      ? String((err as { message: unknown }).message)
      : String(err);
}

function isUniqueConstraintError(err: unknown): boolean {
  return /unique constraint|unique index|duplicate key|UNIQUE constraint failed|duplicate entry/i.test(errorMessageOf(err));
}

/**
 * Field-level VALIDATION_FAILED. metadata `Field` is the request path
 * (e.g. `Admin.Email`, `Organization.CurrencyCode`) the setup page maps to a step.
 */
function setupFieldError(field: string, message: string) {
  return newAuthError({ code: AuthErrCode.VALIDATION_FAILED, message })
    .withGrpcCode(GrpcCode.InvalidArgument)
    .withMetadata({ Field: field });
}

function usernameTakenError(username: string) {
  return newAuthError({
    code: AuthErrCode.USERNAME_TAKEN,
    message: _t('Username is already in use', {
      scope: 'service/models/user/_lifecycle_complete_initial_setup',
    }),
  })
    .withGrpcCode(GrpcCode.AlreadyExists)
    .withMetadata({ username, Field: 'Admin.Username' });
}

function emailTakenError(email: string) {
  return newAuthError({
    code: AuthErrCode.EMAIL_TAKEN,
    message: _t('Email is already registered', {
      scope: 'service/models/user/_lifecycle_complete_initial_setup',
    }),
  })
    .withGrpcCode(GrpcCode.AlreadyExists)
    .withMetadata({ email, Field: 'Admin.Email' });
}

/** Trim; empty means "clear". Throws Admin.Email field errors for bad shape or length. */
export function normalizeSetupEmail(raw: unknown): string {
  const email = String(raw ?? '').trim();
  if (!email) return '';
  if (email.length > SETUP_EMAIL_MAX_LEN) {
    throw setupFieldError(
      'Admin.Email',
      _t('Email is too long', { scope: 'service/models/user/_lifecycle_complete_initial_setup' })
    );
  }
  if (!SETUP_EMAIL_PATTERN.test(email)) {
    throw setupFieldError(
      'Admin.Email',
      _t('Enter a valid email address', { scope: 'service/models/user/_lifecycle_complete_initial_setup' })
    );
  }
  return email;
}

/** Re-scope a base field error (metadata `Field`) under `Organization.`; other errors pass through. */
function mapOrganizationError(err: unknown): unknown {
  const field = String((err as { metadata?: Record<string, string> })?.metadata?.Field || '');
  if (!ORGANIZATION_FIELDS.has(field)) return err;
  return setupFieldError(`Organization.${field}`, errorMessageOf(err));
}

/**
 * Validate CompleteInitialSetup username/password and return a store-ready hash.
 */
export function validateAndHashSetupPassword(username: string, password: string): string {
  if (!username) {
    throw setupFieldError(
      'Admin.Username',
      _t('Username and password are required', { scope: 'service/models/user/_lifecycle_complete_initial_setup' })
    );
  }
  if (!password) {
    throw setupFieldError(
      'Admin.Password',
      _t('Username and password are required', { scope: 'service/models/user/_lifecycle_complete_initial_setup' })
    );
  }
  if (username.length > SETUP_USERNAME_MAX_LEN) {
    throw setupFieldError(
      'Admin.Username',
      _t('Username is too long', { scope: 'service/models/user/_lifecycle_complete_initial_setup' })
    );
  }
  return hashPassword(password);
}

async function readSetupCompletedAt(): Promise<string | null> {
  return await withRecordRuleAndFieldRuleBypass(() =>
    pool<AppSettingModelCtor>('auth', 'AppSetting').Get(SETUP_COMPLETED_AT_KEY, null)
  );
}

async function releaseSetupCompletedAt(): Promise<void> {
  await withRecordRuleAndFieldRuleBypass(() =>
    pool<AppSettingModelCtor>('auth', 'AppSetting').Set(SETUP_COMPLETED_AT_KEY, null)
  );
}

/**
 * True when SetupComplete is recorded via auth.AppSetting setup.completed_at.
 * Read-only: anonymous probes must stay write-free.
 */
export async function isInitialSetupComplete(): Promise<boolean> {
  return Boolean(String((await readSetupCompletedAt()) || '').trim());
}

/** Reject when SetupComplete is already recorded. */
async function assertSetupStillPending(): Promise<void> {
  if (await isInitialSetupComplete()) {
    throw alreadyCompleteError();
  }
}

/**
 * Create-only claim of setup.completed_at. Concurrent losers map unique hits to ALREADY_COMPLETE.
 * A blank existing row (pending) is deleted first so Create can reclaim the unique Key.
 */
async function claimSetupCompletedAt(iso: string): Promise<void> {
  await withRecordRuleAndFieldRuleBypass(async () => {
    const store = pool<AppSettingModelCtor>('auth', 'AppSetting');
    const current = await store.Get(SETUP_COMPLETED_AT_KEY, null);
    if (String(current || '').trim()) {
      throw alreadyCompleteError();
    }
    // Blank row still occupies the unique Key; free it before create-only claim.
    // Set(null) hard-deletes (auth.AppSetting softDelete: false).
    if (current !== null) {
      await store.Set(SETUP_COMPLETED_AT_KEY, null);
    }
    try {
      // Create-only via the auth-scoped store (not Set upsert) so a concurrent loser
      // hits the unique key and maps to ALREADY_COMPLETE.
      await (store as any).Create({ Key: SETUP_COMPLETED_AT_KEY, Value: iso });
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
 * Complete initial setup. Order: read-only admin checks → base writes the main
 * company (its own transaction, idempotent) → claim `setup.completed_at` and
 * update the seed admin in one savepoint. The marker is the single commit point:
 * once it exists the organization has been written; a failure after the base
 * write leaves setup pending and a retry overwrites the organization.
 */
export async function runCompleteInitialSetup(
  params: CompleteInitialSetupParams,
  deps: {
    searchByUsername: (username: string) => Promise<Array<{ Id?: string }>>;
    /** Must include soft-deleted rows (Email is unique across them). */
    searchByEmail: (email: string) => Promise<Array<{ Id?: string }>>;
    browseAdmin: (
      adminId: string
    ) => Promise<{ Id?: string; Username?: string; Email?: string | null } | null | undefined>;
    updateAdmin: (adminId: string, values: InitialSetupAdminValues) => Promise<void>;
    /** Active base.Language id for a POSIX code, or null. */
    lookupLanguageId: (code: string) => Promise<string | null>;
    /** Writes the organization onto CompanyId; base field errors carry metadata `Field`. */
    applyOrganization: (req: CompleteInitialSetupOrganization & { CompanyId: string }) => Promise<void>;
    resolveCompanyId?: () => Promise<string | null>;
    nowIso?: () => string;
    resolveAdminId?: () => Promise<string | null>;
    isSetupComplete?: () => Promise<boolean>;
    /** Create-only marker claim; throws INITIAL_SETUP_ALREADY_COMPLETE when lost. */
    claimSetupComplete?: (iso: string) => Promise<void>;
    withSavepoint?: <T>(fn: () => Promise<T>) => Promise<T>;
    revokeAdminSessions?: (adminId: string) => Promise<void>;
  }
): Promise<CompleteInitialSetupResult> {
  const username = normalizeSetupUsername(params.Admin?.Username);
  const password = typeof params.Admin?.Password === 'string' ? params.Admin.Password : '';
  const passwordHash = validateAndHashSetupPassword(username, password);
  const email = normalizeSetupEmail(params.Admin?.Email);
  const adminLanguageCode = String(params.Admin?.LanguageCode ?? '').trim();
  if (!adminLanguageCode) {
    throw setupFieldError(
      'Admin.LanguageCode',
      _t('Language is required', { scope: 'service/models/user/_lifecycle_complete_initial_setup' })
    );
  }

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
      throw usernameTakenError(username);
    }
  }

  if (email && email !== String(admin.Email ?? '').trim()) {
    const existing = await deps.searchByEmail(email);
    const taken = existing.some(row => String(row?.Id || '').trim() !== resolvedAdminId);
    if (taken) {
      throw emailTakenError(email);
    }
  }

  const languageId = String((await deps.lookupLanguageId(adminLanguageCode)) || '').trim();
  if (!languageId) {
    throw setupFieldError(
      'Admin.LanguageCode',
      _t('Invalid or inactive language: %s', { scope: 'service/models/user/_lifecycle_complete_initial_setup' }, adminLanguageCode)
    );
  }

  const companyId = String(
    (deps.resolveCompanyId
      ? await deps.resolveCompanyId()
      : await withPermissionGraphBypass(async () => MetaModelDataService.RefOrNull(MAIN_COMPANY_XML_ID))) || ''
  ).trim();
  if (!companyId) {
    throw newAuthError({
      code: AuthErrCode.INITIAL_COMPANY_NOT_FOUND,
      message: _t('Seed main company was not found', {
        scope: 'service/models/user/_lifecycle_complete_initial_setup',
      }),
    }).withGrpcCode(GrpcCode.FailedPrecondition);
  }

  const org = params.Organization || ({} as CompleteInitialSetupOrganization);
  try {
    await deps.applyOrganization({
      CompanyId: companyId,
      Name: String(org.Name ?? ''),
      ...(org.CountryCode && String(org.CountryCode).trim() ? { CountryCode: String(org.CountryCode) } : {}),
      CurrencyCode: String(org.CurrencyCode ?? ''),
      Timezone: String(org.Timezone ?? ''),
      LanguageCode: String(org.LanguageCode ?? ''),
    });
  } catch (err) {
    throw mapOrganizationError(err);
  }

  const adminValues: InitialSetupAdminValues = {
    Username: username,
    PasswordHash: passwordHash,
    LanguageId: languageId,
    Email: email || null,
  };

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
        await deps.updateAdmin(resolvedAdminId, adminValues);
      } catch (err) {
        // Production wraps claim+update in withSavepoint (rolls back together).
        // Without a savepoint, free the marker so setup can be retried.
        if (!deps.withSavepoint) {
          try {
            await releaseSetupCompletedAt();
          } catch {
            // Best effort; surface the credential-update error.
          }
        }
        if (isUniqueConstraintError(err)) {
          throw /email/i.test(errorMessageOf(err)) && email ? emailTakenError(email) : usernameTakenError(username);
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

  // After marker+credential commit: drop pre-setup seed-admin sessions.
  // Best-effort — must not fail the RPC (setup is already committed; a hard error
  // would strand retries on INITIAL_SETUP_ALREADY_COMPLETE with tokens still live).
  if (deps.revokeAdminSessions) {
    try {
      await deps.revokeAdminSessions(resolvedAdminId);
    } catch {
      // ignore
    }
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
    searchByEmail: async email =>
      (await UserModel.Search(['Email', '=', email], {
        fields: ['Id'],
        limit: 8,
        withDeleted: true,
      } as any)) as Array<{
        Id?: string;
      }>,
    browseAdmin: async adminId => {
      const row = (await UserModel.Browse(adminId, ['Id', 'Username', 'Email'] as any)) as {
        Id?: string;
        Username?: string;
        Email?: string | null;
      } | null;
      return row;
    },
    updateAdmin: async (adminId, values) => {
      // PasswordHash is field-rule denied for normal roles; bypass for this initial-setup write.
      await withRecordRuleAndFieldRuleBypass(async () => {
        await UserModel.UpdateById(adminId, values as any, ['Id']);
      });
    },
    lookupLanguageId: async code => {
      const rows = await LanguageService.Search(
        { And: [['Code', '=', code], ['IsActive', '=', true]] } as any,
        { fields: ['Id'], limit: 1 } as any
      );
      return String((rows?.[0] as { Id?: string } | undefined)?.Id || '').trim() || null;
    },
    applyOrganization: async req => {
      await CompanyService.ApplyInitialSetup(req);
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
