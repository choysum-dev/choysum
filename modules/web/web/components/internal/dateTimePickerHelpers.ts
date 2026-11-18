// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { CalendarDate, parseDate } from '@internationalized/date';

import { dayjsUtc } from '@/web/web/utils/dayjsUtc';
import {
  displayFormatNeedsZoneOffset,
  formatUtcInTimeZone,
  getUserTimeZone,
  userWallDateToUtc,
} from '@/web/web/utils/datetime';
import {
  formatDatePickerValue,
  parseDatePickerValue,
  todayDatePickerValue,
  type DatePickerDateParts,
} from './datePickerHelpers';

/** Local wall datetime with seconds: YYYY-MM-DDTHH:mm:ss */
export type DateTimePickerParts = DatePickerDateParts & {
  hour: number;
  minute: number;
  second: number;
};

const DATETIME_RE = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})(?::(\d{2}))?$/;

/** Formats local wall parts as YYYY-MM-DDTHH:mm:ss. */
export function formatDateTimePickerValue(parts: DateTimePickerParts): string {
  const date = formatDatePickerValue(parts);
  if (!date) return '';
  const { hour, minute, second } = parts;
  if (!Number.isInteger(hour) || hour < 0 || hour > 23) return '';
  if (!Number.isInteger(minute) || minute < 0 || minute > 59) return '';
  if (!Number.isInteger(second) || second < 0 || second > 59) return '';
  return `${date}T${String(hour).padStart(2, '0')}:${String(minute).padStart(2, '0')}:${String(second).padStart(2, '0')}`;
}

/**
 * Parses YYYY-MM-DDTHH:mm:ss or YYYY-MM-DDTHH:mm (seconds default 0).
 * Returns null for empty / whitespace / invalid input.
 */
export function parseDateTimePickerValue(
  value: string | null | undefined,
  parse: (text: string) => CalendarDate = parseDate,
): DateTimePickerParts | null {
  const text = String(value ?? '').trim();
  if (!text) return null;
  const match = text.match(DATETIME_RE);
  if (!match) return null;
  const date = parseDatePickerValue(
    `${match[1]}-${match[2]}-${match[3]}`,
    parse,
  );
  if (!date) return null;
  const hour = Number(match[4]);
  const minute = Number(match[5]);
  const second = match[6] != null ? Number(match[6]) : 0;
  if (!Number.isInteger(hour) || hour < 0 || hour > 23) return null;
  if (!Number.isInteger(minute) || minute < 0 || minute > 59) return null;
  if (!Number.isInteger(second) || second < 0 || second > 59) return null;
  return {
    year: date.year,
    month: date.month,
    day: date.day,
    hour,
    minute,
    second,
  };
}

/**
 * Display text for the trigger.
 * Default: space instead of T (YYYY-MM-DD HH:mm:ss). Optional dayjs displayFormat overrides.
 * When the format includes offset tokens (Z/z), wall parts are interpreted in `timeZone`
 * (user TZ) so the offset matches query preview — not browser/UTC.
 */
export function formatDateTimePickerDisplay(
  value: string | null | undefined,
  displayFormat?: string | null,
  timeZone?: string | null,
): string {
  const parts = parseDateTimePickerValue(value);
  if (!parts) return String(value ?? '').trim();
  const fmt = String(displayFormat || '').trim();
  const date = formatDatePickerValue(parts);
  const canonical = `${date} ${String(parts.hour).padStart(2, '0')}:${String(parts.minute).padStart(2, '0')}:${String(parts.second).padStart(2, '0')}`;
  if (!fmt) return canonical;

  if (displayFormatNeedsZoneOffset(fmt)) {
    const tz = String(timeZone || '').trim() || getUserTimeZone();
    // Wall-carrier Date: local getters hold user-TZ components (same contract as toView).
    // Use setFullYear — `new Date(year<100, …)` remaps to 19xx.
    const wallLocal = new Date(0);
    wallLocal.setFullYear(parts.year, parts.month - 1, parts.day);
    wallLocal.setHours(parts.hour, parts.minute, parts.second, 0);
    const utc = userWallDateToUtc(wallLocal, tz);
    if (utc) {
      const wall = formatUtcInTimeZone(utc, fmt, tz);
      if (wall) return wall;
    }
    return canonical;
  }

  // No offset token: format validated parts in UTC so year 0001–0099 and DST gaps stay literal.
  const instant = new Date(0);
  instant.setUTCFullYear(parts.year, parts.month - 1, parts.day);
  instant.setUTCHours(parts.hour, parts.minute, parts.second, 0);
  const m = dayjsUtc.utc(instant);
  return m.isValid() ? m.format(fmt) : canonical;
}

/** Now in the local calendar + clock (seconds precision). */
export function nowDateTimePickerValue(date: Date = new Date()): string {
  return formatDateTimePickerValue({
    year: date.getFullYear(),
    month: date.getMonth() + 1,
    day: date.getDate(),
    hour: date.getHours(),
    minute: date.getMinutes(),
    second: date.getSeconds(),
  });
}

/** Merge a calendar day into an existing datetime string (keep time, default 00:00:00). */
export function withDateTimePickerDate(
  current: string | null | undefined,
  date: DatePickerDateParts,
): string {
  const prev = parseDateTimePickerValue(current);
  return formatDateTimePickerValue({
    year: date.year,
    month: date.month,
    day: date.day,
    hour: prev?.hour ?? 0,
    minute: prev?.minute ?? 0,
    second: prev?.second ?? 0,
  });
}

/** Replace hour / minute / second on an existing value (date defaults to today). */
export function withDateTimePickerTime(
  current: string | null | undefined,
  time: { hour?: number; minute?: number; second?: number },
  today: Date = new Date(),
): string {
  const prev = parseDateTimePickerValue(current);
  const base = prev ?? {
    year: today.getFullYear(),
    month: today.getMonth() + 1,
    day: today.getDate(),
    hour: 0,
    minute: 0,
    second: 0,
  };
  return formatDateTimePickerValue({
    ...base,
    hour: time.hour ?? base.hour,
    minute: time.minute ?? base.minute,
    second: time.second ?? base.second,
  });
}

export function clearDateTimePickerValue(): null {
  return null;
}

export { todayDatePickerValue };
