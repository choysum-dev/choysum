// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Pure helpers for the L3 DataTable engine (sorting + row selection).
 * The Vue SFC owns TanStack Table / virtualizer wiring.
 */

export type DataTableRowId = string | number;

export type DataTableSortDir = 'asc' | 'desc';

export type DataTableSortState = {
  id: string;
  desc: boolean;
} | null;

/** Toggles column sort: none → asc → desc → none. */
export function nextDataTableSort(
  current: DataTableSortState,
  columnId: string,
): DataTableSortState {
  if (!current || current.id !== columnId) {
    return { id: columnId, desc: false };
  }
  if (!current.desc) {
    return { id: columnId, desc: true };
  }
  return null;
}

/** Lexicographic / numeric compare for a single accessor. */
export function compareDataTableValues(a: unknown, b: unknown): number {
  if (a == null && b == null) {
    return 0;
  }
  if (a == null) {
    return -1;
  }
  if (b == null) {
    return 1;
  }
  if (typeof a === 'number' && typeof b === 'number') {
    return a - b;
  }
  return String(a).localeCompare(String(b), undefined, { numeric: true, sensitivity: 'base' });
}

/** Stable sort of rows by a column accessor and direction. */
export function sortDataTableRows<T>(
  rows: readonly T[],
  accessor: (row: T) => unknown,
  dir: DataTableSortDir,
): T[] {
  const factor = dir === 'desc' ? -1 : 1;
  return [...rows].sort((left, right) => factor * compareDataTableValues(accessor(left), accessor(right)));
}

/** Toggle a row id in a selection set (returns a new Set). */
export function toggleDataTableSelection(
  selected: ReadonlySet<DataTableRowId>,
  id: DataTableRowId,
): Set<DataTableRowId> {
  const next = new Set(selected);
  if (next.has(id)) {
    next.delete(id);
  } else {
    next.add(id);
  }
  return next;
}

/** Select or clear every id in `ids`. */
export function setDataTableSelectionAll(
  ids: readonly DataTableRowId[],
  selectAll: boolean,
): Set<DataTableRowId> {
  return selectAll ? new Set(ids) : new Set();
}

/** Clamp virtual window start/end against a row count. */
export function clampDataTableVirtualWindow(
  start: number,
  end: number,
  rowCount: number,
): { start: number; end: number } {
  const safeCount = Math.max(0, Math.floor(rowCount));
  const safeStart = Math.min(Math.max(0, Math.floor(start)), safeCount);
  const safeEnd = Math.min(Math.max(safeStart, Math.floor(end)), safeCount);
  return { start: safeStart, end: safeEnd };
}
