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
  const toIndex = (value: number): number => (Number.isFinite(value) ? Math.floor(value) : 0);
  const safeCount = Math.max(0, toIndex(rowCount));
  const safeStart = Math.min(Math.max(0, toIndex(start)), safeCount);
  const safeEnd = Math.min(Math.max(safeStart, toIndex(end)), safeCount);
  return { start: safeStart, end: safeEnd };
}

/**
 * TanStack row keys are strings; encode the original id type so `42` and `'42'`
 * stay distinct in selection state.
 */
export function encodeDataTableRowKey(id: DataTableRowId): string {
  return typeof id === 'number' ? `n:${id}` : `s:${id}`;
}

/** Decodes an internal TanStack key produced by encodeDataTableRowKey. */
export function decodeDataTableRowKey(key: string): DataTableRowId {
  if (key.startsWith('n:')) {
    const n = Number(key.slice(2));
    if (Number.isFinite(n)) {
      return n;
    }
  }
  if (key.startsWith('s:')) {
    return key.slice(2);
  }
  return key;
}

function isUsableDataTableRowId(value: unknown): value is DataTableRowId {
  if (typeof value === 'string') {
    return value.trim() !== '';
  }
  return typeof value === 'number' && Number.isFinite(value);
}

/**
 * Resolves a stable row id for TanStack.
 * Prefers an explicit rowId fn, then `Id`, then `id`. Throws when none are usable
 * (empty-string keys would collide across rows).
 */
export function resolveDataTableRowId<T extends Record<string, unknown>>(
  row: T,
  rowId?: (row: T) => DataTableRowId,
): DataTableRowId {
  if (rowId) {
    const value = rowId(row);
    if (!isUsableDataTableRowId(value)) {
      throw new Error('DataTable rowId() returned an empty id');
    }
    return value;
  }
  const raw = [(row as { Id?: unknown }).Id, (row as { id?: unknown }).id].find(isUsableDataTableRowId);
  if (raw === undefined) {
    throw new Error('DataTable: provide rowId when rows lack Id/id');
  }
  return raw;
}

/** Maps TanStack string selection keys back to original DataTableRowId values. */
export function mapDataTableSelectionKeys(
  keys: readonly string[],
  registry: ReadonlyMap<string, DataTableRowId>,
): DataTableRowId[] {
  return keys.map((key) => {
    const known = registry.get(key);
    if (known !== undefined) {
      return known;
    }
    // Decode the internal key so `n:`/`s:` never leak into public selection ids.
    return decodeDataTableRowKey(key);
  });
}
