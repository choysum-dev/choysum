// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { sql } from 'kysely';
import BaseModel from '../../model/model';
import { MetadataStorage, type FieldMetadata, type ManyToOneMetadata, type ModelCtor, type ModelMetadata, type SelectCtx } from '../../metadata';
import { buildTimeBucketExpr } from './time_bucket_sql';
import type { DialectName } from '../repository_dialect';
import type {
  AggregateFunction,
  BaseQueryCondition,
  RepositoryAliasableLike,
  RepositorySelectCtxFactoryLike,
  RepositoryTableConditionConvertDepsLike,
  RepositoryWhereCapableLike,
} from '../types';
import type { GroupSpecLike, NormalizedAgg, NormalizedCompositeGroupSpec, NormalizedGroupSpec } from './group_spec';
import type { UnknownRecord } from '../../../../utils/types';
import { asObjectRecord } from '../../../../utils/object';

type AliasableExpression = RepositoryAliasableLike;

type SelectBuilderLike = {
  ref: (path: string) => unknown;
};

type RepositoryAggregateWhereCapable<T> = RepositoryWhereCapableLike<T>;

function toModelCtor(meta: ModelMetadata): ModelCtor<BaseModel> {
  return meta.type as unknown as ModelCtor<BaseModel>;
}

function isCompositeGroupSpec(group: GroupSpecLike): group is NormalizedCompositeGroupSpec {
  const record = asObjectRecord(group);
  return record?.composite === true && Array.isArray(record.parts);
}

function asAliasableExpression(value: unknown): AliasableExpression {
  return value as AliasableExpression;
}

function asSelectBuilder(value: unknown): SelectBuilderLike {
  return value as SelectBuilderLike;
}

function getFieldScale(fieldMeta: FieldMetadata): number | undefined {
  const columnRecord = asObjectRecord(fieldMeta.column);
  const scale = columnRecord?.scale;
  return typeof scale === 'number' ? scale : undefined;
}

type RepositoryReadAggregateExpressionDeps = {
  getDialect: () => DialectName;
  makeSelectCtx: RepositorySelectCtxFactoryLike<ModelMetadata, SelectCtx>;
  table: string;
  meta: ModelMetadata;
};

type RepositoryReadAggregateConditionDeps = RepositoryTableConditionConvertDepsLike<BaseQueryCondition>;

function buildRepositoryTimeGroupExpression(params: RepositoryReadAggregateExpressionDeps, builder: unknown, group: NormalizedGroupSpec, timezone?: string) {
  const ctx = params.makeSelectCtx(builder, params.table, params.meta);
  const colExpr = ctx.field(toModelCtor(params.meta), group.field);
  if (!group.granularity) {
    throw new Error(`time group requires granularity: ${group.field}`);
  }
  return buildTimeBucketExpr(params.getDialect(), colExpr, group.granularity, timezone);
}

function buildRepositoryAggregateExpression(
  params: Pick<RepositoryReadAggregateExpressionDeps, 'makeSelectCtx' | 'table' | 'meta'>,
  builder: unknown,
  aggSpec: { field: string; agg: AggregateFunction; alias: string; distinct?: boolean }
) {
  const ctx = params.makeSelectCtx(builder, params.table, params.meta);
  const fieldExpr = ctx.field(toModelCtor(params.meta), aggSpec.field);
  switch (aggSpec.agg) {
    case 'sum':
      return sql`SUM(${fieldExpr})`;
    case 'avg':
      return sql`AVG(${fieldExpr})`;
    case 'min':
      return sql`MIN(${fieldExpr})`;
    case 'max':
      return sql`MAX(${fieldExpr})`;
    case 'count_distinct':
      return sql`COUNT(DISTINCT ${fieldExpr})`;
    case 'count':
    default:
      if (aggSpec.distinct) return sql`COUNT(DISTINCT ${fieldExpr})`;
      return sql`COUNT(${fieldExpr})`;
  }
}

function resolveRepositoryLeafFieldMeta(meta: ModelMetadata, path: string): FieldMetadata | undefined {
  const parts = String(path || '')
    .split('.')
    .filter(Boolean);
  if (!parts.length) return undefined;

  let currentMeta = meta;
  for (let index = 0; index < parts.length; index++) {
    const segment = parts[index];
    const fieldMeta = currentMeta.fields.get(segment) as FieldMetadata | undefined;
    if (!fieldMeta) return undefined;
    const isLeaf = index === parts.length - 1;
    if (isLeaf) return fieldMeta;
    if (fieldMeta.type !== 'ManyToOne' || !fieldMeta.relation) return undefined;
    const ctor = (fieldMeta.relation as ManyToOneMetadata<BaseModel>).targetModel?.();
    if (!ctor) return undefined;
    currentMeta = MetadataStorage.instance.getModelMetadata(ctor);
  }

  return undefined;
}

export function buildRepositoryReadAggregateGroupExprs(
  params: RepositoryReadAggregateExpressionDeps,
  builder: unknown,
  group: GroupSpecLike,
  timezone?: string
): unknown[] {
  if (isCompositeGroupSpec(group)) {
    return group.parts.map(part =>
      part.isTime
        ? buildRepositoryTimeGroupExpression(params, builder, part, timezone)
        : params.makeSelectCtx(builder, params.table, params.meta).field(toModelCtor(params.meta), part.field)
    );
  }

  return [
    group.isTime
      ? buildRepositoryTimeGroupExpression(params, builder, group, timezone)
      : params.makeSelectCtx(builder, params.table, params.meta).field(toModelCtor(params.meta), group.field),
  ];
}

export function buildRepositoryReadAggregateSelections(
  params: RepositoryReadAggregateExpressionDeps,
  selBuilder: unknown,
  group: GroupSpecLike,
  aggs: NormalizedAgg[],
  countAll: () => AliasableExpression,
  timezone?: string
): unknown[] {
  const selections: unknown[] = [];
  const groupExprs = buildRepositoryReadAggregateGroupExprs(params, selBuilder, group, timezone);
  if (isCompositeGroupSpec(group)) {
    const parts = group.parts;
    groupExprs.forEach((expr, index) => selections.push(asAliasableExpression(expr).as(parts[index].alias)));
  } else {
    selections.push(asAliasableExpression(groupExprs[0]).as(group.alias));
  }

  for (const agg of aggs) {
    const aggExpr = buildRepositoryAggregateExpression(params, selBuilder, agg);
    selections.push(asAliasableExpression(aggExpr).as(agg.alias));
  }
  selections.push(countAll().as('__count'));
  return selections;
}

export function buildRepositoryReadAggregateTotalSelections(
  params: Pick<RepositoryReadAggregateExpressionDeps, 'makeSelectCtx' | 'table' | 'meta'>,
  selBuilder: unknown,
  aggs: NormalizedAgg[],
  countAll: () => AliasableExpression
): unknown[] {
  const selections: unknown[] = [];
  for (const agg of aggs) {
    const aggExpr = buildRepositoryAggregateExpression(params, selBuilder, agg);
    selections.push(asAliasableExpression(aggExpr).as(agg.alias));
  }
  selections.push(countAll().as('__count'));
  return selections;
}

export function applyRepositoryReadAggregateCondition<T>(query: T, params: RepositoryReadAggregateConditionDeps, filtered: BaseQueryCondition): T {
  if (params.isEmptyCondition(filtered)) {
    return query;
  }
  const whereCapableQuery = query as RepositoryAggregateWhereCapable<T>;
  return whereCapableQuery.where(({ eb }) => params.convertCondition(eb, filtered, params.table));
}

export function resolveRepositoryReadAggregateKnownAliases(group: GroupSpecLike, aggs: Array<{ alias: string }>): Set<string> {
  const groupAliases = isCompositeGroupSpec(group) ? group.parts.map(part => part.alias) : [group.alias];
  return new Set<string>([...groupAliases, '__count', ...aggs.map(agg => agg.alias)]);
}

export function normalizeRepositoryAggregateDecimals(
  meta: ModelMetadata,
  rows: UnknownRecord[],
  aggs: Array<{ field: string; agg: AggregateFunction; alias: string }>
) {
  if (!rows?.length) return rows;

  const need = new Map<string, FieldMetadata>();
  for (const agg of aggs) {
    if (agg.agg === 'count' || agg.agg === 'count_distinct') continue;
    const fieldMeta = agg.field.includes('.') ? resolveRepositoryLeafFieldMeta(meta, agg.field) : (meta.fields.get(agg.field) as FieldMetadata | undefined);
    if (fieldMeta && (fieldMeta.type === 'decimal' || fieldMeta.type === 'monetary')) {
      need.set(agg.alias, fieldMeta);
    }
  }
  if (!need.size) return rows;

  const toNumber = (value: unknown) => {
    if (typeof value === 'number') return value;
    if (typeof value === 'bigint') return Number(value);
    if (typeof value === 'string') {
      const normalized = Number(value);
      return Number.isNaN(normalized) ? value : normalized;
    }
    return value;
  };

  for (const row of rows) {
    for (const [alias, fieldMeta] of need) {
      const value = row[alias];
      if (value == null) continue;
      const normalized = toNumber(value);
      const scale = getFieldScale(fieldMeta);
      if (typeof normalized === 'number' && Number.isFinite(normalized) && typeof scale === 'number') {
        const factor = Math.pow(10, Math.max(0, scale));
        row[alias] = Math.round(normalized * factor) / factor;
      } else if (typeof normalized === 'number' && Number.isFinite(normalized)) {
        row[alias] = normalized;
      }
    }
  }

  return rows;
}
