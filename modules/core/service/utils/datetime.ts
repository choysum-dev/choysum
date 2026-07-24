// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import moment from 'moment-timezone';
import { getContextCompanyTimezone } from '../runtime/context/scope';

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

let cachedIanaTimezoneSelection: Array<{ value: string; label: string }> | null = null;

/**
 * Lists IANA timezone identifiers as FieldsGet / selection items.
 *
 * Labels are bare technical codes (value === label); do not wrap in `_lt`.
 * The mapped list is cached for the process lifetime (moment-timezone names are static).
 */
export function listIanaTimezoneSelection(): Array<{ value: string; label: string }> {
  if (!cachedIanaTimezoneSelection) {
    cachedIanaTimezoneSelection = Object.freeze(
      moment.tz.names().map(name => Object.freeze({ value: name, label: name }))
    ) as Array<{ value: string; label: string }>;
  }
  return cachedIanaTimezoneSelection;
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

// ---------------------------------------------------------------------------
// Day boundaries / business calendar (moment-timezone)
// ---------------------------------------------------------------------------

function requireIanaTimezone(tz: string | undefined, label = 'timezone'): string {
  const normalized = String(tz ?? '').trim();
  if (!isIanaTimezone(normalized)) {
    throw new Error(`Invalid IANA ${label}: ${String(tz ?? '')}`);
  }
  return normalized;
}

function resolveCalendarDay(date: string | Date, tz: string): string {
  if (date instanceof Date) {
    if (Number.isNaN(date.getTime())) {
      throw new Error('Invalid date');
    }
    return moment.tz(date, tz).format('YYYY-MM-DD');
  }
  const day = String(date ?? '')
    .trim()
    .slice(0, 10);
  if (!/^\d{4}-\d{2}-\d{2}$/.test(day) || !moment(day, 'YYYY-MM-DD', true).isValid()) {
    throw new Error(`Invalid date: ${String(date ?? '')}`);
  }
  return day;
}

/**
 * Half-open UTC range for calendar day `D` in `tz`: `[D 00:00, D+1 00:00)`.
 *
 * DST-safe via moment-timezone (23h / 25h days). Does not rewrite Search — call before filtering.
 */
export function dayRange(date: string | Date, tz: string): { start: Date; end: Date } {
  const zone = requireIanaTimezone(tz);
  const day = resolveCalendarDay(date, zone);
  const startLocal = moment.tz(`${day} 00:00:00`, 'YYYY-MM-DD HH:mm:ss', true, zone);
  if (!startLocal.isValid()) {
    throw new Error(`Invalid date: ${day}`);
  }
  const endLocal = startLocal.clone().add(1, 'day');
  return {
    start: startLocal.clone().utc().toDate(),
    end: endLocal.clone().utc().toDate(),
  };
}

function resolveBusinessTimezone(tz?: string): string {
  if (tz !== undefined && tz !== null && String(tz).trim() !== '') {
    return requireIanaTimezone(tz);
  }
  const companyTz = getContextCompanyTimezone();
  if (companyTz) {
    return requireIanaTimezone(companyTz, 'company timezone');
  }
  return 'UTC';
}

/**
 * Company (or explicit) calendar day for `now` as `YYYY-MM-DD`.
 * Defaults to `getContextCompanyTimezone()`, then `UTC`.
 */
export function businessToday(tz?: string, now: Date = new Date()): string {
  const zone = resolveBusinessTimezone(tz);
  return moment.tz(now, zone).format('YYYY-MM-DD');
}

/**
 * Previous company (or explicit) calendar day for `now` as `YYYY-MM-DD`.
 */
export function businessYesterday(tz?: string, now: Date = new Date()): string {
  const zone = resolveBusinessTimezone(tz);
  return moment.tz(now, zone).subtract(1, 'day').format('YYYY-MM-DD');
}

const WALL_CLOCK_FORMATS = ['YYYY-MM-DD HH:mm:ss', 'YYYY-MM-DDTHH:mm:ss', 'YYYY-MM-DD HH:mm', 'YYYY-MM-DDTHH:mm', 'YYYY-MM-DD'];

/**
 * Convert offset-free wall-clock start/end in `tz` to UTC `Date` bounds for Search filters.
 *
 * Prefer half-open `[start, end)` at the call site (e.g. next-day 00:00 as `endWall`).
 * Does not mutate Search — convert before building conditions.
 */
export function wallClockRangeToUtc(startWall: string, endWall: string, tz: string): { start: Date; end: Date } {
  const zone = requireIanaTimezone(tz);
  const startLocal = moment.tz(String(startWall ?? '').trim(), WALL_CLOCK_FORMATS, true, zone);
  const endLocal = moment.tz(String(endWall ?? '').trim(), WALL_CLOCK_FORMATS, true, zone);
  if (!startLocal.isValid() || !endLocal.isValid()) {
    throw new Error(`Invalid wall-clock range: ${String(startWall)} .. ${String(endWall)}`);
  }
  return {
    start: startLocal.clone().utc().toDate(),
    end: endLocal.clone().utc().toDate(),
  };
}
