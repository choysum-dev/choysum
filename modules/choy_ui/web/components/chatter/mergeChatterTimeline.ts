// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type {
  ChatterFieldChangeRow,
  ChatterMessageRow,
  ChatterTimelineEntry,
} from './chatterTypes';

/**
 * Parses a wire timestamp into epoch ms, or null when unusable.
 */
export function parseChatterTimestamp(value: unknown): number | null {
  if (value == null || value === '') return null;
  if (value instanceof Date) {
    const ms = value.getTime();
    return Number.isNaN(ms) ? null : ms;
  }
  if (typeof value === 'number') {
    return Number.isFinite(value) ? value : null;
  }
  const raw = String(value).trim();
  if (!raw) return null;
  // Protobuf JSON serializes int64 timestamps as strings; Date.parse fails on them.
  if (/^\d+$/.test(raw)) {
    const asNumber = Number(raw);
    return Number.isFinite(asNumber) ? asNumber : null;
  }
  const parsed = Date.parse(raw);
  return Number.isNaN(parsed) ? null : parsed;
}

/**
 * Merges message and field-change rows into a sorted timeline.
 */
export function mergeChatterTimeline(
  messages: ChatterMessageRow[] | null | undefined,
  fieldChanges: ChatterFieldChangeRow[] | null | undefined,
): ChatterTimelineEntry[] {
  const entries: ChatterTimelineEntry[] = [];

  for (const row of messages || []) {
    const id = String(row?.Id || '').trim();
    const at = parseChatterTimestamp(row?.CreatedAt);
    if (!id || at == null) continue;
    entries.push({
      kind: 'message',
      id,
      at,
      type: String(row?.Type || 'comment').trim() || 'comment',
      body: String(row?.Body ?? ''),
      authorUid:
        row?.AuthorUid == null || String(row.AuthorUid).trim() === ''
          ? null
          : String(row.AuthorUid).trim(),
    });
  }

  for (const row of fieldChanges || []) {
    const id = String(row?.Id || '').trim();
    const at = parseChatterTimestamp(row?.At);
    if (!id || at == null) continue;
    entries.push({
      kind: 'fieldChange',
      id,
      at,
      field:
        row?.Field == null || String(row.Field).trim() === '' ? null : String(row.Field).trim(),
      changeKind: String(row?.Kind || '').trim() || 'field',
      oldValue: row?.OldValue == null ? null : String(row.OldValue),
      newValue: row?.NewValue == null ? null : String(row.NewValue),
      actorUid:
        row?.ActorUid == null || String(row.ActorUid).trim() === ''
          ? null
          : String(row.ActorUid).trim(),
    });
  }

  entries.sort(compareChatterTimelineEntries);
  return entries;
}

/**
 * Ascending by time; fieldChange before message on ties; then id.
 */
export function compareChatterTimelineEntries(
  left: ChatterTimelineEntry,
  right: ChatterTimelineEntry,
): number {
  if (left.at !== right.at) return left.at - right.at;
  if (left.kind !== right.kind) return left.kind === 'fieldChange' ? -1 : 1;
  // Code-unit order so tie-breaks are locale-independent.
  return left.id < right.id ? -1 : left.id > right.id ? 1 : 0;
}
