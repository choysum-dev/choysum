// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  activateLanguage,
  deactivateLanguage,
  GROUPING_SELECTION,
  FIRST_DAY_SELECTION,
  createPersonNameFormatRules,
  isLanguageCodeReadonly,
  LANGUAGE_LIST_COLUMN_PROPS,
  reloadActiveLanguagesAfterLanguageSave,
} from './languageFormHelpers';

test('languageFormHelpers: selection option catalogs', () => {
  expect(GROUPING_SELECTION).toContain('[3,0]');
  expect(GROUPING_SELECTION).toContain('[]');
  expect(FIRST_DAY_SELECTION).toEqual(['0', '1', '2', '3', '4', '5', '6']);
  expect(LANGUAGE_LIST_COLUMN_PROPS).toEqual(['Name', 'Code', 'Direction', 'IsActive', 'DateFormat']);
  expect(isLanguageCodeReadonly('create')).toBe(false);
  expect(isLanguageCodeReadonly('edit')).toBe(true);
  expect(isLanguageCodeReadonly('display')).toBe(true);
  expect(isLanguageCodeReadonly(undefined)).toBe(true);
});

test('languageFormHelpers: person name format rules accept templates with tokens', async () => {
  const rules = createPersonNameFormatRules('invalid person name format');
  const validator = rules[0]?.validator;
  expect(typeof validator).toBe('function');
  await new Promise<void>((resolve, reject) => {
    validator!({}, '{LastName}{FirstName}', (err?: Error) => {
      if (err) reject(err);
      else resolve();
    });
  });
});

test('languageFormHelpers: person name format rules reject invalid templates', async () => {
  const rules = createPersonNameFormatRules('invalid person name format');
  const validator = rules[0]?.validator;
  let caught: Error | undefined;
  await new Promise<void>(resolve => {
    validator!({}, 'no-tokens-here', (err?: Error) => {
      caught = err;
      resolve();
    });
  });
  expect(caught).toBeTruthy();
  expect(String(caught?.message || '')).toBe('invalid person name format');
});

test('languageFormHelpers: reloadActiveLanguagesAfterLanguageSave swallows load failures', async () => {
  await reloadActiveLanguagesAfterLanguageSave();
});

test('languageFormHelpers: activateLanguage calls Activate then reloads', async () => {
  const calls: string[] = [];
  await activateLanguage({
    activate: async req => {
      calls.push(`activate:${req.Id}`);
    },
    recordId: 'lang-1',
    reloadRecord: async () => {
      calls.push('reloadRecord');
    },
    reloadActiveLanguages: async () => {
      calls.push('reloadActive');
    },
  });
  expect(calls).toEqual(['activate:lang-1', 'reloadRecord', 'reloadActive']);
});

test('languageFormHelpers: activateLanguage skips reload when Activate rejects', async () => {
  const calls: string[] = [];
  let err: unknown;
  try {
    await activateLanguage({
      activate: async () => {
        calls.push('activate');
        throw new Error('nope');
      },
      recordId: 'lang-1',
      reloadRecord: async () => {
        calls.push('reloadRecord');
      },
      reloadActiveLanguages: async () => {
        calls.push('reloadActive');
      },
    });
  } catch (e) {
    err = e;
  }
  expect(String((err as Error)?.message || err)).toBe('nope');
  expect(calls).toEqual(['activate']);
});

test('languageFormHelpers: deactivateLanguage calls Deactivate then reloads', async () => {
  const calls: string[] = [];
  await deactivateLanguage({
    deactivate: async req => {
      calls.push(`deactivate:${req.Id}`);
    },
    recordId: 'lang-2',
    reloadRecord: async () => {
      calls.push('reloadRecord');
    },
    reloadActiveLanguages: async () => {
      calls.push('reloadActive');
    },
  });
  expect(calls).toEqual(['deactivate:lang-2', 'reloadRecord', 'reloadActive']);
});
