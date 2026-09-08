// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createI18n } from 'vue-i18n';

import { createTranslate } from '@/core/service/i18n';
import { clearContextStack, runWithRequestContextSync } from '@/core/rpc/context';
import {
  exposeBrowserI18nOnWindow,
  installBrowserI18nBridge,
  trackComposerMessageRevision,
} from './index';
import { projectTerminologyMessages } from './terminology';

afterEach(() => {
  clearContextStack();
  delete (globalThis as { $choysum?: unknown }).$choysum;
  const win = (globalThis as { window?: { $i18n?: unknown } }).window;
  if (win && win !== (globalThis as unknown)) {
    (globalThis as { window?: unknown }).window = globalThis;
  } else if (win) {
    delete win.$i18n;
  }
});

test('installBrowserI18nBridge > falls back to msgid when the bridge is not installed', () => {
  const { _t } = createTranslate('core', { scope: 'web/rpc/errors' });
  expect(_t('Unknown API error')).toBe('Unknown API error');
});

test('installBrowserI18nBridge > honors the requested terminology lang when the composer locale differs', () => {
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
  (globalThis as { window?: { $i18n?: unknown } }).window = globalThis as any;
  (globalThis as { window: { $i18n?: unknown } }).window.$i18n = i18n.global;
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

test('installBrowserI18nBridge > exposeBrowserI18nOnWindow wires window.$i18n and createTranslate lookup', () => {
  // ES modules do not resolve bare `window`; keep a real globalThis.window for the bridge.
  (globalThis as { window?: unknown }).window = globalThis as any;
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
  expect((globalThis as { window?: { $i18n?: unknown } }).window?.$i18n).toBe(i18n.global);

  runWithRequestContextSync({ lang: 'zh_CN', locale: 'zh-CN' }, () => {
    const { _t } = createTranslate('core', { scope: 'web/rpc/errors' });
    expect(_t('Unknown API error')).toBe('未知 API 错误');
  });
});

test('installBrowserI18nBridge > exposeBrowserI18nOnWindow is a no-op without window', () => {
  const originalWindow = (globalThis as { window?: unknown }).window;
  delete (globalThis as { window?: unknown }).window;
  expect(() => exposeBrowserI18nOnWindow({})).not.toThrow();
  (globalThis as { window?: unknown }).window = originalWindow ?? globalThis;
});

