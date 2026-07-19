// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export { BaseModel, Field, Model } from './model';

export type { Context } from './context';
export {
  getActiveCompanyId,
  getContextLang,
  getContextTimezone,
  getCtxValue,
  getEnabledCompanyIds,
  getIdentity,
  getReadonlyCtx,
  getReqMeta,
  getUserId,
  withContext,
} from './context';

export {
  createTranslate,
  withI18nScope,
  resolveI18nScope,
  formatScope,
  resolveRequestLang,
} from '../i18n';

export type { SelectResult, Entity, Selectable, Queryable } from './common';
export type { Insertable, Updateable } from './input';
export type { FieldPath, FieldPathType } from './field';
export type { DeepRelationSelection, FieldSelection } from './selection';

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
  BaseCondition,
  BaseQueryCondition,
  QueryCondition,
  SearchOptions,
  CountOptions,
  UpdateOptions,
  DeleteOptions,
  OrderBy,
  AggregateFunction,
  GroupBySpec,
  FieldAggregation,
  TemporalGranularity,
  ReadGroupOptions,
  ReadGroupCountOptions,
  ReadGroupResult,
} from './query';

export type { RecordRuleOp, ConditionExpr, ConditionEnvelope } from './authz';

export type { IdRelationItem, ModelRelationItem, RelationItem, RelationOperations } from './relation';

export { Onchange } from './onchange';
export type { OnchangeContext, OnchangeResult, PreviewModel } from './onchange';

export { ValidationEngine, resolveValidationSummary } from './validation';
export type { ValidationIssueLite, ValidationFieldIssueSummary, ResolvedValidationSummary } from './validation';

export { default as Decimal } from '../../utils/decimal';
