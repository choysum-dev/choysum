// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { formatUtcInTimeZone } from '@/web/web/utils/datetime';
import { valueToPreview } from '@/web/web/query/utils/condition/like';

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
  test('formats and parses YYYY-MM-DDTHH:mm:ss', () => {
    expect(
      formatDateTimePickerValue({
        year: 2026,
        month: 10,
        day: 20,
        hour: 9,
        minute: 49,
        second: 7,
      }),
    ).toBe('2026-10-20T09:49:07');
    expect(parseDateTimePickerValue('2026-10-20T09:49:07')).toEqual({
      year: 2026,
      month: 10,
      day: 20,
      hour: 9,
      minute: 49,
      second: 7,
    });
    expect(formatDateTimePickerDisplay('2026-10-20T09:49:07')).toBe('2026-10-20 09:49:07');
    expect(formatDateTimePickerDisplay('2026-10-20T09:49:07', 'DD.MM.YYYY HH:mm')).toBe(
      '20.10.2026 09:49',
    );
    // Wall-clock parts must format even when the local zone has a DST gap at that clock.
    expect(formatDateTimePickerDisplay('2026-03-08T02:30:00', 'DD.MM.YYYY HH:mm')).toBe(
      '08.03.2026 02:30',
    );
    expect(formatDateTimePickerDisplay('0042-01-05T09:00:00', 'DD.MM.YYYY HH:mm')).toBe(
      '05.01.0042 09:00',
    );
  });

  test('formatDateTimePickerDisplay: Z offset follows user timezone, not UTC', () => {
    // UTC 00:00Z → Asia/Shanghai wall 08:00; offset must be +08:00 (not +00:00).
    const wall = '2026-01-05T08:00:00';
    const fmt = 'YYYY-MM-DD HH:mm Z';
    const tz = 'Asia/Shanghai';
    expect(formatDateTimePickerDisplay(wall, fmt, tz)).toBe('2026-01-05 08:00 +08:00');
    // Same instant via hub / query preview must agree.
    expect(formatUtcInTimeZone('2026-01-05T00:00:00.000Z', fmt, tz)).toBe(
      '2026-01-05 08:00 +08:00',
    );
    expect(
      valueToPreview('=', '2026-01-05T00:00:00.000Z', {
        fieldType: 'datetime',
        timeZone: tz,
        displayFormat: fmt,
      }),
    ).toBe('2026-01-05 08:00 +08:00');
    // No Z: still wall-parts path (Shanghai not required).
    expect(formatDateTimePickerDisplay(wall, 'DD.MM.YYYY HH:mm', tz)).toBe('05.01.2026 08:00');
    // Literal [Z] must not force the zone-offset path.
    expect(formatDateTimePickerDisplay(wall, 'YYYY-MM-DD[T]HH:mm:ss[Z]', tz)).toBe(
      '2026-01-05T08:00:00Z',
    );
    // Invalid IANA: keep wall clock, omit offset (do not claim +00:00).
    expect(formatDateTimePickerDisplay(wall, 'YYYY-MM-DD HH:mm Z', 'Not/A_Zone')).toBe(
      '2026-01-05 08:00',
    );
    // User-TZ spring-forward gap: keep requested clock without a fake UTC offset.
    expect(
      formatDateTimePickerDisplay('2026-03-08T02:30:00', 'YYYY-MM-DD HH:mm Z', 'America/New_York'),
    ).toBe('2026-03-08 02:30');
    // Year 0001–0099 with Z: keep literal year; omit offset when tz remap fails round-trip.
    expect(formatDateTimePickerDisplay('0042-01-05T09:00:00', 'YYYY-MM-DD HH:mm Z', 'UTC')).toBe(
      '0042-01-05 09:00',
    );
    // Offset-only format with failed zone → canonical wall (nothing left after stripping Z).
    expect(formatDateTimePickerDisplay(wall, 'Z', 'Not/A_Zone')).toBe('2026-01-05 08:00:00');
    // Bracket literals in the failed-zone path must stay (covers stripDisplayOffsetTokens).
    expect(formatDateTimePickerDisplay(wall, 'YYYY-MM-DD[T]HH:mm Z', 'Not/A_Zone')).toBe(
      '2026-01-05T08:00',
    );
    expect(formatDateTimePickerDisplay(wall, 'YYYY-MM-DD HH:mm ZZ', 'Not/A_Zone')).toBe(
      '2026-01-05 08:00',
    );
    // Unclosed `[` is kept as a trailing literal fragment after stripping Z.
    expect(formatDateTimePickerDisplay(wall, 'HH:mm Z[', 'Not/A_Zone')).toBe('08:00 [');
  });

  test('parses legacy YYYY-MM-DDTHH:mm with second default 0', () => {
    expect(parseDateTimePickerValue('2026-10-20T09:49')).toEqual({
      year: 2026,
      month: 10,
      day: 20,
      hour: 9,
      minute: 49,
      second: 0,
    });
    expect(formatDateTimePickerDisplay('2026-10-20T09:49')).toBe('2026-10-20 09:49:00');
  });

  test('rejects blank and invalid values', () => {
    expect(parseDateTimePickerValue('')).toBeNull();
    expect(parseDateTimePickerValue('2026-10-20')).toBeNull();
    expect(parseDateTimePickerValue('2026-10-20T24:00')).toBeNull();
    expect(parseDateTimePickerValue('2026-10-20T09:60')).toBeNull();
    expect(parseDateTimePickerValue('2026-10-20T09:00:60')).toBeNull();
    expect(parseDateTimePickerValue('2026-02-30T09:00:00')).toBeNull();
    expect(clearDateTimePickerValue()).toBeNull();
  });

  test('merges date / time parts and now()', () => {
    expect(withDateTimePickerDate('2026-10-20T09:49:07', { year: 2026, month: 11, day: 1 })).toBe(
      '2026-11-01T09:49:07',
    );
    expect(withDateTimePickerDate(null, { year: 2026, month: 11, day: 1 })).toBe(
      '2026-11-01T00:00:00',
    );
    expect(withDateTimePickerTime('2026-10-20T09:49:07', { hour: 15 })).toBe('2026-10-20T15:49:07');
    expect(withDateTimePickerTime('2026-10-20T09:49:07', { minute: 5 })).toBe('2026-10-20T09:05:07');
    expect(withDateTimePickerTime('2026-10-20T09:49:07', { second: 30 })).toBe(
      '2026-10-20T09:49:30',
    );
    const fixed = new Date(2026, 0, 5, 14, 7, 30);
    expect(nowDateTimePickerValue(fixed)).toBe('2026-01-05T14:07:30');
  });
});
