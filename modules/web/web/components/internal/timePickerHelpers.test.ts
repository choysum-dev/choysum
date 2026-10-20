// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  clearTimePickerValue,
  formatTimePickerDisplay,
  formatTimePickerValue,
  nowTimePickerValue,
  parseTimePickerValue,
  withTimePickerParts,
} from './timePickerHelpers';

describe('timePickerHelpers', () => {
  test('formats and parses HH:mm:ss / HH:mm', () => {
    expect(formatTimePickerValue({ hour: 9, minute: 5, second: 7 })).toBe('09:05:07');
    expect(parseTimePickerValue('09:05:07')).toEqual({ hour: 9, minute: 5, second: 7 });
    expect(parseTimePickerValue('09:05')).toEqual({ hour: 9, minute: 5, second: 0 });
    expect(formatTimePickerDisplay('09:05:07')).toBe('09:05:07');
  });

  test('rejects blank and invalid values', () => {
    expect(parseTimePickerValue('')).toBeNull();
    expect(parseTimePickerValue('24:00:00')).toBeNull();
    expect(parseTimePickerValue('09:60')).toBeNull();
    expect(parseTimePickerValue('09:05:60')).toBeNull();
    expect(clearTimePickerValue()).toBeNull();
  });

  test('patches parts and now()', () => {
    expect(withTimePickerParts('09:05:07', { hour: 15 })).toBe('15:05:07');
    expect(withTimePickerParts(null, { minute: 30 })).toBe('00:30:00');
    const fixed = new Date(2026, 0, 5, 14, 7, 9);
    expect(nowTimePickerValue(fixed)).toBe('14:07:09');
  });
});
