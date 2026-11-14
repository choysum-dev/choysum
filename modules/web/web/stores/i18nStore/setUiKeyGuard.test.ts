// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { shouldNoopSetUiKey } from './setUiKeyGuard';

test('shouldNoopSetUiKey: true when initialized on the same locale', () => {
  expect(
    shouldNoopSetUiKey({
      locale: 'zh-CN',
      currentLocaleCode: 'zh-CN',
      isInitialized: true,
    }),
  ).toBe(true);
});

test('shouldNoopSetUiKey: false when locale differs', () => {
  expect(
    shouldNoopSetUiKey({
      locale: 'zh-CN',
      currentLocaleCode: 'en',
      isInitialized: true,
    }),
  ).toBe(false);
});

test('shouldNoopSetUiKey: false before initialize even if locale matches', () => {
  expect(
    shouldNoopSetUiKey({
      locale: 'en',
      currentLocaleCode: 'en',
      isInitialized: false,
    }),
  ).toBe(false);
});

test('shouldNoopSetUiKey: false when current locale is unset', () => {
  expect(
    shouldNoopSetUiKey({
      locale: 'en',
      currentLocaleCode: null,
      isInitialized: true,
    }),
  ).toBe(false);
});
