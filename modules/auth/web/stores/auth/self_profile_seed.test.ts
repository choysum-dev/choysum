// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  needsSelfProfileLoad,
  resolveSelfProfileSeedLanguageId,
  resolveSelfProfileSeedTimezone,
  resolveSessionLanguageCode,
} from './self_profile_seed';

test('resolveSessionLanguageCode: prefers saved User.Language', () => {
  expect(resolveSessionLanguageCode('zh_CN', 'en_US')).toEqual({
    code: 'zh_CN',
    fromSession: false,
  });
});

test('resolveSessionLanguageCode: falls back to session terminology lang', () => {
  expect(resolveSessionLanguageCode(null, 'zh_CN')).toEqual({ code: 'zh_CN', fromSession: true });
  expect(resolveSessionLanguageCode('  ', 'en_US')).toEqual({ code: 'en_US', fromSession: true });
});

test('resolveSessionLanguageCode: falls back to en_US when both empty', () => {
  expect(resolveSessionLanguageCode(null, null)).toEqual({ code: 'en_US', fromSession: true });
});

test('needsSelfProfileLoad: false when currentUser has Id', () => {
  expect(needsSelfProfileLoad({ Id: 'usr_1' })).toBe(false);
  expect(needsSelfProfileLoad({ Id: ' usr_1 ' })).toBe(false);
});

test('needsSelfProfileLoad: true when profile missing', () => {
  expect(needsSelfProfileLoad(null)).toBe(true);
  expect(needsSelfProfileLoad(undefined)).toBe(true);
  expect(needsSelfProfileLoad({})).toBe(true);
  expect(needsSelfProfileLoad({ Id: '' })).toBe(true);
});

test('resolveSelfProfileSeedLanguageId: uses saved LanguageId without Search', async () => {
  let searched = false;
  const resolved = await resolveSelfProfileSeedLanguageId({
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

test('resolveSelfProfileSeedLanguageId: Search once when LanguageId empty', async () => {
  const codes: string[] = [];
  const resolved = await resolveSelfProfileSeedLanguageId({
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

test('resolveSelfProfileSeedLanguageId: Search failure yields empty seed', async () => {
  const resolved = await resolveSelfProfileSeedLanguageId({
    languageId: '',
    sessionTerminologyLang: 'en_US',
    searchLanguageByCode: async () => {
      throw new Error('network');
    },
  });
  expect(resolved.seedLanguageId).toBe('');
  expect(resolved.languageFromSession).toBe(true);
});

test('resolveSelfProfileSeedTimezone: prefers saved Timezone', () => {
  expect(resolveSelfProfileSeedTimezone('UTC', () => 'Asia/Shanghai')).toEqual({
    seedTimezone: 'UTC',
    timezoneFromBrowser: false,
  });
});

test('resolveSelfProfileSeedTimezone: empty Timezone uses browser hint', () => {
  expect(resolveSelfProfileSeedTimezone(null, () => 'Asia/Tokyo')).toEqual({
    seedTimezone: 'Asia/Tokyo',
    timezoneFromBrowser: true,
  });
});

test('resolveSelfProfileSeedTimezone: empty browser yields empty seed', () => {
  expect(resolveSelfProfileSeedTimezone(null, () => '')).toEqual({
    seedTimezone: '',
    timezoneFromBrowser: false,
  });
});

test('resolveSelfProfileSeedTimezone: array Timezone is treated as empty', () => {
  expect(resolveSelfProfileSeedTimezone(['UTC'] as unknown as string, () => 'Asia/Tokyo')).toEqual({
    seedTimezone: 'Asia/Tokyo',
    timezoneFromBrowser: true,
  });
});
