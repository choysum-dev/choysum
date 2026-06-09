// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { sql } from 'kysely';
import type { ModelMetadata } from '../../metadata';
import type { DialectName } from '../repository_dialect';
import {
  normalizeFieldAggregation as normalizeFieldAggregationShared,
  normalizeGroupBySpec as normalizeGroupBySpecShared,
  normalizeGroupBySpecs as normalizeGroupBySpecsShared,
} from './group_spec';
import type { GroupSpecLike, NormalizedAgg, NormalizedCompositeGroupSpec } from './group_spec';
import type {
  BaseQueryCondition,
  FieldAggregation,
  GroupBySpec,
  RepoReadGroupCountOptions,
  RepoReadGroupOptions,
  RepoReadGroupRow,
  RepoReadTotalsOptions,
  RepoReadTotalsRow,
  RepositoryCountAllDbLike,
  RepositoryExecuteUnknownQueryLike,
  RepositoryRecordRuleConditionPipelineDepsLike,
  RepositorySelectCtxFactoryLike,
} from '../types';
import {
  applyRepositoryReadAggregateCondition,
  buildRepositoryReadAggregateGroupExprs,
  buildRepositoryReadAggregateSelections,
  buildRepositoryReadAggregateTotalSelections,
  normalizeRepositoryAggregateDecimals,
  resolveRepositoryReadAggregateKnownAliases,
} from './read_aggregate_helpers';
import type { ObjectRecord } from '../../../../utils/types';
import { asObjectRecord } from '../../../../utils/object';
import type { SelectCtx } from '../../metadata';

type AggregateQueryLike = {
  select: (selection: unknown) => AggregateQueryLike;
  groupBy: (grouping: unknown) => AggregateQueryLike;
  having: (predicate: (args: { eb: unknown }) => unknown) => AggregateQueryLike;
  limit: (count: number) => AggregateQueryLike;
  offset: (count: number) => AggregateQueryLike;
  as: (alias: string) => unknown;
};

type AggregateDbLike = RepositoryCountAllDbLike<AggregateQueryLike>;

function isCompositeGroupSpec(group: GroupSpecLike): group is NormalizedCompositeGroupSpec {
  const record = asObjectRecord(group);
  return record?.composite === true && Array.isArray(record.parts);
}

function resolveTimezone(explicitTimezone: string | undefined, ctx: unknown): string | undefined {
  if (explicitTimezone) return explicitTimezone;
  const record = asObjectRecord(ctx);
  return typeof record?.tz === 'string' ? record.tz : undefined;
}

type RepositoryReadAggregateDeps = {
  db: unknown;
  table: string;
  meta: ModelMetadata;
  ctx: unknown;
  getDialect: () => DialectName;
  makeSelectCtx: RepositorySelectCtxFactoryLike<ModelMetadata, SelectCtx>;
  convertHaving: (eb: unknown, condition: BaseQueryCondition, knownAliases: Set<string>) => unknown;
  normalizeOrderBy: (orderBy: unknown) => unknown;
  applyOrderByToQuery: (query: unknown, meta: unknown, table: string, orderBy: unknown) => unknown;
  execute: RepositoryExecuteUnknownQueryLike;
} & RepositoryRecordRuleConditionPipelineDepsLike<'read', BaseQueryCondition>;

export async function executeRepositoryReadGroup<T>(params: RepositoryReadAggregateDeps, options: RepoReadGroupOptions<T>): Promise<RepoReadGroupRow[]> {
  if (!options || !options.groupby) {
    throw new Error('readGroup requires options.groupby');
  }

  const timezone = resolveTimezone(options.timezone, params.ctx);
  const isComposite = Array.isArray(options.groupby);
  const group: GroupSpecLike = isComposite
    ? normalizeGroupBySpecsShared(options.groupby as GroupBySpec<ObjectRecord>[])
    : normalizeGroupBySpecShared(options.groupby as GroupBySpec<ObjectRecord>);
  const aggs: NormalizedAgg[] = (options.fields ?? []).map(field => normalizeFieldAggregationShared(field as FieldAggregation<ObjectRecord>));

  const db = params.db as AggregateDbLike;
  let qb: AggregateQueryLike = db.selectFrom(params.table);
  qb = qb.select((selBuilder: unknown) => buildRepositoryReadAggregateSelections(params, selBuilder, group, aggs, () => db.fn.countAll(), timezone));

  const condWithRR = await params.applyRecordRuleToCondition((options.condition ?? []) as BaseQueryCondition, 'read');
  const filtered = params.applyDefaultLayers(condWithRR);
  qb = applyRepositoryReadAggregateCondition(qb, params, filtered);

  qb = qb.groupBy((gb: unknown) => buildRepositoryReadAggregateGroupExprs(params, gb, group, timezone));

  if (options.having) {
    const knownAliases = resolveRepositoryReadAggregateKnownAliases(group, aggs);
    qb = qb.having(({ eb }: { eb: unknown }) => params.convertHaving(eb, options.having as BaseQueryCondition, knownAliases));
  }

  const optOrder = params.normalizeOrderBy(options.orderBy);
  if (Array.isArray(optOrder) && optOrder.length) {
    qb = params.applyOrderByToQuery(qb, params.meta, params.table, optOrder) as AggregateQueryLike;
  }

  if (typeof options.limit === 'number') qb = qb.limit(options.limit);
  if (typeof options.offset === 'number') qb = qb.offset(options.offset);

  const rows = ((await params.execute<RepoReadGroupRow>(qb)) || []) as RepoReadGroupRow[];
  normalizeRepositoryAggregateDecimals(params.meta, rows as ObjectRecord[], aggs);
  for (const row of rows) row.__count = Number(row.__count ?? 0);
  return rows;
}

export async function executeRepositoryReadTotals<T>(params: RepositoryReadAggregateDeps, options: RepoReadTotalsOptions<T>): Promise<RepoReadTotalsRow> {
  const aggs: NormalizedAgg[] = (options.fields ?? []).map(field => normalizeFieldAggregationShared(field as FieldAggregation<ObjectRecord>));

  const db = params.db as AggregateDbLike;
  let qb = db
    .selectFrom(params.table)
    .select((selBuilder: unknown) => buildRepositoryReadAggregateTotalSelections(params, selBuilder, aggs, () => db.fn.countAll()));

  const condWithRR = await params.applyRecordRuleToCondition((options.condition ?? []) as BaseQueryCondition, 'read');
  const filtered = params.applyDefaultLayers(condWithRR);
  qb = applyRepositoryReadAggregateCondition(qb, params, filtered);

  const rows = (await params.execute<RepoReadTotalsRow>(qb)) as RepoReadTotalsRow[] | undefined;
  const row = (rows && rows[0]) || ({} as RepoReadTotalsRow);
  normalizeRepositoryAggregateDecimals(params.meta, [row] as ObjectRecord[], aggs);
  row.__count = Number(row.__count ?? 0);
  return row;
}

export async function executeRepositoryReadGroupCount<T>(params: RepositoryReadAggregateDeps, options: RepoReadGroupCountOptions<T>): Promise<number> {
  if (!options || !options.groupby) {
    throw new Error('readGroupCount requires options.groupby');
  }

  const timezone = resolveTimezone(options.timezone, params.ctx);
  const isComposite = Array.isArray(options.groupby);
  const group: GroupSpecLike = isComposite
    ? normalizeGroupBySpecsShared(options.groupby as GroupBySpec<ObjectRecord>[])
    : normalizeGroupBySpecShared(options.groupby as GroupBySpec<ObjectRecord>);
  const aggs: NormalizedAgg[] = (options.fields ?? []).map(field => normalizeFieldAggregationShared(field as FieldAggregation<ObjectRecord>));

  const condWithRR = await params.applyRecordRuleToCondition((options.condition ?? []) as BaseQueryCondition, 'read');
  const filtered = params.applyDefaultLayers(condWithRR);

  if (!options.having) {
    if (!isCompositeGroupSpec(group)) {
      const db = params.db as AggregateDbLike;
      let qb = db.selectFrom(params.table).select((builder: unknown) => {
        const expr = buildRepositoryReadAggregateGroupExprs(params, builder, group, timezone)[0];
        return [sql`COUNT(DISTINCT ${expr})`.as('Total')];
      });
      qb = applyRepositoryReadAggregateCondition(qb, params, filtered);
      const rows = (await params.execute<{ Total: number }>(qb)) as Array<{ Total: number }> | undefined;
      const total = rows && rows[0] ? Number(rows[0].Total ?? 0) : 0;
      return Number.isFinite(total) ? total : 0;
    }
  }

  const db = params.db as AggregateDbLike;
  let sub: AggregateQueryLike = db
    .selectFrom(params.table)
    .select((selBuilder: unknown) => buildRepositoryReadAggregateSelections(params, selBuilder, group, aggs, () => db.fn.countAll(), timezone));

  sub = applyRepositoryReadAggregateCondition(sub, params, filtered);

  sub = sub.groupBy((gb: unknown) => buildRepositoryReadAggregateGroupExprs(params, gb, group, timezone));

  const knownAliases = resolveRepositoryReadAggregateKnownAliases(group, aggs);
  sub = sub.having(({ eb }: { eb: unknown }) => params.convertHaving(eb, options.having as BaseQueryCondition, knownAliases));

  const subAliased = sub.as('t');
  const outer = db.selectFrom(subAliased).select(db.fn.countAll().as('Total'));
  const rows = (await params.execute<{ Total: number }>(outer)) as Array<{ Total: number }> | undefined;
  const total = rows && rows[0] ? Number(rows[0].Total ?? 0) : 0;
  return Number.isFinite(total) ? total : 0;
}
