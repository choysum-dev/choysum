// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ErrorFactory } from '@/core/service/error';

export { ChoysumError, GrpcCode, errorAs, isErrorOf, createDomainErrorHandlers, ErrorFactory, generateErrorId, validateErrorCode } from '@/core/service/error';
export type { ErrorCodePattern, ErrorOptions } from '@/core/service/error';

/**
 * Stable domain name used by document service errors.
 */
export const DOCUMENT_DOMAIN = 'document' as const;

/**
 * Document-domain error codes surfaced by service and model entry points.
 */
export namespace DocumentErrCode {
  export const UNKNOWN = 'UNKNOWN';
  export const INTERNAL = 'INTERNAL';
  export const INVALID_ARGUMENT = 'INVALID_ARGUMENT';
  export const UNAUTHENTICATED = 'UNAUTHENTICATED';
  export const PERMISSION_DENIED = 'PERMISSION_DENIED';
  export const NOT_FOUND = 'NOT_FOUND';
  export const FAILED_PRECONDITION = 'FAILED_PRECONDITION';
  export const IDEMPOTENCY_KEY_REUSED = 'IDEMPOTENCY_KEY_REUSED';
  export const UPLOAD_SESSION_NOT_FOUND = 'UPLOAD_SESSION_NOT_FOUND';
  export const UPLOAD_SESSION_EXPIRED = 'UPLOAD_SESSION_EXPIRED';
  export const UPLOAD_SESSION_FINALIZED = 'UPLOAD_SESSION_FINALIZED';
  export const INVALID_UPLOAD_BODY = 'INVALID_UPLOAD_BODY';
  export const UPLOAD_TOO_LARGE = 'UPLOAD_TOO_LARGE';
  export const MIME_TYPE_NOT_ALLOWED = 'MIME_TYPE_NOT_ALLOWED';
  export const CHECKSUM_MISMATCH = 'CHECKSUM_MISMATCH';
  export const BINDING_NOT_FOUND = 'BINDING_NOT_FOUND';
  export const ATTACHMENT_NOT_FOUND = 'ATTACHMENT_NOT_FOUND';
  export const SKELETON_NOT_IMPLEMENTED = 'SKELETON_NOT_IMPLEMENTED';
}

/**
 * Union of document-domain error code values.
 */
export type DocumentErrCodeType = (typeof DocumentErrCode)[keyof typeof DocumentErrCode];

/**
 * Default HTTP status mapping for document-domain errors.
 */
export const DOCUMENT_ERROR_HTTP_STATUS: Readonly<Record<DocumentErrCodeType, number>> = {
  [DocumentErrCode.UNKNOWN]: 500,
  [DocumentErrCode.INTERNAL]: 500,
  [DocumentErrCode.INVALID_ARGUMENT]: 400,
  [DocumentErrCode.UNAUTHENTICATED]: 401,
  [DocumentErrCode.PERMISSION_DENIED]: 403,
  [DocumentErrCode.NOT_FOUND]: 404,
  [DocumentErrCode.FAILED_PRECONDITION]: 412,
  [DocumentErrCode.IDEMPOTENCY_KEY_REUSED]: 409,
  [DocumentErrCode.UPLOAD_SESSION_NOT_FOUND]: 404,
  [DocumentErrCode.UPLOAD_SESSION_EXPIRED]: 410,
  [DocumentErrCode.UPLOAD_SESSION_FINALIZED]: 409,
  [DocumentErrCode.INVALID_UPLOAD_BODY]: 400,
  [DocumentErrCode.UPLOAD_TOO_LARGE]: 413,
  [DocumentErrCode.MIME_TYPE_NOT_ALLOWED]: 415,
  [DocumentErrCode.CHECKSUM_MISMATCH]: 422,
  [DocumentErrCode.BINDING_NOT_FOUND]: 404,
  [DocumentErrCode.ATTACHMENT_NOT_FOUND]: 404,
  [DocumentErrCode.SKELETON_NOT_IMPLEMENTED]: 501,
};

/**
 * Resolves the default HTTP status for a document-domain error code.
 */
export function documentErrorHttpStatus(code: string | undefined | null): number {
  const normalized = String(code ?? '').trim() as DocumentErrCodeType;
  return DOCUMENT_ERROR_HTTP_STATUS[normalized] ?? 500;
}

const { newError, wrapError, isError } = ErrorFactory.createDomainHandlers<DocumentErrCodeType>(DOCUMENT_DOMAIN);

/**
 * Domain-scoped document error helpers.
 */
export { newError as newDocumentError, wrapError as wrapDocumentError, isError as isDocumentError };

/**
 * Construct and throw a document-domain error with optional gRPC code and metadata.
 *
 * This is a convenience wrapper around the `newDocumentError(...).withGrpcCode(...).withMetadata(...)` chain.
 * Returns `never` — the function always throws.
 */
export function throwDocumentError(code: DocumentErrCodeType, message: string, grpcCode?: number, metadata?: Record<string, unknown>): never {
  const err = newError({ code, message }).withGrpcCode(grpcCode ?? 2 /* UNKNOWN */);
  if (!metadata) throw err;
  const stringified: Record<string, string> = {};
  for (const [k, v] of Object.entries(metadata)) {
    stringified[k] = String(v ?? '');
  }
  throw err.withMetadata(stringified);
}
