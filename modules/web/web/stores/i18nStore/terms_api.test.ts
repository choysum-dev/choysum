// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import { fetchTerms, patchTerms, downloadTerminologyPo } from './terms_api';
import { componentHintFromScope } from './component_hint';

describe('componentHintFromScope', () => {
  it('strips location after @', () => {
    expect(componentHintFromScope('web/pages/Login@title')).toBe('web/pages/Login');
    expect(componentHintFromScope('game.rescue')).toBe('game.rescue');
    expect(componentHintFromScope('')).toBe('');
  });
});

describe('terms_api', () => {
  it('GET /web/i18n/terms with auth and query', async () => {
    const fetchImpl = vi.fn(async (url: string, init?: RequestInit) => {
      expect(url).toContain('/web/i18n/terms?');
      expect(url).toContain('lang=zh_CN');
      expect(url).toContain('application=auth');
      expect(url).toContain('q=Log');
      const headers = new Headers(init?.headers);
      expect(headers.get('Authorization')).toBe('Bearer t1');
      return new Response(
        JSON.stringify({
          lang: 'zh_CN',
          items: [{ application: 'auth', module: 'auth', scope: 'a@t', src: 'Log In', value: '登录', kind: 'literal' }],
          total: 1,
          limit: 50,
          offset: 0,
        }),
        { status: 200, headers: { 'content-type': 'application/json' } }
      );
    });

    const out = await fetchTerms(
      { lang: 'zh_CN', application: 'auth', q: 'Log' },
      { fetchImpl: fetchImpl as any, accessToken: 't1' }
    );
    expect(out.items).toHaveLength(1);
    expect(out.items[0].application).toBe('auth');
  });

  it('PATCH /web/i18n/terms with items body', async () => {
    const fetchImpl = vi.fn(async (url: string, init?: RequestInit) => {
      expect(url).toContain('/web/i18n/terms');
      expect(init?.method).toBe('PATCH');
      const body = JSON.parse(String(init?.body || '{}'));
      expect(body.lang).toBe('zh_CN');
      expect(body.items).toHaveLength(1);
      return new Response(JSON.stringify({ lang: 'zh_CN', items: body.items }), {
        status: 200,
        headers: { 'content-type': 'application/json' },
      });
    });

    const out = await patchTerms(
      {
        lang: 'zh_CN',
        items: [{ application: 'auth', module: 'auth', scope: 'a@t', src: 'Hello', value: '您好', kind: 'literal' }],
      },
      { fetchImpl: fetchImpl as any, accessToken: 't1' }
    );
    expect(out.items[0].value).toBe('您好');
  });

  it('GET /web/i18n/po downloads attachment blob', async () => {
    const fetchImpl = vi.fn(async (url: string, init?: RequestInit) => {
      expect(url).toContain('/web/i18n/po?');
      expect(url).toContain('lang=zh_CN');
      expect(url).toContain('application=auth');
      expect(init?.method).toBe('GET');
      return new Response('msgid ""\nmsgstr ""\n', {
        status: 200,
        headers: {
          'content-type': 'text/x-po; charset=utf-8',
          'content-disposition': 'attachment; filename="auth-zh_CN.po"',
        },
      });
    });

    const out = await downloadTerminologyPo(
      { lang: 'zh_CN', application: 'auth' },
      { fetchImpl: fetchImpl as any, accessToken: 't1' }
    );
    expect(out.filename).toBe('auth-zh_CN.po');
    expect(out.blob.size).toBeGreaterThan(0);
  });
});
