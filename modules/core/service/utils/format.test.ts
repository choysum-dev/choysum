// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { formatPaddedNumber, resolvePaddedNumberFormat, buildPaddedNumberItems } from '@/core/service/utils/format';

// ---------------------------------------------------------------------------
// formatPaddedNumber
// ---------------------------------------------------------------------------

test('formatPaddedNumber with prefix and suffix', () => {
  expect(formatPaddedNumber('SEQ-', '', 5, 42n)).toBe('SEQ-00042');
});

test('formatPaddedNumber without padding', () => {
  expect(formatPaddedNumber('#', '', 0, 7n)).toBe('#7');
});

test('formatPaddedNumber with negative padding treated as zero', () => {
  expect(formatPaddedNumber('', '', -1, 99n)).toBe('99');
});

test('formatPaddedNumber with suffix', () => {
  expect(formatPaddedNumber('', '-END', 3, 1n)).toBe('001-END');
});

test('formatPaddedNumber undefined prefix/suffix treated as empty', () => {
  expect(formatPaddedNumber(undefined, undefined, 3, 5n)).toBe('005');
});

test('resolvePaddedNumberFormat prefers snapshot values and falls back to defaults', () => {
  expect(resolvePaddedNumberFormat({ Prefix: 'S/', Suffix: '-X', Padding: '6' }, { prefix: 'D/', suffix: '-D', padding: 4 })).toEqual({
    prefix: 'S/',
    suffix: '-X',
    padding: 6,
  });

  expect(resolvePaddedNumberFormat({ Prefix: 123, Padding: 'bad' }, { prefix: 'D/', suffix: '-D', padding: 4 })).toEqual({
    prefix: 'D/',
    suffix: '-D',
    padding: 4,
  });

  expect(resolvePaddedNumberFormat(null, { prefix: 'D/', suffix: '-D', padding: 4 })).toEqual({
    prefix: 'D/',
    suffix: '-D',
    padding: 4,
  });
});

test('buildPaddedNumberItems builds contiguous formatted items', () => {
  expect(buildPaddedNumberItems(7n, 3, 'SO/', '', 4)).toEqual([
    { Value: 'SO/0007', Number: 7 },
    { Value: 'SO/0008', Number: 8 },
    { Value: 'SO/0009', Number: 9 },
  ]);

  expect(buildPaddedNumberItems(1n, 0, 'SO/', '', 4)).toEqual([]);
  expect(buildPaddedNumberItems(1n, -1, 'SO/', '', 4)).toEqual([]);
  expect(buildPaddedNumberItems(1n, Number.NaN, 'SO/', '', 4)).toEqual([]);
});
