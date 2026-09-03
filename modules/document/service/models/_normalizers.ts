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
 * Trim optional text, coercing finite numbers to strings.
 */
export function normalizeLooseOptionalText(value: unknown): string | undefined {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return String(value);
  }
  return normalizeOptionalString(value);
}

/**
 * Build a deduplicated company-id list, always including the active company when present.
 */
export function normalizeCompanyIdList(value: unknown, activeCompanyId: string): string[] {
  const out: string[] = [];
  if (Array.isArray(value)) {
    for (const item of value) {
      const text = normalizeOptionalString(item);
      if (text) out.push(text);
    }
  }
  const normalizedActive = normalizeOptionalString(activeCompanyId);
  if (out.length === 0 && normalizedActive) out.push(normalizedActive);
  if (normalizedActive && !out.includes(normalizedActive)) out.unshift(normalizedActive);
  return Array.from(new Set(out));
}

/**
 * Validate and normalize a loose principal input into a typed PrincipalContext.
 */
export function assertPrincipal(raw: unknown): PrincipalContext {
  const principal = asRecord(raw);
  const rawEnabledCompanyIds = principal?.enabledCompanyIds;
  let enabledCompanyIds: string[] | undefined;
  if (rawEnabledCompanyIds !== undefined && rawEnabledCompanyIds !== null) {
    if (!Array.isArray(rawEnabledCompanyIds)) {
      throwDocumentError(
        DocumentErrCode.INVALID_ARGUMENT,
        _t('principal.enabledCompanyIds must be an array', { scope: 'service/models/_normalizers' }),
        GrpcCode.InvalidArgument,
        { field: 'principal.enabledCompanyIds' }
      );
    }
    enabledCompanyIds = rawEnabledCompanyIds
      .map(item => normalizeOptionalString(item))
      .filter((item): item is string => Boolean(item));
  }

  return {
    userId: requireText(principal?.userId, 'principal.userId'),
    activeCompanyId: requireText(principal?.activeCompanyId, 'principal.activeCompanyId'),
    enabledCompanyIds,
  };
}
