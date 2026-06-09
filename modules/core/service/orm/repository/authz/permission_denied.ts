// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { GrpcCode, ChoysumError } from '@/core/service/error';
import type { RepositoryAuthzContextDepsParams } from './deps';

export function resolveRepositoryPermissionDeniedLayer(code: string): string {
  if (String(code).startsWith('company_scope_')) return 'company_filter';
  if (String(code).startsWith('record_rule_')) return 'record_rule';
  if (String(code).startsWith('field_rule_')) return 'field_rule';
  return 'unknown';
}

export function createRepositoryPermissionDeniedError(
  deps: RepositoryAuthzContextDepsParams,
  code: string,
  message: string,
  metadata?: Record<string, string>
): ChoysumError {
  const req = deps.getReqMethodMeta();
  const scope = deps.getCompanyScopeFacts();
  const model = deps.meta.fullModelName || deps.meta.modelName || deps.meta.name;
  deps.emitAuthzDecisionSummary({
    layer: resolveRepositoryPermissionDeniedLayer(code),
    decision: 'deny',
    basis: code,
    fullMethod: req.fullMethod,
    method: req.method,
    model,
    userId: String(deps.userId || '').trim(),
    activeCompanyId: scope.activeCompanyId,
    enabledCompanyIds: scope.enabledCompanyIds,
    companyMode: req.companyMode,
    recordRuleMode: req.recordRuleMode,
    fieldRuleMode: req.fieldRuleMode,
    message,
    metadata: metadata || {},
  });

  const error = new ChoysumError({ domain: 'core.repository', code, message }).withGrpcCode(GrpcCode.PermissionDenied);
  if (metadata && Object.keys(metadata).length) error.withMetadata(metadata);
  return error;
}
