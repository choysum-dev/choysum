// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

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

export { toUIErrorState } from './ui';
export type { UIErrorState } from './ui';
export { errorMessageKey } from './message';
export { errorAction } from './action';
export type { ErrorAction } from './action';
