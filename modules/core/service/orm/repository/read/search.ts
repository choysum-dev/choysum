// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata } from '../../metadata';
import { REL_ALIAS_PREFIX } from '../../relation/relation_alias';
import type {
  BaseQueryCondition,
  Entity,
  RepositoryGetScalarFieldsDepsLike,
  RepositoryRecordRuleConditionPipelineDepsLike,
  SearchOptions,
  RepositoryExecute,
  RepositoryRefBuilderLike,
  RepositorySelectionAliaserLike,
  RepositorySelectFromDbLike,
  RepositorySelectCtxFactoryLike,
  RepositoryWhereCapableLike,
  RepositorySelectCallbackCapableLike,
  RepositoryLimitCapableLike,
  RepositoryOffsetCapableLike,
  RepositoryForUpdateCapableLike,
} from '../types';
import type { DialectName } from '../repository_dialect';
import { buildHiddenScaleAlias } from '../hidden_scale_alias';
import type { SelectionNode, SelectionRelationEntry } from '../projection';
import type { RepositoryOrderSpec } from '../query/ordering';
import { hasRepositorySqlComputeExpression, resolveRepositorySqlComputeExpression } from '../query';
import { asObjectRecord } from '../../../../utils/object';
import type { ObjectRecord } from '../../../../utils/types';

type SelectCtxLike = {
  field: (modelCtor: unknown, path: string) => unknown;
};

interface RepositorySearchSelectQueryLike
  extends
    RepositoryWhereCapableLike<RepositorySearchSelectQueryLike>,
    RepositoryLimitCapableLike<RepositorySearchSelectQueryLike>,
    RepositoryOffsetCapableLike<RepositorySearchSelectQueryLike>,
    RepositoryForUpdateCapableLike<RepositorySearchSelectQueryLike> {}

type RepositorySearchSelectFromBuilderLike = RepositorySelectCallbackCapableLike<RepositorySearchSelectQueryLike, unknown, unknown[]>;

type RepositorySearchDbLike = RepositorySelectFromDbLike<RepositorySearchSelectFromBuilderLike, string>;

type RepositorySearchReadDeps = {
  db: unknown;
  table: string;
  meta: ModelMetadata;
  getDialect: () => DialectName;
  isTopLevelGrpcCall: () => boolean;
  buildSelectionTree: (meta: ModelMetadata, fields: unknown[]) => unknown;
  pruneSelectionTreeForFieldRule: (meta: ModelMetadata, node: unknown, denyCache: Map<unknown, string[]>) => Promise<void>;
  makeSelectCtx: RepositorySelectCtxFactoryLike<ModelMetadata>;
  aliasSelection: RepositorySelectionAliaserLike;
  buildRelationJsonSelect: (qb: unknown, parentMeta: ModelMetadata, relKey: string, entry: SelectionRelationEntry) => unknown;
  normalizeOrderBy: (input: unknown) => unknown;
  resolveEffectiveOrder: (
    overrideOrder: RepositoryOrderSpec[] | undefined | null,
    metaOrder: RepositoryOrderSpec[] | undefined | null,
    meta: unknown
  ) => RepositoryOrderSpec[] | null | undefined;
  applyOrderByToQuery: (query: unknown, targetMeta: unknown, targetTable: string, orderList: RepositoryOrderSpec[]) => unknown;
  execute: RepositoryExecute;
  decodeRowWithTree: (meta: ModelMetadata, node: unknown, row: Entity) => Entity;
} & RepositoryGetScalarFieldsDepsLike<ModelMetadata> &
  RepositoryRecordRuleConditionPipelineDepsLike<'read', BaseQueryCondition>;

function toSelectionNode(input: unknown): SelectionNode {
  const record = asObjectRecord(input);
  if (record?.columns instanceof Set && record?.relations instanceof Map) {
    return input as SelectionNode;
  }
  const columns = record?.columns instanceof Set ? (record.columns as Set<string>) : new Set<string>();
  const relations = record?.relations instanceof Map ? (record.relations as Map<string, SelectionRelationEntry>) : new Map<string, SelectionRelationEntry>();
  return { columns, relations };
}

export async function executeRepositorySearch(
  params: RepositorySearchReadDeps,
  condition: BaseQueryCondition,
  options?: SearchOptions<ObjectRecord>
): Promise<Entity[]> {
  const requestedList = Array.isArray(options?.fields) ? ([...options.fields] as unknown[]) : undefined;
  const requestedFields = requestedList ? (requestedList.includes('Id') ? requestedList : [...requestedList, 'Id']) : undefined;

  const selectFromBuilder = (params.db as RepositorySearchDbLike).selectFrom(params.table);

  let selectionTree: SelectionNode;
  if (requestedFields && requestedFields.length > 0) {
    selectionTree = toSelectionNode(params.buildSelectionTree(params.meta, requestedFields));
  } else {
    const defaultColumns = params.getScalarFields(params.meta);
    selectionTree = { columns: new Set(defaultColumns), relations: new Map() };
  }

  if (params.isTopLevelGrpcCall() && requestedFields) {
    try {
      const denyCache = new Map<unknown, string[]>();
      await params.pruneSelectionTreeForFieldRule(params.meta, selectionTree, denyCache);
    } catch {
      // Best-effort only; reads must remain available even if pruning fails.
    }
  }

  let selectQuery = selectFromBuilder.select((qb: unknown) => {
    const selections: unknown[] = [];
    const refBuilder = qb as RepositoryRefBuilderLike<string>;
    const scalarCols = Array.from(selectionTree.columns);
    if (!scalarCols.includes('Id')) scalarCols.unshift('Id');

    for (const column of scalarCols) {
      const fieldMeta = params.meta.fields.get(column);
      const selectCtx = params.makeSelectCtx(qb, params.table, params.meta) as SelectCtxLike;
      if (column.includes('.')) {
        const expr = selectCtx.field(params.meta.type, column);
        selections.push(params.aliasSelection(expr, column));
      } else if (hasRepositorySqlComputeExpression(params.meta, column)) {
        const out = resolveRepositorySqlComputeExpression(params.meta, column, selectCtx);
        if (out === undefined) {
          throw new Error(`field sql compute handler is missing: ${params.meta.fullModelName || params.meta.modelName || params.meta.name}.${column}`);
        }
        selections.push(params.aliasSelection(out, column));
      } else {
        selections.push(refBuilder.ref(`${params.table}.${column}`).as(column));
      }

      if (fieldMeta?.type === 'decimal') {
        const spec = asObjectRecord(fieldMeta.column || {});
        const scale = spec?.scale;
        const scaleField = typeof spec?.scaleField === 'string' ? spec.scaleField : undefined;
        if (typeof scale !== 'number' && scaleField) {
          const scaleMeta = params.meta.fields.get(scaleField);
          if (hasRepositorySqlComputeExpression(params.meta, scaleField)) {
            const out = resolveRepositorySqlComputeExpression(params.meta, scaleField, selectCtx);
            if (out === undefined) {
              throw new Error(`field sql compute handler is missing: ${params.meta.fullModelName || params.meta.modelName || params.meta.name}.${scaleField}`);
            }
            selections.push(params.aliasSelection(out, buildHiddenScaleAlias(column)));
          } else if (scaleMeta?.column) {
            selections.push(refBuilder.ref(`${params.table}.${scaleField}`).as(buildHiddenScaleAlias(column)));
          }
        }
      }
    }

    for (const [relKey, relNode] of selectionTree.relations) {
      const expr = params.buildRelationJsonSelect(qb, params.meta, relKey, relNode);
      if (expr) selections.push(params.aliasSelection(expr, `${REL_ALIAS_PREFIX}${relKey}`));
    }

    return selections;
  });

  const condWithRR = await params.applyRecordRuleToCondition(condition, 'read');
  const filtered = params.applyDefaultLayers(condWithRR);
  if (!params.isEmptyCondition(filtered)) {
    selectQuery = selectQuery.where(({ eb }: { eb: unknown }) => params.convertCondition(eb, filtered, params.table));
  }

  const optOrder = params.normalizeOrderBy(options?.orderBy) as RepositoryOrderSpec[] | undefined;
  const metaOrder = params.normalizeOrderBy(params.meta.orderBy as unknown) as RepositoryOrderSpec[] | undefined;
  const effectiveOrder = params.resolveEffectiveOrder(optOrder, metaOrder, params.meta);
  selectQuery = params.applyOrderByToQuery(
    selectQuery,
    params.meta,
    params.table,
    Array.isArray(effectiveOrder) ? effectiveOrder : []
  ) as RepositorySearchSelectQueryLike;

  if (options?.limit) {
    selectQuery = selectQuery.limit(options.limit);
  }
  if (options?.offset) {
    selectQuery = selectQuery.offset(options.offset);
  }
  if (options?.forUpdate && params.getDialect() !== 'sqlite') {
    selectQuery = selectQuery.forUpdate();
  }

  const results = await params.execute(selectQuery as never);
  if (!results?.length) return [];
  return results.map(row => params.decodeRowWithTree(params.meta, selectionTree, row as Entity));
}
