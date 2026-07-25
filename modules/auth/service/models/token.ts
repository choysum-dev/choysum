// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Model, Field, SqlCompute } from '@/core/service';
import { newAuthError, wrapAuthError, GrpcCode, AuthErrCode } from '../error';
import { _t, _lt } from '../i18n';
import User from './user';

/**
 * Token persists access and refresh token metadata for revocation and audit flows.
 */
@Model('Token')
export default class Token extends BaseModel {
  /**
   * Display name derived from TokenId for list and form views.
   */
  @Field({
    type: 'varchar',
    size: 36,
    string: _lt('Display Name', { scope: 'auth.model.Token.fields' }),
  })
  public readonly DisplayName!: string;

  @SqlCompute<Token>('DisplayName')
  sqlDisplayName() {
    return this.$sql.field('TokenId');
  }

  /**
   * User that owns the token row.
   */
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => User, onDelete: 'CASCADE' },
    string: _lt('User', { scope: 'auth.model.Token.fields' }),
  })
  UserId: User;

  /**
   * Stable token identifier embedded in the signed token payload.
   */
  @Field({
    type: 'varchar',
    size: 36,
    notNull: true,
    unique: true,
    copy: false,
    string: _lt('Token ID', { scope: 'auth.model.Token.fields' }),
  })
  TokenId: string;

  /**
   * Token category such as access or refresh.
   */
  @Field({
    type: 'varchar',
    size: 10,
    notNull: true,
    index: true,
    string: _lt('Type', { scope: 'auth.model.Token.fields' }),
  })
  TokenType: string;

  /**
   * Token expiration timestamp.
   */
  @Field({
    type: 'datetime',
    notNull: true,
    index: true,
    copy: false,
    string: _lt('Expires At', { scope: 'auth.model.Token.fields' }),
  })
  ExpiresAt: Date;

  /**
   * Whether the token has been revoked.
   */
  @Field({
    type: 'boolean',
    default: () => false,
    index: true,
    copy: false,
    string: _lt('Revoked', { scope: 'auth.model.Token.fields' }),
  })
  Revoked: boolean;

  /**
   * Time when the token was revoked.
   */
  @Field({
    type: 'datetime',
    index: true,
    copy: false,
    string: _lt('Revoked At', { scope: 'auth.model.Token.fields' }),
  })
  RevokedAt: Date;

  /**
   * Human-readable explanation for the revocation.
   */
  @Field({
    type: 'varchar',
    size: 255,
    copy: false,
    string: _lt('Revocation Reason', { scope: 'auth.model.Token.fields' }),
  })
  RevocationReason: string;

  /**
   * Additional token metadata stored alongside the persisted token row.
   */
  @Field({
    type: 'jsonobject',
    copy: false,
    string: _lt('Metadata', { scope: 'auth.model.Token.fields' }),
  })
  Metadata: Record<string, any>;

  /**
   * Create and persist a new access and refresh token pair.
   */
  static async CreateTokenPair(userId: string, metadata: TokenMetadata = {}): Promise<TokenPair> {
    if (!$choysum.auth.enabled) {
      throw newAuthError({
        code: AuthErrCode.AUTH_SERVICE_DISABLED,
        message: _t('Auth service is disabled', { scope: 'service/models/token' }),
      }).withGrpcCode(GrpcCode.Unavailable);
    }

    try {
      // Delegate token issuance to the auth runtime and persist the resulting identities.
      const tokens = $choysum.auth.createTokens(userId, metadata);

      const accessIdentity = $choysum.auth.validateToken(tokens.accessToken, 'access');
      const refreshIdentity = $choysum.auth.validateToken(tokens.refreshToken, 'refresh');

      // Persist the access token identity.
      await this.Create({
        UserId: { Id: userId },
        TokenId: accessIdentity.tokenId,
        TokenType: 'access',
        ExpiresAt: new Date(tokens.expiresAt),
        Revoked: false,
        Metadata: { ...metadata, payload: accessIdentity },
      });

      // Persist the refresh token identity.
      await this.Create({
        UserId: { Id: userId },
        TokenId: refreshIdentity.tokenId,
        TokenType: 'refresh',
        ExpiresAt: new Date(tokens.refreshExpiresAt),
        Revoked: false,
        Metadata: { ...metadata, payload: refreshIdentity },
      });

      return tokens;
    } catch (error) {
      throw wrapAuthError(error, {
        code: AuthErrCode.TOKEN_CREATION_FAILED,
        message: _t('Failed to create token pair', { scope: 'service/models/token' }),
      }).withMetadata({ userId });
    }
  }

  /**
   * Refresh a token pair and persist the replacement identities.
   */
  static async RefreshTokens(refreshToken: string, metadata?: TokenMetadata): Promise<TokenPair> {
    if (!$choysum.auth.enabled) {
      throw newAuthError({
        code: AuthErrCode.AUTH_SERVICE_DISABLED,
        message: _t('Auth service is disabled', { scope: 'service/models/token' }),
      }).withGrpcCode(GrpcCode.Unavailable);
    }

    try {
      // Pass the latest metadata to the Go auth runtime during refresh.
      const tokens = $choysum.auth.refreshTokens(refreshToken, metadata);

      // Persist the newly issued token identities.
      const accessIdentity = $choysum.auth.validateToken(tokens.accessToken, 'access');
      const refreshIdentity = $choysum.auth.validateToken(tokens.refreshToken, 'refresh');

      // Read the userId from the access token payload.
      const userId = accessIdentity.userId;

      await this.Create({
        UserId: { Id: userId },
        TokenId: accessIdentity.tokenId,
        TokenType: 'access',
        ExpiresAt: new Date(tokens.expiresAt),
        Revoked: false,
        Metadata: { refreshed: true, payload: accessIdentity, ...(metadata || {}) },
      });

      await this.Create({
        UserId: { Id: userId },
        TokenId: refreshIdentity.tokenId,
        TokenType: 'refresh',
        ExpiresAt: new Date(tokens.refreshExpiresAt),
        Revoked: false,
        Metadata: { refreshed: true, payload: refreshIdentity },
      });

      return tokens;
    } catch (error) {
      throw wrapAuthError(error, {
        code: AuthErrCode.TOKEN_REFRESH_FAILED,
        message: _t('Failed to refresh tokens', { scope: 'service/models/token' }),
      });
    }
  }

  /**
   * Revoke one token through the auth runtime.
   */
  static async RevokeToken(token: string, reason: string = ''): Promise<boolean> {
    if (!$choysum.auth.enabled) {
      throw newAuthError({
        code: AuthErrCode.AUTH_SERVICE_DISABLED,
        message: _t('Auth service is disabled', { scope: 'service/models/token' }),
      }).withGrpcCode(GrpcCode.Unavailable);
    }

    try {
      // Delegate revocation to the Go auth runtime to avoid duplicate database work.
      return $choysum.auth.revokeToken(token, reason);
    } catch (error) {
      throw wrapAuthError(error, {
        code: AuthErrCode.TOKEN_REVOCATION_FAILED,
        message: _t('Failed to revoke token', { scope: 'service/models/token' }),
      }).withMetadata({ reason });
    }
  }

  /**
   * Revoke all tokens for a user, optionally excluding one token Id.
   */
  static async RevokeAllUserTokens(userId: string, exceptTokenId?: string, reason: string = ''): Promise<number> {
    if (!$choysum.auth.enabled) {
      throw newAuthError({
        code: AuthErrCode.AUTH_SERVICE_DISABLED,
        message: _t('Auth service is disabled', { scope: 'service/models/token' }),
      }).withGrpcCode(GrpcCode.Unavailable);
    }

    try {
      // Delegate revocation to the Go auth runtime to avoid duplicate database work.
      return $choysum.auth.revokeAllUserTokens(userId, exceptTokenId, reason);
    } catch (error) {
      throw wrapAuthError(error, {
        code: AuthErrCode.TOKEN_REVOCATION_FAILED,
        message: _t('Failed to revoke user tokens', { scope: 'service/models/token' }),
      }).withMetadata({
        userId,
        exceptTokenId: exceptTokenId || '',
        reason,
      });
    }
  }

  /**
   * Revoke all access tokens for a user without touching refresh tokens.
   */
  static async RevokeUserAccessTokens(userId: string, reason: string = ''): Promise<number> {
    if (!$choysum.auth.enabled) {
      throw newAuthError({
        code: AuthErrCode.AUTH_SERVICE_DISABLED,
        message: _t('Auth service is disabled', { scope: 'service/models/token' }),
      }).withGrpcCode(GrpcCode.Unavailable);
    }

    if (!userId) {
      throw newAuthError({
        code: AuthErrCode.VALIDATION_FAILED,
        message: _t('userId is required', { scope: 'service/models/token' }),
      }).withGrpcCode(GrpcCode.InvalidArgument);
    }

    try {
      // Mark the persisted token rows as revoked instead of depending on raw bearer token values.
      const now = new Date();
      const effectiveReason = String(reason || '').trim() || 'revoked in batch';
      const updated = await this.Update(
        {
          And: [
            ['UserId', '=', userId],
            ['TokenType', '=', 'access'],
            ['Revoked', '=', false],
          ],
        },
        {
          Revoked: true,
          RevokedAt: now,
          RevocationReason: effectiveReason,
        }
      );

      return updated.length;
    } catch (error) {
      throw wrapAuthError(error, {
        code: AuthErrCode.TOKEN_REVOCATION_FAILED,
        message: _t('Failed to revoke user access tokens', { scope: 'service/models/token' }),
      }).withMetadata({ userId, reason });
    }
  }

  /**
   * Validate a token and enforce revocation checks.
   */
  static async ValidateToken(token: string, tokenType: string): Promise<TokenIdentity> {
    if (!$choysum.auth.enabled) {
      throw newAuthError({
        code: AuthErrCode.AUTH_SERVICE_DISABLED,
        message: _t('Auth service is disabled', { scope: 'service/models/token' }),
      }).withGrpcCode(GrpcCode.Unavailable);
    }

    try {
      // Ask the Go auth runtime to validate the token and revocation state together.
      return $choysum.auth.validateToken(token, tokenType, true);
    } catch (error) {
      throw wrapAuthError(error, {
        code: AuthErrCode.TOKEN_VALIDATION_FAILED,
        message: _t('Token validation failed', { scope: 'service/models/token' }),
      }).withMetadata({ tokenType });
    }
  }

  /**
   * Mark expired non-revoked tokens as revoked.
   */
  static async CleanExpiredTokens(): Promise<number> {
    // Expired tokens stay queryable, but should no longer be considered active.
    const expiredTokens = await this.Search({
      And: [
        ['ExpiresAt', '<', new Date()],
        ['Revoked', '=', false],
      ],
    });

    for (const token of expiredTokens) {
      await this.UpdateById(token.Id, {
        Revoked: true,
        RevokedAt: new Date(),
        RevocationReason: 'token expired',
      });
    }

    return expiredTokens.length;
  }
}
