// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { applyUserLanguagePreference, terminologyCodeFromLanguageId } from './language_preference';

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

test('applyUserLanguagePreference: sets display overrides before language resolution', async () => {
  const order: string[] = [];
  await applyUserLanguagePreference({
    languageId: 'lang_zh',
    displayOverrides: { dateFormat: 'YYYY-MM-DD' },
    browseLanguage: async () => {
      order.push('browse');
      return { Code: 'zh_CN' };
    },
    setUiKey: async key => {
      order.push(`ui:${key}`);
    },
    setDisplayOverrides: value => {
      order.push(`overrides:${JSON.stringify(value)}`);
    },
    langToUiKey: lang => `ui:${lang}`,
  });
  expect(order[0]).toBe('overrides:{"dateFormat":"YYYY-MM-DD"}');
  expect(order).toEqual(['overrides:{"dateFormat":"YYYY-MM-DD"}', 'browse', 'ui:ui:zh_CN']);
});

test('applyUserLanguagePreference: keeps overrides when setUiKey rejects', async () => {
  let overrides: unknown = 'unset';
  let caught: unknown;
  try {
    await applyUserLanguagePreference({
      languageId: 'lang_zh',
      displayOverrides: { dateFormat: 'YYYY-MM-DD' },
      browseLanguage: async () => ({ Code: 'zh_CN' }),
      setUiKey: async () => {
        throw new Error('setUiKey failed');
      },
      setDisplayOverrides: value => {
        overrides = value;
      },
      langToUiKey: lang => lang,
    });
  } catch (err) {
    caught = err;
  }
  expect(overrides).toEqual({ dateFormat: 'YYYY-MM-DD' });
  expect(String((caught as Error)?.message || '')).toContain('setUiKey failed');
});

test('applyUserLanguagePreference: skips setUiKey when LanguageId has no Code', async () => {
  const setUiKeyCalls: string[] = [];
  await applyUserLanguagePreference({
    languageId: 'lang_missing',
    browseLanguage: async () => null,
    setUiKey: async key => {
      setUiKeyCalls.push(key);
    },
    setDisplayOverrides: () => undefined,
    langToUiKey: lang => lang,
  });
  expect(setUiKeyCalls).toEqual([]);
});

test('applyUserLanguagePreference: skips setUiKey when uiKey already matches', async () => {
  const setUiKeyCalls: string[] = [];
  let overrides: unknown = null;
  await applyUserLanguagePreference({
    languageId: 'lang_zh',
    displayOverrides: { dateFormat: 'YYYY-MM-DD' },
    browseLanguage: async () => ({ Code: 'zh_CN' }),
    setUiKey: async key => {
      setUiKeyCalls.push(key);
    },
    setDisplayOverrides: value => {
      overrides = value;
    },
    langToUiKey: lang => (lang === 'zh_CN' ? 'zh-CN' : 'en'),
    getCurrentUiKey: () => 'zh-CN',
  });
  expect(overrides).toEqual({ dateFormat: 'YYYY-MM-DD' });
  expect(setUiKeyCalls).toEqual([]);
});

test('applyUserLanguagePreference: calls setUiKey when uiKey differs', async () => {
  const setUiKeyCalls: string[] = [];
  await applyUserLanguagePreference({
    languageId: 'lang_zh',
    browseLanguage: async () => ({ Code: 'zh_CN' }),
    setUiKey: async key => {
      setUiKeyCalls.push(key);
    },
    setDisplayOverrides: () => undefined,
    langToUiKey: lang => (lang === 'zh_CN' ? 'zh-CN' : 'en'),
    getCurrentUiKey: () => 'en',
  });
  expect(setUiKeyCalls).toEqual(['zh-CN']);
});

test('applyUserLanguagePreference: calls setUiKey when getCurrentUiKey omitted', async () => {
  const setUiKeyCalls: string[] = [];
  await applyUserLanguagePreference({
    languageId: 'lang_zh',
    browseLanguage: async () => ({ Code: 'zh_CN' }),
    setUiKey: async key => {
      setUiKeyCalls.push(key);
    },
    setDisplayOverrides: () => undefined,
    langToUiKey: lang => lang,
  });
  expect(setUiKeyCalls).toEqual(['zh_CN']);
});
