// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { CalendarDate, parseDate } from '@internationalized/date';

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

/** Display text for the trigger (space instead of T). */
export function formatDateTimePickerDisplay(value: string | null | undefined): string {
  const parts = parseDateTimePickerValue(value);
  if (!parts) return String(value ?? '').trim();
  const date = formatDatePickerValue(parts);
  return `${date} ${String(parts.hour).padStart(2, '0')}:${String(parts.minute).padStart(2, '0')}:${String(parts.second).padStart(2, '0')}`;
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
