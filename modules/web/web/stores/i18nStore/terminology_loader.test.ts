// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { uiKeyToLang, langToUiKey } from './lang';
import { fetchWebTranslations } from './terminology_loader';

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

describe('fetchWebTranslations', () => {
  test('requests lang+hash and returns payload', async () => {
    const fetchImpl = fnRecorder(async () => ({
      ok: true,
      json: async () => ({
        lang: 'zh_CN',
        locale: 'zh-CN',
        hash: 'abc',
        unchanged: false,
        messages: { auth: { 'a@t': { Hello: '你好' } } },
      }),
    }));

    const out = await fetchWebTranslations('zh_CN', 'prev', { fetchImpl: fetchImpl as any });
    expect(fetchImpl.calls.length).toBe(1);
    const url = String(fetchImpl.calls[0]![0]);
    expect(url).toContain('/web/i18n/translations?');
    expect(url).toContain('lang=zh_CN');
    expect(url).toContain('hash=prev');
    expect(url).not.toContain('moduleNames');
    expect(out.unchanged).toBe(false);
    expect(out.messages?.auth?.['a@t']?.Hello).toBe('你好');
  });

  test('nulls messages when unchanged', async () => {
    const fetchImpl = fnRecorder(async () => ({
      ok: true,
      json: async () => ({
        lang: 'zh_CN',
        locale: 'zh-CN',
        hash: 'abc',
        unchanged: true,
        messages: { should: 'ignore' },
      }),
    }));

    const out = await fetchWebTranslations('zh_CN', 'abc', { fetchImpl: fetchImpl as any });
    expect(out.unchanged).toBe(true);
    expect(out.messages).toBeNull();
  });

  test('throws when gateway fails', async () => {
    const fetchImpl = fnRecorder(async () => ({
      ok: false,
      status: 502,
      json: async () => ({}),
    }));

    await expect(fetchWebTranslations('en_US', undefined, { fetchImpl: fetchImpl as any })).rejects.toThrow(/502/);
  });
});
