// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';

import { uiKeyToLang, langToUiKey } from './lang';
import { createTerminologyCatalogMerger, shouldMergeTerminology } from './merge';

describe('uiKeyToLang / langToUiKey', () => {
  it('maps zh-CN ↔ zh_CN and en ↔ en_US', () => {
    expect(uiKeyToLang('zh-CN')).toBe('zh_CN');
    expect(langToUiKey('zh_CN')).toBe('zh-CN');
    expect(uiKeyToLang('en')).toBe('en_US');
    expect(langToUiKey('en_US')).toBe('en');
  });

  it('does not treat locale as lang (D12d)', () => {
    expect(uiKeyToLang('zh-CN')).not.toBe('zh-CN');
  });
});

describe('shouldMergeTerminology', () => {
  it('merges only when Gateway returned a fresh non-empty identified catalog', () => {
    expect(
      shouldMergeTerminology({
        lang: 'zh_CN',
        locale: 'zh-CN',
        hash: 'h',
        unchanged: false,
        messages: { auth: { scope: { Login: '登录' } } },
      })
    ).toBe(true);
  });

  it('does not merge an empty payload', () => {
    expect(
      shouldMergeTerminology({
        lang: 'zh_CN',
        locale: 'zh-CN',
        hash: 'h',
        unchanged: false,
        messages: {},
      })
    ).toBe(false);
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

describe('createTerminologyCatalogMerger', () => {
  it('merges and invalidates once, ignores duplicates, and refreshes changed hashes', () => {
    const merge = vi.fn();
    const notify = vi.fn();
    const apply = createTerminologyCatalogMerger({ merge, notify });
    const first = {
      lang: 'zh_CN',
      locale: 'zh-CN',
      hash: 'hash-1',
      unchanged: false,
      messages: { base: { menu: { Settings: '设置' } } },
    };

    expect(apply(first, 'zh-CN')).toBe(true);
    expect(apply(first, 'zh-CN')).toBe(false);
    expect(merge).toHaveBeenCalledTimes(1);
    expect(notify).toHaveBeenCalledTimes(1);

    expect(apply({
      ...first,
      hash: 'hash-2',
      messages: { base: { menu: { Settings: '系统设置' } } },
    }, 'zh-CN')).toBe(true);
    expect(merge).toHaveBeenCalledTimes(2);
    expect(notify).toHaveBeenCalledTimes(2);
  });

  it('does not consume identity when merging fails', () => {
    const merge = vi.fn()
      .mockImplementationOnce(() => {
        throw new Error('merge failed');
      })
      .mockImplementationOnce(() => undefined);
    const notify = vi.fn();
    const apply = createTerminologyCatalogMerger({ merge, notify });
    const load = {
      lang: 'zh_CN',
      locale: 'zh-CN',
      hash: 'hash-1',
      unchanged: false,
      messages: { base: { menu: { Settings: '设置' } } },
    };

    expect(() => apply(load, 'zh-CN')).toThrow('merge failed');
    expect(apply(load, 'zh-CN')).toBe(true);
    expect(notify).toHaveBeenCalledTimes(1);
  });
});
