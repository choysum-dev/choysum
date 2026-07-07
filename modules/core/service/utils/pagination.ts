// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { normalizeLimit, normalizeOffset } from './normalization';

export type PaginationParams = {
  limit?: number;
  offset?: number;
};

export type NormalizedPagination = {
  limit: number | undefined;
  offset: number;
};

/**
 * Normalize limit/offset from loose user input into safe numeric values.
 *
 * Delegates to {@link normalizeLimit} and {@link normalizeOffset} so that
 * string-encoded numbers (common in query parameters) are coerced consistently.
 */
export function normalizePagination(options?: PaginationParams): NormalizedPagination {
  const limit = normalizeLimit(options?.limit) ?? undefined;
  const offset = normalizeOffset(options?.offset);
  return { limit, offset };
}

export type PagedResponse<T, K extends string = string> = {
  [key in K]: T[];
} & {
  total: number;
  filtered: number;
  offset: number;
  limit?: number;
  returned: number;
};

/**
 * Slice a (pre-filtered) result set according to normalized pagination and wrap
 * it into the standard diagnostic response envelope.
 *
 * @param items     - the pre-filtered array (before pagination)
 * @param resultKey - the key under which the paged slice is placed
 * @param pagination - output from {@link normalizePagination}
 * @param total      - total count before any filtering; defaults to items.length
 * @param extra      - additional top-level fields merged into the response
 */
export function paginateAndWrap<T, K extends string>(
  items: T[],
  resultKey: K,
  pagination: NormalizedPagination,
  total?: number,
  extra?: Record<string, unknown>
): PagedResponse<T, K> {
  const { limit, offset } = pagination;
  const paged = typeof limit === 'number' ? items.slice(offset, offset + limit) : items.slice(offset);
  const effectiveTotal = typeof total === 'number' && Number.isFinite(total) && total >= 0 ? total : items.length;
  return {
    [resultKey]: paged,
    total: effectiveTotal,
    filtered: items.length,
    offset,
    limit,
    returned: paged.length,
    ...(extra || {}),
  } as PagedResponse<T, K>;
}
