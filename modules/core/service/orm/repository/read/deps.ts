// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata } from '../../metadata';
import type { DialectName } from '../repository_dialect';
import { convertRepositoryHavingCondition } from '../query';
import type {
  BaseQueryCondition,
  Entity,
  RepositoryGetScalarFieldsDepsLike,
  RepositoryRecordRuleConditionPipelineDepsLike,
  RepositoryTableConditionConvertDepsLike,
  RepositoryExecuteUnknownQueryLike,
  RepositorySelectionAliaserLike,
  RepositorySelectCtxFactoryLike,
} from '../types';
import type { SelectionRelationEntry } from '../projection';
import type { RepositoryOrderSpec } from '../query/ordering';

type RepositoryConditionQueryDepsParams = {
  db: unknown;
  applyConditionLayers: (condition: BaseQueryCondition) => BaseQueryCondition;
  execute: RepositoryExecuteUnknownQueryLike;
} & RepositoryTableConditionConvertDepsLike<BaseQueryCondition>;

type RepositoryReadConditionDepsParams = RepositoryRecordRuleConditionPipelineDepsLike<'read', BaseQueryCondition> & {
  execute: RepositoryExecuteUnknownQueryLike;
};

type RepositoryReadOrderDepsParams = {
  normalizeOrderBy: (input: unknown) => unknown;
  applyOrderByToQuery: (query: unknown, meta: unknown, table: string, orderBy: RepositoryOrderSpec[]) => unknown;
};

type RepositoryReadQueryFacadeDepsParams = RepositoryReadConditionDepsParams & RepositoryReadOrderDepsParams;

type RepositorySearchFacadeDepsParams = RepositoryReadQueryFacadeDepsParams & {
  db: unknown;
  table: string;
  meta: ModelMetadata;
  getDialect: () => DialectName;
  isTopLevelGrpcCall: () => boolean;
  buildSelectionTree: (meta: ModelMetadata, fields: unknown[]) => unknown;
  getScalarFields: RepositoryGetScalarFieldsDepsLike<ModelMetadata>['getScalarFields'];
  pruneSelectionTreeForFieldRule: (meta: ModelMetadata, node: unknown, denyCache: Map<unknown, string[]>) => Promise<void>;
  makeSelectCtx: RepositorySelectCtxFactoryLike<ModelMetadata>;
  aliasSelection: RepositorySelectionAliaserLike;
  buildRelationJsonSelect: (qb: unknown, parentMeta: ModelMetadata, relKey: string, entry: SelectionRelationEntry) => unknown;
  resolveEffectiveOrder: (
    overrideOrder: RepositoryOrderSpec[] | undefined | null,
    metaOrder: RepositoryOrderSpec[] | undefined | null,
    meta: unknown
  ) => RepositoryOrderSpec[] | null | undefined;
  decodeRowWithTree: (meta: ModelMetadata, node: unknown, row: Entity) => Entity;
};

type RepositoryReadAggregateDepsParams = RepositoryReadConditionDepsParams &
  RepositoryReadOrderDepsParams & {
    db: unknown;
    table: string;
    meta: ModelMetadata;
    ctx: unknown;
    getDialect: () => DialectName;
    makeSelectCtx: RepositorySelectCtxFactoryLike<ModelMetadata>;
  };

export function createRepositoryConditionQueryDeps(params: RepositoryConditionQueryDepsParams) {
  return {
    db: params.db,
    table: params.table,
    applyConditionLayers: params.applyConditionLayers,
    isEmptyCondition: params.isEmptyCondition,
    convertCondition: params.convertCondition,
    execute: params.execute,
  };
}

export function createRepositoryReadConditionDeps(params: RepositoryReadConditionDepsParams) {
  return {
    applyRecordRuleToCondition: params.applyRecordRuleToCondition,
    applyDefaultLayers: params.applyDefaultLayers,
    isEmptyCondition: params.isEmptyCondition,
    convertCondition: params.convertCondition,
    execute: params.execute,
  };
}

export function createRepositoryReadOrderDeps(params: RepositoryReadOrderDepsParams) {
  return {
    normalizeOrderBy: params.normalizeOrderBy,
    applyOrderByToQuery: params.applyOrderByToQuery,
  };
}

export function createRepositoryReadQueryFacadeDeps(params: RepositoryReadQueryFacadeDepsParams) {
  return {
    ...createRepositoryReadConditionDeps(params),
    ...createRepositoryReadOrderDeps(params),
  };
}

export function createRepositorySearchFacadeDeps(params: RepositorySearchFacadeDepsParams) {
  return {
    db: params.db,
    table: params.table,
    meta: params.meta,
    getDialect: params.getDialect,
    isTopLevelGrpcCall: params.isTopLevelGrpcCall,
    buildSelectionTree: params.buildSelectionTree,
    getScalarFields: params.getScalarFields,
    pruneSelectionTreeForFieldRule: params.pruneSelectionTreeForFieldRule,
    makeSelectCtx: params.makeSelectCtx,
    aliasSelection: params.aliasSelection,
    buildRelationJsonSelect: params.buildRelationJsonSelect,
    ...createRepositoryReadQueryFacadeDeps(params),
    resolveEffectiveOrder: params.resolveEffectiveOrder,
    decodeRowWithTree: params.decodeRowWithTree,
  };
}

export function createRepositoryReadAggregateDeps(params: RepositoryReadAggregateDepsParams) {
  return {
    db: params.db,
    table: params.table,
    meta: params.meta,
    ctx: params.ctx,
    getDialect: params.getDialect,
    makeSelectCtx: params.makeSelectCtx,
    convertCondition: params.convertCondition,
    convertHaving: (eb: unknown, condition: BaseQueryCondition, knownAliases: Set<string>) =>
      convertRepositoryHavingCondition(
        {
          convertCondition: (builder, conditionValue, selfTable) => params.convertCondition(builder, conditionValue, selfTable),
          selfTable: params.table,
        },
        eb,
        condition,
        knownAliases
      ),
    applyRecordRuleToCondition: params.applyRecordRuleToCondition,
    applyDefaultLayers: params.applyDefaultLayers,
    isEmptyCondition: params.isEmptyCondition,
    normalizeOrderBy: params.normalizeOrderBy,
    applyOrderByToQuery: params.applyOrderByToQuery,
    execute: params.execute,
  };
}

export function createRepositoryReadAggregateFacadeDeps(params: RepositoryReadAggregateDepsParams) {
  return createRepositoryReadAggregateDeps(params);
}
