// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { withContext, getContextClientTimezone } from '@/core/service/api/context';
import { createServiceByModel } from '@/core/service/rpc';
import { isIanaTimezone } from '@/core/service/utils/datetime';
import { newAuthError, AuthErrCode, GrpcCode } from '../error';
import { _t } from '../i18n';
import type Company from '@/base/service/models/company';
import Role from './role';
import Session from './session';
import Token from './token';
import UserRole from './user_role';
import { hashPassword, verifyPassword, withPermissionGraphBypass } from './_user_authz_shared';
import { buildScopePreferences } from './_user_lifecycle_scope';

const CompanyService = createServiceByModel<typeof Company>('base.Company');

export type LoginUserLike = {
  Id: string;
  Username: string;
  PasswordHash: string;
  IsActive: boolean;
  Timezone?: string | null;
  load: (fields: string[]) => Promise<void>;
};

/**
 * D20: when User.Timezone is empty, return a valid baggage client IANA to persist.
 * Never returns a value when the user already has a timezone or clientTz is missing/invalid.
 */
export function resolveTimezoneToPersist(
  userTimezone: string | null | undefined,
  clientTimezone: string | null | undefined
): string | undefined {
  if (String(userTimezone || '').trim()) {
    return undefined;
  }
  const candidate = String(clientTimezone || '').trim();
  if (!candidate || !isIanaTimezone(candidate)) {
    return undefined;
  }
  return candidate;
}

/**
 * Persist browser/client IANA onto User when Timezone is empty (D20).
 * Returns the user (possibly reloaded) so token metadata sees the new value.
 */
export async function persistBrowserTimezoneIfEmpty<T extends { Id: string; Timezone?: string | null }>(
  user: T,
  deps: {
    updateTimezone: (userId: string, timezone: string) => Promise<void>;
    reloadUser?: (userId: string) => Promise<T>;
    clientTimezone?: string | null;
  }
): Promise<T> {
  const clientTz = deps.clientTimezone !== undefined ? deps.clientTimezone : getContextClientTimezone();
  const next = resolveTimezoneToPersist(user.Timezone, clientTz);
  if (!next) {
    return user;
  }
  const userId = String(user.Id || '').trim();
  if (!userId) {
    return user;
  }
  await withPermissionGraphBypass(async () => {
    await deps.updateTimezone(userId, next);
  });
  if (deps.reloadUser) {
    return await deps.reloadUser(userId);
  }
  (user as any).Timezone = next;
  return user;
}

/**
 * Validate registration input and return a hashed password ready for persistence.
 */
export function validateAndHashRegistrationInput(userData: { Username?: string }, password: string): string {
  if (!userData?.Username || !password) {
    throw newAuthError({
      code: AuthErrCode.VALIDATION_FAILED,
      message: _t('Username and password are required', { scope: 'service/models/_user_lifecycle_auth' }),
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
      message: _t('Username is required', { scope: 'service/models/_user_lifecycle_auth' }),
    }).withGrpcCode(GrpcCode.InvalidArgument);
  }
  const existing = await deps.searchByUsername(username);
  if (existing.length > 0) {
    throw newAuthError({
      code: AuthErrCode.USERNAME_TAKEN,
      message: _t('Username is already in use', { scope: 'service/models/_user_lifecycle_auth' }),
    })
      .withGrpcCode(GrpcCode.AlreadyExists)
      .withMetadata({ username: String(userData?.Username || '') });
  }

  const email = String(userData?.Email ?? '').trim();
  if (email) {
    const emailExists = await deps.searchByEmail(email);
    if (emailExists.length > 0) {
      throw newAuthError({
        code: AuthErrCode.EMAIL_TAKEN,
        message: _t('Email is already registered', { scope: 'service/models/_user_lifecycle_auth' }),
      })
        .withGrpcCode(GrpcCode.AlreadyExists)
        .withMetadata({ email });
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
      message: _t('User registration failed: missing user id', { scope: 'service/models/_user_lifecycle_auth' }),
    }).withGrpcCode(GrpcCode.Internal);
  }
  return userId;
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
        message: _t('Registration failed: Main Company (Code=MAIN) was not found', { scope: 'service/models/_user_lifecycle_auth' }),
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
          message: _t('Registration failed: base.user role is not initialized; load auth bootstrap data first', { scope: 'service/models/_user_lifecycle_auth' }),
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
      message: _t('User not found or password is incorrect', { scope: 'service/models/_user_lifecycle_auth' }),
    }).withGrpcCode(GrpcCode.NotFound);
  }

  if (!verifyPassword(password, user.PasswordHash)) {
    throw newAuthError({
      code: AuthErrCode.INVALID_PASSWORD,
      message: _t('User not found or password is incorrect', { scope: 'service/models/_user_lifecycle_auth' }),
    })
      .withGrpcCode(GrpcCode.Unauthenticated)
      .withMetadata({ username: usernameOrEmail });
  }

  if (!user.IsActive) {
    throw newAuthError({
      code: AuthErrCode.ACCOUNT_DISABLED,
      message: _t('Account is disabled', { scope: 'service/models/_user_lifecycle_auth' }),
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
      message: _t('Invalid token payload: missing user id', { scope: 'service/models/_user_lifecycle_auth' }),
    }).withGrpcCode(GrpcCode.InvalidArgument);
  }

  const user = await deps.browseUser(userId);
  if (!user) {
    throw newAuthError({
      code: AuthErrCode.USER_NOT_FOUND,
      message: _t('User not found', { scope: 'service/models/_user_lifecycle_auth' }),
    }).withGrpcCode(GrpcCode.NotFound);
  }
  if (!user.IsActive) {
    throw newAuthError({
      code: AuthErrCode.ACCOUNT_DISABLED,
      message: _t('Account is disabled', { scope: 'service/models/_user_lifecycle_auth' }),
    })
      .withGrpcCode(GrpcCode.PermissionDenied)
      .withMetadata({ userId: user.Id, username: user.Username });
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
      message: _t('Invalid token payload: missing user id', { scope: 'service/models/_user_lifecycle_auth' }),
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
