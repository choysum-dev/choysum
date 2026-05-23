// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ErrorFactory } from '../../core/service/error';

export {
  ChoysumError,
  GrpcCode,
  errorAs,
  isErrorOf,
  createDomainErrorHandlers,
  ErrorFactory,
  generateErrorId,
  validateErrorCode,
} from '../../core/service/error';
export type { ErrorCodePattern, ErrorOptions } from '../../core/service/error';

/**
 * Auth module error domain.
 */
export const AUTH_DOMAIN = 'auth' as const;

/**
 * Auth module error codes.
 */
export namespace AuthErrCode {
  export const UNKNOWN = 'UNKNOWN';

  // User errors.
  export const USERNAME_TAKEN = 'USERNAME_TAKEN';
  export const EMAIL_TAKEN = 'EMAIL_TAKEN';
  export const USER_NOT_FOUND = 'USER_NOT_FOUND';
  export const USER_CREATION_FAILED = 'USER_CREATION_FAILED';
  export const INVALID_PASSWORD = 'INVALID_PASSWORD';
  export const ACCOUNT_DISABLED = 'ACCOUNT_DISABLED';

  // Token errors.
  export const TOKEN_CREATION_FAILED = 'TOKEN_CREATION_FAILED';
  export const TOKEN_VALIDATION_FAILED = 'TOKEN_VALIDATION_FAILED';
  export const TOKEN_REFRESH_FAILED = 'TOKEN_REFRESH_FAILED';
  export const TOKEN_REVOCATION_FAILED = 'TOKEN_REVOCATION_FAILED';

  // Session errors.
  export const SESSION_INVALID = 'SESSION_INVALID';
  export const SESSION_REVOCATION_FAILED = 'SESSION_REVOCATION_FAILED';

  // Role errors.
  export const ROLE_NOT_FOUND = 'ROLE_NOT_FOUND';
  export const ROLE_CREATION_FAILED = 'ROLE_CREATION_FAILED';
  export const ROLE_ASSIGNMENT_FAILED = 'ROLE_ASSIGNMENT_FAILED';

  // Permission errors.
  export const PERMISSION_NOT_FOUND = 'PERMISSION_NOT_FOUND';
  export const PERMISSION_CREATION_FAILED = 'PERMISSION_CREATION_FAILED';

  // Service errors.
  export const AUTH_SERVICE_DISABLED = 'AUTH_SERVICE_DISABLED';
  export const VALIDATION_FAILED = 'VALIDATION_FAILED';
}

/**
 * Auth error code union.
 */
export type AuthErrCodeType = (typeof AuthErrCode)[keyof typeof AuthErrCode];

/**
 * Auth-scoped error helpers.
 */
const { newError, wrapError, isError } = ErrorFactory.createDomainHandlers<AuthErrCodeType>(AUTH_DOMAIN);

// Export auth-scoped helpers.
export { newError as newAuthError, wrapError as wrapAuthError, isError as isAuthError };
