// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  clearDateTimePickerValue,
  formatDateTimePickerDisplay,
  formatDateTimePickerValue,
  nowDateTimePickerValue,
  parseDateTimePickerValue,
  withDateTimePickerDate,
  withDateTimePickerTime,
} from './dateTimePickerHelpers';

describe('dateTimePickerHelpers', () => {
  test('formats and parses YYYY-MM-DDTHH:mm', () => {
    expect(
      formatDateTimePickerValue({
        year: 2026,
        month: 10,
        day: 20,
        hour: 9,
        minute: 49,
      }),
    ).toBe('2026-10-20T09:49');
    expect(parseDateTimePickerValue('2026-10-20T09:49')).toEqual({
      year: 2026,
      month: 10,
      day: 20,
      hour: 9,
      minute: 49,
    });
    expect(formatDateTimePickerDisplay('2026-10-20T09:49')).toBe('2026-10-20 09:49');
  });

  test('rejects blank and invalid values', () => {
    expect(parseDateTimePickerValue('')).toBeNull();
    expect(parseDateTimePickerValue('2026-10-20')).toBeNull();
    expect(parseDateTimePickerValue('2026-10-20T24:00')).toBeNull();
    expect(parseDateTimePickerValue('2026-10-20T09:60')).toBeNull();
    expect(parseDateTimePickerValue('2026-02-30T09:00')).toBeNull();
    expect(clearDateTimePickerValue()).toBeNull();
  });

  test('merges date / time parts and now()', () => {
    expect(withDateTimePickerDate('2026-10-20T09:49', { year: 2026, month: 11, day: 1 })).toBe(
      '2026-11-01T09:49',
    );
    expect(withDateTimePickerDate(null, { year: 2026, month: 11, day: 1 })).toBe('2026-11-01T00:00');
    expect(withDateTimePickerTime('2026-10-20T09:49', { hour: 15 })).toBe('2026-10-20T15:49');
    expect(withDateTimePickerTime('2026-10-20T09:49', { minute: 5 })).toBe('2026-10-20T09:05');
    const fixed = new Date(2026, 0, 5, 14, 7, 30);
    expect(nowDateTimePickerValue(fixed)).toBe('2026-01-05T14:07');
  });
});
