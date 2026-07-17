// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { reactive, ref } from 'vue';
import { describe, expect, it, vi, afterEach } from 'vitest';

import { createTranslate } from './translate';

describe('createTranslate', () => {
  afterEach(() => {
    delete (globalThis as { window?: unknown }).window;
  });

  it('falls back to msgid when i18n is missing', () => {
    (globalThis as { window?: unknown }).window = {};
    const { _t } = createTranslate('web');
    expect(_t('Hello', { path: 'web/pages/Home', location: 'title' })).toBe('Hello');
  });

  it('reads module/scope/msgid directly from the active vue-i18n catalog', () => {
    const getLocaleMessage = vi.fn(() => ({
      web: {
        'web/pages/Home@title': {
          Hello: '你好',
          'Sentence with a dot.': '带句号的句子。',
        },
      },
    }));
    (globalThis as { window?: { $i18n?: unknown } }).window = {
      $i18n: { locale: ref('zh-CN'), getLocaleMessage },
    };

    const { _t } = createTranslate('web', { path: 'web/pages/Home', location: 'title' });
    expect(_t('Hello')).toBe('你好');
    expect(_t('Sentence with a dot.')).toBe('带句号的句子。');
    expect(getLocaleMessage).toHaveBeenCalledWith('zh-CN');
  });

  it('falls back to msgid when key missing', () => {
    (globalThis as { window?: { $i18n?: unknown } }).window = {
      $i18n: {
        locale: ref('zh-CN'),
        getLocaleMessage: () => ({}),
      },
    };
    const { _t } = createTranslate('web');
    expect(_t('Missing', { scope: 'web/x@y' })).toBe('Missing');
  });

  it('reacts when the Gateway catalog is merged after the consumer loads', () => {
    const messages = reactive<Record<string, any>>({ 'zh-CN': {} });
    (globalThis as { window?: { $i18n?: unknown } }).window = {
      $i18n: {
        locale: ref('zh-CN'),
        getLocaleMessage: (locale: string) => messages[locale] || {},
      },
    };
    const { _tr } = createTranslate('auth', { scope: 'web/pages/Login' });
    const label = _tr('User Login');

    expect(label.value).toBe('User Login');
    messages['zh-CN'] = {
      auth: {
        'web/pages/Login': {
          'User Login': '用户登录',
        },
      },
    };
    expect(label.value).toBe('用户登录');
  });
});
