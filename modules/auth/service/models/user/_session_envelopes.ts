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
 * (Id, Company*, ActiveCompanyId, EnabledCompanyIds, PasswordHash, roles, …) are intentionally excluded.
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

/** Seed-admin section of CompleteInitialSetup. */
export type CompleteInitialSetupAdmin = {
  Username: string;
  /** Client-hashed password (same envelope as Login / Register). */
  Password: string;
  /** Admin UI language (POSIX code of an active base.Language). */
  LanguageCode: string;
  /** Optional; empty or omitted clears the seed placeholder email. */
  Email?: string;
};

/** Organization section of CompleteInitialSetup; written by base.Company/ApplyInitialSetup. */
export type CompleteInitialSetupOrganization = {
  Name: string;
  CountryCode?: string;
  CurrencyCode: string;
  /** Business timezone (IANA) written to Company.Timezone. */
  Timezone: string;
};

/**
 * First-run setup: replace seed admin credentials, write the main company,
 * and mark SetupComplete. Result is `{ UserId }` only; callers Login for a session.
 */
export type CompleteInitialSetupReq = {
  Admin: CompleteInitialSetupAdmin;
  Organization: CompleteInitialSetupOrganization;
};

export type CompleteInitialSetupResp = {
  UserId: string;
};

function pickString(src: Record<string, unknown>, key: string): string | undefined {
  const value = src[key];
  return typeof value === 'string' ? value : undefined;
}

function asRecord(value: unknown): Record<string, unknown> | null {
  return value && typeof value === 'object' && !Array.isArray(value) ? (value as Record<string, unknown>) : null;
}

/**
 * Whitelist CompleteInitialSetup sections; unknown keys and non-string values are dropped.
 * Returns null for a section that is missing or not an object.
 */
export function pickCompleteInitialSetupReq(req: Record<string, unknown>): {
  Admin: Partial<CompleteInitialSetupAdmin> | null;
  Organization: Partial<CompleteInitialSetupOrganization> | null;
} {
  const admin = asRecord(req.Admin);
  const org = asRecord(req.Organization);
  return {
    Admin: admin
      ? {
          Username: pickString(admin, 'Username'),
          Password: pickString(admin, 'Password'),
          LanguageCode: pickString(admin, 'LanguageCode'),
          Email: pickString(admin, 'Email'),
        }
      : null,
    Organization: org
      ? {
          Name: pickString(org, 'Name'),
          CountryCode: pickString(org, 'CountryCode'),
          CurrencyCode: pickString(org, 'CurrencyCode'),
          Timezone: pickString(org, 'Timezone'),
        }
      : null,
  };
}

/**
 * Anonymous setup probe for FE routing. Empty payload; Complete mirrors
 * AppSetting setup.completed_at.
 */
export type GetInitialSetupStateReq = Record<string, never>;

export type GetInitialSetupStateResp = {
  Complete: boolean;
};

export type RefreshTokensReq = {
  RefreshToken: string;
};

export type SwitchCompanyScopeReq = {
  ActiveCompanyId: string;
  /** Readable company scope; omit to default from stored EnabledCompanyIds / [ActiveCompanyId]. */
  EnabledCompanyIds?: string[];
};

export type LogoutReq = {
  Token: string;
  AllDevices?: boolean;
  DeviceInfo?: string;
};

/**
 * Self-service password change. Identity comes from the session; passwords are
 * client-hashed the same way as Login (`$CH$…` accepted by hashPassword/verifyPassword).
 */
export type ChangePasswordReq = {
  CurrentPassword: string;
  NewPassword: string;
};

export type ChangePasswordResp = {
  Ok: boolean;
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
