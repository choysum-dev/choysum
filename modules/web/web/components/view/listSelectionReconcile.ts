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
 * Replace selected rows with matching fresh rows by id; drop entries absent from freshRows.
 */
export function reconcileListSelection<T>(selected: T[], freshRows: T[]): T[] {
  if (!selected.length) return selected;
  if (!freshRows.length) return [];
  const byKey = new Map<string, T>();
  for (const row of freshRows) {
    const k = listSelectionRowKey(row);
    if (k != null) byKey.set(k, row);
  }
  const out: T[] = [];
  for (const sel of selected) {
    const k = listSelectionRowKey(sel);
    if (k != null && byKey.has(k)) out.push(byKey.get(k)!);
  }
  return out;
}

/** Keep only selection keys that still exist among fresh rows (same key space as useTableSelection). */
export function reconcileListSelectionKeys(
  selectedKeys: Iterable<string | number>,
  freshRows: unknown[],
  keyOf: (row: unknown) => string | number | undefined | null,
): Set<string | number> {
  const fresh = new Set<string | number>();
  for (const row of freshRows) {
    const k = keyOf(row);
    if (k != null) fresh.add(k);
  }
  const next = new Set<string | number>();
  for (const k of selectedKeys) {
    if (fresh.has(k) || fresh.has(String(k))) next.add(k);
  }
  return next;
}
