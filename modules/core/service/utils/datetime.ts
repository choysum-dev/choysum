// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import moment from 'moment-timezone';

// ---------------------------------------------------------------------------
// Date helpers
// ---------------------------------------------------------------------------

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
  if (typeof value === 'string' && /^\d+$/.test(value) && value.length >= 10) {
    const parsed = new Date(Number(value));
    return Number.isNaN(parsed.getTime()) ? undefined : parsed;
  }
  const parsed = new Date(String(value ?? ''));
  return Number.isNaN(parsed.getTime()) ? undefined : parsed;
}

// ---------------------------------------------------------------------------
// Timezone helpers
// ---------------------------------------------------------------------------

/**
 * Reports whether a timezone string is a valid IANA timezone.
 */
export function isIanaTimezone(tz?: string): boolean {
  if (!tz) return false;
  return Boolean(moment.tz.zone(tz));
}

/**
 * Lists IANA timezone identifiers as FieldsGet / selection items.
 *
 * Labels are bare technical codes (value === label); do not wrap in `_lt`.
 */
export function listIanaTimezoneSelection(): Array<{ value: string; label: string }> {
  return moment.tz.names().map(name => ({ value: name, label: name }));
}

/**
 * Parses a fixed UTC offset string into minutes.
 *
 * Accepts forms like `+08:00`, `+8`, `-05:00`, `UTC`, `GMT`, `Z`.
 * Returns undefined for unrecognised or out-of-range inputs.
 */
export function parseTimezoneOffsetMinutes(tz?: string): number | undefined {
  if (!tz) return undefined;
  const raw = tz.trim();
  if (!raw) return undefined;
  if (raw === 'UTC' || raw === 'GMT' || raw === 'Z') return 0;

  const match = raw.match(/^([+-])(\d{1,2})(?::?(\d{2}))?$/);
  if (!match) return undefined;
  const sign = match[1] === '-' ? -1 : 1;
  const hours = Number(match[2]);
  const minutes = Number(match[3] ?? '0');
  if (Number.isNaN(hours) || Number.isNaN(minutes)) return undefined;
  if (hours > 14 || minutes >= 60) return undefined;
  return sign * (hours * 60 + minutes);
}
