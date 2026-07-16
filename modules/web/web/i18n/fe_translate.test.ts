// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, afterEach } from 'vitest';

import { createFeTranslate } from './fe_translate';

describe('createFeTranslate', () => {
  afterEach(() => {
    delete (globalThis as { window?: unknown }).window;
  });

  it('falls back to msgid when i18n is missing', () => {
    (globalThis as { window?: unknown }).window = {};
    const { _t } = createFeTranslate('web');
    expect(_t('Hello', { path: 'web/pages/Home', location: 'title' })).toBe('Hello');
  });

  it('reads namespace module.scope from vue-i18n', () => {
    const t = vi.fn((_key: string, _arg?: unknown, opts?: { namespace?: string }) => {
      if (opts?.namespace === 'web.web/pages/Home@title') {
        return '你好';
      }
      return _key;
    });
    const te = vi.fn(() => true);
    (globalThis as { window?: { $i18n?: unknown } }).window = { $i18n: { t, te } };

    const { _t } = createFeTranslate('web');
    expect(_t('Hello', { path: 'web/pages/Home', location: 'title' })).toBe('你好');
    expect(t).toHaveBeenCalled();
  });

  it('falls back to msgid when key missing', () => {
    (globalThis as { window?: { $i18n?: unknown } }).window = {
      $i18n: {
        t: () => 'missing',
        te: () => false,
      },
    };
    const { _t } = createFeTranslate('web');
    expect(_t('Missing', { scope: 'web/x@y' })).toBe('Missing');
  });
});
