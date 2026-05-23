// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export { DeleteResult, UpdateResult, InsertResult, ExpressionBuilder, ExpressionWrapper, SelectQueryBuilder, Compilable } from './common';
export { SimplifyResult } from './common';

export type { RepositoryQueryLike, RepositoryExecute } from './execution';
export type { SelectResult, Entity, FilteredQueryProperties, FilteredInputProperties, Selectable, Queryable } from './common';
export type { IdRelationItem, ModelRelationItem, RelationItem, RelationOperations, Insertable, Updateable } from './input';
export type { DeepRelationSelection, FieldSelection } from './selection';
export type {
  Operator,
  BaseCondition,
  BaseQueryCondition,
  NestedPath,
  QueryCondition,
  OrderBy,
  SoftDeleteMode,
  SoftDeleteOptions,
  SearchOptions,
  CountOptions,
  UpdateOptions,
  DeleteOptions,
} from './query';
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
export type {
  NonNil,
  RepositoryAliasableLike,
  RepositoryRefBuilderLike,
  RepositoryCountAllFnLike,
  RepositorySelectFromDbLike,
  RepositoryCountAllDbLike,
  RepositoryWherePredicateLike,
  RepositoryWhereCapableLike,
  RepositorySelectCallbackLike,
  RepositorySelectCallbackCapableLike,
  RepositorySelectColumnsCapableLike,
  RepositoryLimitCapableLike,
  RepositoryOffsetCapableLike,
  RepositoryForUpdateCapableLike,
  RepositoryInsertIntoDbLike,
  RepositoryUpdateTableDbLike,
  RepositoryDeleteFromDbLike,
  RepositoryConditionConverterLike,
  RepositoryConditionLayerFnLike,
  RepositoryConditionEmptyCheckFnLike,
  RepositoryConditionConvertDepsLike,
  RepositoryDefaultConditionPipelineDepsLike,
  RepositorySoftConditionPipelineDepsLike,
  RepositoryRecordRuleConditionPipelineDepsLike,
  RepositoryTableConditionConvertDepsLike,
  RepositoryTableDefaultConditionPipelineDepsLike,
  RepositoryTableSoftConditionPipelineDepsLike,
  RepositoryMutationPayloadGuardDepsLike,
  RepositoryMutationPayloadDefaultsDepsLike,
  RepositoryMutationPayloadValidateDepsLike,
  RepositoryMutationPayloadEncodeDepsLike,
  RepositoryMutationPayloadGuardEncodeDepsLike,
  RepositoryGetScalarFieldsDepsLike,
  RepositorySelectCtxFactoryLike,
  RepositorySelectionAliaserLike,
  RepositoryExecuteUnknownQueryLike,
} from './shared';
