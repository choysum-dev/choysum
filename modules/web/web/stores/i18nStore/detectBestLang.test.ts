// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { detectBestLang, matchBestLang } from './utils';

test('matchBestLang: empty preferred returns catalog fallback', () => {
  expect(matchBestLang([], ['zh_CN', 'en_US'])).toBe('en_US');
  expect(matchBestLang([], ['zh_CN'])).toBe('zh_CN');
  expect(matchBestLang([], [])).toBe('en_US');
});

test('matchBestLang: case-insensitive exact match returns catalog code', () => {
  expect(matchBestLang(['en-us'], ['en_GB', 'en_US'])).toBe('en_US');
  expect(matchBestLang(['EN-us'], ['en_GB', 'en_US'])).toBe('en_US');
});

test('matchBestLang: prefix match on language subtag', () => {
  expect(matchBestLang(['zh'], ['en_US', 'zh_CN'])).toBe('zh_CN');
  expect(matchBestLang(['zh-Hans'], ['en_US', 'zh_CN'])).toBe('zh_CN');
});

test('detectBestLang: SSR-style catalog fallback without navigator', () => {
  expect(detectBestLang(['fr_FR', 'en_US'])).toBe('en_US');
  expect(detectBestLang(['fr_FR'])).toBe('fr_FR');
});

test('detectBestLang: empty allowed catalog falls back to en_US', () => {
  expect(detectBestLang([])).toBe('en_US');
});