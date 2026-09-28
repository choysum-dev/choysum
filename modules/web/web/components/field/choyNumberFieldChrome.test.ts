// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  isChoyNumberHostCompatibleWithMode,
  resolveChoyNumberChromeParseMode,
  resolveChoyNumberInputMode,
} from './choyNumberFieldChrome';

test('resolveChoyNumberChromeParseMode: bigint→integer; undefined→decimal', () => {
  expect(resolveChoyNumberChromeParseMode('bigint')).toBe('integer');
  expect(resolveChoyNumberChromeParseMode('integer')).toBe('integer');
  expect(resolveChoyNumberChromeParseMode('float')).toBe('float');
  expect(resolveChoyNumberChromeParseMode('decimal')).toBe('decimal');
  expect(resolveChoyNumberChromeParseMode(undefined)).toBe('decimal');
});

test('resolveChoyNumberInputMode: integer/bigint numeric else decimal', () => {
  expect(resolveChoyNumberInputMode('integer')).toBe('numeric');
  expect(resolveChoyNumberInputMode('bigint')).toBe('numeric');
  expect(resolveChoyNumberInputMode('float')).toBe('decimal');
  expect(resolveChoyNumberInputMode('decimal')).toBe('decimal');
  expect(resolveChoyNumberInputMode(undefined)).toBe('decimal');
});

test('isChoyNumberHostCompatibleWithMode: finite + mode rules', () => {
  expect(isChoyNumberHostCompatibleWithMode(Number.NaN, 'float')).toBe(false);
  expect(isChoyNumberHostCompatibleWithMode(Number.POSITIVE_INFINITY, 'decimal')).toBe(false);
  expect(isChoyNumberHostCompatibleWithMode(12, 'integer')).toBe(true);
  expect(isChoyNumberHostCompatibleWithMode(12.5, 'integer')).toBe(false);
  expect(isChoyNumberHostCompatibleWithMode(12, 'bigint')).toBe(true);
  expect(isChoyNumberHostCompatibleWithMode(12.5, 'bigint')).toBe(false);
  expect(isChoyNumberHostCompatibleWithMode(12.5, 'float')).toBe(true);
  expect(isChoyNumberHostCompatibleWithMode(Number.MAX_SAFE_INTEGER + 2, 'integer')).toBe(false);
});
