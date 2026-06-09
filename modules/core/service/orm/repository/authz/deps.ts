// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata } from '../../metadata';
import type { RepositoryCompanyScopeFacts, RepositoryReqMethodMeta } from './authz_runtime';
import type { RepositoryEmitAuthzDecisionSummary, RepositoryPermissionDeniedFn } from './types';
import type { BaseQueryCondition, ConditionEnvelope, RecordRuleOp, RepositoryExecute, RepositoryTableSoftConditionPipelineDepsLike } from '../types';

export type RepositoryAuthzContextDepsParams = {
  meta: ModelMetadata;
  userId?: string;
  getReqMethodMeta: () => RepositoryReqMethodMeta;
  getCompanyScopeFacts: () => RepositoryCompanyScopeFacts;
  emitAuthzDecisionSummary: RepositoryEmitAuthzDecisionSummary;
};

type RepositoryAuthPolicyCommonDepsParams = {
  meta: ModelMetadata;
  userId?: string;
  requestContext: unknown;
  normalizeCompanyIds: () => string[];
  permissionDenied: RepositoryPermissionDeniedFn;
};

type RepositoryRecordRuleDepsParams = RepositoryAuthPolicyCommonDepsParams & {
  normalizeCompanyIdForWrite: () => string | undefined;
  isControlPlaneMetaModel: () => boolean;
  recordRuleEnabled: () => boolean;
  getRecordRuleBypassDepth: () => number;
  withRecordRuleBypass: <T>(fn: () => Promise<T>) => Promise<T>;
};

type RepositoryRecordRuleCoordinatorDepsParams = RepositoryAuthzContextDepsParams & {
  recordRuleEnabled: () => boolean;
  getRecordRuleEnvelope: (op: RecordRuleOp) => Promise<ConditionEnvelope>;
  replaceRecordRuleTokens: (condition: BaseQueryCondition) => BaseQueryCondition;
  permissionDenied: RepositoryPermissionDeniedFn;
  countConditionMatches: (condition: BaseQueryCondition) => Promise<number>;
};

type RepositoryFieldRuleDepsParams = RepositoryAuthPolicyCommonDepsParams & {
  isControlPlaneMetaModel: () => boolean;
  isFieldRuleControlPlaneModel: () => boolean;
  withRecordRuleBypass: <T>(fn: () => Promise<T>) => Promise<T>;
  withFieldRuleBypass: <T>(fn: () => Promise<T>) => Promise<T>;
};

type RepositoryCompanyScopeDepsParams = RepositoryAuthzContextDepsParams & {
  ctx: unknown;
  companyLayerSkipped: () => boolean;
  permissionDenied: RepositoryPermissionDeniedFn;
};

type RepositoryCompanyScopeQueryDepsParams = RepositoryCompanyScopeDepsParams &
  RepositoryTableSoftConditionPipelineDepsLike<BaseQueryCondition> & {
    db: unknown;
    execute: RepositoryExecute;
  };

function createRepositoryAuthPolicyCommonDeps(params: RepositoryAuthPolicyCommonDepsParams) {
  return {
    meta: params.meta,
    userId: params.userId,
    requestContext: params.requestContext,
    normalizeCompanyIds: params.normalizeCompanyIds,
    permissionDenied: params.permissionDenied,
  };
}

export function createRepositoryAuthzContextDeps(params: RepositoryAuthzContextDepsParams) {
  return {
    meta: params.meta,
    userId: params.userId,
    getReqMethodMeta: params.getReqMethodMeta,
    getCompanyScopeFacts: params.getCompanyScopeFacts,
    emitAuthzDecisionSummary: params.emitAuthzDecisionSummary,
  };
}

export function createRepositoryRecordRuleDeps(params: RepositoryRecordRuleDepsParams) {
  return {
    ...createRepositoryAuthPolicyCommonDeps(params),
    normalizeCompanyIdForWrite: params.normalizeCompanyIdForWrite,
    isControlPlaneMetaModel: params.isControlPlaneMetaModel,
    recordRuleEnabled: params.recordRuleEnabled,
    getRecordRuleBypassDepth: params.getRecordRuleBypassDepth,
    withRecordRuleBypass: params.withRecordRuleBypass,
  };
}

export function createRepositoryRecordRulePolicyDeps(params: RepositoryRecordRuleDepsParams) {
  return createRepositoryRecordRuleDeps(params);
}

export function createRepositoryRecordRuleCoordinatorDeps(params: RepositoryRecordRuleCoordinatorDepsParams) {
  return {
    ...createRepositoryAuthzContextDeps(params),
    recordRuleEnabled: params.recordRuleEnabled,
    getRecordRuleEnvelope: params.getRecordRuleEnvelope,
    replaceRecordRuleTokens: params.replaceRecordRuleTokens,
    permissionDenied: params.permissionDenied,
    countConditionMatches: params.countConditionMatches,
  };
}

export function createRepositoryRecordRuleCoordinatorPolicyDeps(params: RepositoryRecordRuleCoordinatorDepsParams) {
  return createRepositoryRecordRuleCoordinatorDeps(params);
}

export function createRepositoryFieldRuleDeps(params: RepositoryFieldRuleDepsParams) {
  return {
    ...createRepositoryAuthPolicyCommonDeps(params),
    isControlPlaneMetaModel: params.isControlPlaneMetaModel,
    isFieldRuleControlPlaneModel: params.isFieldRuleControlPlaneModel,
    withRecordRuleBypass: params.withRecordRuleBypass,
    withFieldRuleBypass: params.withFieldRuleBypass,
  };
}

export function createRepositoryFieldRulePolicyDeps(params: RepositoryFieldRuleDepsParams) {
  return createRepositoryFieldRuleDeps(params);
}

export function createRepositoryFieldRuleSelectionDeps(params: Pick<RepositoryFieldRuleDepsParams, 'isControlPlaneMetaModel'>) {
  return {
    isControlPlaneMetaModel: params.isControlPlaneMetaModel,
  };
}

export function createRepositoryCompanyScopeDeps(params: RepositoryCompanyScopeDepsParams) {
  return {
    ...createRepositoryAuthzContextDeps(params),
    ctx: params.ctx,
    companyLayerSkipped: params.companyLayerSkipped,
    permissionDenied: params.permissionDenied,
  };
}

export function createRepositoryCompanyScopePolicyDeps(params: RepositoryCompanyScopeDepsParams) {
  return createRepositoryCompanyScopeDeps(params);
}

export function createRepositoryCompanyScopeQueryDeps(params: RepositoryCompanyScopeQueryDepsParams) {
  return {
    ...createRepositoryCompanyScopeDeps(params),
    db: params.db,
    table: params.table,
    applySoftLayer: params.applySoftLayer,
    isEmptyCondition: params.isEmptyCondition,
    convertCondition: params.convertCondition,
    execute: params.execute,
  };
}
