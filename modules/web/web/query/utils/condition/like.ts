// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { formatUtcInTimeZone, getUserTimeZone } from '@/web/web/utils/datetime';

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

function looksLikeUtcDatetime(value: unknown): boolean {
  if (value instanceof Date) return !Number.isNaN(value.getTime());
  if (typeof value !== 'string') return false;
  const raw = value.trim();
  // ISO instants (with time); calendar dates stay literal.
  return /^\d{4}-\d{2}-\d{2}T/.test(raw);
}

export type ValuePreviewOptions = {
  /** Field metadata type (`datetime` | `date` | …). */
  fieldType?: string;
  /** Override display timezone (defaults to getUserTimeZone()). */
  timeZone?: string;
  /** Display format for datetime preview. */
  displayFormat?: string;
};

/**
 * Converts a value into the preview string shown for an operator.
 * Datetime wire values (UTC ISO / Date) render as user wall-clock; date literals stay unchanged.
 */
export function valueToPreview(op: string, value: any, options?: ValuePreviewOptions): string {
  if (isLikeOperator(op)) return toLikePattern(value);
  if (Array.isArray(value)) {
    return `(${value.map(v => valueToPreview('=', v, options)).join(', ')})`;
  }

  const fieldType = String(options?.fieldType || '').toLowerCase();
  if (fieldType === 'date' || fieldType === 'time') {
    if (value instanceof Date) {
      return fieldType === 'date' ? value.toISOString().slice(0, 10) : String(value);
    }
    return String(value ?? '');
  }

  if (fieldType === 'datetime' || (!fieldType && looksLikeUtcDatetime(value))) {
    const tz = options?.timeZone || getUserTimeZone();
    const format = options?.displayFormat || 'YYYY-MM-DD HH:mm:ss';
    const wall = formatUtcInTimeZone(value, format, tz);
    if (wall) return wall;
  }

  if (value instanceof Date) return value.toISOString();
  return String(value ?? '');
}
