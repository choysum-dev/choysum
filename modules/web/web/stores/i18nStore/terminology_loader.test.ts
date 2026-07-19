// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, afterEach } from 'vitest';

import { localeToLang, langToLocale } from './lang';
import { fetchWebTranslations } from './terminology_loader';

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

describe('fetchWebTranslations', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('requests lang+hash and returns payload', async () => {
    const fetchImpl = vi.fn(async () => ({
      ok: true,
      json: async () => ({
        lang: 'zh_CN',
        locale: 'zh-CN',
        hash: 'abc',
        unchanged: false,
        messages: { auth: { 'a@t': { Hello: '你好' } } },
      }),
    })) as unknown as typeof fetch;

    const out = await fetchWebTranslations('zh_CN', 'prev', { fetchImpl });
    expect(fetchImpl).toHaveBeenCalledOnce();
    const url = String((fetchImpl.mock.calls[0] as unknown[])[0]);
    expect(url).toContain('/web/i18n/translations?');
    expect(url).toContain('lang=zh_CN');
    expect(url).toContain('hash=prev');
    expect(url).not.toContain('moduleNames');
    expect(out.unchanged).toBe(false);
    expect(out.messages?.auth?.['a@t']?.Hello).toBe('你好');
  });

  it('nulls messages when unchanged', async () => {
    const fetchImpl = vi.fn(async () => ({
      ok: true,
      json: async () => ({
        lang: 'zh_CN',
        locale: 'zh-CN',
        hash: 'abc',
        unchanged: true,
        messages: { should: 'ignore' },
      }),
    })) as unknown as typeof fetch;

    const out = await fetchWebTranslations('zh_CN', 'abc', { fetchImpl });
    expect(out.unchanged).toBe(true);
    expect(out.messages).toBeNull();
  });

  it('throws when gateway fails', async () => {
    const fetchImpl = vi.fn(async () => ({
      ok: false,
      status: 502,
      json: async () => ({}),
    })) as unknown as typeof fetch;

    await expect(fetchWebTranslations('en_US', undefined, { fetchImpl })).rejects.toThrow(/502/);
  });
});
