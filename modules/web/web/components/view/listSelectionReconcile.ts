// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** Stable id for list selection / table rows (controller nodes or flat records). */
export function listSelectionRowKey(row: unknown): string | undefined {
  if (row == null || typeof row !== 'object') return undefined;
  const r = row as Record<string, any>;
  if (r.key != null) return String(r.key);
  if (r.__rowKey != null) return String(r.__rowKey);
  if (r.Id != null) return String(r.Id);
  if (r.type === 'record') return r.record?.Id != null ? String(r.record.Id) : undefined;
  if (r.kind === 'record') return r.payload?.Id != null ? String(r.payload.Id) : undefined;
  const nested = r.payload?.Id ?? r.record?.Id;
  return nested != null ? String(nested) : undefined;
}

/**
 * Replace selected rows with matching fresh rows by id; keep unmatched selection entries.
 */
export function reconcileListSelection<T>(selected: T[], freshRows: T[]): T[] {
  if (!selected.length || !freshRows.length) return selected;
  const byKey = new Map<string, T>();
  for (const row of freshRows) {
    const k = listSelectionRowKey(row);
    if (k != null) byKey.set(k, row);
  }
  return selected.map((sel) => {
    const k = listSelectionRowKey(sel);
    if (k != null && byKey.has(k)) return byKey.get(k)!;
    return sel;
  });
}
