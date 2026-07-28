// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata } from '../../metadata';
import { andAll } from '../query';
import { fetchRepositoryRecordRuleEnvelope, replaceRepositoryRecordRuleConditionTokens, type RepositoryRecordRuleDeps } from './record_rule_helpers';
import type { BaseQueryCondition, ConditionEnvelope, RecordRuleOp } from '../types';
import type { RepositoryCompanyScopeFacts, RepositoryReqMethodMeta } from './authz_runtime';
import type { RepositoryEmitAuthzDecisionSummary, RepositoryPermissionDeniedFn } from './types';
import { _t } from '@/core/service/i18n_binder';

export async function getRepositoryRecordRuleEnvelope(params: RepositoryRecordRuleDeps, op: RecordRuleOp): Promise<ConditionEnvelope> {
  return await fetchRepositoryRecordRuleEnvelope(params, op);
}

export function replaceRepositoryRecordRuleTokens(params: RepositoryRecordRuleDeps, condition: BaseQueryCondition): BaseQueryCondition {
  return replaceRepositoryRecordRuleConditionTokens(params, condition);
}

type RepositoryRecordRuleCoordinatorDeps = {
  meta: ModelMetadata;
  userId?: string;
  recordRuleEnabled: () => boolean;
  getRecordRuleEnvelope: (op: RecordRuleOp) => Promise<ConditionEnvelope>;
  replaceRecordRuleTokens: (condition: BaseQueryCondition) => BaseQueryCondition;
  getReqMethodMeta: () => RepositoryReqMethodMeta;
  getCompanyScopeFacts: () => RepositoryCompanyScopeFacts;
  emitAuthzDecisionSummary: RepositoryEmitAuthzDecisionSummary;
  permissionDenied: RepositoryPermissionDeniedFn;
  countConditionMatches: (condition: BaseQueryCondition) => Promise<number>;
};

function encodeHitRuleIds(ids: string[] | undefined): string | undefined {
  const cleaned = Array.from(new Set((ids || []).map(id => String(id ?? '').trim()).filter(Boolean))).sort();
  return cleaned.length ? cleaned.join(',') : undefined;
}

export async function applyRepositoryRecordRuleToCondition(
  params: RepositoryRecordRuleCoordinatorDeps,
  condition: BaseQueryCondition,
  op: RecordRuleOp
): Promise<BaseQueryCondition> {
  if (!params.recordRuleEnabled()) return condition;

  const env = await params.getRecordRuleEnvelope(op);
  if (env.kind === 'true') return condition;
  if (env.kind === 'false') {
    if (op === 'read') {
      const model = params.meta.fullModelName || params.meta.modelName || params.meta.name;
      const req = params.getReqMethodMeta();
      const scope = params.getCompanyScopeFacts();
      params.emitAuthzDecisionSummary({
        layer: 'record_rule',
        decision: 'deny',
        basis: 'record_rule_denied_read_empty_set',
        fullMethod: req.fullMethod,
        method: req.method,
        model,
        op,
        userId: String(params.userId || '').trim(),
        activeCompanyId: scope.activeCompanyId,
        enabledCompanyIds: scope.enabledCompanyIds,
        reason: env.reason || 'denied',
        hitRuleIds: env.hitRuleIds,
      });
      const never: BaseQueryCondition = ['Id', '=', '__choysum_never__'];
      return andAll(condition, never) as BaseQueryCondition;
    }
    const metadata: Record<string, string> = {
      model: params.meta.fullModelName || params.meta.modelName || params.meta.name,
      op,
      reason: env.reason || 'denied',
    };
    const hitRuleIds = encodeHitRuleIds(env.hitRuleIds);
    if (hitRuleIds) metadata.hitRuleIds = hitRuleIds;
    throw params.permissionDenied('record_rule_denied', _t('record rule denied', { scope: 'service/orm/repository/authz/record_rule' }), metadata);
  }

  const expr = params.replaceRecordRuleTokens(env.expr);
  const req = params.getReqMethodMeta();
  const scope = params.getCompanyScopeFacts();
  const model = params.meta.fullModelName || params.meta.modelName || params.meta.name;
  params.emitAuthzDecisionSummary({
    layer: 'record_rule',
    decision: 'allow',
    basis: 'record_rule_expr_applied',
    fullMethod: req.fullMethod,
    method: req.method,
    model,
    op,
    userId: String(params.userId || '').trim(),
    activeCompanyId: scope.activeCompanyId,
    enabledCompanyIds: scope.enabledCompanyIds,
    reason: env.reason || '',
    hitRuleIds: env.hitRuleIds,
  });

  return andAll(condition, expr) as BaseQueryCondition;
}

export async function assertRepositoryRecordRuleAllTargetsAllowed(
  params: RepositoryRecordRuleCoordinatorDeps,
  op: Extract<RecordRuleOp, 'write' | 'delete'>,
  targetIds: string[]
): Promise<void> {
  if (!params.recordRuleEnabled()) return;
  if (!targetIds.length) return;

  const env = await params.getRecordRuleEnvelope(op);
  if (env.kind === 'true') return;
  if (env.kind === 'false') {
    const metadata: Record<string, string> = {
      model: params.meta.fullModelName || params.meta.modelName || params.meta.name,
      op,
      reason: env.reason || 'denied',
    };
    const hitRuleIds = encodeHitRuleIds(env.hitRuleIds);
    if (hitRuleIds) metadata.hitRuleIds = hitRuleIds;
    throw params.permissionDenied('record_rule_denied', _t('record rule denied', { scope: 'service/orm/repository/authz/record_rule' }), metadata);
  }

  const rrExpr = params.replaceRecordRuleTokens(env.expr);
  const checkCondition: BaseQueryCondition = { And: [['Id', 'in', targetIds], rrExpr] };
  const allowed = await params.countConditionMatches(checkCondition);
  if (allowed !== targetIds.length) {
    const metadata: Record<string, string> = {
      model: params.meta.fullModelName || params.meta.modelName || params.meta.name,
      op,
      targetCount: String(targetIds.length),
      allowedCount: String(allowed),
      reason: env.reason || 'denied',
    };
    const hitRuleIds = encodeHitRuleIds(env.hitRuleIds);
    if (hitRuleIds) metadata.hitRuleIds = hitRuleIds;
    throw params.permissionDenied(
      'record_rule_violation',
      _t('target set violates record rule', { scope: 'service/orm/repository/authz/record_rule' }),
      metadata
    );
  }
}

export async function assertRepositoryRecordRuleCreateAllowed(params: RepositoryRecordRuleCoordinatorDeps): Promise<void> {
  if (!params.recordRuleEnabled()) return;
  const env = await params.getRecordRuleEnvelope('create');
  if (env.kind === 'false') {
    const metadata: Record<string, string> = {
      model: params.meta.fullModelName || params.meta.modelName || params.meta.name,
      op: 'create',
      reason: env.reason || 'denied',
    };
    const hitRuleIds = encodeHitRuleIds(env.hitRuleIds);
    if (hitRuleIds) metadata.hitRuleIds = hitRuleIds;
    throw params.permissionDenied('record_rule_denied', _t('record rule denied', { scope: 'service/orm/repository/authz/record_rule' }), metadata);
  }
}

export async function assertRepositoryRecordRuleAllCreatedAllowed(
  params: RepositoryRecordRuleCoordinatorDeps,
  createdIds: string[],
  env: ConditionEnvelope
): Promise<void> {
  if (!params.recordRuleEnabled()) return;
  if (!createdIds.length) return;
  if (env.kind !== 'expr') return;

  const rrExpr = params.replaceRecordRuleTokens(env.expr);
  const checkCondition: BaseQueryCondition = { And: [['Id', 'in', createdIds], rrExpr] };
  const allowed = await params.countConditionMatches(checkCondition);
  if (allowed !== createdIds.length) {
    const metadata: Record<string, string> = {
      model: params.meta.fullModelName || params.meta.modelName || params.meta.name,
      op: 'create',
      createdCount: String(createdIds.length),
      allowedCount: String(allowed),
      reason: env.reason || 'denied',
    };
    const hitRuleIds = encodeHitRuleIds(env.hitRuleIds);
    if (hitRuleIds) metadata.hitRuleIds = hitRuleIds;
    throw params.permissionDenied(
      'record_rule_violation',
      _t('created set violates record rule', { scope: 'service/orm/repository/authz/record_rule' }),
      metadata
    );
  }
}
