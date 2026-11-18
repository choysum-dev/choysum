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
  languageRowFields,
  languageRowMatchesId,
  languageRowOverrideFields,
  mergeLanguageFormFields,
  reloadActiveLanguagesAfterLanguageSave,
  resolveLanguageCurrentRow,
  resolveLanguageRecordId,
  shouldPreferBrowseReloadAfterLanguageAction,
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

test('languageFormHelpers: languageRowFields unwraps payload/raw/flat', () => {
  expect(languageRowFields(null)).toBe(null);
  expect(languageRowFields('x')).toBe(null);
  expect(languageRowFields({ payload: { Id: '1', Code: 'zh_CN' } })).toEqual({ Id: '1', Code: 'zh_CN' });
  expect(languageRowFields({ raw: { Id: '2', Code: 'en_US' } })).toEqual({ Id: '2', Code: 'en_US' });
  expect(languageRowFields({ Id: '3', Code: 'ja_JP' })).toEqual({ Id: '3', Code: 'ja_JP' });
});

test('languageFormHelpers: languageRowMatchesId matches key or payload Id', () => {
  expect(languageRowMatchesId(null, '1')).toBe(false);
  expect(languageRowMatchesId({ Id: '1' }, '')).toBe(false);
  expect(languageRowMatchesId({ key: 'lang-1', payload: { Id: 'lang-1' } }, 'lang-1')).toBe(true);
  expect(languageRowMatchesId({ Id: 'lang-2', Code: 'zh_CN' }, 'lang-2')).toBe(true);
  expect(languageRowMatchesId({ Id: 'other' }, 'lang-2')).toBe(false);
});

test('languageFormHelpers: resolveLanguageCurrentRow prefers records then rows', () => {
  expect(
    resolveLanguageCurrentRow({
      recordId: 'a',
      records: { a: { kind: 'record', key: 'a', payload: { Id: 'a', IsActive: false, Code: 'zh_HK' } } },
    })
  ).toEqual({ Id: 'a', IsActive: false, Code: 'zh_HK' });

  expect(
    resolveLanguageCurrentRow({
      recordId: 'b',
      records: {},
      rows: [{ kind: 'record', key: 'b', payload: { Id: 'b', IsActive: true, Code: 'zh_CN' } }],
    })
  ).toEqual({ Id: 'b', IsActive: true, Code: 'zh_CN' });

  expect(resolveLanguageCurrentRow({ recordId: 'missing', records: {}, rows: [] })).toEqual({});
});

test('languageFormHelpers: languageRowOverrideFields is scoped to recordId', () => {
  const override = { id: 'lang-a', fields: { Id: 'lang-a', IsActive: true, Code: 'zh_HK' } };
  expect(languageRowOverrideFields(override, 'lang-a')).toEqual(override.fields);
  expect(languageRowOverrideFields(override, 'lang-b')).toBe(null);
  expect(languageRowOverrideFields(null, 'lang-a')).toBe(null);
  expect(languageRowOverrideFields(override, '')).toBe(null);
});

test('languageFormHelpers: resolveLanguageRecordId prefers prop then route :id', () => {
  expect(resolveLanguageRecordId({ recordId: 'from-prop', routeParams: { id: 'from-route' } })).toBe('from-prop');
  expect(resolveLanguageRecordId({ routeParams: { id: 'route-id' } })).toBe('route-id');
  expect(resolveLanguageRecordId({ routeParams: { recordId: 'rid' } })).toBe('rid');
  expect(resolveLanguageRecordId({ routeQuery: { id: 'qid' } })).toBe('qid');
  expect(resolveLanguageRecordId({ routeParams: { id: 'new' } })).toBe('');
  expect(resolveLanguageRecordId({})).toBe('');
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

test('languageFormHelpers: activateLanguage still refreshes overlays when reloadRecord rejects', async () => {
  const calls: string[] = [];
  await activateLanguage({
    activate: async () => {
      calls.push('activate');
    },
    recordId: 'lang-1',
    reloadRecord: async () => {
      calls.push('reloadRecord');
      throw new Error('browse failed');
    },
    reloadActiveLanguages: async () => {
      calls.push('reloadActive');
    },
  });
  expect(calls).toEqual(['activate', 'reloadRecord', 'reloadActive']);
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

test('languageFormHelpers: deactivateLanguage still refreshes overlays when reloadRecord rejects', async () => {
  const calls: string[] = [];
  await deactivateLanguage({
    deactivate: async () => {
      calls.push('deactivate');
    },
    recordId: 'lang-2',
    reloadRecord: async () => {
      calls.push('reloadRecord');
      throw new Error('browse failed');
    },
    reloadActiveLanguages: async () => {
      calls.push('reloadActive');
    },
  });
  expect(calls).toEqual(['deactivate', 'reloadRecord', 'reloadActive']);
});

test('languageFormHelpers: activate/deactivate no-op on empty recordId', async () => {
  let called = false;
  await activateLanguage({
    activate: async () => {
      called = true;
    },
    recordId: '  ',
  });
  await deactivateLanguage({
    deactivate: async () => {
      called = true;
    },
    recordId: '',
  });
  expect(called).toBe(false);
});

test('languageFormHelpers: mergeLanguageFormFields only merges matching draft Id', () => {
  const fromStore = { Id: 'b', Code: 'en_US', IsActive: true };
  expect(
    mergeLanguageFormFields({
      recordId: 'b',
      fromStore,
      formData: { Id: 'a', Code: 'zh_CN', IsActive: false },
    })
  ).toEqual(fromStore);
  expect(
    mergeLanguageFormFields({
      recordId: 'b',
      fromStore,
      formData: { Id: 'b', IsActive: false },
    })
  ).toEqual({ Id: 'b', Code: 'en_US', IsActive: false });
  expect(mergeLanguageFormFields({ recordId: 'b', fromStore, formData: null })).toEqual(fromStore);
});

test('languageFormHelpers: shouldPreferBrowseReloadAfterLanguageAction', () => {
  expect(shouldPreferBrowseReloadAfterLanguageAction('edit')).toBe(true);
  expect(shouldPreferBrowseReloadAfterLanguageAction('create')).toBe(true);
  expect(shouldPreferBrowseReloadAfterLanguageAction('display')).toBe(false);
  expect(shouldPreferBrowseReloadAfterLanguageAction(undefined)).toBe(false);
});
