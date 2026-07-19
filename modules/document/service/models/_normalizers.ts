// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { normalizeOptionalString, asRecord } from '@/core/service/utils/normalization';
import { createTranslate } from '@/core/service/i18n';
import { GrpcCode } from '../error';
import { DocumentErrCode, throwDocumentError } from '../error';
import type { PrincipalContext } from '../contracts';

const { _t } = createTranslate('document');

/**
 * Require a non-empty trimmed string value; throws INVALID_ARGUMENT otherwise.
 */
export function requireText(value: unknown, fieldName: string): string {
  const text = normalizeOptionalString(value);
  if (!text) {
    throw throwDocumentError(
      DocumentErrCode.INVALID_ARGUMENT,
      _t('%s is required', { scope: 'service/models/_normalizers' }, fieldName),
      GrpcCode.InvalidArgument,
      { field: fieldName }
    );
  }
  return text;
}

/**
 * Resolve and require the current user identity; throws UNAUTHENTICATED otherwise.
 */
export function requireUserId(rawUserId: unknown): string {
  const userId = normalizeOptionalString(rawUserId);
  if (!userId) {
    throw throwDocumentError(
      DocumentErrCode.UNAUTHENTICATED,
      _t('Authentication is required', { scope: 'service/models/_normalizers' }),
      GrpcCode.Unauthenticated
    );
  }
  return userId;
}

/**
 * Resolve and require the current company identity; throws PERMISSION_DENIED otherwise.
 */
export function requireCompanyId(rawCompanyId: unknown, stage: string): string {
  const companyId = normalizeOptionalString(rawCompanyId);
  if (!companyId) {
    throw throwDocumentError(
      DocumentErrCode.PERMISSION_DENIED,
      _t('activeCompanyId is required for document operations', { scope: 'service/models/_normalizers' }),
      GrpcCode.PermissionDenied,
      { stage }
    );
  }
  return companyId;
}

/**
 * Normalize a loose principal input into a typed PrincipalContext.
 */
export function normalizePrincipal(raw: unknown): PrincipalContext {
  const principal = asRecord(raw);
  return {
    userId: requireText(principal?.userId, 'principal.userId'),
    activeCompanyId: requireText(principal?.activeCompanyId, 'principal.activeCompanyId'),
    enabledCompanyIds: Array.isArray(principal?.enabledCompanyIds)
      ? (principal?.enabledCompanyIds as unknown[]).map(item => normalizeOptionalString(item)).filter((item): item is string => Boolean(item))
      : undefined,
  };
}
