// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Default List-column `sortable` by field meta type.
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
