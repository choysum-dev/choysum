// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { downloadTerminologyPo } from './po_download';
import { asyncFnRecorder } from '@/web/web/__tests__/mountApp';

describe('downloadTerminologyPo', () => {
  test('requires lang, application, and module before fetching', async () => {
    const fetchImpl = asyncFnRecorder();
    await expectRejects(
      () => downloadTerminologyPo({ lang: '', application: 'web', module: 'web', fetchImpl: fetchImpl as any }),
      'lang is required'
    );
    await expectRejects(
      () => downloadTerminologyPo({ lang: 'zh-CN', application: '', module: 'web', fetchImpl: fetchImpl as any }),
      'application is required'
    );
    await expectRejects(
      () => downloadTerminologyPo({ lang: 'zh-CN', application: 'web', module: '', fetchImpl: fetchImpl as any }),
      'module is required'
    );
    expect(fetchImpl.calls.length).toBe(0);
  });

  test('GETs /web/i18n/po with Bearer token and returns blob', async () => {
    const blob = new Blob(['msgid ""'], { type: 'text/x-po' });
    const fetchImpl = asyncFnRecorder(async () => ({
      ok: true,
      blob: async () => blob,
    }));

    const out = await downloadTerminologyPo({
      lang: 'zh-CN',
      application: 'web',
      module: 'web',
      accessToken: 'tok',
      fetchImpl: fetchImpl as any,
    });

    expect(out).toBe(blob);
    expect(fetchImpl.calls).toEqual([
      ['/web/i18n/po?lang=zh-CN&application=web&module=web', { headers: { Authorization: 'Bearer tok' } }],
    ]);
  });

  test('omits Authorization when accessToken is empty', async () => {
    const blob = new Blob(['x']);
    const fetchImpl = asyncFnRecorder(async () => ({
      ok: true,
      blob: async () => blob,
    }));

    await downloadTerminologyPo({
      lang: 'zh-CN',
      application: 'web',
      module: 'web',
      accessToken: '   ',
      fetchImpl: fetchImpl as any,
    });
    expect(fetchImpl.calls).toEqual([
      ['/web/i18n/po?lang=zh-CN&application=web&module=web', { headers: {} }],
    ]);
  });

  test('uses global fetch when fetchImpl is omitted', async () => {
    const blob = new Blob(['x']);
    const fetchImpl = asyncFnRecorder(async () => ({
      ok: true,
      blob: async () => blob,
    }));
    const prev = globalThis.fetch;
    (globalThis as any).fetch = fetchImpl;
    try {
      const out = await downloadTerminologyPo({
        lang: 'zh-CN',
        application: 'web',
        module: 'web',
      });
      expect(out).toBe(blob);
      expect(fetchImpl.calls.length).toBe(1);
    } finally {
      globalThis.fetch = prev;
    }
  });

  test('surfaces gateway error body', async () => {
    const fetchImpl = asyncFnRecorder(async () => ({
      ok: false,
      status: 400,
      statusText: 'Bad Request',
      json: async () => ({ error: 'module is required' }),
    }));

    await expectRejects(
      () => downloadTerminologyPo({ lang: 'zh-CN', application: 'web', module: 'web', fetchImpl: fetchImpl as any }),
      'module is required'
    );
  });

  test('falls back to statusText when error JSON is invalid', async () => {
    const fetchImpl = asyncFnRecorder(async () => ({
      ok: false,
      status: 502,
      statusText: 'Bad Gateway',
      json: async () => {
        throw new Error('not json');
      },
    }));

    await expectRejects(
      () => downloadTerminologyPo({ lang: 'zh-CN', application: 'web', module: 'web', fetchImpl: fetchImpl as any }),
      'Bad Gateway'
    );
  });

  test('falls back to status code when body and statusText are empty', async () => {
    const fetchImpl = asyncFnRecorder(async () => ({
      ok: false,
      status: 503,
      statusText: '',
      json: async () => ({}),
    }));

    await expectRejects(
      () => downloadTerminologyPo({ lang: 'zh-CN', application: 'web', module: 'web', fetchImpl: fetchImpl as any }),
      'PO download failed (503)'
    );
  });
});
