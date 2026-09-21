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
 * Anonymous-callable Register user fields. Server-managed columns
 * (Id, Company*, PasswordHash, Preferences, roles, …) are intentionally excluded.
 */
const REGISTER_USER_KEYS = [
  'Username',
  'Email',
  'FirstName',
  'LastName',
  'LanguageId',
  'Timezone',
] as const satisfies readonly (keyof Insertable<User>)[];

export type RegisterUserInput = Partial<Pick<Insertable<User>, (typeof REGISTER_USER_KEYS)[number]>>;

/**
 * Register input. Result is frozen as B: `{ UserId }` (not TokenPair);
 * clients that want a session call Login after Register.
 */
export type RegisterReq = {
  User: RegisterUserInput;
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
  EnabledCompanyIds?: string[];
};

export type LogoutReq = {
  Token: string;
  AllDevices?: boolean;
  DeviceInfo?: string;
};

/** Keep only anonymous-callable Register string fields (runtime over-posting guard). */
export function pickRegisterUserInput(raw: Record<string, unknown>): RegisterUserInput {
  const out: Record<string, unknown> = {};
  for (const key of REGISTER_USER_KEYS) {
    if (!Object.prototype.hasOwnProperty.call(raw, key)) continue;
    const value = raw[key];
    if (typeof value === 'string' && value.trim() !== '') {
      out[key] = value.trim();
    }
  }
  return out as RegisterUserInput;
}
