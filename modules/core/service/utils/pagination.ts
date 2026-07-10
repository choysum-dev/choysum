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

const DEFAULT_BATCH_SIZE = 200;

/**
 * Generic batch processor for paginated Search loops.
 *
 * Replaces the repeated `for (;;)` + `Search` + `break` pattern found in
 * GC / retention / expiry loops.
 *
 * @param searchFn  Callback that performs a single page Search.
 * @param condition Search condition forwarded to each page.
 * @param processor Async callback invoked for every row in every page.
 * @param opts      Batch size and optional offset-based pagination flags.
 * @returns Total number of rows processed across all pages.
 */
export async function paginateBatch<T>(
  searchFn: (
    condition: unknown,
    opts: {
      limit: number;
      offset?: number;
      orderBy?: { field: string; order: 'asc' | 'desc' };
      fields?: string[];
    }
  ) => Promise<T[]>,
  condition: unknown,
  processor: (item: T) => Promise<void>,
  opts?: {
    batch?: number;
    offsetMode?: boolean;
    orderBy?: { field: string; order: 'asc' | 'desc' };
    fields?: string[];
  }
): Promise<number> {
  const batch = opts?.batch ?? DEFAULT_BATCH_SIZE;
  const offsetMode = opts?.offsetMode ?? false;
  let processed = 0;
  let offset = 0;

  for (;;) {
    const pageOpts: {
      limit: number;
      offset?: number;
      orderBy?: { field: string; order: 'asc' | 'desc' };
      fields?: string[];
    } = { limit: batch };

    if (offsetMode) {
      pageOpts.offset = offset;
      if (opts?.orderBy) pageOpts.orderBy = opts.orderBy;
    }
    if (opts?.fields) pageOpts.fields = opts.fields;

    const rows = await searchFn(condition, pageOpts);
    if (!rows.length) break;

    for (const row of rows) {
      await processor(row);
      processed += 1;
    }

    if (offsetMode) offset += rows.length;
    if (rows.length < batch) break;
  }

  return processed;
}
