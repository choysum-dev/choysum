// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import { useI18nStore } from '../../stores/i18nStore';
import {
  languageMenuCurrentCode,
  languageMenuRows,
  runLanguageMenuSelect,
} from './shellLanguageMenuModel';

describe('shellLanguageMenuModel', () => {
  test('languageMenuRows: uses store rows or default catalog', () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    const store = useI18nStore();
    store.activeLanguages = [{ Code: 'zh_CN', Name: 'Chinese' }];
    expect(languageMenuRows(store).map(r => r.Code)).toEqual(['zh_CN']);
    expect(languageMenuRows(null, ['en_US']).map(r => r.Code)).toEqual(['en_US']);
    store.activeLanguages = [];
    expect(languageMenuRows(store, ['en_US']).map(r => r.Code)).toEqual(['en_US']);
  });

  test('languageMenuCurrentCode: active language or en_US', () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    const store = useI18nStore();
    store.lang = 'zh_CN';
    store.activeLanguages = [{ Code: 'zh_CN' }];
    expect(languageMenuCurrentCode(store)).toBe('zh_CN');
    expect(languageMenuCurrentCode(null)).toBe('en_US');
  });

  test('runLanguageMenuSelect: no-ops without store', async () => {
    await runLanguageMenuSelect('zh_CN', null);
  });

  test('runLanguageMenuSelect: delegates to setLang when store present', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    const store = useI18nStore();
    store.isInitialized = true;
    store.lang = 'en_US';
    store.activeLanguages = [{ Code: 'en_US' }, { Code: 'zh_CN' }];
    let called = '';
    const stub = {
      ...store,
      setLang: async (code: string) => {
        called = code;
        return true;
      },
    } as typeof store;
    await runLanguageMenuSelect('zh_CN', stub, async () => undefined);
    expect(called).toBe('zh_CN');
  });

  test('runLanguageMenuSelect: swallows remount errors', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    const store = useI18nStore();
    const stub = {
      ...store,
      setLang: async () => true,
    } as typeof store;
    const prevError = console.error;
    const errors: unknown[] = [];
    console.error = (...args: unknown[]) => {
      errors.push(args[0]);
    };
    try {
      await runLanguageMenuSelect('zh_CN', stub, async () => {
        throw new Error('reload boom');
      });
      expect(String(errors[0] || '')).toMatch(/Failed to select language/);
    } finally {
      console.error = prevError;
    }
  });
});