// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ErrorFactory as CoreErrorFactory } from '../../error';

export {
  ChoysumError,
  GrpcCode,
  errorAs,
  isErrorOf,
  createDomainErrorHandlers,
  ErrorFactory,
  generateErrorId,
  validateErrorCode,
  ErrorInfoSchema,
} from '../../error';
export type { ErrorCodePattern, ErrorOptions, ErrorInfo } from '../../error';

/**
 * Database error domain.
 */
export const DOMAIN = 'core.db' as const;

/**
 * Database error codes.
 */
export namespace DbErrCode {
  export const UNKNOWN = 'UNKNOWN';
  export const TRANSACTION_NOT_STARTED = 'TRANSACTION_NOT_STARTED';
  export const TRANSACTION_BEGIN_FAILED = 'TRANSACTION_BEGIN_FAILED';
  export const TRANSACTION_COMMIT_FAILED = 'TRANSACTION_COMMIT_FAILED';
  export const TRANSACTION_ROLLBACK_FAILED = 'TRANSACTION_ROLLBACK_FAILED';
  export const TRANSACTION_ID_FAILED = 'TRANSACTION_ID_FAILED';
  export const EXECUTION_FAILED = 'EXECUTION_FAILED';
  export const QUERY_FAILED = 'QUERY_FAILED';
  export const STREAMING_NOT_SUPPORTED = 'STREAMING_NOT_SUPPORTED';
  export const SAVEPOINT_CREATE_FAILED = 'SAVEPOINT_CREATE_FAILED';
  export const SAVEPOINT_ROLLBACK_FAILED = 'SAVEPOINT_ROLLBACK_FAILED';
  export const SAVEPOINT_RELEASE_FAILED = 'SAVEPOINT_RELEASE_FAILED';
  export const SAVEPOINT_OPERATION_FAILED = 'SAVEPOINT_OPERATION_FAILED';
}

/**
 * Database error code type.
 */
export type DbErrCodeType = (typeof DbErrCode)[keyof typeof DbErrCode];

/**
 * Database error helpers.
 */
const { newError, wrapError, isError } = CoreErrorFactory.createDomainHandlers<DbErrCodeType>(DOMAIN);

// Export database-specific helper aliases.
export { newError as newDbError, wrapError as wrapDbError, isError as isDbError };
