// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { QueryContext } from './context';
import type { GroupBySpec } from './types';

/**
 * Query execution plan kinds produced from a query context.
 */
export type PlanKind = 'search' | 'count' | 'readGroup' | 'readGroupCount' | 'browse';

/**
 * Concrete driver-level execution plan.
 */
export interface QueryPlan {
  /** Plan category. */
  kind: PlanKind;

  /** Driver-specific payload passed to the service layer. */
  params: any;

  /** Stable signature independent of UI state object identity. */
  hash: string;
}

/**
 * Main query plan plus any supporting plans such as counts.
 */
export interface PlanBundle {
  /** Primary execution plan. */
  main: QueryPlan;

  /** Secondary plans required to complete the query response. */
  auxiliary: QueryPlan[];
}

/**
 * Serializes a value with deterministic object-key ordering.
 */
export function stableStringify(obj: any): string {
  const seen = new WeakSet();
  return JSON.stringify(obj, function replacer(key, value) {
    if (typeof value === 'object' && value !== null) {
      if (seen.has(value)) return '[Circular]';
      seen.add(value);
      if (Array.isArray(value)) return value.map(v => v);
      const sorted: any = Object.keys(value)
        .sort()
        .reduce((acc: any, k) => {
          acc[k] = (value as any)[k];
          return acc;
        }, {});
      return sorted;
    }
    return value;
  });
}

/**
 * Computes a stable hash for a query planning payload.
 */
export function stableHash(obj: any): string {
  const seen = new WeakSet();
  const str = JSON.stringify(obj, function replacer(key, value) {
    if (typeof value === 'object' && value !== null) {
      if (seen.has(value)) return '[Circular]';
      seen.add(value);
      if (Array.isArray(value)) return value.map(v => v);
      const sorted: any = Object.keys(value)
        .sort()
        .reduce((acc: any, k) => {
          acc[k] = (value as any)[k];
          return acc;
        }, {});
      return sorted;
    }
    return value;
  });
  let h = 2166136261 >>> 0;
  for (let i = 0; i < str.length; i++) {
    h ^= str.charCodeAt(i);
    h += (h << 1) + (h << 4) + (h << 7) + (h << 8) + (h << 24);
  }
  return ('0000000' + (h >>> 0).toString(16)).slice(-8);
}

/**
 * Returns the merged query condition payload for planning.
 */
function mergeConditions(ctx: QueryContext) {
  return ctx.filters;
}

/**
 * Builds executable query plans from a query context.
 */
export function buildPlan(ctx: QueryContext): PlanBundle {
  if (ctx.recordId != null) {
    const params = { id: ctx.recordId, fields: undefined };
    const main = { kind: 'browse' as const, params, hash: stableHash(['b', ctx.model, params.id]) };
    return { main, auxiliary: [] };
  }

  if (ctx.shape === 'groups') {
    const g = ctx.queryState;
    const fullGroupby: GroupBySpec<any>[] = g.appliedGroups || [];
    const useFull = !!ctx.options?.fullGroupHierarchy;
    const groupbyToUse: GroupBySpec<any>[] = useFull ? fullGroupby : fullGroupby.length > 1 ? [fullGroupby[0]] : fullGroupby;
    const skipPg = !!ctx.options?.skipPagination;
    const params = {
      groupby: groupbyToUse,
      condition: mergeConditions(ctx) ?? [],
      orderBy: g.orderBy,
      limit: skipPg ? undefined : ctx.range?.limit,
      offset: skipPg ? undefined : ctx.range?.offset,
      fields: undefined,
    };
    const main = {
      kind: 'readGroup' as const,
      params,
      hash: stableHash(['rg', ctx.model, params, useFull ? 'full' : 'top', skipPg ? 'nopg' : 'pg']),
    };
    const countParams = { groupby: groupbyToUse, condition: params.condition, fields: undefined };
    const count = {
      kind: 'readGroupCount' as const,
      params: countParams,
      hash: stableHash(['rgc', ctx.model, countParams, useFull ? 'full' : 'top']),
    };
    return { main, auxiliary: ctx.options?.skipCount ? [] : [count] };
  }

  const s = ctx.queryState;
  const params = {
    condition: mergeConditions(ctx),
    orderBy: s?.orderBy,
    limit: ctx.range?.limit,
    offset: ctx.range?.offset,
    fields: undefined,
  };
  const main = { kind: 'search' as const, params, hash: stableHash(['s', ctx.model, params]) };
  const countParams = { condition: params.condition };
  const count = { kind: 'count' as const, params: countParams, hash: stableHash(['c', ctx.model, countParams]) };
  return { main, auxiliary: ctx.options?.skipCount ? [] : [count] };
}
