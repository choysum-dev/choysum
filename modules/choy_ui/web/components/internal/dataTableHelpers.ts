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
  // Coerce Dates to epoch millis so mixed Date/number columns compare numerically;
  // an invalid Date yields NaN and keeps the NaN ordering below.
  const left = a instanceof Date ? a.getTime() : a;
  const right = b instanceof Date ? b.getTime() : b;
  if (typeof left === 'number' && typeof right === 'number') {
    if (Number.isNaN(left) || Number.isNaN(right)) {
      return Number.isNaN(left) === Number.isNaN(right) ? 0 : Number.isNaN(left) ? 1 : -1;
    }
    return left - right;
  }
  // localeCompare (not Intl.Collator): QuickJS FE unit runtime has no constructible Collator.
  return String(left).localeCompare(String(right), undefined, { numeric: true, sensitivity: 'base' });
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
 * stay distinct in selection state. String ids are trimmed so padded controlled
 * ids match `resolveDataTableRowId` keys.
 */
export function normalizeDataTableRowId(id: DataTableRowId): DataTableRowId | null {
  if (typeof id === 'string') {
    const trimmed = id.trim();
    return trimmed === '' ? null : trimmed;
  }
  return Number.isFinite(id) ? id : null;
}

export function encodeDataTableRowKey(id: DataTableRowId): string {
  const normalized = normalizeDataTableRowId(id);
  if (normalized === null) {
    // Non-finite numbers cannot round-trip through `n:` (decode would not yield a number).
    return `s:${String(id).trim()}`;
  }
  return typeof normalized === 'number' ? `n:${normalized}` : `s:${normalized}`;
}

/** Decodes an internal TanStack key produced by encodeDataTableRowKey. */
export function decodeDataTableRowKey(key: string): DataTableRowId {
  if (key.startsWith('n:')) {
    const raw = key.slice(2).trim();
    // `Number('')` is 0, which would silently decode a malformed key to row 0.
    const n = raw === '' ? Number.NaN : Number(raw);
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
    return typeof value === 'string' ? value.trim() : value;
  }
  const raw = [(row as { Id?: unknown }).Id, (row as { id?: unknown }).id].find(isUsableDataTableRowId);
  if (raw === undefined) {
    throw new Error('DataTable: provide rowId when rows lack Id/id');
  }
  return typeof raw === 'string' ? raw.trim() : raw;
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

/**
 * Keeps only selected keys that are still present after a data swap.
 * Returns null when the selection is already clean (no write needed).
 */
export function pruneDataTableSelection(
  selection: Readonly<Record<string, boolean>>,
  presentKeys: ReadonlySet<string>,
): Record<string, boolean> | null {
  const pruned: Record<string, boolean> = {};
  for (const [key, isSelected] of Object.entries(selection)) {
    if (isSelected && presentKeys.has(key)) {
      pruned[key] = true;
    }
  }
  const prevKeys = Object.keys(selection);
  const nextKeys = Object.keys(pruned);
  const unchanged =
    prevKeys.length === nextKeys.length && nextKeys.every((key) => selection[key] === true);
  // All-false TanStack leftovers have nothing selected — skip a no-op `{}` write.
  const hadSelected = prevKeys.some((key) => selection[key] === true);
  if (unchanged || !hadSelected) {
    return null;
  }
  return pruned;
}

/** Set-equality of selection ids using typed TanStack keys (dedupes duplicates). */
export function dataTableSelectionIdsEqual(
  left: readonly DataTableRowId[],
  right: readonly DataTableRowId[],
): boolean {
  const leftKeys = new Set(left.map(encodeDataTableRowKey));
  const rightKeys = new Set(right.map(encodeDataTableRowKey));
  if (leftKeys.size !== rightKeys.size) {
    return false;
  }
  for (const key of leftKeys) {
    if (!rightKeys.has(key)) {
      return false;
    }
  }
  return true;
}

/**
 * Merges controlled ids that are off the current page with the visible selection.
 * Preserves parent-owned ids across data swaps while reflecting on-page toggles.
 */
export function mergeDataTableControlledSelection(
  controlled: readonly DataTableRowId[],
  visibleSelected: readonly DataTableRowId[],
  presentKeys: ReadonlySet<string>,
): DataTableRowId[] {
  const kept: DataTableRowId[] = [];
  const seen = new Set<string>();
  for (const id of controlled) {
    // Blank / non-finite ids can never match a page key; keeping them would leak ghosts forever.
    if (normalizeDataTableRowId(id) === null) {
      continue;
    }
    const key = encodeDataTableRowKey(id);
    if (presentKeys.has(key) || seen.has(key)) {
      continue;
    }
    seen.add(key);
    kept.push(id);
  }
  const out = [...kept];
  for (const id of visibleSelected) {
    const key = encodeDataTableRowKey(id);
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    out.push(id);
  }
  return out;
}
