// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { normalizeLooseOptionalText, asRecord } from '@/core/service/utils/normalization';
import { createTranslate } from '@/core/service/i18n';
import { GrpcCode } from '../error';
import { DocumentErrCode, throwDocumentError } from '../error';
import type { PrincipalContext } from '../contracts';

const { _t } = createTranslate('document');

/**
 * Require a non-empty trimmed string value; throws INVALID_ARGUMENT otherwise.
 */
export function requireText(value: unknown, fieldName: string): string {
  const text = normalizeLooseOptionalText(value);
  if (!text) {
    throw throwDocumentError(
      DocumentErrCode.INVALID_ARGUMENT,
      _t('%s is required', { scope: 'service/models/_document_bridge' }, fieldName),
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
  const userId = normalizeLooseOptionalText(rawUserId);
  if (!userId) {
    throw throwDocumentError(
      DocumentErrCode.UNAUTHENTICATED,
      _t('Authentication is required', { scope: 'service/models/_document_bridge' }),
      GrpcCode.Unauthenticated
    );
  }
  return userId;
}

/**
 * Resolve and require the current company identity; throws PERMISSION_DENIED otherwise.
 */
export function requireCompanyId(rawCompanyId: unknown, stage: string): string {
  const companyId = normalizeLooseOptionalText(rawCompanyId);
  if (!companyId) {
    throw throwDocumentError(
      DocumentErrCode.PERMISSION_DENIED,
      _t('activeCompanyId is required for document operations', { scope: 'service/models/_document_bridge' }),
      GrpcCode.PermissionDenied,
      { stage }
    );
  }
  return companyId;
}

/**
 * Build a deduplicated company-id list, always including the active company when present.
 * Finite numeric ids are coerced to strings (same as {@link normalizeLooseOptionalText}).
 */
export function normalizeCompanyIdList(value: unknown, activeCompanyId: string): string[] {
  const out: string[] = [];
  if (Array.isArray(value)) {
    for (const item of value) {
      const text = normalizeLooseOptionalText(item);
      if (text) out.push(text);
    }
  }
  const normalizedActive = normalizeLooseOptionalText(activeCompanyId);
  if (normalizedActive && !out.includes(normalizedActive)) {
    out.unshift(normalizedActive);
  }
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
      throw throwDocumentError(
        DocumentErrCode.INVALID_ARGUMENT,
        _t('principal.enabledCompanyIds must be an array', { scope: 'service/models/_document_bridge' }),
        GrpcCode.InvalidArgument,
        { field: 'principal.enabledCompanyIds' }
      );
    }
    enabledCompanyIds = Array.from(
      new Set(
        rawEnabledCompanyIds
          .map(item => normalizeLooseOptionalText(item))
          .filter((item): item is string => Boolean(item))
      )
    );
  }

  return {
    userId: requireText(principal?.userId, 'principal.userId'),
    activeCompanyId: requireText(principal?.activeCompanyId, 'principal.activeCompanyId'),
    enabledCompanyIds,
  };
}

/**
 * Build PrincipalContext from the trusted request/session runtime axes.
 */
export function principalFromRuntime(
  runtime: { userId?: unknown; companyId?: unknown; companyIds?: unknown },
  stage: string
): PrincipalContext {
  const userId = requireUserId(runtime.userId);
  const activeCompanyId = requireCompanyId(runtime.companyId, stage);
  return {
    userId,
    activeCompanyId,
    enabledCompanyIds: normalizeCompanyIdList(runtime.companyIds, activeCompanyId),
  };
}

/**
 * Fail closed when a stale client still supplies wire `principal`.
 */
export function rejectLegacyPrincipalField(req: unknown, stage: string): void {
  if (req != null && typeof req === 'object' && Object.prototype.hasOwnProperty.call(req, 'principal')) {
    throwDocumentError(
      DocumentErrCode.INVALID_ARGUMENT,
      _t('principal is derived from the session', { scope: 'service/models/_document_bridge' }),
      GrpcCode.InvalidArgument,
      { stage, reason: 'legacy_principal_supplied' }
    );
  }
}
