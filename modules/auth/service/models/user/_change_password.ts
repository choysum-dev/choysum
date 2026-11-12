// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { newAuthError, AuthErrCode, GrpcCode } from '../../error';
import { _t } from '../../i18n';
import { hashPassword, verifyPassword } from './_authz_shared';

const MIN_PASSWORD_LENGTH = 6;

/**
 * Validate ChangePassword envelope strings and return trimmed values.
 */
export function normalizeChangePasswordInput(req: {
  CurrentPassword?: unknown;
  NewPassword?: unknown;
}): { currentPassword: string; newPassword: string } {
  const currentPassword = typeof req.CurrentPassword === 'string' ? req.CurrentPassword : '';
  const newPassword = typeof req.NewPassword === 'string' ? req.NewPassword : '';
  if (!currentPassword) {
    throw newAuthError({
      code: AuthErrCode.VALIDATION_FAILED,
      message: _t('Current password is required', { scope: 'service/models/user' }),
    }).withGrpcCode(GrpcCode.InvalidArgument);
  }
  if (!newPassword) {
    throw newAuthError({
      code: AuthErrCode.VALIDATION_FAILED,
      message: _t('New password is required', { scope: 'service/models/user' }),
    }).withGrpcCode(GrpcCode.InvalidArgument);
  }
  // Client-hashed payloads are long; only enforce length on non-$CH$ plaintext.
  if (!newPassword.startsWith('$CH$') && newPassword.length < MIN_PASSWORD_LENGTH) {
    throw newAuthError({
      code: AuthErrCode.VALIDATION_FAILED,
      message: _t('Password must be at least 6 characters', { scope: 'service/models/user' }),
    }).withGrpcCode(GrpcCode.InvalidArgument);
  }
  if (currentPassword === newPassword) {
    throw newAuthError({
      code: AuthErrCode.VALIDATION_FAILED,
      message: _t('New password must differ from the current password', { scope: 'service/models/user' }),
    }).withGrpcCode(GrpcCode.InvalidArgument);
  }
  return { currentPassword, newPassword };
}

/**
 * Verify the current password and build the next PasswordHash.
 */
export function buildChangedPasswordHash(opts: {
  currentPassword: string;
  newPassword: string;
  passwordHash: string;
}): string {
  if (!verifyPassword(opts.currentPassword, opts.passwordHash)) {
    throw newAuthError({
      code: AuthErrCode.INVALID_PASSWORD,
      message: _t('Current password is incorrect', { scope: 'service/models/user' }),
    }).withGrpcCode(GrpcCode.Unauthenticated);
  }
  return hashPassword(opts.newPassword);
}
