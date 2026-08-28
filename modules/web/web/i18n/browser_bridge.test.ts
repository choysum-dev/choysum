// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, afterEach } from 'vitest';
import { createI18n } from 'vue-i18n';

import { createTranslate } from '@/core/service/i18n';
import { clearContextStack, runWithRequestContextSync } from '@/core/rpc/context';
import {
  exposeBrowserI18nOnWindow,
  installBrowserI18nBridge,
  trackComposerMessageRevision,
} from './index';
import { projectTerminologyMessages } from './terminology';

describe('installBrowserI18nBridge', () => {
  afterEach(() => {
    clearContextStack();
    delete (globalThis as { $choysum?: unknown }).$choysum;
    delete (globalThis as { window?: { $i18n?: unknown } }).window?.$i18n;
  });

  it('falls back to msgid when the bridge is not installed', () => {
    const { _t } = createTranslate('core', { scope: 'web/rpc/errors' });
    expect(_t('Unknown API error')).toBe('Unknown API error');
  });

  it('honors the requested terminology lang when the composer locale differs', () => {
    const i18n = createI18n({
      legacy: false,
      locale: 'en',
      missingWarn: false,
      fallbackWarn: false,
      messages: {
        en: {},
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

    const translated = (globalThis as {
      $choysum?: {
        i18n?: {
          t: (module: string, lang: string, scope: string, src: string, kind?: string) => string;
        };
      };
    }).$choysum?.i18n?.t('core', 'zh_CN', 'web/rpc/errors', 'Unknown API error');

    expect(translated).toBe('未知 API 错误');
  });

  it('exposeBrowserI18nOnWindow wires window.$i18n and createTranslate lookup', () => {
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
    exposeBrowserI18nOnWindow(i18n.global);
    expect((window as { $i18n?: unknown }).$i18n).toBe(i18n.global);

    runWithRequestContextSync({ lang: 'zh_CN', locale: 'zh-CN' }, () => {
      const { _t } = createTranslate('core', { scope: 'web/rpc/errors' });
      expect(_t('Unknown API error')).toBe('未知 API 错误');
    });
  });

  it('exposeBrowserI18nOnWindow is a no-op without window', () => {
    const originalWindow = globalThis.window;
    // @ts-expect-error simulate non-browser runtime
    delete globalThis.window;
    expect(() => exposeBrowserI18nOnWindow({})).not.toThrow();
    globalThis.window = originalWindow;
  });
});
