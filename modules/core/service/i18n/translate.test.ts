// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, test, beforeEach, afterEach } from 'bun:test';
import { createTranslate } from './translate';
import { __resetI18nScopeStackForTests } from './scope';
import { resolveRequestLang } from './request_lang';
import {
  clearGlobalRequestContextProvider,
  setGlobalRequestContextProvider,
} from '../../rpc/context';

beforeEach(() => {
  __resetI18nScopeStackForTests();
  clearGlobalRequestContextProvider();
});

afterEach(() => {
  clearGlobalRequestContextProvider();
  const root = globalThis as { $choysum?: { i18n?: unknown } };
  if (root.$choysum) {
    delete root.$choysum.i18n;
  }
});

describe('resolveRequestLang', () => {
  test('uses lang not locale', () => {
    setGlobalRequestContextProvider({ locale: 'zh-CN' });
    expect(resolveRequestLang()).toBe('en_US');
  });
  test('prefers lang', () => {
    setGlobalRequestContextProvider({ lang: 'zh_CN', locale: 'zh-CN' });
    expect(resolveRequestLang()).toBe('zh_CN');
  });
  test('fallbacks', () => {
    expect(resolveRequestLang({}, { userLanguage: 'ja_JP' })).toBe('ja_JP');
  });
});

describe('createTranslate', () => {
  test('miss falls back to src', () => {
    const { _t } = createTranslate('auth');
    expect(_t('Sign in')).toBe('Sign in');
  });

  test('hit via bridge', () => {
    const root = globalThis as { $choysum: { i18n: { t: (...args: string[]) => string } } };
    root.$choysum = {
      i18n: {
        t: (module, lang, scope, src) => {
          if (module === 'auth' && src === 'Sign in' && scope === 'login@title') {
            return '登录';
          }
          return '';
        },
      },
    };
    setGlobalRequestContextProvider({ lang: 'zh_CN' });
    const { _t } = createTranslate('auth');
    expect(_t('Sign in', { scope: 'login@title' })).toBe('登录');
  });

  test('hit via bridge with kind', () => {
    const root = globalThis as { $choysum: { i18n: { t: (...args: string[]) => string } } };
    root.$choysum = {
      i18n: {
        t: (module, _lang, scope, src, kind = 'literal') => {
          if (module === 'auth' && src === 'Company' && scope === 'm@id' && kind === 'menu') {
            return '公司';
          }
          return '';
        },
      },
    };
    setGlobalRequestContextProvider({ lang: 'zh_CN' });
    const { _t } = createTranslate('auth');
    expect(_t('Company', { scope: 'm@id', kind: 'menu' })).toBe('公司');
  });

  test('interpolation', () => {
    const root = globalThis as { $choysum: { i18n: { t: () => string } } };
    root.$choysum = { i18n: { t: () => '用户 %s 不存在' } };
    setGlobalRequestContextProvider({ lang: 'zh_CN' });
    const { _t } = createTranslate('auth');
    expect(_t('User %s not found', 'alice')).toBe('用户 alice 不存在');
  });

  test('_lt is lazy', () => {
    let calls = 0;
    const root = globalThis as { $choysum: { i18n: { t: () => string } } };
    root.$choysum = {
      i18n: {
        t: () => {
          calls += 1;
          return '访问错误';
        },
      },
    };
    setGlobalRequestContextProvider({ lang: 'zh_CN' });
    const { _lt } = createTranslate('auth');
    const lazy = _lt('Access Error');
    expect(calls).toBe(0);
    expect(String(lazy)).toBe('访问错误');
    expect(calls).toBe(1);
  });
});
