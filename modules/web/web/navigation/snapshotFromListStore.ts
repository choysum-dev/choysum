// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { filtersToQuery } from '@/web/web/query/utils/condition/builder';
import { combinePresentConditions } from '@/web/web/query/utils/condition/absent';
import type { RecordSetEntry } from '@/web/web/stores/recordSetStore';

type ListStoreLike = {
  modelName?: string;
  state?: {
    queryState?: {
      appliedFilters?: unknown[];
      keyword?: string;
      keywordFields?: string[];
      forcedCondition?: unknown;
      orderBy?: unknown;
      pagination?: { offset?: number; limit?: number };
    };
    result?: {
      rows?: unknown[];
      total?: number;
    };
  };
  metadata?: { fields?: Record<string, unknown> };
};

function unwrapRowRecord(row: unknown): Record<string, unknown> | null {
  if (!row || typeof row !== 'object') return null;
  const r = row as Record<string, unknown>;
  if (r.kind === 'group' || r.kind === 'more') return null;
  if (r.kind === 'record' && r.payload && typeof r.payload === 'object') {
    return r.payload as Record<string, unknown>;
  }
  if (r.type === 'record' && r.record && typeof r.record === 'object') {
    return r.record as Record<string, unknown>;
  }
  if (r.Id != null || r.id != null) return r;
  return null;
}

/** Collect record Ids from List/Kanban row wrappers (skips group/more nodes). */
export function collectRecordIdsFromRows(rows: unknown[] | undefined): string[] {
  const out: string[] = [];
  for (const row of rows || []) {
    const rec = unwrapRowRecord(row);
    if (!rec) continue;
    const id = String(rec.Id ?? rec.id ?? '').trim();
    if (id) out.push(id);
  }
  return out;
}

/** Build the same merged condition List apply uses for Search/Count. */
export function buildListMergedCondition(store: ListStoreLike): unknown {
  const qs = store.state?.queryState;
  const ui = filtersToQuery(
    (Array.isArray(qs?.appliedFilters) ? qs!.appliedFilters : []) as any,
    qs?.keyword,
    qs?.keywordFields,
    store.metadata?.fields as any
  );
  return combinePresentConditions(ui, qs?.forcedCondition);
}

export type SnapshotFromListStoreOpts = {
  model: string;
  key: string;
  /** 0-based index within the current page / visible collection. */
  rowIndex: number;
  /** Preferred record id (find in page to correct index). */
  recordId?: string;
  /**
   * Optional id window (e.g. grouped List visibleNodes / Kanban lane cards).
   * When set, replaces ids derived from store.result.rows and starts at window 0.
   */
  pageIds?: string[];
};

/**
 * Snapshot List/Kanban store query + current page ids into a Form pager entry.
 */
export function snapshotFromListStore(
  store: ListStoreLike | null | undefined,
  opts: SnapshotFromListStoreOpts
): RecordSetEntry | null {
  if (!store?.state) return null;
  const model = String(opts.model || store.modelName || '').trim();
  if (!model) return null;

  const qs = store.state.queryState;
  const cleanedPageIds = Array.isArray(opts.pageIds)
    ? opts.pageIds.map(id => String(id ?? '').trim()).filter(Boolean)
    : null;
  // Empty pageIds must not wipe Search offset / rows — treat as no override.
  const overrideIds = cleanedPageIds && cleanedPageIds.length > 0 ? cleanedPageIds : null;
  // Visible/lane windows are not Search page offsets — start at 0.
  const offset = overrideIds
    ? 0
    : Math.max(0, Math.trunc(Number(qs?.pagination?.offset) || 0));
  const ids = overrideIds ?? collectRecordIdsFromRows(store.state.result?.rows);
  const totalRaw = Number(store.state.result?.total);
  const pageSpan = ids.length ? offset + ids.length : offset;
  // Prefer Search total, but never under-count the known page span before clamping index.
  const total =
    Number.isFinite(totalRaw) && totalRaw >= 0
      ? Math.max(Math.trunc(totalRaw), pageSpan)
      : Math.max(pageSpan, ids.length);

  let index = offset + Math.max(0, Math.trunc(Number(opts.rowIndex) || 0));
  const wantId = String(opts.recordId ?? '').trim();
  if (wantId) {
    const local = ids.indexOf(wantId);
    if (local >= 0) index = offset + local;
  }
  if (total > 0) {
    index = Math.max(0, Math.min(index, total - 1));
  } else {
    index = 0;
  }

  return {
    key: String(opts.key || `${model}|collection`),
    model,
    condition: buildListMergedCondition(store),
    orderBy: qs?.orderBy,
    total,
    index,
    ids,
    windowStart: offset,
  };
}
