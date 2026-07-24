// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * FE datetime hub (D21 / D22): dayjs utc + timezone.
 * Wire values stay UTC; UI wall clocks use getUserTimeZone().
 * Do not use this hub for calendar `date` fields (no zone conversion).
 */

import dayjs, { type Dayjs, type ConfigType } from 'dayjs';
import utc from 'dayjs/plugin/utc';
import timezone from 'dayjs/plugin/timezone';
import customParseFormat from 'dayjs/plugin/customParseFormat';
import { detectBrowserTimezone, resolveRequestTimezone } from './request_timezone';

dayjs.extend(utc);
dayjs.extend(timezone);
dayjs.extend(customParseFormat);

/** dayjs + utc/timezone plugin surface (peer dayjs may lack ambient plugin typings). */
type TzDayjs = Dayjs & {
  tz: (timezone?: string, keepLocalTime?: boolean) => TzDayjs;
  utc: (keepLocalTime?: boolean) => TzDayjs;
};

type DayjsTzFactory = {
  (date?: ConfigType, format?: string, strict?: boolean): TzDayjs;
  utc: (date?: ConfigType, format?: string, strict?: boolean) => TzDayjs;
  tz: ((date: ConfigType, timezone: string) => TzDayjs) & ((date: ConfigType, format: string, timezone: string) => TzDayjs);
  extend: typeof dayjs.extend;
};

const tzDayjs = dayjs as unknown as DayjsTzFactory;

export type UserTimeZoneResolver = () => string | null | undefined;

let userTimeZoneResolver: UserTimeZoneResolver | undefined;

/**
 * Optional override for tests / app bootstrap (e.g. auth store).
 * When unset, getUserTimeZone falls back to browser → UTC.
 */
export function setUserTimeZoneResolver(resolver: UserTimeZoneResolver | undefined): void {
  userTimeZoneResolver = resolver;
}

/**
 * Client-side display TZ aligned with §5.1 (without company fallback):
 * User.Timezone → browser IANA → UTC.
 */
export function getUserTimeZone(): string {
  let fromUser = '';
  try {
    fromUser = String(userTimeZoneResolver?.() || '').trim();
  } catch {
    fromUser = '';
  }
  const resolved = resolveRequestTimezone(fromUser, detectBrowserTimezone());
  return resolved || 'UTC';
}

function asUtcDayjs(input: Date | string | number): TzDayjs {
  if (input instanceof Date) {
    return tzDayjs.utc(input.toISOString());
  }
  if (typeof input === 'number') {
    return tzDayjs.utc(input);
  }
  const raw = String(input).trim();
  const parsed = tzDayjs.utc(raw);
  if (parsed.isValid()) return parsed;
  return tzDayjs(raw).utc();
}

/**
 * Convert a UTC instant to a Date whose local Y/M/D/h/m/s match the wall clock in `tz`.
 * Intended for Element Plus datetime pickers (browser-local Date getters).
 */
export function utcToUserWallDate(input: Date | string | number | null | undefined, tz: string = getUserTimeZone()): Date | null {
  if (input == null || input === '') return null;
  const wall = asUtcDayjs(input).tz(tz);
  if (!wall.isValid()) return null;
  return new Date(wall.year(), wall.month(), wall.date(), wall.hour(), wall.minute(), wall.second(), wall.millisecond());
}

/**
 * Interpret a picker Date's local components as wall clock in `tz` and return a UTC Date.
 * Ambiguous DST-fold wall times (e.g. 01:30 on America/New_York fall-back) follow dayjs.tz
 * default occurrence selection — no earlier/later fold flag is available in the FE stack (D21).
 */
export function userWallDateToUtc(wallLocal: Date | null | undefined, tz: string = getUserTimeZone()): Date | null {
  if (wallLocal == null || Number.isNaN(wallLocal.getTime())) return null;
  const stamp = [
    String(wallLocal.getFullYear()).padStart(4, '0'),
    String(wallLocal.getMonth() + 1).padStart(2, '0'),
    String(wallLocal.getDate()).padStart(2, '0'),
  ].join('-');
  const time = [
    String(wallLocal.getHours()).padStart(2, '0'),
    String(wallLocal.getMinutes()).padStart(2, '0'),
    String(wallLocal.getSeconds()).padStart(2, '0'),
  ].join(':');
  const ms = String(wallLocal.getMilliseconds()).padStart(3, '0');
  const wall = tzDayjs.tz(`${stamp} ${time}.${ms}`, 'YYYY-MM-DD HH:mm:ss.SSS', tz);
  if (!wall.isValid()) return null;
  return wall.utc().toDate();
}

/**
 * Format a UTC instant as a wall-clock string in `tz`.
 */
export function formatUtcInTimeZone(
  input: Date | string | number | null | undefined,
  format: string,
  tz: string = getUserTimeZone()
): string {
  if (input == null || input === '') return '';
  const wall = asUtcDayjs(input).tz(tz);
  return wall.isValid() ? wall.format(format) : '';
}

/**
 * Format a UTC instant to storage ISO (UTC offset / Z) using dayjs format tokens.
 */
export function formatUtcIso(input: Date | string | number | null | undefined, format: string): string | null {
  if (input == null || input === '') return null;
  const m = asUtcDayjs(input);
  if (!m.isValid()) return null;
  let out = m.format(format);
  // dayjs formats the `Z` token as `+00:00`; keep a stable UTC wire with literal Z.
  if (out.endsWith('+00:00')) {
    out = `${out.slice(0, -6)}Z`;
  }
  return out;
}

/** Parse a UTC ISO-like string with an optional dayjs format (strict). */
export function parseUtc(input: string, format?: string, strict?: boolean): TzDayjs {
  if (format) return tzDayjs.utc(input, format, strict);
  return tzDayjs.utc(input);
}

function addCalendarDay(ymd: string): string {
  const [y, m, d] = ymd.split('-').map(Number);
  const next = new Date(Date.UTC(y, m - 1, d + 1));
  return next.toISOString().slice(0, 10);
}

/**
 * Half-open UTC range for calendar day `D` in `tz`: `[D 00:00, D+1 00:00)`.
 * Call before building Search conditions; DST-safe via dayjs timezone.
 */
export function dayRange(date: string | Date, tz: string = getUserTimeZone()): { start: Date; end: Date } {
  const zone = String(tz || '').trim() || 'UTC';
  let day: string;
  if (date instanceof Date) {
    if (Number.isNaN(date.getTime())) {
      throw new Error('Invalid date');
    }
    day = asUtcDayjs(date).tz(zone).format('YYYY-MM-DD');
  } else {
    day = String(date ?? '')
      .trim()
      .slice(0, 10);
    if (!/^\d{4}-\d{2}-\d{2}$/.test(day)) {
      throw new Error(`Invalid date: ${String(date ?? '')}`);
    }
  }
  // Keep wall midnight explicit; advance by calendar day (not +24h) for DST.
  const startLocal = tzDayjs.tz(`${day} 00:00:00`, zone) as TzDayjs;
  const endLocal = tzDayjs.tz(`${addCalendarDay(day)} 00:00:00`, zone) as TzDayjs;
  if (!startLocal.isValid() || !endLocal.isValid()) {
    throw new Error(`Invalid date: ${day}`);
  }
  return {
    start: startLocal.utc().toDate(),
    end: endLocal.utc().toDate(),
  };
}

export { tzDayjs as hubDayjs };
export type { TzDayjs };
