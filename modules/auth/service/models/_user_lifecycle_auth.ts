// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { withContext } from '@/core/service/api/context';
import { createServiceByModel } from '@/core/service/rpc';
import { newAuthError, AuthErrCode, GrpcCode } from '../error';
import type IrModelModel from '@/meta/service/models/ir_model';
import type IrServiceModel from '@/meta/service/models/ir_service';
import type Company from '@/base/service/models/company';
import Role from './role';
import RoleMethodAccess from './role_method_access';
import Session from './session';
import Token from './token';
import UserRole from './user_role';
import { hashPassword, verifyPassword, withPermissionGraphBypass } from './_user_authz_shared';
import { buildScopePreferences } from './_user_lifecycle_scope';

const IrModel = createServiceByModel<typeof IrModelModel>('meta.IrModel');
const IrService = createServiceByModel<typeof IrServiceModel>('meta.IrService');
const CompanyService = createServiceByModel<typeof Company>('base.Company');

export type LoginUserLike = {
  Id: string;
  Username: string;
  PasswordHash: string;
  IsActive: boolean;
  load: (fields: string[]) => Promise<void>;
};

/**
 * Validate registration input and return a hashed password ready for persistence.
 */
export function validateAndHashRegistrationInput(userData: { Username?: string }, password: string): string {
  if (!userData?.Username || !password) {
    throw newAuthError({
      code: AuthErrCode.VALIDATION_FAILED,
      message: 'Username and password are required',
    }).withGrpcCode(GrpcCode.InvalidArgument);
  }
  return hashPassword(password);
}

/**
 * Enforce registration uniqueness for Username and optional Email.
 */
export async function ensureRegistrationIdentityUnique(
  userData: { Username?: string; Email?: string } | null | undefined,
  deps: {
    searchByUsername: (username: string) => Promise<any[]>;
    searchByEmail: (email: string) => Promise<any[]>;
  }
): Promise<void> {
  const existing = await deps.searchByUsername(String(userData?.Username || '').trim());
  if (existing.length > 0) {
    throw newAuthError({
      code: AuthErrCode.USERNAME_TAKEN,
      message: 'Username is already in use',
    })
      .withGrpcCode(GrpcCode.AlreadyExists)
      .withMetadata({ username: String(userData?.Username || '') });
  }

  if (userData?.Email) {
    const emailExists = await deps.searchByEmail(String(userData.Email || '').trim());
    if (emailExists.length > 0) {
      throw newAuthError({
        code: AuthErrCode.EMAIL_TAKEN,
        message: 'Email is already registered',
      })
        .withGrpcCode(GrpcCode.AlreadyExists)
        .withMetadata({ email: userData.Email });
    }
  }
}

/**
 * Validate a created user id and raise an internal registration error when missing.
 */
export function ensureCreatedUserIdOrThrow(createdUserId: any): string {
  const userId = String(createdUserId || '').trim();
  if (!userId) {
    throw newAuthError({
      code: AuthErrCode.USER_CREATION_FAILED,
      message: 'User registration failed: missing user id',
    }).withGrpcCode(GrpcCode.Internal);
  }
  return userId;
}

let baseUserRpcAllowedEnsured = false;

async function ensureBaseUserRpcAllow(roleId: string): Promise<void> {
  if (baseUserRpcAllowedEnsured) return;
  const rid = String(roleId || '').trim();
  if (!rid) return;

  const targets: Array<{ app: string; model: string; method: string }> = [
    { app: 'auth', model: 'User', method: 'Browse' },
    { app: 'auth', model: 'User', method: 'GetPermissionState' },
    { app: 'auth', model: 'User', method: 'SwitchCompanyScope' },
    { app: 'base', model: 'Company', method: 'Search' },
  ];

  const resolveModelId = async (app: string, name: string): Promise<string> => {
    const hit = (
      await IrModel.Search(
        {
          And: [
            ['Application', '=', app],
            ['Name', '=', name],
          ],
        } as any,
        { fields: ['Id'], limit: 1 } as any
      )
    )?.[0] as any;
    return String(hit?.Id || '').trim();
  };

  const resolveService = async (modelId: string, method: string): Promise<{ id: string; name: string }> => {
    const rows = await IrService.Search({ And: [['ModelId', '=', modelId]] } as any, { fields: ['Id', 'Name'], limit: 5000 } as any);
    const target = String(method || '')
      .trim()
      .toLowerCase();
    const hit = (rows || []).find(
      (r: any) =>
        String((r as any).Name || '')
          .trim()
          .toLowerCase() === target
    ) as any;
    const id = String(hit?.Id || '').trim();
    const name = String(hit?.Name || '').trim();
    return { id, name };
  };

  for (const t of targets) {
    const modelId = await resolveModelId(t.app, t.model);
    if (!modelId) continue;

    const svc = await resolveService(modelId, t.method);
    if (!svc.id) continue;

    const existing = await RoleMethodAccess.Search(
      {
        And: [
          ['RoleId', '=', rid],
          ['IrServiceId', '=', svc.id],
          ['IrModelId', 'is', null],
          ['IrApplicationId', 'is', null],
        ],
      } as any,
      { fields: ['Id', 'Mode'], limit: 1 } as any
    );

    if (!existing || existing.length === 0) {
      await RoleMethodAccess.Create(
        {
          RoleId: { Id: rid } as any,
          IrServiceId: svc.id,
          IrModelId: null,
          IrApplicationId: null,
          Mode: 'allow',
        } as any,
        ['Id'] as any
      );
    } else {
      const row: any = existing[0];
      const id = String(row?.Id || '').trim();
      const mode = String(row?.Mode || '').toLowerCase();
      if (id && mode !== 'allow') {
        await RoleMethodAccess.UpdateById(
          id,
          {
            IrServiceId: svc.id,
            IrModelId: null,
            IrApplicationId: null,
            Mode: 'allow',
          } as any,
          ['Id'] as any
        );
      }
    }
  }
  baseUserRpcAllowedEnsured = true;
}

/**
 * Provision the baseline company/role graph for a newly registered user.
 */
export async function provisionRegisteredUserBaseline(
  userId: string,
  rawPreferences: any,
  deps: {
    updateUserCompanyContext: (values: { CompanyId: string; CompanyIds: string[]; Preferences: Record<string, any> }) => Promise<void>;
  }
): Promise<void> {
  await withPermissionGraphBypass(async () => {
    const main = await CompanyService.Search(['Code', '=', 'MAIN'] as any, { fields: ['Id'], limit: 1 } as any);
    const mainCompanyId = String((main as any)?.[0]?.Id || '').trim();
    if (!mainCompanyId) {
      throw newAuthError({
        code: AuthErrCode.VALIDATION_FAILED,
        message: 'Registration failed: Main Company (Code=MAIN) was not found',
      }).withGrpcCode(GrpcCode.FailedPrecondition);
    }

    await withContext({ activeCompanyId: mainCompanyId, enabledCompanyIds: [mainCompanyId] }, async () => {
      const basePrefs: Record<string, any> = rawPreferences && typeof rawPreferences === 'object' && !Array.isArray(rawPreferences) ? rawPreferences : {};
      const nextPrefs = buildScopePreferences(basePrefs, mainCompanyId, [mainCompanyId]);

      await deps.updateUserCompanyContext({
        CompanyId: mainCompanyId,
        CompanyIds: [mainCompanyId],
        Preferences: nextPrefs,
      });

      const baseRole = await Role.Search(
        ['Code', '=', 'base.user'] as any,
        {
          fields: ['Id'],
          limit: 1,
        } as any
      );
      const baseUserRoleId = String((baseRole as any)?.[0]?.Id || '').trim();
      if (!baseUserRoleId) {
        throw newAuthError({
          code: AuthErrCode.ROLE_NOT_FOUND,
          message: 'Registration failed: base.user role is not initialized; load auth bootstrap data first',
        })
          .withGrpcCode(GrpcCode.FailedPrecondition)
          .withMetadata({ roleCode: 'base.user', externalId: 'auth.role_base_user' });
      }

      const existingUserRole = await UserRole.Search(
        {
          And: [
            ['UserId', '=', userId],
            ['RoleId', '=', baseUserRoleId],
            ['CompanyId', '=', mainCompanyId],
          ],
        } as any,
        { fields: ['Id'], limit: 1 } as any
      );

      if (!existingUserRole || existingUserRole.length === 0) {
        await UserRole.Create(
          {
            UserId: { Id: userId } as any,
            RoleId: { Id: baseUserRoleId } as any,
            CompanyId: mainCompanyId,
          } as any,
          ['Id'] as any
        );
      }

      await ensureBaseUserRpcAllow(baseUserRoleId);
    });
  });
}

/**
 * Validate login candidate and enforce password/account checks.
 */
export function validateLoginCandidateOrThrow(user: LoginUserLike | undefined, usernameOrEmail: string, password: string): LoginUserLike {
  if (!user) {
    throw newAuthError({
      code: AuthErrCode.USER_NOT_FOUND,
      message: 'User not found or password is incorrect',
    }).withGrpcCode(GrpcCode.NotFound);
  }

  if (!verifyPassword(password, user.PasswordHash)) {
    throw newAuthError({
      code: AuthErrCode.INVALID_PASSWORD,
      message: 'User not found or password is incorrect',
    })
      .withGrpcCode(GrpcCode.Unauthenticated)
      .withMetadata({ username: usernameOrEmail });
  }

  if (!user.IsActive) {
    throw newAuthError({
      code: AuthErrCode.ACCOUNT_DISABLED,
      message: 'Account is disabled',
    })
      .withGrpcCode(GrpcCode.PermissionDenied)
      .withMetadata({ userId: user.Id, username: user.Username });
  }

  return user;
}

/**
 * Issue a login token pair using the freshest metadata and persist session when needed.
 */
export async function issueLoginTokensAndSession(
  user: LoginUserLike,
  deps: {
    extractUserMetadata: (user: LoginUserLike) => Promise<any>;
    updateLastLogin: (userId: string, timestamp: Date) => Promise<void>;
  },
  opts: {
    ipAddress?: string;
    deviceInfo?: string;
    rememberMe?: boolean;
  }
): Promise<any> {
  await user.load(['CompanyIds']);
  const metadata = await deps.extractUserMetadata(user);
  const tokens = await Token.CreateTokenPair(user.Id, metadata);

  await deps.updateLastLogin(user.Id, new Date());

  if (opts.ipAddress || opts.deviceInfo) {
    const accessIdentity = (globalThis as any).$choysum.auth.validateToken(tokens.accessToken, 'access', false);
    await Session.Create({
      UserId: { Id: user.Id },
      AccessTokenId: String(accessIdentity?.tokenId || '').trim(),
      DeviceInfo: opts.deviceInfo || '',
      IpAddress: opts.ipAddress || '',
      ExpiresAt: new Date(tokens.expiresAt),
      LastActivityAt: new Date(),
      Status: 'active',
      Metadata: {
        rememberMe: !!opts.rememberMe,
        createdAt: Date.now(),
      },
    });
  }

  return tokens;
}

/**
 * Refresh token pair using latest user metadata from storage.
 */
export async function refreshTokensWithLatestMetadata(
  refreshToken: string,
  deps: {
    browseUser: (userId: string) => Promise<any>;
    extractUserMetadata: (user: any) => Promise<any>;
  }
): Promise<any> {
  const id = await Token.ValidateToken(refreshToken, 'refresh');
  const userId = String((id as any)?.userId || '').trim();
  if (!userId) {
    throw newAuthError({
      code: AuthErrCode.VALIDATION_FAILED,
      message: 'Invalid token payload: missing user id',
    }).withGrpcCode(GrpcCode.InvalidArgument);
  }

  const user = await deps.browseUser(userId);
  await user.load(['CompanyIds']);
  const metadata = await deps.extractUserMetadata(user);

  return await Token.RefreshTokens(refreshToken, metadata);
}

/**
 * Revoke token/session artifacts for one-device or all-device logout.
 */
export async function revokeLogoutArtifacts(token: string, allDevices: boolean): Promise<void> {
  const identity = await Token.ValidateToken(token, 'access');
  const userId = String((identity as any)?.userId || '').trim();
  if (!userId) {
    throw newAuthError({
      code: AuthErrCode.VALIDATION_FAILED,
      message: 'Invalid token payload: missing user id',
    }).withGrpcCode(GrpcCode.InvalidArgument);
  }

  if (allDevices) {
    await Token.RevokeAllUserTokens(userId, undefined, 'User initiated logout on all devices');
    await Session.RevokeAllForUser(userId);
    return;
  }

  await Token.RevokeToken(token, 'User initiated logout');

  try {
    const tokenId = String((identity as any)?.tokenId || '').trim();
    const sessions = tokenId ? await Session.Search(['AccessTokenId', '=', tokenId] as any) : [];
    if (sessions.length > 0) {
      await Session.RevokeSession(sessions[0].Id);
    }
  } catch {
    // Ignore lookup failures because a session record may not exist.
  }
}
