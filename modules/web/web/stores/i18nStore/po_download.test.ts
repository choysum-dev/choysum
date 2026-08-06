// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';

import { downloadTerminologyPo } from './po_download';

describe('downloadTerminologyPo', () => {
  it('requires lang, application, and module before fetching', async () => {
    const fetchImpl = vi.fn();
    await expect(downloadTerminologyPo({ lang: '', application: 'web', module: 'web', fetchImpl })).rejects.toThrow(
      'lang is required'
    );
    await expect(downloadTerminologyPo({ lang: 'zh-CN', application: '', module: 'web', fetchImpl })).rejects.toThrow(
      'application is required'
    );
    await expect(downloadTerminologyPo({ lang: 'zh-CN', application: 'web', module: '', fetchImpl })).rejects.toThrow(
      'module is required'
    );
    expect(fetchImpl).not.toHaveBeenCalled();
  });

  it('GETs /web/i18n/po with Bearer token and returns blob', async () => {
    const blob = new Blob(['msgid ""'], { type: 'text/x-po' });
    const fetchImpl = vi.fn(async () => ({
      ok: true,
      blob: async () => blob,
    })) as unknown as typeof fetch;

    const out = await downloadTerminologyPo({
      lang: 'zh-CN',
      application: 'web',
      module: 'web',
      accessToken: 'tok',
      fetchImpl,
    });

    expect(out).toBe(blob);
    expect(fetchImpl).toHaveBeenCalledWith('/web/i18n/po?lang=zh-CN&application=web&module=web', {
      headers: { Authorization: 'Bearer tok' },
    });
  });

  it('surfaces gateway error body', async () => {
    const fetchImpl = vi.fn(async () => ({
      ok: false,
      status: 400,
      statusText: 'Bad Request',
      json: async () => ({ error: 'module is required' }),
    })) as unknown as typeof fetch;

    await expect(
      downloadTerminologyPo({ lang: 'zh-CN', application: 'web', module: 'web', fetchImpl })
    ).rejects.toThrow('module is required');
  });
});
