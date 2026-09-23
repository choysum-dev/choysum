// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { CalendarDate, parseDate } from '@internationalized/date';

/** Formats a CalendarDate as YYYY-MM-DD. */
export function formatDatePickerValue(date: CalendarDate): string {
  const month = String(date.month).padStart(2, '0');
  const day = String(date.day).padStart(2, '0');
  return `${date.year}-${month}-${day}`;
}

/**
 * Parses a YYYY-MM-DD string (or blank) into a CalendarDate.
 * Returns null for empty / whitespace / invalid input.
 */
export function parseDatePickerValue(value: string | null | undefined): CalendarDate | null {
  const text = String(value ?? '').trim();
  if (!text) {
    return null;
  }
  try {
    return parseDate(text);
  } catch {
    return null;
  }
}

/** Today in the local calendar (Gregorian). */
export function todayDatePickerValue(date: Date = new Date()): CalendarDate {
  return new CalendarDate(date.getFullYear(), date.getMonth() + 1, date.getDate());
}

/** Clears the picker model (null). */
export function clearDatePickerValue(): null {
  return null;
}
