// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { CalendarDate } from '@internationalized/date';
import {
  clearDatePickerValue,
  formatDatePickerValue,
  parseDatePickerValue,
  todayDatePickerValue,
} from './datePickerHelpers';

describe('datePickerHelpers', () => {
  test('formats and parses YYYY-MM-DD', () => {
    const date = new CalendarDate(2026, 9, 23);
    expect(formatDatePickerValue(date)).toBe('2026-09-23');
    expect(parseDatePickerValue('2026-09-23')?.toString()).toBe(date.toString());
    // Years below 1000 must round-trip through the four-digit parse regex.
    expect(formatDatePickerValue(new CalendarDate(999, 1, 5))).toBe('0999-01-05');
    expect(parseDatePickerValue('0999-01-05')?.toString()).toBe('0999-01-05');
    // Years outside 1–9999 are not format/parse round-trippable.
    expect(formatDatePickerValue({ year: 10000, month: 1, day: 1 })).toBe('');
    expect(formatDatePickerValue({ year: 0, month: 1, day: 1 })).toBe('');
    // Out-of-range month/day must not emit strings the parser rejects.
    expect(formatDatePickerValue({ year: 2026, month: 13, day: 1 })).toBe('');
    expect(formatDatePickerValue({ year: 2026, month: 1, day: 40 })).toBe('');
    expect(formatDatePickerValue({ year: 2026, month: 2, day: 30 })).toBe('');
  });

  test('treats blank and invalid as null', () => {
    expect(parseDatePickerValue('')).toBeNull();
    expect(parseDatePickerValue('   ')).toBeNull();
    expect(parseDatePickerValue('not-a-date')).toBeNull();
    expect(parseDatePickerValue('2026-02-30')).toBeNull();
    // Reject clamped parser results that no longer match the requested parts.
    expect(
      parseDatePickerValue('2026-02-30', () => new CalendarDate(2026, 2, 28)),
    ).toBeNull();
    expect(clearDatePickerValue()).toBeNull();
  });

  test('todayDatePickerValue uses local calendar parts', () => {
    const fixed = new Date(2026, 0, 5);
    expect(formatDatePickerValue(todayDatePickerValue(fixed))).toBe('2026-01-05');
  });
});
