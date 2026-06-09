// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Code, ConnectError } from '@connectrpc/connect';
import { ErrorInfoSchema, ChoysumError } from '../../error';

export function normalizeTransportError(error: unknown): unknown {
  if (!(error instanceof ConnectError)) {
    return error;
  }

  const errorInfos = error.findDetails(ErrorInfoSchema);
  if (errorInfos && errorInfos.length > 0) {
    return ChoysumError.fromErrorInfo(errorInfos[0]);
  }

  const grpcCode = error.code as Code;
  const choysumError = new ChoysumError({
    domain: 'api',
    code: 'UNKNOWN',
    message: error.message || 'Unknown API error',
  })
    .withGrpcCode(grpcCode)
    .withCause(error);

  if (error.metadata) {
    const metadataObj: Record<string, string> = {};
    error.metadata.forEach((value, key) => {
      metadataObj[key] = value;
    });
    choysumError.withMetadata(metadataObj);
  }

  return choysumError;
}
