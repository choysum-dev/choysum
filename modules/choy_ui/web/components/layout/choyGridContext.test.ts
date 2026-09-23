// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { normalizeChoyGridCols, resolveChoyColSpan } from './choyGridContext';

describe('normalizeChoyGridCols', () => {
  test('defaults and clamps to a positive integer', () => {
    expect(normalizeChoyGridCols()).toBe(12);
    expect(normalizeChoyGridCols(8)).toBe(8);
    expect(normalizeChoyGridCols(0)).toBe(1);
    expect(normalizeChoyGridCols(-2)).toBe(1);
    expect(normalizeChoyGridCols(Number.NaN)).toBe(12);
    expect(normalizeChoyGridCols(10000)).toBe(24);
  });
});

describe('resolveChoyColSpan', () => {
  test('defaults to full width of twelve tracks', () => {
    expect(resolveChoyColSpan()).toBe(12);
    expect(resolveChoyColSpan(undefined, 8)).toBe(8);
  });

  test('clamps invalid cols and spans to positive integers within the grid', () => {
    expect(resolveChoyColSpan(0, 12)).toBe(1);
    expect(resolveChoyColSpan(-3, 12)).toBe(1);
    expect(resolveChoyColSpan(20, 12)).toBe(12);
    expect(resolveChoyColSpan(2.9, 12)).toBe(2);
    expect(resolveChoyColSpan(3, Number.NaN)).toBe(3);
    expect(resolveChoyColSpan(3, 0)).toBe(1);
    expect(resolveChoyColSpan(null, 12)).toBe(12);
    expect(resolveChoyColSpan(3, 10000)).toBe(3);
  });
});
