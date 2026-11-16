// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  applyUserLanguagePreference,
  normalizeLanguageId,
  terminologyCodeFromLanguageId,
} from './language_preference';

test('normalizeLanguageId: unwraps ManyToOne and scalars', () => {
  expect(normalizeLanguageId({ Id: 'lang-1' })).toBe('lang-1');
  expect(normalizeLanguageId('lang-2')).toBe('lang-2');
  expect(normalizeLanguageId({ Id: '' })).toBe(null);
  expect(normalizeLanguageId(null)).toBe(null);
  expect(normalizeLanguageId(['lang-tuple'])).toBe(null);
});

test('terminologyCodeFromLanguageId: empty id skips Browse', async () => {
  let browsed = false;
  const code = await terminologyCodeFromLanguageId('', async () => {
    browsed = true;
    return { Code: 'zh_CN' };
  });
  expect(code).toBe('');
  expect(browsed).toBe(false);
});

test('terminologyCodeFromLanguageId: returns trimmed Code from Browse', async () => {
  const code = await terminologyCodeFromLanguageId(' lang_1 ', async (id, fields) => {
    expect(id).toBe('lang_1');
    expect(fields).toEqual(['Code', 'IsActive']);
    return { Code: '  en_US  ' };
  });
  expect(code).toBe('en_US');
});

test('terminologyCodeFromLanguageId: unwraps a ManyToOne Id reference', async () => {
  const code = await terminologyCodeFromLanguageId({ Id: ' lang_2 ' }, async id => {
    expect(id).toBe('lang_2');
    return { Code: 'zh_CN' };
  });
  expect(code).toBe('zh_CN');
});

test('terminologyCodeFromLanguageId: empty object skips Browse', async () => {
  let browsed = false;
  const code = await terminologyCodeFromLanguageId({}, async () => {
    browsed = true;
    return { Code: 'zh_CN' };
  });
  expect(code).toBe('');
  expect(browsed).toBe(false);
});

test('terminologyCodeFromLanguageId: inactive language yields empty code', async () => {
  const code = await terminologyCodeFromLanguageId('lang_off', async () => ({ Code: 'zh_CN', IsActive: false }));
  expect(code).toBe('');
});

test('applyUserLanguagePreference: resolves language then setLang', async () => {
  const order: string[] = [];
  await applyUserLanguagePreference({
    languageId: 'lang_zh',
    browseLanguage: async () => {
      order.push('browse');
      return { Code: 'zh_CN' };
    },
    setLang: async key => {
      order.push(`lang:${key}`);
    },
  });
  expect(order).toEqual(['browse', 'lang:zh_CN']);
});

test('applyUserLanguagePreference: propagates setLang rejection', async () => {
  let caught: unknown;
  try {
    await applyUserLanguagePreference({
      languageId: 'lang_zh',
      browseLanguage: async () => ({ Code: 'zh_CN' }),
      setLang: async () => {
        throw new Error('setLang failed');
      },
    });
  } catch (err) {
    caught = err;
  }
  expect(String((caught as Error)?.message || '')).toContain('setLang failed');
});

test('applyUserLanguagePreference: skips setLang when LanguageId has no Code', async () => {
  const setLangCalls: string[] = [];
  await applyUserLanguagePreference({
    languageId: 'lang_missing',
    browseLanguage: async () => null,
    setLang: async key => {
      setLangCalls.push(key);
    },
  });
  expect(setLangCalls).toEqual([]);
});

test('applyUserLanguagePreference: skips setLang when POSIX code already matches', async () => {
  const setLangCalls: string[] = [];
  await applyUserLanguagePreference({
    languageId: 'lang_zh',
    browseLanguage: async () => ({ Code: 'zh_CN' }),
    setLang: async key => {
      setLangCalls.push(key);
    },
    getCurrentLang: () => 'zh_CN',
  });
  expect(setLangCalls).toEqual([]);
});

test('applyUserLanguagePreference: calls setLang when code differs', async () => {
  const setLangCalls: string[] = [];
  await applyUserLanguagePreference({
    languageId: 'lang_zh',
    browseLanguage: async () => ({ Code: 'zh_CN' }),
    setLang: async key => {
      setLangCalls.push(key);
    },
    getCurrentLang: () => 'en_US',
  });
  expect(setLangCalls).toEqual(['zh_CN']);
});

test('applyUserLanguagePreference: calls setLang when getCurrentLang omitted', async () => {
  const setLangCalls: string[] = [];
  await applyUserLanguagePreference({
    languageId: 'lang_zh',
    browseLanguage: async () => ({ Code: 'zh_CN' }),
    setLang: async key => {
      setLangCalls.push(key);
    },
  });
  expect(setLangCalls).toEqual(['zh_CN']);
});

test('applyUserLanguagePreference: single setLang when preference resolves', async () => {
  const order: string[] = [];
  await applyUserLanguagePreference({
    languageId: 'lang_zh',
    browseLanguage: async () => ({ Code: 'zh_CN' }),
    setLang: async key => {
      order.push(`setLang:${key}`);
      return false;
    },
  });
  expect(order).toEqual(['setLang:zh_CN']);
});