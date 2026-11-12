// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  needsUserLoadOnPreferencesOpen,
  resolvePreferencesSeedLanguageId,
  resolvePreferencesSeedTimezone,
} from './preferences_open';

test('needsUserLoadOnPreferencesOpen: false when currentUser has Id', () => {
  expect(needsUserLoadOnPreferencesOpen({ Id: 'usr_1' })).toBe(false);
  expect(needsUserLoadOnPreferencesOpen({ Id: ' usr_1 ' })).toBe(false);
});

test('needsUserLoadOnPreferencesOpen: true when profile missing', () => {
  expect(needsUserLoadOnPreferencesOpen(null)).toBe(true);
  expect(needsUserLoadOnPreferencesOpen(undefined)).toBe(true);
  expect(needsUserLoadOnPreferencesOpen({})).toBe(true);
  expect(needsUserLoadOnPreferencesOpen({ Id: '' })).toBe(true);
});

test('resolvePreferencesSeedLanguageId: uses saved LanguageId without Search', async () => {
  let searched = false;
  const resolved = await resolvePreferencesSeedLanguageId({
    languageId: { Id: ' lang_zh ' },
    sessionTerminologyLang: 'en_US',
    searchLanguageByCode: async () => {
      searched = true;
      return 'should-not-run';
    },
  });
  expect(resolved).toEqual({ seedLanguageId: 'lang_zh', languageFromSession: false });
  expect(searched).toBe(false);
});

test('resolvePreferencesSeedLanguageId: Search once when LanguageId empty', async () => {
  const codes: string[] = [];
  const resolved = await resolvePreferencesSeedLanguageId({
    languageId: null,
    sessionTerminologyLang: 'zh_CN',
    searchLanguageByCode: async code => {
      codes.push(code);
      return 'lang_from_session';
    },
  });
  expect(codes).toEqual(['zh_CN']);
  expect(resolved).toEqual({ seedLanguageId: 'lang_from_session', languageFromSession: true });
});

test('resolvePreferencesSeedLanguageId: Search failure yields empty seed', async () => {
  const resolved = await resolvePreferencesSeedLanguageId({
    languageId: '',
    sessionTerminologyLang: 'en_US',
    searchLanguageByCode: async () => {
      throw new Error('network');
    },
  });
  expect(resolved.seedLanguageId).toBe('');
  expect(resolved.languageFromSession).toBe(true);
});

test('resolvePreferencesSeedTimezone: prefers saved Timezone without FieldsGet', () => {
  const resolved = resolvePreferencesSeedTimezone('UTC');
  expect(resolved).toEqual({ seedTimezone: 'UTC', timezoneFromBrowser: false });
});

test('resolvePreferencesSeedTimezone: empty Timezone uses browser hint', () => {
  const resolved = resolvePreferencesSeedTimezone(null);
  // Browser IANA zone is environment-dependent; only assert fromBrowser / non-throw.
  if (resolved.seedTimezone) {
    expect(resolved.timezoneFromBrowser).toBe(true);
  } else {
    expect(resolved.timezoneFromBrowser).toBe(false);
  }
});
