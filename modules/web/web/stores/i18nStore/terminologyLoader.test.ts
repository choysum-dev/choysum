// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { fetchWebTranslations } from './terminologyLoader';
import { asyncFnRecorder } from '@/web/web/__tests__/mountApp';

describe('fetchWebTranslations', () => {
  test('requests lang+hash and returns payload', async () => {
    const fetchImpl = asyncFnRecorder(async () => ({
      ok: true,
      json: async () => ({
        lang: 'zh_CN',
        bcp47: 'zh-CN',
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
    expect(out.bcp47).toBe('zh-CN');
    expect(out.messages?.auth?.['a@t']?.Hello).toBe('你好');
  });

  test('nulls messages when unchanged', async () => {
    const fetchImpl = asyncFnRecorder(async () => ({
      ok: true,
      json: async () => ({
        lang: 'zh_CN',
        bcp47: 'zh-CN',
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
    const fetchImpl = asyncFnRecorder(async () => ({
      ok: false,
      status: 502,
      json: async () => ({}),
    }));

    await expectRejects(() => fetchWebTranslations('en_US', undefined, { fetchImpl: fetchImpl as any }), /502/);
  });
});