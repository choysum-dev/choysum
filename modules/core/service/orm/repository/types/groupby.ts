// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { BaseQueryCondition, QueryCondition, SoftDeleteOptions } from './query';
import type { ObjectRecord } from '../../../../utils/types';

export type TemporalGranularity = 'year' | 'quarter' | 'month' | 'week' | 'day';
export type AggregateFunction = 'count' | 'count_distinct' | 'sum' | 'avg' | 'min' | 'max';
export type QueryPath = string;
/** @deprecated Prefer QueryPath. Kept for compatibility. */
export type QueryablePath<T = ObjectRecord> = QueryPath;

type IsAny<T> = 0 extends 1 & T ? true : false;
type FieldKeyOf<T> = Extract<keyof T, string>;
type GroupByField<T> = IsAny<T> extends true ? string : FieldKeyOf<T>;
type GroupByShorthandOf<T> = GroupByField<T> | `${GroupByField<T>}:${TemporalGranularity}`;
type GroupOrderBy<T> = { field: GroupByField<T>; order: 'asc' | 'desc' } | Array<{ field: GroupByField<T>; order: 'asc' | 'desc' }>;

export type GroupBySpec<T = ObjectRecord> =
  | GroupByShorthandOf<T>
  | {
      field: GroupByField<T>;
      granularity?: TemporalGranularity;
      alias?: string;
      range?: { start: Date | string; end: Date | string };
    };

export type FieldAggregation<T = ObjectRecord> =
  | `${string}:${AggregateFunction}`
  | {
      field: GroupByField<T> | string;
      agg: AggregateFunction;
      alias?: string;
      distinct?: boolean;
    };

export interface RepoReadGroupOptions<T = ObjectRecord> {
  groupby: GroupBySpec<T> | GroupBySpec<T>[];
  fields?: Array<FieldAggregation<T>>;
  condition?: QueryCondition<T> | [] | BaseQueryCondition;
  having?: BaseQueryCondition;
  orderBy?: GroupOrderBy<T>;
  limit?: number;
  offset?: number;
  timezone?: string;
}

export type RepoReadGroupRow = ObjectRecord & { __count: number };

export interface RepoReadTotalsOptions<T = ObjectRecord> {
  fields?: Array<FieldAggregation<T>>;
  condition?: QueryCondition<T> | [] | BaseQueryCondition;
  timezone?: string;
}

export type RepoReadTotalsRow = ObjectRecord & { __count: number };

export interface RepoReadGroupCountOptions<T = ObjectRecord> {
  groupby: GroupBySpec<T> | GroupBySpec<T>[];
  fields?: Array<FieldAggregation<T>>;
  condition?: QueryCondition<T> | [] | BaseQueryCondition;
  having?: BaseQueryCondition;
  timezone?: string;
}

export type ReadGroupShape = 'flat' | 'tree';

export interface ReadGroupOptions<T = ObjectRecord> extends SoftDeleteOptions {
  fields?: Array<FieldAggregation<T>>;
  having?: BaseQueryCondition | BaseQueryCondition[];
  orderBy?: GroupOrderBy<T>;
  limit?: number | { perLevel?: number[] };
  offset?: number;
  timezone?: string;
  fillTemporalGaps?: boolean;
}

export interface ReadGroupCountOptions<T = ObjectRecord> extends SoftDeleteOptions {
  fields?: Array<FieldAggregation<T>>;
  having?: BaseQueryCondition | BaseQueryCondition[];
  timezone?: string;
}

export interface GroupRow {
  depth: number;
  keys: ObjectRecord;
  labels: Record<string, string>;
  metrics: ObjectRecord;
  count: number;
  condition?: BaseQueryCondition;
  remainingGroupby?: Array<GroupBySpec<ObjectRecord> | GroupBySpec<ObjectRecord>[]>;
  children?: GroupRow[];
}

export type ReadGroupResult = GroupRow[];
