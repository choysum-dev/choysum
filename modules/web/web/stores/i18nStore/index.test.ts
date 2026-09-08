// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { uiKeyToLang, langToUiKey } from './lang';
import { createTerminologyCatalogMerger, shouldMergeTerminology } from './merge';

type CallRecorder = { calls: unknown[][] };

function fnRecorder<T = undefined, A extends unknown[] = unknown[]>(
  impl?: (...args: A) => T | Promise<T>
): CallRecorder & ((...args: A) => T | Promise<T>) {
  const rec: CallRecorder & ((...args: A) => T | Promise<T>) = Object.assign(
    (...args: A) => {
      rec.calls.push(args);
      return impl ? impl(...args) : (undefined as T);
    },
    { calls: [] as unknown[][] }
  );
  return rec;
}

describe('uiKeyToLang / langToUiKey', () => {
  test('maps zh-CN ↔ zh_CN and en ↔ en_US', () => {
    expect(uiKeyToLang('zh-CN')).toBe('zh_CN');
    expect(langToUiKey('zh_CN')).toBe('zh-CN');
    expect(uiKeyToLang('en')).toBe('en_US');
    expect(langToUiKey('en_US')).toBe('en');
  });

  test('does not treat locale as lang (D12d)', () => {
    expect(uiKeyToLang('zh-CN')).not.toBe('zh-CN');
  });
});

describe('shouldMergeTerminology', () => {
  test('merges only when Gateway returned a fresh non-empty identified catalog', () => {
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

  test('does not merge an empty payload', () => {
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

  test('does not merge when unchanged', () => {
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

  test('does not merge on gatewayError so UI keeps msgid', () => {
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
  test('merges and invalidates once, ignores duplicates, and refreshes changed hashes', () => {
    const merge = fnRecorder();
    const notify = fnRecorder();
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
    expect(merge.calls.length).toBe(1);
    expect(notify.calls.length).toBe(1);

    expect(apply({
      ...first,
      hash: 'hash-2',
      messages: { base: { menu: { Settings: '系统设置' } } },
    }, 'zh-CN')).toBe(true);
    expect(merge.calls.length).toBe(2);
    expect(notify.calls.length).toBe(2);
  });

  test('does not consume identity when merging fails', () => {
    let n = 0;
    const merge = fnRecorder(() => {
      n += 1;
      if (n === 1) throw new Error('merge failed');
    });
    const notify = fnRecorder();
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
    expect(notify.calls.length).toBe(1);
  });
});
