// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ErrorFactory } from '@/core/service/error';

export { ChoysumError, GrpcCode, errorAs, isErrorOf, createDomainErrorHandlers, ErrorFactory } from '@/core/service/error';
export type { ErrorCodePattern, ErrorOptions } from '@/core/service/error';

/**
 * Stable domain name used by message service errors.
 */
export const MESSAGE_DOMAIN = 'message' as const;

/**
 * Message-domain error codes.
 */
export namespace MessageErrCode {
  export const INVALID_ARGUMENT = 'INVALID_ARGUMENT';
  export const INVALID_TYPE = 'INVALID_TYPE';
  export const ATTACHMENT_BIND_FAILED = 'ATTACHMENT_BIND_FAILED';
  export const PERMISSION_DENIED = 'PERMISSION_DENIED';
}

/**
 * Union of message-domain error code values.
 */
export type MessageErrCodeType = (typeof MessageErrCode)[keyof typeof MessageErrCode];

const { newError, wrapError, isError } = ErrorFactory.createDomainHandlers<MessageErrCodeType>(MESSAGE_DOMAIN);

/**
 * Domain-scoped message error helpers.
 */
export { newError as newMessageError, wrapError as wrapMessageError, isError as isMessageError };
