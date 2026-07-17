// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTranslate } from './translate';
import { __resetI18nScopeStackForTests } from './scope';
import { resolveRequestLang } from './request_lang';
import {
  clearGlobalRequestContextProvider,
  setGlobalRequestContextProvider,
} from '../../rpc/context';

type TestI18nBridge = { t: (...args: string[]) => string };
type TestChoysumRoot = { i18n?: TestI18nBridge };

const initialI18nBridge = (globalThis as { $choysum?: TestChoysumRoot }).$choysum?.i18n;

function setTestI18nBridge(i18n: TestI18nBridge): void {
  const root = globalThis as { $choysum?: TestChoysumRoot };
  if (!root.$choysum) {
    root.$choysum = {};
  }
  root.$choysum.i18n = i18n;
}

function resetI18nTestState(): void {
  __resetI18nScopeStackForTests();
  clearGlobalRequestContextProvider();
  const root = globalThis as { $choysum?: TestChoysumRoot };
  if (root.$choysum) {
    if (initialI18nBridge) {
      root.$choysum.i18n = initialI18nBridge;
    } else {
      delete root.$choysum.i18n;
    }
  }
}

function withResetI18nState(run: () => void): void {
  resetI18nTestState();
  try {
    run();
  } finally {
    resetI18nTestState();
  }
}

test('i18n resolveRequestLang: uses lang not locale', () => {
  withResetI18nState(() => {
    setGlobalRequestContextProvider({ locale: 'zh-CN' });
    expect(resolveRequestLang()).toBe('en_US');
  });
});

test('i18n resolveRequestLang: prefers lang', () => {
  withResetI18nState(() => {
    setGlobalRequestContextProvider({ lang: 'zh_CN', locale: 'zh-CN' });
    expect(resolveRequestLang()).toBe('zh_CN');
  });
});

test('i18n resolveRequestLang: fallbacks', () => {
  withResetI18nState(() => {
    expect(resolveRequestLang({}, { userLanguage: 'ja_JP' })).toBe('ja_JP');
  });
});

test('i18n createTranslate: miss falls back to src', () => {
  withResetI18nState(() => {
    const { _t } = createTranslate('auth');
    expect(_t('Sign in')).toBe('Sign in');
  });
});

test('i18n createTranslate: hit via bridge', () => {
  withResetI18nState(() => {
    setTestI18nBridge({
      t: (module, lang, scope, src) => {
        if (module === 'auth' && src === 'Sign in' && scope === 'login@title') {
          return '登录';
        }
        return '';
      },
    });
    setGlobalRequestContextProvider({ lang: 'zh_CN' });
    const { _t } = createTranslate('auth');
    expect(_t('Sign in', { scope: 'login@title' })).toBe('登录');
  });
});

test('i18n createTranslate: hit via bridge with kind', () => {
  withResetI18nState(() => {
    setTestI18nBridge({
      t: (module, _lang, scope, src, kind = 'literal') => {
        if (module === 'auth' && src === 'Company' && scope === 'm@id' && kind === 'menu') {
          return '公司';
        }
        return '';
      },
    });
    setGlobalRequestContextProvider({ lang: 'zh_CN' });
    const { _t } = createTranslate('auth');
    expect(_t('Company', { scope: 'm@id', kind: 'menu' })).toBe('公司');
  });
});

test('i18n createTranslate: interpolation', () => {
  withResetI18nState(() => {
    setTestI18nBridge({ t: () => '用户 %s 不存在' });
    setGlobalRequestContextProvider({ lang: 'zh_CN' });
    const { _t } = createTranslate('auth');
    expect(_t('User %s not found', 'alice')).toBe('用户 alice 不存在');
  });
});

test('i18n createTranslate: _lt is lazy', () => {
  withResetI18nState(() => {
    let calls = 0;
    setTestI18nBridge({
      t: () => {
        calls += 1;
        return '访问错误';
      },
    });
    setGlobalRequestContextProvider({ lang: 'zh_CN' });
    const { _lt } = createTranslate('auth');
    const lazy = _lt('Access Error');
    expect(calls).toBe(0);
    expect(String(lazy)).toBe('访问错误');
    expect(calls).toBe(1);
  });
});
