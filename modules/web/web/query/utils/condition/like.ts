// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Reports whether an operator belongs to the LIKE family.
 */
export function isLikeOperator(op?: string): boolean {
  if (!op) return false;
  const o = op.toLowerCase();
  return o === 'like' || o === 'not like' || o === 'contains';
}

/**
 * Wraps a raw value in SQL LIKE wildcards.
 */
export function toLikePattern(raw: any): string {
  const s = raw == null ? '' : String(raw);
  return `%${s}%`;
}

/**
 * Normalizes a value for LIKE-based operators.
 */
export function normalizeLikeValue(val: any): string {
  return toLikePattern(val);
}

/**
 * Converts a value into the preview string shown for an operator.
 */
export function valueToPreview(op: string, value: any): string {
  if (isLikeOperator(op)) return toLikePattern(value);
  if (Array.isArray(value)) return `(${value.map(v => String(v)).join(', ')})`;
  if (value instanceof Date) return value.toISOString();
  return String(value);
}
