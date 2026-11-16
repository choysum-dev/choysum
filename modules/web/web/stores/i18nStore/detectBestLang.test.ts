// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { detectBestLang, looksLikeLegacyUiKey } from './utils';

test('looksLikeLegacyUiKey: detects short and hyphen UI keys', () => {
  expect(looksLikeLegacyUiKey('en')).toBe(true);
  expect(looksLikeLegacyUiKey('zh-CN')).toBe(true);
  expect(looksLikeLegacyUiKey('')).toBe(true);
  expect(looksLikeLegacyUiKey('zh_CN')).toBe(false);
  expect(looksLikeLegacyUiKey('en_US')).toBe(false);
});

test('detectBestLang: matches navigator prefs against allowed POSIX codes', () => {
  expect(detectBestLang(['en_US', 'zh_CN'])).toBe('en_US');
});