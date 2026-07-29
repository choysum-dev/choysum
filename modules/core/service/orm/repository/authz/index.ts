// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export type { RepositoryAuthzContextDepsParams } from './deps';
export type { RepositoryAuthzDecisionSummary, RepositoryEmitAuthzDecisionSummary, RepositoryPermissionDeniedFn } from './types';
export {
  createRepositoryAuthzContextDeps,
  createRepositoryRecordRuleDeps,
  createRepositoryRecordRulePolicyDeps,
  createRepositoryRecordRuleCoordinatorDeps,
  createRepositoryRecordRuleCoordinatorPolicyDeps,
  createRepositoryFieldRuleDeps,
  createRepositoryFieldRulePolicyDeps,
  createRepositoryFieldRuleSelectionDeps,
  createRepositoryCompanyScopeDeps,
  createRepositoryCompanyScopePolicyDeps,
  createRepositoryCompanyScopeQueryDeps,
} from './deps';
export { AuthUserService, isAuthServiceNotPresent, isAuthServiceUnavailable } from './auth_user_service';
export type { RepositoryAuthzDecisionLogMode, RepositoryReqMethodMeta, RepositoryCompanyScopeFacts } from './authz_runtime';
export {
  getRepositoryAuthzDecisionLogMode,
  repositoryAuthzDecisionAuditEnabled,
  emitRepositoryAuthzDecisionSummary,
  getRepositoryCurrentReq,
  getRepositoryCurrentReqWrapper,
  isRepositoryTopLevelGrpcCall,
  getOrInitRepositoryReqServiceState,
  getRepositoryReqMethodMeta,
  getRepositoryCompanyScopeFacts,
  getRepositoryRecordRuleBypassDepth,
  withRepositoryRecordRuleBypass,
  getRepositoryFieldRuleBypassDepth,
  withRepositoryFieldRuleBypass,
  withRepositoryAuthzRuleBypass,
  getRepositoryValidationBypassState,
  getRepositoryValidationBypassDepth,
  withRepositoryValidationBypass,
} from './authz_runtime';
export {
  normalizeRepositoryCompanyIds,
  normalizeRepositoryCompanyIdForWrite,
  validateRepositoryCompanyIdInScope,
  resolveRepositoryCompanyField,
  repositoryHasCompanyField,
  requireRepositoryOwnershipField,
  isRepositoryOwnershipFieldNotNull,
  validateRepositoryOwnershipNullability,
  repositoryCompanyFieldEnabled,
  applyRepositoryCompanyLayer,
  applyRepositoryDefaultCompanyIdOnCreate,
  applyRepositoryDefaultCompanyIdOnUpdate,
  assertRepositoryCompanyWriteAccessForCondition,
} from './company_scope';
export {
  getRepositoryRecordRuleEnvelope,
  replaceRepositoryRecordRuleTokens,
  applyRepositoryRecordRuleToCondition,
  assertRepositoryRecordRuleAllTargetsAllowed,
  assertRepositoryRecordRuleCreateAllowed,
  assertRepositoryRecordRuleAllCreatedAllowed,
} from './record_rule';
export type { RepositoryRecordRuleDeps } from './record_rule_helpers';
export { fetchRepositoryRecordRuleEnvelope, replaceRepositoryRecordRuleConditionTokens } from './record_rule_helpers';
export type { RepositoryFieldRuleDeps, RepositoryFieldRuleSelectionDeps, RepositoryFieldRuleSpec } from './field_rule';
export {
  assertRepositoryFieldRuleWriteAllowed,
  buildFailClosedFieldRuleSpec,
  getRepositoryFieldRuleSpec,
  getRepositoryTopLevelFieldRuleMode,
  pruneRepositorySelectionTreeForFieldRule,
  repositoryFieldRuleEnabled,
  repositoryFieldRuleLayerSkipped,
} from './field_rule';
export { createRepositoryPermissionDeniedError, resolveRepositoryPermissionDeniedLayer } from './permission_denied';
