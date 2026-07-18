// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Model, Field } from '@/core/service';
import { wrapAuthError, AuthErrCode } from '../error';
import { _t } from '../i18n';
import User from './user';

/**
 * Session tracks one authenticated browser or device session for a user.
 */
@Model('Session')
export default class Session extends BaseModel {
  /**
   * User that owns the session.
   */
  @Field({ type: 'ManyToOne', relation: { targetModel: () => User, onDelete: 'CASCADE' } })
  UserId: User;

  /**
   * Access token identifier bound to the session. This is the token JTI, not a bearer secret.
   */
  @Field({ type: 'varchar', size: 36, index: true})
  AccessTokenId: string;

  /**
   * Device fingerprint or other client metadata captured at login.
   */
  @Field({ type: 'text' })
  DeviceInfo: string;

  /**
   * Client IP address recorded for the session.
   */
  @Field({ type: 'varchar', size: 45})
  IpAddress: string;

  /**
   * Session expiration timestamp.
   */
  @Field({ type: 'datetime', notNull: true, index: true})
  ExpiresAt: Date;

  /**
   * Last observed activity time for the session.
   */
  @Field({ type: 'datetime', index: true})
  LastActivityAt: Date;

  /**
   * Session lifecycle state such as active, expired, or revoked.
   */
  @Field({ type: 'varchar', size: 20, default: () => 'active', index: true})
  Status: string;

  /**
   * Arbitrary session metadata captured by the auth flow.
   */
  @Field({ type: 'jsonobject' })
  Metadata: Record<string, any>;

  /**
   * Revoke one session by Id.
   */
  static async RevokeSession(sessionId: string): Promise<boolean> {
    try {
      await this.UpdateById(sessionId, {
        Status: 'revoked',
        LastActivityAt: new Date(),
      });
      return true;
    } catch (error) {
      throw wrapAuthError(error, {
        code: AuthErrCode.SESSION_REVOCATION_FAILED,
        message: _t('Failed to revoke session', { scope: 'service/models/session' }),
      }).withMetadata({ sessionId });
    }
  }

  /**
   * Revoke all active sessions for a user, optionally keeping one session alive.
   */
  static async RevokeAllForUser(userId: string, exceptSessionId?: string): Promise<number> {
    try {
      const condition: any = {
        And: [
          ['UserId', '=', userId],
          ['Status', '=', 'active'],
        ],
      };

      if (exceptSessionId) {
        condition.And.push(['Id', '!=', exceptSessionId]);
      }

      const result = await this.Update(condition, {
        Status: 'revoked',
        LastActivityAt: new Date(),
      });
      return result.length;
    } catch (error) {
      throw wrapAuthError(error, {
        code: AuthErrCode.SESSION_REVOCATION_FAILED,
        message: _t('Failed to revoke user sessions', { scope: 'service/models/session' }),
      }).withMetadata({
        userId,
        exceptSessionId: exceptSessionId || '',
      });
    }
  }
}
