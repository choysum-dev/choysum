// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { GrpcCode, ChoysumError } from '../../error';

export function ensureServiceError(error: unknown, fallbackCode = GrpcCode.Unknown, fallbackMessage = 'service error'): ChoysumError {
  if (error instanceof ChoysumError) {
    return error;
  }
  if (error instanceof Error) {
    return ChoysumError.wrap(error, {
      domain: 'core.service',
      code: 'UNKNOWN',
      message: error.message,
    }).withGrpcCode(fallbackCode);
  }
  return new ChoysumError({
    domain: 'core.service',
    code: 'UNKNOWN',
    message: fallbackMessage,
  }).withGrpcCode(fallbackCode);
}
