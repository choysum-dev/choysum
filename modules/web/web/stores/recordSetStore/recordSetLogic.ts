// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { RecordSetEntry, SearchRecordSetIds } from './types';

/** Default prefetch window when resolving neighbors outside the current ids slice. */
export const RECORD_SET_NEIGHBOR_WINDOW = 40;

export function clampRecordSetIndex(index: number, total: number): number {
  if (!Number.isFinite(total) || total <= 0) return 0;
  if (!Number.isFinite(index)) return 0;
  return Math.max(0, Math.min(Math.trunc(index), Math.trunc(total) - 1));
}

export function idAtGlobalIndex(entry: RecordSetEntry, globalIndex: number): string | undefined {
  const local = globalIndex - entry.windowStart;
  if (local < 0 || local >= entry.ids.length) return undefined;
  const id = String(entry.ids[local] ?? '').trim();
  return id || undefined;
}

export function canMoveRecordSet(entry: RecordSetEntry | null | undefined, delta: -1 | 1): boolean {
  if (!entry || entry.total <= 0) return false;
  const next = entry.index + delta;
  return next >= 0 && next < entry.total;
}

async function fetchWindowAt(
  entry: RecordSetEntry,
  fetchOffset: number,
  searchIds: SearchRecordSetIds,
  windowSize: number
): Promise<{ ids: string[]; windowStart: number } | null> {
  const limit = Math.max(1, Math.trunc(windowSize) || RECORD_SET_NEIGHBOR_WINDOW);
  let ids: string[];
  try {
    ids = await searchIds({
      condition: entry.condition,
      orderBy: entry.orderBy,
      offset: fetchOffset,
      limit,
    });
  } catch {
    return null;
  }
  const cleaned = (Array.isArray(ids) ? ids : [])
    .map(id => String(id ?? '').trim())
    .filter(Boolean);
  if (!cleaned.length) return null;
  return { ids: cleaned, windowStart: fetchOffset };
}

/**
 * Resolve neighbor id within the sliding window, or via Search when out of window.
 * Returns an updated entry (new index / window) and the target id, or null when blocked.
 */
export async function resolveRecordSetNeighbor(
  entry: RecordSetEntry,
  delta: -1 | 1,
  searchIds?: SearchRecordSetIds,
  windowSize: number = RECORD_SET_NEIGHBOR_WINDOW
): Promise<{ entry: RecordSetEntry; id: string } | null> {
  if (!canMoveRecordSet(entry, delta)) return null;
  return resolveRecordSetAtIndex(entry, entry.index + delta, searchIds, windowSize);
}

/**
 * Jump to an absolute 0-based index within the record set (clamped).
 * Returns null when the target matches the current index or cannot be resolved.
 */
export async function resolveRecordSetAtIndex(
  entry: RecordSetEntry,
  targetIndex: number,
  searchIds?: SearchRecordSetIds,
  windowSize: number = RECORD_SET_NEIGHBOR_WINDOW
): Promise<{ entry: RecordSetEntry; id: string } | null> {
  if (!entry || entry.total <= 0) return null;
  const nextIndex = clampRecordSetIndex(targetIndex, entry.total);
  if (nextIndex === entry.index) return null;

  const inWindow = idAtGlobalIndex(entry, nextIndex);
  if (inWindow) {
    return { entry: { ...entry, index: nextIndex }, id: inWindow };
  }
  if (!searchIds) return null;

  // Prefetch forward from the target index so subsequent next clicks stay in-window.
  const fetched = await fetchWindowAt(entry, nextIndex, searchIds, windowSize);
  if (!fetched) return null;
  const id = fetched.ids[0];
  if (!id) return null;

  return {
    entry: {
      ...entry,
      index: nextIndex,
      ids: fetched.ids,
      windowStart: fetched.windowStart,
    },
    id,
  };
}
