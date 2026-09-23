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
 * `parse` is injectable so tests can exercise the clamp-rejection path.
 */
export function parseDatePickerValue(
  value: string | null | undefined,
  parse: (text: string) => CalendarDate = parseDate,
): CalendarDate | null {
  const text = String(value ?? '').trim();
  if (!text) {
    return null;
  }
  const parts = text.match(/^(\d{4})-(\d{2})-(\d{2})$/);
  if (!parts) {
    return null;
  }
  const rawYear = Number(parts[1]);
  const rawMonth = Number(parts[2]);
  const rawDay = Number(parts[3]);
  try {
    const parsed = parse(text);
    // Some parsers clamp out-of-range days (2026-02-30 → 2026-02-28); reject those.
    if (parsed.year !== rawYear || parsed.month !== rawMonth || parsed.day !== rawDay) {
      return null;
    }
    return parsed;
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
