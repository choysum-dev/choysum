// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata } from '../../metadata';
import type {
  BaseQueryCondition,
  Entity,
  RepositoryExecute,
  RepositorySelectColumnsCapableLike,
  RepositorySelectFromDbLike,
  RepositoryTableSoftConditionPipelineDepsLike,
  RepositoryWhereCapableLike,
} from '../types';
import type { RepositoryCompanyScopeFacts, RepositoryReqMethodMeta } from './authz_runtime';
import type { RepositoryEmitAuthzDecisionSummary, RepositoryPermissionDeniedFn } from './types';
import { getRuntimeEnvFlag } from '@/core/utils/env';
import { asObjectRecord } from '../../../../utils/object';

type RepositoryCompanyScopeDeps = {
  meta: ModelMetadata;
  ctx: unknown;
  userId?: string;
  companyLayerSkipped: () => boolean;
  getReqMethodMeta: () => RepositoryReqMethodMeta;
  getCompanyScopeFacts: () => RepositoryCompanyScopeFacts;
  emitAuthzDecisionSummary: RepositoryEmitAuthzDecisionSummary;
  permissionDenied: RepositoryPermissionDeniedFn;
};

type RepositoryCompanyScopeQueryDeps = RepositoryCompanyScopeDeps &
  RepositoryTableSoftConditionPipelineDepsLike<BaseQueryCondition> & {
    db: unknown;
    execute: RepositoryExecute;
  };

interface RepositoryCompanyScopeSelectQueryLike extends RepositoryWhereCapableLike<RepositoryCompanyScopeSelectQueryLike> {}

type RepositoryCompanyScopeSelectFromBuilderLike = RepositorySelectColumnsCapableLike<RepositoryCompanyScopeSelectQueryLike, string[]>;

type RepositoryCompanyScopeDbLike = RepositorySelectFromDbLike<RepositoryCompanyScopeSelectFromBuilderLike, string>;

export function normalizeRepositoryCompanyIds(ctx: unknown): string[] {
  const requestContext = asObjectRecord(ctx);
  const raw = requestContext?.enabledCompanyIds ?? requestContext?.EnabledCompanyIds ?? requestContext?.activeCompanyId ?? requestContext?.ActiveCompanyId;

  const ids: string[] = [];
  if (Array.isArray(raw)) {
    for (const value of raw) {
      const normalized = String(value ?? '').trim();
      if (normalized) ids.push(normalized);
    }
  } else if (raw != null) {
    const normalized = String(raw).trim();
    if (normalized) ids.push(normalized);
  }

  return [...new Set(ids)];
}

export function normalizeRepositoryCompanyIdForWrite(ctx: unknown): string | undefined {
  const requestContext = asObjectRecord(ctx);
  const raw = requestContext?.activeCompanyId ?? requestContext?.ActiveCompanyId;
  if (raw != null) {
    const value = String(raw).trim();
    if (value) return value;
  }

  const rawIds = requestContext?.enabledCompanyIds;
  if (Array.isArray(rawIds) && rawIds.length === 1) {
    const value = String(rawIds[0] ?? '').trim();
    if (value) return value;
  }

  return undefined;
}

export function validateRepositoryCompanyIdInScope(params: RepositoryCompanyScopeDeps, companyId: unknown, companyIds: string[]): void {
  if (companyId === null || companyId === undefined) return;
  const normalized = String(companyId).trim();
  if (!normalized) return;
  if (!companyIds.includes(normalized)) {
    throw params.permissionDenied('company_scope_violation', 'company is not in ctx.enabledCompanyIds', {
      model: params.meta.fullModelName || params.meta.modelName || params.meta.name,
      companyId: normalized,
    });
  }
}

export function repositoryCompanyScopedEnabled(params: RepositoryCompanyScopeDeps): boolean {
  const grpcEnabled = getRuntimeEnvFlag('CHOYSUM_GRPC_COMPANY_FILTER_ENABLED', true);
  if (!grpcEnabled) return false;
  if (params.companyLayerSkipped()) return false;
  if (!params.meta.companyScoped) return false;

  const hasCompanyIdField = params.meta.fields instanceof Map && params.meta.fields.has('CompanyId');
  if (!hasCompanyIdField) {
    throw params.permissionDenied('company_scope_missing_company_id_field', 'companyScoped model is missing CompanyId field', {
      model: params.meta.fullModelName || params.meta.modelName || params.meta.name,
    });
  }

  const ids = normalizeRepositoryCompanyIds(params.ctx);
  if (!ids.length) {
    throw params.permissionDenied('company_scope_missing_ctx_company', 'missing ctx.enabledCompanyIds/activeCompanyId for company scoped operation', {
      model: params.meta.fullModelName || params.meta.modelName || params.meta.name,
    });
  }

  return true;
}

export function applyRepositoryCompanyLayer(params: RepositoryCompanyScopeDeps, condition: BaseQueryCondition): BaseQueryCondition {
  if (params.companyLayerSkipped()) return condition;
  if (!repositoryCompanyScopedEnabled(params)) return condition;

  const companyIds = normalizeRepositoryCompanyIds(params.ctx);
  const companyCondition: BaseQueryCondition = {
    Or: [
      ['CompanyId', 'in', companyIds],
      ['CompanyId', 'is', null],
    ],
  };

  const req = params.getReqMethodMeta();
  const scope = params.getCompanyScopeFacts();
  const model = params.meta.fullModelName || params.meta.modelName || params.meta.name;
  params.emitAuthzDecisionSummary({
    layer: 'company_filter',
    decision: 'allow',
    basis: 'company_filter_applied',
    fullMethod: req.fullMethod,
    method: req.method,
    model,
    userId: String(params.userId || '').trim(),
    activeCompanyId: scope.activeCompanyId,
    enabledCompanyIds: scope.enabledCompanyIds,
  });

  return Array.isArray(condition) && condition.length === 0 ? companyCondition : { And: [condition, companyCondition] };
}

export function applyRepositoryDefaultCompanyIdOnCreate(params: RepositoryCompanyScopeDeps, entity: Entity): Entity {
  if (!params.meta.companyScoped) return entity;

  repositoryCompanyScopedEnabled(params);
  const companyIds = normalizeRepositoryCompanyIds(params.ctx);
  const entityRecord = asObjectRecord(entity);

  if (Object.prototype.hasOwnProperty.call(entity || {}, 'CompanyId')) {
    validateRepositoryCompanyIdInScope(params, entityRecord?.CompanyId, companyIds);
    return entity;
  }

  const companyId = normalizeRepositoryCompanyIdForWrite(params.ctx);
  if (!companyId) {
    throw params.permissionDenied('company_scope_missing_default_company_id', 'missing ctx.activeCompanyId for company scoped create', {
      model: params.meta.fullModelName || params.meta.modelName || params.meta.name,
    });
  }

  return { ...entity, CompanyId: companyId };
}

export function applyRepositoryDefaultCompanyIdOnUpdate(params: RepositoryCompanyScopeDeps, vals: Entity): Entity {
  if (!params.meta.companyScoped) return vals;

  repositoryCompanyScopedEnabled(params);
  if (!Object.prototype.hasOwnProperty.call(vals || {}, 'CompanyId')) return vals;

  validateRepositoryCompanyIdInScope(params, asObjectRecord(vals)?.CompanyId, normalizeRepositoryCompanyIds(params.ctx));
  return vals;
}

export async function assertRepositoryCompanyWriteAccessForCondition(
  params: RepositoryCompanyScopeQueryDeps,
  condition: BaseQueryCondition
): Promise<string[]> {
  if (!params.meta.companyScoped) return [];

  repositoryCompanyScopedEnabled(params);

  let selectQuery = (params.db as RepositoryCompanyScopeDbLike).selectFrom(params.table).select(['Id', 'CompanyId']);
  const filtered = params.applySoftLayer(condition);
  if (!params.isEmptyCondition(filtered)) {
    selectQuery = selectQuery.where(({ eb }) => params.convertCondition(eb, filtered, params.table));
  }

  const rows = ((await params.execute(selectQuery as never)) || []) as Array<{ Id?: string; CompanyId?: unknown }>;
  const companyIds = normalizeRepositoryCompanyIds(params.ctx);
  for (const row of rows) {
    validateRepositoryCompanyIdInScope(params, row?.CompanyId, companyIds);
  }

  return rows.map(row => String(row?.Id || '')).filter(Boolean);
}
