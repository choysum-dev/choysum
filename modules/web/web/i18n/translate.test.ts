// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, afterEach } from 'vitest';
import { createI18n } from 'vue-i18n';
import { computed, nextTick } from 'vue';

import {
  createTranslate,
  createTermReference,
  createTermReferenceKey,
  withI18nScope,
} from '@/core/service/i18n';
import {
  installBrowserI18nBridge,
  notifyComposerMessagesChanged,
  trackComposerMessageRevision,
  translateTerm,
  type ComposerLike,
} from './index';
import { projectTerminologyMessages } from './terminology';

/**
 * Erase vue-i18n Composer generics. Passing `MessageRecord` into `createI18n`
 * otherwise makes `.te()` / assignability hit TS2589 (excessively deep).
 */
type TestComposer = ComposerLike & {
  te: (key: string, locale?: string) => boolean;
  locale: { value: string };
  mergeLocaleMessage: (locale: string, message: Record<string, any>) => void;
};

function asTestComposer(composer: unknown): TestComposer {
  return composer as TestComposer;
}

function installI18n(locale = 'zh-CN'): TestComposer {
  const i18n = createI18n({
    legacy: false,
    locale,
    missingWarn: false,
    fallbackWarn: false,
    messages: { en: {}, 'zh-CN': {} },
    postTranslation: trackComposerMessageRevision,
  });
  const composer = asTestComposer(i18n.global);
  (globalThis as { window?: { $i18n?: unknown } }).window = {
    $i18n: composer,
  };
  installBrowserI18nBridge();
  return composer;
}

describe('createTranslate', () => {
  afterEach(() => {
    delete (globalThis as { window?: unknown }).window;
    delete (globalThis as { $choysum?: unknown }).$choysum;
  });

  it('falls back to msgid when i18n is missing', () => {
    (globalThis as { window?: unknown }).window = {};
    const { _t } = createTranslate('web');
    expect(_t('Hello', { path: 'web/pages/Home', location: 'title' })).toBe('Hello');
  });

  it('uses Composer t against projected Gateway terms', () => {
    const composer = installI18n();
    composer.mergeLocaleMessage('zh-CN', projectTerminologyMessages({
      web: {
        'web/pages/Home@title': {
          Hello: '你好',
          'Sentence with a dot.': '带句号的句子。',
        },
      },
    }));
    notifyComposerMessagesChanged();

    const { _t } = createTranslate('web', { path: 'web/pages/Home', location: 'title' });
    expect(_t('Hello')).toBe('你好');
    expect(_t('Sentence with a dot.')).toBe('带句号的句子。');
  });

  it('falls back to msgid when key missing', () => {
    installI18n();
    const { _t } = createTranslate('web');
    expect(_t('Missing', { scope: 'web/x@y' })).toBe('Missing');
  });

  it('uses vue-i18n locale fallback without a te preflight', () => {
    const reference = createTermReference('base', 'Users', { scope: 'base.route.users' });
    const composer = asTestComposer(createI18n({
      legacy: false,
      locale: 'zh-CN',
      fallbackLocale: 'en',
      missingWarn: false,
      fallbackWarn: false,
      messages: {
        en: projectTerminologyMessages({
          base: { 'base.route.users': { Users: 'People' } },
        }),
        'zh-CN': {},
      },
    }).global);

    expect(composer.te(reference.key, 'zh-CN')).toBe(false);
    expect(translateTerm(composer, reference, 'Legacy users')).toBe('People');
  });

  it('safely handles missing references, composers, and translations', () => {
    const reference = createTermReference('base', '', { scope: 'base.route.empty' });
    expect(translateTerm(undefined, reference, 'Legacy title')).toBe('Legacy title');
    expect(translateTerm({ t: () => '' }, reference, 'Legacy title')).toBe('Legacy title');
    expect(translateTerm({ t: () => { throw new Error('unavailable'); } }, reference, 'Legacy title'))
      .toBe('Legacy title');
    expect(translateTerm(undefined, undefined, 'Plain title')).toBe('Plain title');
  });

  it('reacts to locale changes through the caller composer', () => {
    const reference = createTermReference('base', 'Settings', { scope: 'base.route.settings' });
    const composer = asTestComposer(createI18n({
      legacy: false,
      locale: 'en',
      missingWarn: false,
      fallbackWarn: false,
      messages: {
        en: {},
        'zh-CN': projectTerminologyMessages({
          base: { 'base.route.settings': { Settings: '设置' } },
        }),
      },
    }).global);
    const title = computed(() => translateTerm(composer, reference, 'Legacy settings'));

    expect(title.value).toBe('Settings');
    composer.locale.value = 'zh-CN';
    expect(title.value).toBe('设置');
  });

  it('reacts when the Gateway catalog is merged after the consumer loads', async () => {
    const composer = installI18n();
    const { _t } = createTranslate('auth', { scope: 'web/pages/Login' });
    const label = computed(() => _t('User Login'));

    expect(label.value).toBe('User Login');
    composer.mergeLocaleMessage('zh-CN', projectTerminologyMessages({
      auth: {
        'web/pages/Login': {
          'User Login': '用户登录',
        },
      },
    }));
    notifyComposerMessagesChanged();
    await nextTick();
    expect(label.value).toBe('用户登录');
  });

  it('uses default and explicit scopes for reactive translations', () => {
    const composer = installI18n();
    composer.mergeLocaleMessage('zh-CN', projectTerminologyMessages({
      auth: {
        'web/pages/Login': {
          'Welcome %s': '欢迎 %s',
        },
        'web/pages/Admin': {
          'Welcome %s': '管理员欢迎 %s',
        },
      },
    }));
    notifyComposerMessagesChanged();

    const { _t } = createTranslate('auth', { scope: 'web/pages/Login' });
    expect(computed(() => _t('Welcome %s', 'Alice')).value).toBe('欢迎 Alice');
    expect(computed(() => _t('Welcome %s', { scope: 'web/pages/Admin' }, 'Alice')).value)
      .toBe('管理员欢迎 Alice');
  });

  it('separates _t text lookup from _lt term references', () => {
    const reference = createTranslate('base')._lt('Left to right', {
      scope: 'base.Language.Direction.ltr',
    });
    expect(reference).toEqual(createTermReference('base', 'Left to right', {
      scope: 'base.Language.Direction.ltr',
    }));

    const text = createTranslate('base', {
      scope: 'base.Language.Direction.ltr',
    })._t('Direction: %s', 'LTR');
    expect(text).toBe('Direction: LTR');
  });

  it('rejects _lt interpolation at runtime', () => {
    const { _lt } = createTranslate('base', { scope: 'base.menu' });
    // Widen past LazyTranslateFn so we can exercise the runtime guards.
    const call = _lt as (src: string, ...args: unknown[]) => unknown;
    expect(() => call('Users', undefined, 'unused')).toThrow(
      '_lt does not accept interpolation arguments'
    );
    expect(() => call('Hello %s', 'world')).toThrow(
      '_lt does not accept interpolation arguments'
    );
    expect(() => call('Hello', [1])).toThrow(
      '_lt does not accept interpolation arguments'
    );
  });

  it('caches factory-default _lt references', () => {
    const { _lt } = createTranslate('web', { scope: 'web/menu/menus' });
    const first = _lt('Home');
    const second = _lt('Home');
    expect(first).toBe(second);
    expect(_lt('Home', { scope: 'web/menu/menus' })).not.toBe(first);
  });

  it('captures the term reference before computed reevaluation', async () => {
    const composer = installI18n();
    composer.mergeLocaleMessage('zh-CN', projectTerminologyMessages({
      auth: {
        'scope.at.creation': {
          Users: '用户',
        },
        'scope.during.reevaluation': {
          Users: '错误范围',
        },
      },
    }));
    notifyComposerMessagesChanged();

    const { _t } = withI18nScope(
      'scope.at.creation',
      () => createTranslate('auth')
    );
    const label = computed(() => _t('Users'));
    expect(label.value).toBe('用户');

    withI18nScope('scope.during.reevaluation', () => {
      notifyComposerMessagesChanged();
    });
    await nextTick();
    expect(label.value).toBe('用户');
  });

  it('creates deterministic Unicode-safe JSON term references', () => {
    const { _lt } = createTranslate('base');
    const reference = _lt('设置.菜单 🍀', { scope: 'base.menu.设置' });
    const same = _lt('设置.菜单 🍀', { scope: 'base.menu.设置' });
    expect(reference.key).toBe(same.key);
    expect(reference.key).toMatch(/^__terms\.[0-9a-f]+$/);
    expect(reference.key.slice('__terms.'.length)).not.toContain('.');
    expect(JSON.parse(JSON.stringify(reference))).toEqual(reference);
    expect(_lt('不同', { scope: reference.scope }).key).not.toBe(reference.key);
    expect(_lt(reference.src, { scope: 'base.menu.other' }).key).not.toBe(reference.key);
    expect(createTranslate('other')._lt(
      reference.src,
      { scope: reference.scope }
    ).key)
      .not.toBe(reference.key);
    expect(createTermReferenceKey(reference.module, reference.scope, reference.src, 'other'))
      .not.toBe(reference.key);
  });

  it('preserves legacy messages while projecting a flat reserved namespace', () => {
    const nested = {
      web: { 'scope.with.dots': { 'Source.with.dots': '译文' } },
      legacy: { title: 'Legacy' },
    };
    const projected = projectTerminologyMessages(nested);
    const key = createTermReferenceKey('web', 'scope.with.dots', 'Source.with.dots', 'literal');
    const segment = key.slice('__terms.'.length);
    expect(projected.web).toBe(nested.web);
    expect(projected.legacy).toBe(nested.legacy);
    expect((projected.__terms as Record<string, string>)[segment]).toBe('译文');
  });
});
