// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { clampChoyPage, choyPageOffset, choyTotalPages } from './paginationHelpers';

describe('paginationHelpers', () => {
  test('choyTotalPages', () => {
    expect(choyTotalPages(0, 20)).toBe(1);
    expect(choyTotalPages(20, 20)).toBe(1);
    expect(choyTotalPages(21, 20)).toBe(2);
    expect(choyTotalPages(100, 0)).toBe(1);
  });

  test('clampChoyPage', () => {
    expect(clampChoyPage(0, 5)).toBe(1);
    expect(clampChoyPage(3, 5)).toBe(3);
    expect(clampChoyPage(9, 5)).toBe(5);
    expect(clampChoyPage(NaN, 5)).toBe(1);
  });

  test('choyPageOffset', () => {
    expect(choyPageOffset(1, 20)).toBe(0);
    expect(choyPageOffset(2, 20)).toBe(20);
    expect(choyPageOffset(0, 20)).toBe(0);
  });
});
