// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { normalizeOptionalString } from '@/core/service/utils/normalization';
import { getBackendEnvPositiveInt } from '@/core/service/runtime/env/backend_env';
import { GrpcCode } from '../error';
import { newDocumentError, DocumentErrCode } from '../error';

const DEFAULT_GC_BATCH_SIZE = 200;

/**
 * Require a non-empty trimmed string value; throws INVALID_ARGUMENT otherwise.
 */
export function requireText(value: unknown, fieldName: string): string {
  const text = normalizeOptionalString(value);
  if (!text) {
    throw newDocumentError({
      code: DocumentErrCode.INVALID_ARGUMENT,
      message: `${fieldName} is required`,
    })
      .withGrpcCode(GrpcCode.InvalidArgument)
      .withMetadata({ field: fieldName });
  }
  return text;
}

/**
 * Resolve and require the current user identity; throws UNAUTHENTICATED otherwise.
 */
export function requireUserId(rawUserId: unknown): string {
  const userId = normalizeOptionalString(rawUserId);
  if (!userId) {
    throw newDocumentError({
      code: DocumentErrCode.UNAUTHENTICATED,
      message: 'Authentication is required',
    }).withGrpcCode(GrpcCode.Unauthenticated);
  }
  return userId;
}

/**
 * Resolve and require the current company identity; throws PERMISSION_DENIED otherwise.
 */
export function requireCompanyId(rawCompanyId: unknown, stage: string): string {
  const companyId = normalizeOptionalString(rawCompanyId);
  if (!companyId) {
    throw newDocumentError({
      code: DocumentErrCode.PERMISSION_DENIED,
      message: 'activeCompanyId is required for document operations',
    })
      .withGrpcCode(GrpcCode.PermissionDenied)
      .withMetadata({ stage });
  }
  return companyId;
}

/**
 * Resolve the GC batch size from backend environment or fall back to 200.
 */
export function resolveGcBatchSize(): number {
  return getBackendEnvPositiveInt(['CHOYSUM_DOCUMENT_GC_BATCH_SIZE'], DEFAULT_GC_BATCH_SIZE);
}
