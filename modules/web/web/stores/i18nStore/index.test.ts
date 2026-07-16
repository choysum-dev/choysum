// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';

import { localeToLang, langToLocale } from './lang';
import { shouldMergeTerminology } from './merge';

describe('localeToLang / langToLocale', () => {
  it('maps zh-CN ↔ zh_CN and en ↔ en_US', () => {
    expect(localeToLang('zh-CN')).toBe('zh_CN');
    expect(langToLocale('zh_CN')).toBe('zh-CN');
    expect(localeToLang('en')).toBe('en_US');
    expect(langToLocale('en_US')).toBe('en');
  });

  it('does not treat locale as lang (D12d)', () => {
    expect(localeToLang('zh-CN')).not.toBe('zh-CN');
  });
});

describe('shouldMergeTerminology', () => {
  it('merges only when Gateway returned fresh messages', () => {
    expect(
      shouldMergeTerminology({
        lang: 'zh_CN',
        locale: 'zh-CN',
        hash: 'h',
        unchanged: false,
        messages: { auth: {} },
      })
    ).toBe(true);
  });

  it('does not merge when unchanged', () => {
    expect(
      shouldMergeTerminology({
        lang: 'zh_CN',
        locale: 'zh-CN',
        hash: 'h',
        unchanged: true,
        messages: null,
      })
    ).toBe(false);
  });

  it('does not merge on gatewayError so UI keeps msgid', () => {
    expect(
      shouldMergeTerminology({
        lang: 'zh_CN',
        locale: 'zh-CN',
        hash: '',
        unchanged: false,
        messages: null,
        gatewayError: true,
      })
    ).toBe(false);
  });
});
