// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  createTermIdentity,
  createTermReference,
  createTermReferenceKey,
  createTranslate,
} from './translate';
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

test('i18n createTranslate: _t and _lt infer precise return types', () => {
  const text: string = createTranslate('auth')._t('Users');
  const reference: import('./translate').TermReference =
    createTranslate('auth', { scope: 'auth.users' })._lt('Users');

  expect(text).toBe('Users');
  expect(reference.src).toBe('Users');
});

test('i18n createTranslate: _lt rejects interpolation at runtime', () => {
  const { _lt } = createTranslate('base', { scope: 'base.Language.Direction.ltr' });
  // Widen past LazyTranslateFn so we can exercise the runtime guards.
  const call = _lt as (src: string, ...args: unknown[]) => unknown;
  expect(() => call('Left to right', undefined, 'unused')).toThrow(
    '_lt does not accept interpolation arguments'
  );
  expect(() => call('Hello %s', 'world')).toThrow(
    '_lt does not accept interpolation arguments'
  );
  expect(() => call('Hello', ['world'])).toThrow(
    '_lt does not accept interpolation arguments'
  );
});

test('i18n createTranslate: _lt caches factory-default references', () => {
  const { _lt } = createTranslate('auth', { scope: 'auth.menu.users' });
  const first = _lt('Users');
  const second = _lt('Users');
  expect(first).toBe(second);
  expect(_lt('Users', { scope: 'auth.menu.users' })).not.toBe(first);
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

test('i18n term identity normalizes fields and defaults kind', () => {
  expect(createTermIdentity('  auth  ', ' Sign in ', { scope: ' login@title ' })).toEqual({
    module: 'auth',
    scope: 'login@title',
    src: ' Sign in ',
    kind: 'literal',
  });
  expect(createTermIdentity(' auth ', 'Company', {
    scope: ' m@id ',
    kind: ' menu ',
  })).toEqual({
    module: 'auth',
    scope: 'm@id',
    src: 'Company',
    kind: 'menu',
  });
});

test('i18n term reference derives from literal canonical identity and key', () => {
  const identity = createTermIdentity(' auth ', 'Users', {
    scope: ' auth.menu.users ',
    kind: 'menu',
  });
  const reference = createTermReference(' auth ', 'Users', {
    scope: ' auth.menu.users ',
    kind: 'menu',
  });

  expect(identity.kind).toBe('menu');
  expect(reference).toEqual({
    ...identity,
    kind: 'literal',
    key: createTermReferenceKey(
      identity.module,
      identity.scope,
      identity.src,
      'literal'
    ),
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

test('i18n createTranslate: explicit closure is lazy', () => {
  withResetI18nState(() => {
    let calls = 0;
    setTestI18nBridge({
      t: () => {
        calls += 1;
        return '访问错误';
      },
    });
    setGlobalRequestContextProvider({ lang: 'zh_CN' });
    const { _t } = createTranslate('auth');
    const lazy = () => _t('Access Error');
    expect(calls).toBe(0);
    expect(lazy()).toBe('访问错误');
    expect(calls).toBe(1);
  });
});

test('i18n createTranslate: closure can capture canonical factory defaults', () => {
  withResetI18nState(() => {
    const calls: string[][] = [];
    setTestI18nBridge({
      t: (...args) => {
        calls.push(args);
        return '';
      },
    });
    setGlobalRequestContextProvider({ lang: 'zh_CN' });
    const options = { scope: ' original ', kind: ' menu ' };
    const { _t } = createTranslate(' auth ', options);
    const lazy = () => _t('Users');

    options.scope = 'changed';
    options.kind = 'literal';
    expect(lazy()).toBe('Users');
    expect(calls).toEqual([
      ['auth', 'zh_CN', 'original', 'Users', 'menu'],
    ]);
  });
});

test('i18n createTranslate: _lt is serializable and does not translate', () => {
  withResetI18nState(() => {
    let calls = 0;
    setTestI18nBridge({
      t: () => {
        calls += 1;
        return '用户';
      },
    });
    const { _lt } = createTranslate('auth');
    const reference = _lt('Users', { scope: 'auth.menu.users' });

    expect(reference).toEqual({
      key: createTermReferenceKey('auth', 'auth.menu.users', 'Users', 'literal'),
      module: 'auth',
      scope: 'auth.menu.users',
      src: 'Users',
      kind: 'literal',
    });
    expect(JSON.parse(JSON.stringify(reference))).toEqual(reference);
    expect(calls).toBe(0);
  });
});

test('i18n term reference keys preserve the fixed legacy identity encoding', () => {
  const identity = ['模块', 'scope.with.dots', '用户 🍀', 'literal'] as const;
  const key = createTermReferenceKey(...identity);
  expect(key).toBe('__terms.363ae6a8a1e59d9731353a73636f70652e776974682e646f747331313ae794a8e688b720f09f8d80373a6c69746572616c');
  expect(createTermReferenceKey(...identity)).toBe(key);
  expect(createTermReference(identity[0], identity[2], {
    scope: identity[1],
    kind: identity[3],
  }).key).toBe(key);
  expect(key).toMatch(/^__terms\.[0-9a-f]+$/);
  for (let index = 0; index < identity.length; index += 1) {
    const changed = [...identity] as [string, string, string, string];
    changed[index] += '!';
    expect(createTermReferenceKey(...changed)).not.toBe(key);
  }
});
