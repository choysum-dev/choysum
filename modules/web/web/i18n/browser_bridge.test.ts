// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, afterEach } from 'vitest';
import { createI18n } from 'vue-i18n';

import { createTranslate } from '@/core/service/i18n';
import { installBrowserI18nBridge, trackComposerMessageRevision } from './index';
import { projectTerminologyMessages } from './terminology';

describe('installBrowserI18nBridge', () => {
  afterEach(() => {
    delete (globalThis as { window?: unknown }).window;
    delete (globalThis as { $choysum?: unknown }).$choysum;
  });

  it('falls back to msgid when the bridge is not installed', () => {
    const { _t } = createTranslate('core', { scope: 'web/rpc/errors' });
    expect(_t('Unknown API error')).toBe('Unknown API error');
  });

  it('resolves terminology through $choysum.i18n.t', () => {
    const i18n = createI18n({
      legacy: false,
      locale: 'zh-CN',
      missingWarn: false,
      fallbackWarn: false,
      messages: {
        'zh-CN': projectTerminologyMessages({
          core: {
            'web/rpc/errors': {
              'Unknown API error': '未知 API 错误',
            },
          },
        }),
      },
      postTranslation: trackComposerMessageRevision,
    });
    (globalThis as { window?: { $i18n?: unknown } }).window = {
      $i18n: i18n.global,
    };
    installBrowserI18nBridge();

    const { _t } = createTranslate('core', { scope: 'web/rpc/errors' });
    expect(_t('Unknown API error')).toBe('未知 API 错误');
  });
});
