// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  defaultChoyColSpanBands,
  normalizeChoyGridCols,
  resolveChoyColSpan,
  resolveChoyColSpanBands,
  scaleChoyColSpanFromLg,
} from './choyGridContext';

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

  test('coerces template attribute strings for span and cols', () => {
    expect(resolveChoyColSpan('3' as unknown as number, 12)).toBe(3);
    expect(resolveChoyColSpan(2, '6' as unknown as number)).toBe(2);
    expect(normalizeChoyGridCols('8' as unknown as number)).toBe(8);
  });

  test('treats blank attribute strings as unset', () => {
    expect(normalizeChoyGridCols('' as unknown as number)).toBe(12);
    expect(normalizeChoyGridCols('   ' as unknown as number)).toBe(12);
    expect(resolveChoyColSpan('' as unknown as number, 12)).toBe(12);
    expect(resolveChoyColSpan(' ' as unknown as number, 8)).toBe(8);
  });
});

describe('scaleChoyColSpanFromLg', () => {
  test('maps 421 examples from the responsive design note', () => {
    expect(scaleChoyColSpanFromLg(3)).toEqual({ base: 12, md: 6, lg: 3 });
    expect(scaleChoyColSpanFromLg(6)).toEqual({ base: 12, md: 12, lg: 6 });
    expect(scaleChoyColSpanFromLg(4)).toEqual({ base: 12, md: 12, lg: 6 });
    expect(scaleChoyColSpanFromLg(12)).toEqual({ base: 12, md: 12, lg: 12 });
  });

  test('default one-slot bands match span 3', () => {
    expect(defaultChoyColSpanBands()).toEqual(scaleChoyColSpanFromLg(3));
  });
});

describe('resolveChoyColSpanBands', () => {
  test('omitted span uses 421 defaults when responsive', () => {
    expect(resolveChoyColSpanBands(undefined, 12, true)).toEqual({
      base: 12,
      md: 6,
      lg: 3,
    });
  });

  test('omitted span stays full width when not responsive', () => {
    expect(resolveChoyColSpanBands(undefined, 12, false)).toEqual({
      base: 12,
      md: 12,
      lg: 12,
    });
    expect(resolveChoyColSpanBands(undefined, 8, false)).toEqual({
      base: 8,
      md: 8,
      lg: 8,
    });
  });

  test('numeric span scales under responsive and stays fixed otherwise', () => {
    expect(resolveChoyColSpanBands(3, 12, true)).toEqual({ base: 12, md: 6, lg: 3 });
    expect(resolveChoyColSpanBands(3, 12, false)).toEqual({ base: 3, md: 3, lg: 3 });
  });

  test('blank attribute string matches omitted under responsive', () => {
    expect(resolveChoyColSpanBands('', 12, true)).toEqual(defaultChoyColSpanBands());
    expect(resolveChoyColSpanBands('   ', 12, true)).toEqual(defaultChoyColSpanBands());
  });

  test('invalid non-blank span falls back to full width under responsive', () => {
    expect(resolveChoyColSpanBands(Number.NaN, 12, true)).toEqual({
      base: 12,
      md: 12,
      lg: 12,
    });
    expect(resolveChoyColSpanBands('foo', 12, true)).toEqual({
      base: 12,
      md: 12,
      lg: 12,
    });
  });

  test('object span uses mobile-first fallbacks without scale-up', () => {
    expect(resolveChoyColSpanBands({ base: 12, md: 6, lg: 3 }, 12, true)).toEqual({
      base: 12,
      md: 6,
      lg: 3,
    });
    expect(resolveChoyColSpanBands({ sm: 12, lg: 4 }, 12, true)).toEqual({
      base: 12,
      md: 12,
      lg: 4,
    });
  });

  test('object span collapses to lg??md??sm??base when not responsive', () => {
    expect(resolveChoyColSpanBands({ base: 12, md: 6, lg: 3 }, 12, false)).toEqual({
      base: 3,
      md: 3,
      lg: 3,
    });
    expect(resolveChoyColSpanBands({ base: 12, md: 6 }, 12, false)).toEqual({
      base: 6,
      md: 6,
      lg: 6,
    });
  });
});
