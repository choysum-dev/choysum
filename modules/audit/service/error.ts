// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ErrorFactory } from '@/core/service/error';

export { ChoysumError, GrpcCode, errorAs, isErrorOf, createDomainErrorHandlers, ErrorFactory } from '@/core/service/error';
export type { ErrorCodePattern, ErrorOptions } from '@/core/service/error';

/**
 * Stable domain name used by audit service errors.
 */
export const AUDIT_DOMAIN = 'audit' as const;

/**
 * Audit-domain error codes.
 */
export namespace AuditErrCode {
  export const INVALID_ARGUMENT = 'INVALID_ARGUMENT';
  export const APPEND_ONLY = 'APPEND_ONLY';
  export const INVALID_KIND = 'INVALID_KIND';
}

/**
 * Union of audit-domain error code values.
 */
export type AuditErrCodeType = (typeof AuditErrCode)[keyof typeof AuditErrCode];

const { newError, wrapError, isError } = ErrorFactory.createDomainHandlers<AuditErrCodeType>(AUDIT_DOMAIN);

/**
 * Domain-scoped audit error helpers.
 */
export { newError as newAuditError, wrapError as wrapAuditError, isError as isAuditError };
