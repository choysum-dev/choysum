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
import { _t } from '@/core/service/i18n_binder';

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

/** Non-empty ownership field name when the model is company-isolated. */
export function resolveRepositoryCompanyField(meta: ModelMetadata | undefined | null): string | undefined {
  const field = String(meta?.companyField ?? '').trim();
  return field || undefined;
}

/** True when meta declares a company ownership field (L1 isolated). */
export function repositoryHasCompanyField(meta: ModelMetadata | undefined | null): boolean {
  return Boolean(resolveRepositoryCompanyField(meta));
}

/**
 * Ownership column for an isolated model. Throws PermissionDenied-style via
 * the provided callback when the field is missing from metadata.
 */
export function requireRepositoryOwnershipField(
  meta: ModelMetadata,
  permissionDenied: RepositoryPermissionDeniedFn
): string {
  const field = resolveRepositoryCompanyField(meta);
  if (!field) {
    throw permissionDenied(
      'company_field_not_isolated',
      _t('model is not company-isolated', { scope: 'service/orm/repository/authz/company_scope' }),
      {
        model: meta.fullModelName || meta.modelName || meta.name,
      }
    );
  }
  if (!(meta.fields instanceof Map) || !meta.fields.has(field)) {
    throw permissionDenied(
      'company_field_missing',
      _t('companyField model is missing ownership field', { scope: 'service/orm/repository/authz/company_scope' }),
      {
        model: meta.fullModelName || meta.modelName || meta.name,
        companyField: field,
      }
    );
  }
  return field;
}

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
    throw params.permissionDenied(
      'company_scope_violation',
      _t('company is not in ctx.enabledCompanyIds', { scope: 'service/orm/repository/authz/company_scope' }),
      {
        model: params.meta.fullModelName || params.meta.modelName || params.meta.name,
        companyId: normalized,
      }
    );
  }
}

export function repositoryCompanyFieldEnabled(params: RepositoryCompanyScopeDeps): boolean {
  const grpcEnabled = getRuntimeEnvFlag('CHOYSUM_GRPC_COMPANY_FILTER_ENABLED', true);
  if (!grpcEnabled) return false;
  if (params.companyLayerSkipped()) return false;
  if (!repositoryHasCompanyField(params.meta)) return false;

  requireRepositoryOwnershipField(params.meta, params.permissionDenied);

  const ids = normalizeRepositoryCompanyIds(params.ctx);
  if (!ids.length) {
    throw params.permissionDenied(
      'company_scope_missing_ctx_company',
      _t(
        'missing ctx.enabledCompanyIds/activeCompanyId for company scoped operation',
        { scope: 'service/orm/repository/authz/company_scope' }
      ),
      {
        model: params.meta.fullModelName || params.meta.modelName || params.meta.name,
      }
    );
  }

  return true;
}

export function applyRepositoryCompanyLayer(params: RepositoryCompanyScopeDeps, condition: BaseQueryCondition): BaseQueryCondition {
  if (params.companyLayerSkipped()) return condition;
  if (!repositoryCompanyFieldEnabled(params)) return condition;

  const ownershipField = requireRepositoryOwnershipField(params.meta, params.permissionDenied);
  const companyIds = normalizeRepositoryCompanyIds(params.ctx);
  const companyCondition: BaseQueryCondition = {
    Or: [
      [ownershipField, 'in', companyIds],
      [ownershipField, 'is', null],
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
    reason: 'company_filter_applied',
  });

  return Array.isArray(condition) && condition.length === 0 ? companyCondition : { And: [condition, companyCondition] };
}

export function applyRepositoryDefaultCompanyIdOnCreate(params: RepositoryCompanyScopeDeps, entity: Entity): Entity {
  if (!repositoryHasCompanyField(params.meta)) return entity;

  repositoryCompanyFieldEnabled(params);
  const ownershipField = requireRepositoryOwnershipField(params.meta, params.permissionDenied);
  const companyIds = normalizeRepositoryCompanyIds(params.ctx);
  const entityRecord = asObjectRecord(entity);

  if (Object.prototype.hasOwnProperty.call(entity || {}, ownershipField)) {
    validateRepositoryCompanyIdInScope(params, entityRecord?.[ownershipField], companyIds);
    return entity;
  }

  const companyId = normalizeRepositoryCompanyIdForWrite(params.ctx);
  if (!companyId) {
    throw params.permissionDenied(
      'company_scope_missing_default_company_id',
      _t('missing ctx.activeCompanyId for company scoped create', { scope: 'service/orm/repository/authz/company_scope' }),
      {
        model: params.meta.fullModelName || params.meta.modelName || params.meta.name,
      }
    );
  }

  return { ...entity, [ownershipField]: companyId };
}

export function applyRepositoryDefaultCompanyIdOnUpdate(params: RepositoryCompanyScopeDeps, vals: Entity): Entity {
  if (!repositoryHasCompanyField(params.meta)) return vals;

  repositoryCompanyFieldEnabled(params);
  const ownershipField = requireRepositoryOwnershipField(params.meta, params.permissionDenied);
  if (!Object.prototype.hasOwnProperty.call(vals || {}, ownershipField)) return vals;

  validateRepositoryCompanyIdInScope(params, asObjectRecord(vals)?.[ownershipField], normalizeRepositoryCompanyIds(params.ctx));
  return vals;
}

export async function assertRepositoryCompanyWriteAccessForCondition(
  params: RepositoryCompanyScopeQueryDeps,
  condition: BaseQueryCondition
): Promise<string[]> {
  if (!repositoryHasCompanyField(params.meta)) return [];

  repositoryCompanyFieldEnabled(params);
  const ownershipField = requireRepositoryOwnershipField(params.meta, params.permissionDenied);

  let selectQuery = (params.db as RepositoryCompanyScopeDbLike).selectFrom(params.table).select(['Id', ownershipField]);
  const filtered = params.applySoftLayer(condition);
  if (!params.isEmptyCondition(filtered)) {
    selectQuery = selectQuery.where(({ eb }) => params.convertCondition(eb, filtered, params.table));
  }

  const rows = ((await params.execute(selectQuery as never)) || []) as Array<Record<string, unknown>>;
  const companyIds = normalizeRepositoryCompanyIds(params.ctx);
  for (const row of rows) {
    validateRepositoryCompanyIdInScope(params, row?.[ownershipField], companyIds);
  }

  return rows.map(row => String(row?.Id || '')).filter(Boolean);
}
