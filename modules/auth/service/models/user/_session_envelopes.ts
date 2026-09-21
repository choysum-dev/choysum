// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { Insertable } from '@/core/service/api/input';
import type User from './user';

/**
 * Session command envelopes for auth.User public verbs (PR-W2).
 * Identity axes still come from the runtime session where applicable;
 * these envelopes carry only command inputs.
 */

export type LoginReq = {
  UsernameOrEmail: string;
  Password: string;
  IpAddress?: string;
  DeviceInfo?: string;
  RememberMe?: boolean;
};

/**
 * Register input. Result is frozen as B: `{ UserId }` (not TokenPair);
 * clients that want a session call Login after Register.
 */
export type RegisterReq = {
  User: Partial<Insertable<User>>;
  Password: string;
};

export type RegisterResp = {
  UserId: string;
};

export type RefreshTokensReq = {
  RefreshToken: string;
};

export type SwitchCompanyScopeReq = {
  ActiveCompanyId: string;
  /** Readable company scope; omit to default from Preferences / [ActiveCompanyId]. */
  EnabledCompanyIds?: string[] | null;
};

export type LogoutReq = {
  Token: string;
  AllDevices?: boolean;
  DeviceInfo?: string;
};
