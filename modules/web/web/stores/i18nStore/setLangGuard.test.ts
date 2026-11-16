// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { shouldNoopSetLang } from './setLangGuard';

test('shouldNoopSetLang: skips when initialized on same POSIX lang', () => {
  expect(
    shouldNoopSetLang({ lang: 'en_US', currentLang: 'en_US', isInitialized: true }),
  ).toBe(true);
  expect(
    shouldNoopSetLang({ lang: 'zh_CN', currentLang: 'en_US', isInitialized: true }),
  ).toBe(false);
  expect(
    shouldNoopSetLang({ lang: 'en_US', currentLang: 'en_US', isInitialized: false }),
  ).toBe(false);
});