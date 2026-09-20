// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Internal barrel for orm/repository/** only (SF-1). Not re-exported from
// repository/index or service/api.

export {
  DeleteResult,
  UpdateResult,
  InsertResult,
  ExpressionBuilder,
  ExpressionWrapper,
  SelectQueryBuilder,
  Compilable,
  SimplifyResult,
} from './common';
export type { RepositoryQueryLike, RepositoryExecute } from './execution';
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
export * from './index';
