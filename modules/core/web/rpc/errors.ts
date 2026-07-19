// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Code, ConnectError } from '@connectrpc/connect';
import { ErrorInfoSchema, ChoysumError } from '../../error';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('core', { scope: 'web/rpc/errors' });

export function normalizeTransportError(error: unknown): unknown {
  if (!(error instanceof ConnectError)) {
    return error;
  }

  const errorInfos =
    typeof error === 'object' &&
    error !== null &&
    'findDetails' in error &&
    typeof (error as { findDetails?: unknown }).findDetails === 'function'
      ? (error as { findDetails: (schema: unknown) => unknown[] }).findDetails(ErrorInfoSchema as any)
      : undefined;
  if (errorInfos && errorInfos.length > 0) {
    return ChoysumError.fromErrorInfo(errorInfos[0] as any);
  }

  const grpcCode = error.code as Code;
  const choysumError = new ChoysumError({
    domain: 'api',
    code: 'UNKNOWN',
    message: error.message || _t('Unknown API error'),
  })
    .withGrpcCode(grpcCode)
    .withCause(error instanceof Error ? error : new Error(String(error)));

  if (error.metadata) {
    const metadataObj: Record<string, string> = {};
    error.metadata.forEach((value, key) => {
      metadataObj[key] = value;
    });
    choysumError.withMetadata(metadataObj);
  }

  return choysumError;
}
