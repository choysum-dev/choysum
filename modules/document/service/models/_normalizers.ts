// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { normalizeOptionalString } from '@/core/service/utils/normalization';
import { GrpcCode } from '../error';
import { DocumentErrCode, throwDocumentError } from '../error';

/**
 * Require a non-empty trimmed string value; throws INVALID_ARGUMENT otherwise.
 */
export function requireText(value: unknown, fieldName: string): string {
  const text = normalizeOptionalString(value);
  if (!text) {
    throw throwDocumentError(DocumentErrCode.INVALID_ARGUMENT, `${fieldName} is required`, GrpcCode.InvalidArgument, { field: fieldName });
  }
  return text;
}

/**
 * Resolve and require the current user identity; throws UNAUTHENTICATED otherwise.
 */
export function requireUserId(rawUserId: unknown): string {
  const userId = normalizeOptionalString(rawUserId);
  if (!userId) {
    throw throwDocumentError(DocumentErrCode.UNAUTHENTICATED, 'Authentication is required', GrpcCode.Unauthenticated);
  }
  return userId;
}

/**
 * Resolve and require the current company identity; throws PERMISSION_DENIED otherwise.
 */
export function requireCompanyId(rawCompanyId: unknown, stage: string): string {
  const companyId = normalizeOptionalString(rawCompanyId);
  if (!companyId) {
    throw throwDocumentError(DocumentErrCode.PERMISSION_DENIED, 'activeCompanyId is required for document operations', GrpcCode.PermissionDenied, { stage });
  }
  return companyId;
}

/**
 * Generic single-row loader: Search with limit=1, throw NOT_FOUND if absent.
 *
 * Replaces the repeated `Search + check rows[0] + throwDocumentError(NOT_FOUND)`
 * pattern found in mustLoad* helpers across binding, object, and stored_content.
 */
export async function mustLoadOne<T>(
  searchFn: (condition: unknown, options?: unknown) => Promise<T[]>,
  condition: unknown,
  notFoundMessage: string,
  metadata?: Record<string, unknown>
): Promise<T> {
  const rows = await searchFn(condition, { limit: 1 });
  const record = rows[0] as T | undefined;
  if (!record) {
    throw throwDocumentError(DocumentErrCode.NOT_FOUND, notFoundMessage, GrpcCode.NotFound, metadata ?? {});
  }
  return record;
}
