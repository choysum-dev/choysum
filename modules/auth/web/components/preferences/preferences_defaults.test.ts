// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { resolvePreferenceLanguage, resolvePreferenceTimezone } from './preferences_defaults';

test('resolvePreferenceLanguage: prefers saved User.Language', () => {
  expect(resolvePreferenceLanguage('zh_CN', 'en_US')).toEqual({ code: 'zh_CN', fromSession: false });
});

test('resolvePreferenceLanguage: falls back to session terminology lang when User.Language is empty', () => {
  expect(resolvePreferenceLanguage(null, 'zh_CN')).toEqual({ code: 'zh_CN', fromSession: true });
  expect(resolvePreferenceLanguage('  ', 'en_US')).toEqual({ code: 'en_US', fromSession: true });
});

test('resolvePreferenceLanguage: falls back to en_US when both are empty', () => {
  expect(resolvePreferenceLanguage(null, null)).toEqual({ code: 'en_US', fromSession: true });
});

test('resolvePreferenceTimezone: prefers saved User.Timezone', () => {
  expect(resolvePreferenceTimezone('UTC', 'Asia/Shanghai', ['UTC', 'Asia/Shanghai'])).toEqual({
    timezone: 'UTC',
    fromBrowser: false,
});
  });

test('resolvePreferenceTimezone: suggests browser timezone when User.Timezone is empty and allowed', () => {
  expect(resolvePreferenceTimezone(null, 'Asia/Shanghai', ['UTC', 'Asia/Shanghai'])).toEqual({
    timezone: 'Asia/Shanghai',
    fromBrowser: true,
});
  });

test('resolvePreferenceTimezone: leaves empty when browser timezone is not in the allowed selection', () => {
  expect(resolvePreferenceTimezone(null, 'Asia/Shanghai', ['UTC', 'Europe/London'])).toEqual({
    timezone: null,
    fromBrowser: false,
});
  });

test('resolvePreferenceTimezone: suggests browser timezone when the allowed list is empty', () => {
  // FieldsGet may fail; still offer the browser value as an editable suggestion.
  expect(resolvePreferenceTimezone(null, 'Asia/Shanghai', [])).toEqual({
    timezone: 'Asia/Shanghai',
    fromBrowser: true,
});
  });
