// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getBackendEnvPositiveInt } from '@/core/service/runtime/env/backend_env';

const DEFAULT_GC_BATCH_SIZE = 200;

export function resolveGcBatchSize(): number {
  return getBackendEnvPositiveInt(['CHOYSUM_DOCUMENT_GC_BATCH_SIZE'], DEFAULT_GC_BATCH_SIZE);
}

/**
 * Generic batch processor for paginated Search loops.
 *
 * Replaces the repeated `for (;;)` + `Search` + `break` pattern found in
 * GC / retention / expiry loops across mutation_ledger, upload_session,
 * and attachment models.
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
  const batch = opts?.batch ?? resolveGcBatchSize();
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
