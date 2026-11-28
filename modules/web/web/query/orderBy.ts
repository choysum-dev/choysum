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
  return String(raw ?? 'asc').trim().toLowerCase() === 'desc' ? 'desc' : 'asc';
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

/**
 * Map static `metadata.orderBy` (`order`) onto FE query-state rows (`direction`).
 */
export function metadataOrderByToState(input: unknown): OrderByState[] | undefined {
  const rpc = toRpcOrderBy(input);
  if (!rpc?.length) return undefined;
  return rpc.map(row => ({ field: row.field, direction: row.order }));
}

/**
 * Keep only orderBy rows whose field is in the groupby key set (ReadGroup / chart).
 * Empty result is undefined so RPC omits ORDER BY. Callers must not treat this as a
 * store wipe — collection orderBy can remain in queryState across grouped views.
 */
export function filterOrderByForGroupKeys(
  orderBy: OrderByState[] | undefined | null,
  groupKeys: Array<string | null | undefined> | undefined | null,
): OrderByState[] | undefined {
  if (!orderBy?.length) return undefined;
  const keySet = new Set(
    (groupKeys || []).map(k => String(k ?? '').trim()).filter(k => k.length > 0),
  );
  if (!keySet.size) return undefined;
  const filtered = orderBy.filter(row => keySet.has(String(row?.field || '').trim()));
  return filtered.length ? filtered : undefined;
}

/**
 * Resolve List header sort from queryState (including explicit clear) or props.
 */
export function resolveTableOrderBy(
  queryState: Record<string, unknown> | null | undefined,
  propsOrderBy: unknown
): OrderByState[] | undefined {
  const raw =
    queryState && Object.prototype.hasOwnProperty.call(queryState, 'orderBy')
      ? queryState.orderBy
      : propsOrderBy;
  if (raw == null) return undefined;
  // Preserve explicit empty clear; otherwise normalize order/direction into OrderByState.
  if (Array.isArray(raw) && raw.length === 0) return [];
  return metadataOrderByToState(raw);
}

/**
 * Write orderBy into store.queryState (including explicit undefined clears).
 * Defined values are normalized to OrderByState[] so a single-object prop
 * (allowed by ChoyListView) is stored as a one-item array that applyImpl accepts.
 */
export function writeQueryStateOrderBy(
  state: { queryState?: Record<string, unknown> } | null | undefined,
  orderBy: unknown
): void {
  if (!state) return;
  let next: unknown = orderBy;
  if (orderBy !== undefined && orderBy !== null) {
    if (Array.isArray(orderBy) && orderBy.length === 0) {
      next = [];
    } else {
      next = metadataOrderByToState(orderBy) ?? [];
    }
  }
  state.queryState = {
    ...(state.queryState || {}),
    orderBy: next,
  };
}

/**
 * Mount-time sync: only copy a defined controlled props.orderBy into queryState.
 */
export function syncControlledOrderByToQueryState(
  state: { queryState?: Record<string, unknown> } | null | undefined,
  orderBy: unknown
): void {
  if (orderBy === undefined) return;
  writeQueryStateOrderBy(state, orderBy);
}

/**
 * Handle-column reorder is allowed only when unsorted, or sorted solely by
 * handleField ascending (matches renumber semantics).
 */
export function isHandleReorderSortAllowed(
  orderByList: Array<{ field?: string; direction?: string; order?: string }> | undefined | null,
  handleField: string
): boolean {
  const list = orderByList || [];
  if (!list.length) return true;
  if (list.length > 1) return false;
  const primary = list[0];
  const field = primary?.field;
  // Match toRpcOrderBy: prefer `order` over `direction`, then normalize.
  const dir = normalizeSortDir(primary?.order ?? primary?.direction ?? 'asc');
  return field === handleField && dir === 'asc';
}
