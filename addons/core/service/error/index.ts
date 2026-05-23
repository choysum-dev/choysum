// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Service error facade for addons/core.
 * Re-exports the shared error primitives together with runtime, context, and gRPC helpers
 * so callers can import service-layer error handling from a single entry point.
 */
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

export { ensureServiceError } from './runtime';
export { withServiceErrorContext } from './context';
export type { ServiceErrorContext } from './context';
export { serviceErrorCode } from './grpc';
