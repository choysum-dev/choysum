// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export { BaseModel, Field, Model } from './model';

export type { Context } from './context';
export {
  getActiveCompanyId,
  getContextLang,
  getContextTimezone,
  getContextCompanyTimezone,
  getContextClientTimezone,
  getCtxValue,
  getEnabledCompanyIds,
  getIdentity,
  getReadonlyCtx,
  getReqMeta,
  getUserId,
  withContext,
  withUser,
} from './context';

export {
  createTranslate,
  withI18nScope,
  resolveI18nScope,
  formatScope,
  resolveRequestLang,
} from '../i18n';

export type { SelectResult, Selectable } from './common';
export type { Insertable, Updateable } from './input';
export type { FieldPath, FieldPathType } from './field';
export type { DeepRelationSelection, FieldSelection, Projected, PartialOrProjected, RowOrProjected } from './selection';
export { fields, projectToSelection } from './selection';

export { Constraint, getEffectiveConstraints, ValidationPipelineError } from './constraint';
export type {
  ConstraintMode,
  ConstraintField,
  ConstraintOptions,
  ConstraintMeta,
  EffectiveConstraintMeta,
  ValidationIssue,
  ConstraintContext,
  ConstraintMethod,
} from './constraint';

export type {
  Operator,
  UntypedCondition,
  UntypedQueryCondition,
  QueryCondition,
  QueryConditionLeaf,
  QueryConditionNode,
  SearchOptions,
  SoftDeleteOptions,
  OrderBy,
  AggregateFunction,
  GroupBySpec,
  FieldAggregation,
  TemporalGranularity,
  ReadGroupOptions,
  ReadGroupCountOptions,
  ReadGroupResult,
  RelationConditionSource,
} from './query';
export { condition } from './query';

export type { RecordRuleOp, ConditionEnvelope, FieldRuleSpec } from './authz';

export type { IdRelationItem, ModelRelationItem, RelationItem, RelationPatch } from './relation';

export { Onchange } from './onchange';
export type { OnchangeContext, OnchangeResult, PreviewModel } from './onchange';

export { ValidationEngine, resolveValidationSummary } from './validation';
export type { ValidationIssueLite, ValidationFieldIssueSummary, ResolvedValidationSummary } from './validation';

export { default as Decimal } from '../../utils/decimal';
