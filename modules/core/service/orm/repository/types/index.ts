// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Author / SSOT type surface. Engine ducks (*Like / *DepsLike) and Kysely
// builders live in ./shared, ./common, ./execution — import those paths inside
// orm/repository/** only; do not re-export them here (SF-1 / TS2).

export type { SelectResult, Entity, FilteredQueryProperties, FilteredInputProperties, Selectable, Queryable } from './common';
export type { IdRelationItem, ModelRelationItem, RelationItem, RelationOperations, Insertable, Updateable } from './input';
export type { DeepRelationSelection, FieldSelection, Projected, PartialOrProjected, RowOrProjected } from './selection';
export { fields, projectToSelection } from './selection';
export type {
  Operator,
  BaseCondition,
  BaseQueryCondition,
  NestedPath,
  QueryCondition,
  OrderBy,
  SoftDeleteMode,
  SoftDeleteOptions,
  ForField,
  SearchOptions,
  CountOptions,
  UpdateOptions,
  DeleteOptions,
} from './query';
export { condition } from './query';
export type {
  TemporalGranularity,
  AggregateFunction,
  QueryPath,
  QueryablePath,
  GroupBySpec,
  FieldAggregation,
  RepoReadGroupOptions,
  RepoReadGroupRow,
  RepoReadTotalsOptions,
  RepoReadTotalsRow,
  RepoReadGroupCountOptions,
  ReadGroupShape,
  ReadGroupOptions,
  ReadGroupCountOptions,
  GroupRow,
  ReadGroupResult,
} from './groupby';
export type { RecordRuleOp, ConditionExpr, ConditionEnvelope } from './authz';
