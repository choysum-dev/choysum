// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Author / SSOT type surface. Engine ducks (*Like / *DepsLike) and Kysely
// builders live in ./shared, ./common, ./execution — import those paths inside
// orm/repository/** only; do not re-export them here (SF-1 / TS2).

export type { SelectResult, FilteredQueryProperties, FilteredInputProperties, Selectable } from './common';
export type { IdRelationItem, ModelRelationItem, RelationItem, RelationPatch, Insertable, Updateable } from './input';
export type { DeepRelationSelection, FieldSelection, Projected, PartialOrProjected, RowOrProjected } from './selection';
export { fields, projectToSelection } from './selection';
export type {
  Operator,
  UntypedCondition,
  UntypedQueryCondition,
  NestedPath,
  QueryCondition,
  QueryConditionLeaf,
  QueryConditionNode,
  OrderBy,
  SoftDeleteOptions,
  RelationConditionSource,
  SearchOptions,
} from './query';
export { condition } from './query';
export type {
  TemporalGranularity,
  AggregateFunction,
  QueryPath,
  GroupBySpec,
  FieldAggregation,
  RepositoryReadGroupOptions,
  RepositoryReadGroupRow,
  RepositoryReadTotalsOptions,
  RepositoryReadTotalsRow,
  RepositoryReadGroupCountOptions,
  ReadGroupShape,
  ReadGroupOptions,
  ReadGroupCountOptions,
  GroupRow,
  ReadGroupResult,
} from './groupby';
export type { RecordRuleOp, ConditionEnvelope } from './authz';
