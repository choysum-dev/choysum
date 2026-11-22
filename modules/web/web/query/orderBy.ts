// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { OrderByState } from '@/web/web/query/state';

/** Repository / RPC sort row (`Search` / `ReadGroup` options.orderBy). */
export type RpcOrderBy = {
  field: string;
  order: 'asc' | 'desc';
};

type OrderByLike = {
  field?: unknown;
  order?: unknown;
  direction?: unknown;
};

function asOrderByLike(value: unknown): OrderByLike | undefined {
  if (!value || typeof value !== 'object') return undefined;
  return value as OrderByLike;
}

function normalizeSortDir(raw: unknown): 'asc' | 'desc' {
  return String(raw ?? 'asc').toLowerCase() === 'desc' ? 'desc' : 'asc';
}

/**
 * Map FE query-state / UI sort rows (`direction`) onto RPC `order`.
 * Accepts either key so callers that already use `order` stay valid.
 */
export function toRpcOrderBy(input: unknown): RpcOrderBy[] | undefined {
  if (input == null) return undefined;
  const arr = Array.isArray(input) ? input : [input];
  const out: RpcOrderBy[] = [];
  for (const item of arr) {
    const candidate = asOrderByLike(item);
    if (!candidate) continue;
    const field = String(candidate.field ?? '').trim();
    if (!field) continue;
    const dirRaw = candidate.order ?? candidate.direction ?? 'asc';
    out.push({ field, order: normalizeSortDir(dirRaw) });
  }
  return out.length ? out : undefined;
}

/**
 * TanStack SortingState from query-state / props orderBy (direction or order).
 */
export function orderByToSortingState(
  input: unknown
): Array<{ id: string; desc: boolean }> {
  const rpc = toRpcOrderBy(input);
  if (!rpc?.length) return [];
  const primary = rpc[0]!;
  return [{ id: primary.field, desc: primary.order === 'desc' }];
}

/** Narrow helper when the caller already holds OrderByState[]. */
export function orderByStateToRpc(orderBy: OrderByState[] | undefined): RpcOrderBy[] | undefined {
  return toRpcOrderBy(orderBy);
}
