// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export type NonNil<V> = Exclude<V, null | undefined>;

export type RepositoryAliasableLike = {
  as: (alias: string) => unknown;
};

export type RepositoryRefBuilderLike<TPath = string, TAliasable extends RepositoryAliasableLike = RepositoryAliasableLike> = {
  ref: (path: TPath) => TAliasable;
};

export type RepositoryCountAllFnLike<TAliasable extends RepositoryAliasableLike = RepositoryAliasableLike> = {
  countAll: () => TAliasable;
};

export type RepositorySelectFromDbLike<TQuery, TTable = unknown> = {
  selectFrom: (table: TTable) => TQuery;
};

export type RepositoryCountAllDbLike<
  TQuery,
  TTable = unknown,
  TAliasable extends RepositoryAliasableLike = RepositoryAliasableLike,
> = RepositorySelectFromDbLike<TQuery, TTable> & {
  fn: RepositoryCountAllFnLike<TAliasable>;
};

export type RepositoryWherePredicateLike = (args: { eb: unknown }) => unknown;

export type RepositoryWhereCapableLike<TSelf> = {
  where: (callback: RepositoryWherePredicateLike) => TSelf;
};

export type RepositorySelectCallbackLike<TBuilder = unknown, TResult = unknown[]> = (qb: TBuilder) => TResult;

export type RepositorySelectCallbackCapableLike<TSelf, TBuilder = unknown, TResult = unknown[]> = {
  select: (callback: RepositorySelectCallbackLike<TBuilder, TResult>) => TSelf;
};

export type RepositorySelectColumnsCapableLike<TSelf, TColumns = string[]> = {
  select: (columns: TColumns) => TSelf;
};

export type RepositoryLimitCapableLike<TSelf> = {
  limit: (value: number) => TSelf;
};

export type RepositoryOffsetCapableLike<TSelf> = {
  offset: (value: number) => TSelf;
};

export type RepositoryForUpdateCapableLike<TSelf> = {
  forUpdate: () => TSelf;
};

export type RepositoryInsertIntoDbLike<TInsertBuilder, TTable = string> = {
  insertInto: (table: TTable) => TInsertBuilder;
};

export type RepositoryUpdateTableDbLike<TSetResult, TValue = unknown, TTable = string> = {
  updateTable: (table: TTable) => {
    set: (values: TValue) => TSetResult;
  };
};

export type RepositoryDeleteFromDbLike<TDeleteQuery, TTable = string> = {
  deleteFrom: (table: TTable) => TDeleteQuery;
};

export type RepositoryConditionConverterLike<TCondition = unknown> = {
  (eb: unknown, condition: TCondition, selfTable?: string): unknown;
};

export type RepositoryConditionLayerFnLike<TCondition = unknown> = {
  (condition: TCondition): TCondition;
};

export type RepositoryConditionEmptyCheckFnLike<TCondition = unknown> = {
  (condition: TCondition): boolean;
};

export type RepositoryConditionConvertDepsLike<TCondition = unknown> = {
  isEmptyCondition: RepositoryConditionEmptyCheckFnLike<TCondition>;
  convertCondition: RepositoryConditionConverterLike<TCondition>;
};

export type RepositoryDefaultConditionPipelineDepsLike<TCondition = unknown> = {
  applyDefaultLayers: RepositoryConditionLayerFnLike<TCondition>;
} & RepositoryConditionConvertDepsLike<TCondition>;

export type RepositorySoftConditionPipelineDepsLike<TCondition = unknown> = {
  applySoftLayer: RepositoryConditionLayerFnLike<TCondition>;
} & RepositoryConditionConvertDepsLike<TCondition>;

export type RepositoryRecordRuleConditionPipelineDepsLike<TOp = unknown, TCondition = unknown> = {
  applyRecordRuleToCondition: (condition: TCondition, op: TOp) => Promise<TCondition>;
} & RepositoryDefaultConditionPipelineDepsLike<TCondition>;

export type RepositoryTableConditionConvertDepsLike<TCondition = unknown> = {
  table: string;
} & RepositoryConditionConvertDepsLike<TCondition>;

export type RepositoryTableDefaultConditionPipelineDepsLike<TCondition = unknown> = {
  table: string;
} & RepositoryDefaultConditionPipelineDepsLike<TCondition>;

export type RepositoryTableSoftConditionPipelineDepsLike<TCondition = unknown> = {
  table: string;
} & RepositorySoftConditionPipelineDepsLike<TCondition>;

export type RepositoryMutationPayloadGuardDepsLike<TEntity = unknown> = {
  assertFieldRuleWriteAllowed: (payload: TEntity) => Promise<void>;
};

export type RepositoryMutationPayloadDefaultsDepsLike<TEntity = unknown> = {
  applyDefaultMutationValues: (payload: TEntity) => TEntity;
};

export type RepositoryMutationPayloadValidateDepsLike<TEntity = unknown, TMode = unknown, TCurrent = unknown> = {
  validateFields: (input: TEntity, mode: TMode, current?: TCurrent) => Promise<void>;
};

export type RepositoryMutationPayloadEncodeDepsLike<TEntity = unknown> = {
  encodeForDb: (input: TEntity) => TEntity;
};

export type RepositoryMutationPayloadGuardEncodeDepsLike<TEntity = unknown> = RepositoryMutationPayloadGuardDepsLike<TEntity> &
  RepositoryMutationPayloadEncodeDepsLike<TEntity>;

export type RepositoryGetScalarFieldsDepsLike<TMeta = unknown> = {
  getScalarFields: (meta: TMeta) => string[];
};

export type RepositorySelectCtxFactoryLike<TMeta = unknown, TCtx = unknown> = {
  (builder: unknown, selfTable: string, curMeta?: TMeta): TCtx;
};

export type RepositorySelectionAliaserLike = {
  (selection: unknown, alias: string): unknown;
};

export type RepositoryExecuteUnknownQueryLike = {
  <T = unknown>(query: unknown): Promise<T[]>;
};
