// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { bcp47FromCode } from './bcp47';
import { createTerminologyCatalogMerger, shouldMergeTerminology } from './merge';
import { fnRecorder } from '@/web/web/__tests__/mountApp';

describe('bcp47FromCode', () => {
  test('maps zh_CN and en_US to BCP 47', () => {
    expect(bcp47FromCode('zh_CN')).toBe('zh-CN');
    expect(bcp47FromCode('en_US')).toBe('en-US');
  });
});

describe('shouldMergeTerminology', () => {
  test('merges only when Gateway returned a fresh non-empty identified catalog', () => {
    expect(
      shouldMergeTerminology({
        lang: 'zh_CN',
        bcp47: 'zh-CN',
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
        bcp47: 'zh-CN',
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
        bcp47: 'zh-CN',
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
        bcp47: 'zh-CN',
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
      bcp47: 'zh-CN',
      hash: 'hash-1',
      unchanged: false,
      messages: { base: { menu: { Settings: '设置' } } },
    };

    expect(apply(first, 'zh_CN')).toBe(true);
    expect(apply(first, 'zh_CN')).toBe(false);
    expect(merge.calls.length).toBe(1);
    expect(notify.calls.length).toBe(1);

    expect(apply({
      ...first,
      hash: 'hash-2',
      messages: { base: { menu: { Settings: '系统设置' } } },
    }, 'zh_CN')).toBe(true);
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
      bcp47: 'zh-CN',
      hash: 'hash-1',
      unchanged: false,
      messages: { base: { menu: { Settings: '设置' } } },
    };

    expect(() => apply(load, 'zh_CN')).toThrow('merge failed');
    expect(apply(load, 'zh_CN')).toBe(true);
    expect(notify.calls.length).toBe(1);
  });
});