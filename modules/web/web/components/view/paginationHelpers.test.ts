// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  applyChoyPagerRangeEdit,
  clampChoyPage,
  choyPageOffset,
  choyTotalPages,
  formatChoyPagerRangeValue,
  parseChoyPagerRange,
} from './paginationHelpers';

describe('paginationHelpers', () => {
  test('choyTotalPages', () => {
    expect(choyTotalPages(0, 20)).toBe(1);
    expect(choyTotalPages(20, 20)).toBe(1);
    expect(choyTotalPages(21, 20)).toBe(2);
    expect(choyTotalPages(100, 0)).toBe(1);
    expect(choyTotalPages(100, -5)).toBe(1);
    expect(choyTotalPages(37.9, 10.5)).toBe(4);
    expect(choyTotalPages(NaN, 10)).toBe(1);
  });

  test('clampChoyPage', () => {
    expect(clampChoyPage(0, 5)).toBe(1);
    expect(clampChoyPage(3, 5)).toBe(3);
    expect(clampChoyPage(9, 5)).toBe(5);
    expect(clampChoyPage(NaN, 5)).toBe(1);
    expect(clampChoyPage(2.9, 5)).toBe(2);
    expect(clampChoyPage(-3, 5)).toBe(1);
    expect(clampChoyPage(2, 0)).toBe(1);
    expect(clampChoyPage(2, NaN)).toBe(1);
  });

  test('choyPageOffset', () => {
    expect(choyPageOffset(1, 20)).toBe(0);
    expect(choyPageOffset(2, 20)).toBe(20);
    expect(choyPageOffset(0, 20)).toBe(0);
    expect(choyPageOffset(3, 0)).toBe(0);
    expect(choyPageOffset(2.7, 10.2)).toBe(10);
  });

  test('formatChoyPagerRangeValue', () => {
    expect(formatChoyPagerRangeValue(1, 12)).toBe('1-12');
    expect(formatChoyPagerRangeValue(5, 5)).toBe('5');
    expect(formatChoyPagerRangeValue(0, 0)).toBe('0');
  });

  test('parseChoyPagerRange', () => {
    expect(parseChoyPagerRange('21')).toEqual({ minimum: 21 });
    expect(parseChoyPagerRange('1-50')).toEqual({ minimum: 1, maximum: 50 });
    expect(parseChoyPagerRange('1\u201350')).toEqual({ minimum: 1, maximum: 50 });
    expect(parseChoyPagerRange('50,1')).toEqual({ minimum: 1, maximum: 50 });
    expect(parseChoyPagerRange('')).toBeNull();
    expect(parseChoyPagerRange('abc')).toBeNull();
  });

  test('applyChoyPagerRangeEdit', () => {
    expect(applyChoyPagerRangeEdit({ draft: '21', total: 100, pageSize: 10 })).toEqual({
      page: 3,
      pageSize: 10,
    });
    expect(applyChoyPagerRangeEdit({ draft: '1-50', total: 100, pageSize: 20 })).toEqual({
      page: 1,
      pageSize: 50,
    });
    // Degenerate "5-5" jumps to that index without collapsing pageSize to 1.
    expect(applyChoyPagerRangeEdit({ draft: '5-5', total: 100, pageSize: 20 })).toEqual({
      page: 1,
      pageSize: 20,
    });
    expect(applyChoyPagerRangeEdit({ draft: 'x', total: 100, pageSize: 20 })).toBeNull();
  });
});
