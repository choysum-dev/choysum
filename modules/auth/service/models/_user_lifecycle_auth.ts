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
  const username = String(userData?.Username || '').trim();
  if (!username) {
    throw newAuthError({
      code: AuthErrCode.VALIDATION_FAILED,
      message: 'Username is required',
    }).withGrpcCode(GrpcCode.InvalidArgument);
  }
  const existing = await deps.searchByUsername(username);
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

// TODO: Move base.user permission seeding to system bootstrap/migration to avoid
// redundant DB queries and race conditions on registration.
async function ensureBaseUserRpcAllow(roleId: string): Promise<void> {
  const rid = String(roleId || '').trim();
  if (!rid) return;

  const targets: Array<{ app: string; model: string; method: string }> = [
    { app: 'auth', model: 'User', method: 'Browse' },
    { app: 'auth', model: 'User', method: 'GetPermissionState' },
    { app: 'auth', model: 'User', method: 'SwitchCompanyScope' },
    { app: 'base', model: 'Company', method: 'Search' },
  ];

  const modelCache = new Map<string, string>();
  const serviceCache = new Map<string, Array<{ Id: string; Name: string }>>();

  const resolveModelId = async (app: string, name: string): Promise<string> => {
    const key = app + '.' + name;
    if (modelCache.has(key)) return modelCache.get(key)!;
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
    const id = String(hit?.Id || '').trim();
    modelCache.set(key, id);
    return id;
  };

  const resolveService = async (modelId: string, method: string): Promise<{ id: string; name: string }> => {
    if (!serviceCache.has(modelId)) {
      const rows = await IrService.Search({ And: [['ModelId', '=', modelId]] } as any, { fields: ['Id', 'Name'], limit: 5000 } as any);
      serviceCache.set(
        modelId,
        (rows || []).map((r: any) => ({ Id: String(r?.Id || '').trim(), Name: String(r?.Name || '').trim() }))
      );
    }
    const cached = serviceCache.get(modelId)!;
    const target = String(method || '')
      .trim()
      .toLowerCase();
    const hit = cached.find(r => r.Name.toLowerCase() === target);
    return { id: hit?.Id || '', name: hit?.Name || '' };
  };

  const resolvedServiceIds: string[] = [];
  for (const t of targets) {
    const modelId = await resolveModelId(t.app, t.model);
    if (!modelId) continue;

    const svc = await resolveService(modelId, t.method);
    if (!svc.id) continue;
    resolvedServiceIds.push(svc.id);
  }

  const serviceIds = Array.from(new Set(resolvedServiceIds));
  if (serviceIds.length === 0) return;

  const existingRows = await RoleMethodAccess.Search(
    {
      And: [
        ['RoleId', '=', rid],
        ['IrServiceId', 'in', serviceIds],
        ['IrModelId', 'is', null],
        ['IrApplicationId', 'is', null],
      ],
    } as any,
    { fields: ['Id', 'IrServiceId', 'Mode'], limit: Math.max(16, serviceIds.length * 2) } as any
  );

  const existingByServiceId = new Map<string, any>();
  for (const row of existingRows || []) {
    const serviceId = String((row as any)?.IrServiceId || '').trim();
    if (!serviceId) continue;
    if (!existingByServiceId.has(serviceId)) {
      existingByServiceId.set(serviceId, row as any);
    }
  }

  const createPayloads: any[] = [];
  const updatePayloads: Array<{ id: string; serviceId: string }> = [];

  for (const serviceId of serviceIds) {
    const existing = existingByServiceId.get(serviceId);
    if (!existing) {
      createPayloads.push({
        RoleId: { Id: rid } as any,
        IrServiceId: serviceId,
        IrModelId: null,
        IrApplicationId: null,
        Mode: 'allow',
      });
      continue;
    }

    const id = String((existing as any)?.Id || '').trim();
    const mode = String((existing as any)?.Mode || '').toLowerCase();
    if (id && mode !== 'allow') {
      updatePayloads.push({ id, serviceId });
    }
  }

  if (createPayloads.length > 0) {
    await RoleMethodAccess.CreateMany(createPayloads as any, ['Id'] as any);
  }

  for (const item of updatePayloads) {
    await RoleMethodAccess.UpdateById(
      item.id,
      {
        IrServiceId: item.serviceId,
        IrModelId: null,
        IrApplicationId: null,
        Mode: 'allow',
      } as any,
      ['Id'] as any
    );
  }
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
  } = {}
): Promise<any> {
  await user.load(['CompanyIds']);
  const metadata = await deps.extractUserMetadata(user);
  const tokens = await Token.CreateTokenPair(user.Id, metadata);
  if (!tokens || !tokens.accessToken) {
    throw new Error('Token creation failed: missing access token');
  }

  await deps.updateLastLogin(user.Id, new Date());

  if (opts.ipAddress || opts.deviceInfo) {
    const auth = (globalThis as any)?.$choysum?.auth;
    if (!auth) {
      throw new Error('Choysum auth subsystem is not initialized');
    }
    const accessIdentity = auth.validateToken(tokens.accessToken, 'access', false);
    const tokenId = String(accessIdentity?.tokenId || '').trim();
    if (!tokenId) {
      throw new Error('Failed to validate issued access token');
    }
    await Session.Create({
      UserId: { Id: user.Id },
      AccessTokenId: tokenId,
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
  const identity = await Token.ValidateToken(refreshToken, 'refresh');
  const userId = String((identity as any)?.userId || '').trim();
  if (!userId) {
    throw newAuthError({
      code: AuthErrCode.VALIDATION_FAILED,
      message: 'Invalid token payload: missing user id',
    }).withGrpcCode(GrpcCode.InvalidArgument);
  }

  const user = await deps.browseUser(userId);
  if (!user) {
    throw newAuthError({
      code: AuthErrCode.USER_NOT_FOUND,
      message: 'User not found',
    }).withGrpcCode(GrpcCode.NotFound);
  }
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
