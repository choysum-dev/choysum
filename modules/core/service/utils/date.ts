// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Parse an optional ISO-8601 date-time string into a Date.
 *
 * Returns the current Date when input is empty or invalid.
 */
export function parseISODate(iso?: string): Date {
  if (!iso) return new Date();
  const parsed = new Date(iso);
  if (Number.isNaN(parsed.getTime())) return new Date();
  return parsed;
}

/**
 * Coerce a loose value into a Date, or undefined on failure.
 *
 * - Date instances are returned directly (NaN → undefined).
 * - Everything else is coerced via `new Date(String(value))`.
 */
export function toDate(value: unknown): Date | undefined {
  if (value instanceof Date) return Number.isNaN(value.getTime()) ? undefined : value;
  if (typeof value === 'number') {
    const parsed = new Date(value);
    return Number.isNaN(parsed.getTime()) ? undefined : parsed;
  }
  if (typeof value === 'string' && /^\d+$/.test(value) && value.length > 4) {
    const parsed = new Date(Number(value));
    return Number.isNaN(parsed.getTime()) ? undefined : parsed;
  }
  const parsed = new Date(String(value ?? ''));
  return Number.isNaN(parsed.getTime()) ? undefined : parsed;
}
