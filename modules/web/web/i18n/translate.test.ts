// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, afterEach } from 'vitest';
import { createI18n } from 'vue-i18n';
import { computed, nextTick } from 'vue';

import {
  createTranslate,
  notifyComposerMessagesChanged,
  trackComposerMessageRevision,
  translateTerm,
} from './translate';
import { projectTerminologyMessages } from './terminology';
import {
  createTextDescriptor,
  createTextDescriptorKey,
  withI18nScope,
} from '@/core/service/i18n';

function installI18n(locale = 'zh-CN') {
  const i18n = createI18n({
    legacy: false,
    locale,
    missingWarn: false,
    fallbackWarn: false,
    messages: { en: {}, 'zh-CN': {} },
    postTranslation: trackComposerMessageRevision,
  });
  (globalThis as { window?: { $i18n?: unknown } }).window = {
    $i18n: i18n.global,
  };
  return i18n.global;
}

describe('createTranslate', () => {
  afterEach(() => {
    delete (globalThis as { window?: unknown }).window;
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
    const descriptor = createTextDescriptor('base', 'Users', { scope: 'base.route.users' });
    const composer = createI18n({
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
    }).global;

    expect(composer.te(descriptor.key, 'zh-CN')).toBe(false);
    expect(translateTerm(composer, descriptor, 'Legacy users')).toBe('People');
  });

  it('safely handles missing descriptors, composers, and translations', () => {
    const descriptor = createTextDescriptor('base', '', { scope: 'base.route.empty' });
    expect(translateTerm(undefined, descriptor, 'Legacy title')).toBe('Legacy title');
    expect(translateTerm({ t: () => '' }, descriptor, 'Legacy title')).toBe('Legacy title');
    expect(translateTerm({ t: () => { throw new Error('unavailable'); } }, descriptor, 'Legacy title'))
      .toBe('Legacy title');
    expect(translateTerm(undefined, undefined, 'Plain title')).toBe('Plain title');
  });

  it('reacts to locale changes through the caller composer', () => {
    const descriptor = createTextDescriptor('base', 'Settings', { scope: 'base.route.settings' });
    const composer = createI18n({
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
    }).global;
    const title = computed(() => translateTerm(composer, descriptor, 'Legacy settings'));

    expect(title.value).toBe('Settings');
    composer.locale.value = 'zh-CN';
    expect(title.value).toBe('设置');
  });

  it('reacts when the Gateway catalog is merged after the consumer loads', async () => {
    const composer = installI18n();
    const { _tr } = createTranslate('auth', { scope: 'web/pages/Login' });
    const label = _tr('User Login');

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

    const { _tr } = createTranslate('auth', { scope: 'web/pages/Login' });
    expect(_tr('Welcome %s', 'Alice').value).toBe('欢迎 Alice');
    expect(_tr('Welcome %s', { scope: 'web/pages/Admin' }, 'Alice').value)
      .toBe('管理员欢迎 Alice');
  });

  it('captures the descriptor before computed reevaluation', async () => {
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

    const { _tr } = createTranslate('auth');
    const label = withI18nScope('scope.at.creation', () => _tr('Users'));
    expect(label.value).toBe('用户');

    withI18nScope('scope.during.reevaluation', () => {
      notifyComposerMessagesChanged();
    });
    await nextTick();
    expect(label.value).toBe('用户');
  });

  it('creates deterministic Unicode-safe JSON descriptors', () => {
    const { _td } = createTranslate('base');
    const descriptor = _td('设置.菜单 🍀', { scope: 'base.menu.设置' });
    const same = _td('设置.菜单 🍀', { scope: 'base.menu.设置' });
    expect(descriptor.key).toBe(same.key);
    expect(descriptor.key).toMatch(/^__terms\.[0-9a-f]+$/);
    expect(descriptor.key.slice('__terms.'.length)).not.toContain('.');
    expect(JSON.parse(JSON.stringify(descriptor))).toEqual(descriptor);
    expect(_td('不同', { scope: descriptor.scope }).key).not.toBe(descriptor.key);
    expect(_td(descriptor.src, { scope: 'base.menu.other' }).key).not.toBe(descriptor.key);
    expect(createTranslate('other')._td(descriptor.src, { scope: descriptor.scope }).key)
      .not.toBe(descriptor.key);
    expect(createTextDescriptorKey(descriptor.module, descriptor.scope, descriptor.src, 'other'))
      .not.toBe(descriptor.key);
  });

  it('preserves legacy messages while projecting a flat reserved namespace', () => {
    const nested = {
      web: { 'scope.with.dots': { 'Source.with.dots': '译文' } },
      legacy: { title: 'Legacy' },
    };
    const projected = projectTerminologyMessages(nested);
    const key = createTextDescriptorKey('web', 'scope.with.dots', 'Source.with.dots', 'literal');
    const segment = key.slice('__terms.'.length);
    expect(projected.web).toBe(nested.web);
    expect(projected.legacy).toBe(nested.legacy);
    expect((projected.__terms as Record<string, string>)[segment]).toBe('译文');
  });
});
