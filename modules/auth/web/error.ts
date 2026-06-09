// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ErrorFactory } from '../../core/web/error';

export { ChoysumError, GrpcCode, errorAs, isErrorOf, createDomainErrorHandlers, ErrorFactory, generateErrorId, validateErrorCode } from '../../core/web/error';
export type { ErrorCodePattern, ErrorOptions } from '../../core/web/error';

/**
 * Auth module error domain.
 */
export const AUTH_DOMAIN = 'auth' as const;

/**
 * Auth module error codes.
 */
export namespace AuthErrCode {
  export const UNKNOWN = 'UNKNOWN';

  // JWT token errors.
  export const TOKEN_INVALID = 'TOKEN_INVALID';
  export const TOKEN_PARSING_FAILED = 'TOKEN_PARSING_FAILED';
  export const IDENTITY_EXTRACTION_FAILED = 'IDENTITY_EXTRACTION_FAILED';

  // Action failures.
  export const LOGOUT_FAILED = 'LOGOUT_FAILED';
  export const REFRESH_FAILED = 'REFRESH_FAILED';
  export const USER_LOADING_FAILED = 'USER_LOADING_FAILED';
  export const INITIALIZATION_FAILED = 'INITIALIZATION_FAILED';
  export const REGISTRATION_FAILED = 'REGISTRATION_FAILED';
  export const INVALID_CREDENTIALS = 'INVALID_CREDENTIALS';

  // Client-side errors.
  export const PASSWORD_HASHING_FAILED = 'PASSWORD_HASHING_FAILED';
  export const CSRF_TOKEN_FAILED = 'CSRF_TOKEN_FAILED';
  export const DEVICE_INFO_FAILED = 'DEVICE_INFO_FAILED';
}

/**
 * Auth error code union.
 */
export type AuthErrCodeType = (typeof AuthErrCode)[keyof typeof AuthErrCode];

/**
 * Auth-scoped error helpers.
 */
const { newError, wrapError, isError } = ErrorFactory.createDomainHandlers<AuthErrCodeType>(AUTH_DOMAIN);

/**
 * Export auth-scoped error helpers.
 */
export { newError as newAuthError, wrapError as wrapAuthError, isError as isAuthError };
