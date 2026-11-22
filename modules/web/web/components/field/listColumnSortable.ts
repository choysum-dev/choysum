// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Default List-column `sortable` by field meta type + storage.
 * Explicit `vColumnProps.sortable` / `:sortable` on the column still wins.
 */

const SORTABLE_LIST_FIELD_TYPES = new Set([
  'varchar',
  'char',
  'text',
  'int',
  'integer',
  'bigint',
  'float',
  'double',
  'number',
  'decimal',
  'monetary',
  'boolean',
  'date',
  'datetime',
  'time',
  'selection',
  'manytoone',
  'manytooneref',
]);

const NON_SORTABLE_LIST_FIELD_TYPES = new Set([
  'manytomany',
  'manytomanyref',
  'onetomany',
  'image',
  'binary',
  'html',
  'json',
  'properties',
  'propertiesdefinition',
]);

export type ListColumnSortableMeta = {
  shouldCreateColumn?: boolean;
  storageKind?: string;
  computedKind?: string;
};

function normalizeListFieldType(fieldType: unknown): string {
  return String(fieldType || '')
    .trim()
    .toLowerCase();
}

/**
 * Default list-column sortable flag for a field type.
 * Unknown types stay non-sortable (opt-in via vColumnProps).
 */
export function defaultListColumnSortableForFieldType(fieldType: unknown): boolean {
  const t = normalizeListFieldType(fieldType);
  if (!t) return false;
  if (NON_SORTABLE_LIST_FIELD_TYPES.has(t)) return false;
  return SORTABLE_LIST_FIELD_TYPES.has(t);
}

/**
 * True when ORDER BY can use a stored column or SQL-compute expression.
 * Missing storage meta keeps the type-only default (legacy / incomplete stores).
 */
export function isListColumnOrderableByStorage(meta?: ListColumnSortableMeta | null): boolean {
  if (!meta) return true;
  if (meta.shouldCreateColumn === true) return true;
  if (String(meta.computedKind || '').toLowerCase() === 'sql') return true;
  if (meta.shouldCreateColumn === false) return false;
  const sk = String(meta.storageKind || '').toLowerCase();
  if (sk === 'virtual' || sk === 'virtualsql') return false;
  return true;
}

/**
 * Default list-column sortable: type allowlist, then storage/sql-compute gate.
 */
export function defaultListColumnSortable(
  fieldType: unknown,
  meta?: ListColumnSortableMeta | null
): boolean {
  if (!defaultListColumnSortableForFieldType(fieldType)) return false;
  return isListColumnOrderableByStorage(meta);
}
