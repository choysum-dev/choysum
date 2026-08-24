// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it, vi } from 'vitest';

const prepareUpload = vi.fn();
const finalizeUpload = vi.fn();

vi.mock('@/web/web/stores/registry', () => ({
  createStoreByModel: vi.fn(() => ({
    PrepareUpload: prepareUpload,
    FinalizeUpload: finalizeUpload,
  })),
}));

vi.mock('@/core/rpc/context', () => ({
  getCurrentRequestContext: vi.fn(() => ({ activeCompanyId: 'cmp-1', lang: 'en' })),
}));

const getCSRFProvider = vi.fn();
const getTokenProvider = vi.fn();
vi.mock('@/core/web/rpc/providers', () => ({
  getCSRFProvider,
  getTokenProvider,
}));

describe('uploadImportCsv', () => {
  beforeEach(() => {
    prepareUpload.mockReset();
    finalizeUpload.mockReset();
    getCSRFProvider.mockReset();
    getTokenProvider.mockReset();
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => ({ ok: true, status: 200 }) as Response),
    );
  });

  it('uploads CSV and returns attachment object id', async () => {
    prepareUpload.mockResolvedValue({
      uploadId: 'upl-1',
      uploadTarget: { method: 'PUT', url: 'https://example/upload', headers: { 'Content-Type': 'text/csv' } },
    });
    finalizeUpload.mockResolvedValue({ attachmentObjectId: 'att-obj-1' });

    const { uploadImportCsv } = await import('./upload_csv');
    const file = new File(['Name,Code\nA,1\n'], 'partners.csv', { type: 'text/csv' });
    const sourceRef = await uploadImportCsv({ ownerModel: 'partner.Partner', file });
    expect(sourceRef).toBe('att-obj-1');
    expect(prepareUpload).toHaveBeenCalled();
    expect(finalizeUpload).toHaveBeenCalledWith({ uploadId: 'upl-1', businessRequestId: expect.any(String) });
    expect(fetch).toHaveBeenCalled();
  });

  it('applies internal upload auth headers', async () => {
    getCSRFProvider.mockReturnValue({ getCSRFToken: vi.fn(async () => 'csrf-token') });
    getTokenProvider.mockReturnValue({
      shouldRefreshToken: vi.fn(async () => false),
      getToken: vi.fn(async () => 'jwt-token'),
    });
    prepareUpload.mockResolvedValue({
      uploadId: 'upl-2',
      uploadTarget: { method: 'PUT', url: '/_document/uploads/upl-2', headers: {} },
    });
    finalizeUpload.mockResolvedValue({ attachmentObjectId: 'att-obj-2' });

    const { uploadImportCsv } = await import('./upload_csv');
    const file = new File(['x'], 'partners.csv', { type: 'text/csv' });
    await uploadImportCsv({ ownerModel: 'partner.Partner', file, businessRequestId: 'req.fixed' });

    const fetchCall = (fetch as any).mock.calls[0];
    const headers = fetchCall[1].headers as Headers;
    expect(headers.get('Authorization')).toBe('Bearer jwt-token');
    expect(headers.get('X-XSRF-TOKEN')).toBe('csrf-token');
    expect(headers.get('baggage')).toContain('ctx.activecompanyid');
  });

  it('fails when prepare upload returns no target', async () => {
    prepareUpload.mockResolvedValue({ uploadId: '' });
    const { uploadImportCsv } = await import('./upload_csv');
    const file = new File(['x'], 'partners.csv', { type: 'text/csv' });
    await expect(uploadImportCsv({ ownerModel: 'partner.Partner', file })).rejects.toThrow('PrepareUpload did not return upload target');
  });

  it('fails when upload target url is empty', async () => {
    prepareUpload.mockResolvedValue({
      uploadId: 'upl-3',
      uploadTarget: { method: 'PUT', url: '', headers: {} },
    });
    const { uploadImportCsv } = await import('./upload_csv');
    const file = new File(['x'], 'partners.csv', { type: 'text/csv' });
    await expect(uploadImportCsv({ ownerModel: 'partner.Partner', file })).rejects.toThrow('upload target url is empty');
  });

  it('fails when upload uses unsupported method', async () => {
    prepareUpload.mockResolvedValue({
      uploadId: 'upl-4',
      uploadTarget: { method: 'POST', url: 'https://example/upload', headers: {} },
    });
    const { uploadImportCsv } = await import('./upload_csv');
    const file = new File(['x'], 'partners.csv', { type: 'text/csv' });
    await expect(uploadImportCsv({ ownerModel: 'partner.Partner', file })).rejects.toThrow('unsupported upload method POST');
  });

  it('fails when finalize returns no attachment id', async () => {
    prepareUpload.mockResolvedValue({
      uploadId: 'upl-5',
      uploadTarget: { method: 'PUT', url: 'https://example/upload', headers: {} },
    });
    finalizeUpload.mockResolvedValue({});
    const { uploadImportCsv } = await import('./upload_csv');
    const file = new File(['x'], 'partners.csv', { type: 'text/csv' });
    await expect(uploadImportCsv({ ownerModel: 'partner.Partner', file })).rejects.toThrow('FinalizeUpload did not return attachmentObjectId');
  });

  it('fails when attachment service is unavailable', async () => {
    const registry = await import('@/web/web/stores/registry');
    (registry.createStoreByModel as any).mockReturnValueOnce({});
    const { uploadImportCsv } = await import('./upload_csv');
    const file = new File(['x'], 'partners.csv', { type: 'text/csv' });
    await expect(uploadImportCsv({ ownerModel: 'partner.Partner', file })).rejects.toThrow('document.AttachmentContent service is unavailable');
  });

  it('continues when sha256 digest fails', async () => {
    const originalCrypto = globalThis.crypto;
    try {
      Object.defineProperty(globalThis, 'crypto', {
        value: {
          subtle: {
            digest: vi.fn(async () => {
              throw new Error('digest failed');
            }),
          },
        },
        configurable: true,
      });
      prepareUpload.mockResolvedValue({
        uploadId: 'upl-7',
        uploadTarget: { method: 'PUT', url: 'https://example/upload', headers: {} },
      });
      finalizeUpload.mockResolvedValue({ attachmentObjectId: 'att-obj-7' });
      const { uploadImportCsv } = await import('./upload_csv');
      const file = new File(['x'], 'partners.csv', { type: 'text/csv' });
      await expect(uploadImportCsv({ ownerModel: 'partner.Partner', file })).resolves.toBe('att-obj-7');
    } finally {
      if (originalCrypto === undefined) {
        Reflect.deleteProperty(globalThis, 'crypto');
      } else {
        Object.defineProperty(globalThis, 'crypto', { value: originalCrypto, configurable: true });
      }
    }
  });

  it('uploads without crypto.subtle and without randomUUID', async () => {
    const originalCrypto = globalThis.crypto;
    try {
      Object.defineProperty(globalThis, 'crypto', { value: {}, configurable: true });
      prepareUpload.mockResolvedValue({
        uploadId: 'upl-8',
        uploadTarget: { method: 'PUT', url: 'https://example/upload', headers: {} },
      });
      finalizeUpload.mockResolvedValue({ attachmentObjectId: 'att-obj-8' });
      const { uploadImportCsv } = await import('./upload_csv');
      const file = new File(['x'], 'partners.csv', { type: 'text/csv' });
      await expect(uploadImportCsv({ ownerModel: 'partner.Partner', file, fieldName: ' CustomField ' })).resolves.toBe('att-obj-8');
      expect(prepareUpload.mock.calls[0][0].fieldName).toBe('CustomField');
    } finally {
      if (originalCrypto === undefined) {
        Reflect.deleteProperty(globalThis, 'crypto');
      } else {
        Object.defineProperty(globalThis, 'crypto', { value: originalCrypto, configurable: true });
      }
    }
  });

  it('ignores auth provider failures and skips duplicate headers', async () => {
    getCSRFProvider.mockReturnValue({
      getCSRFToken: vi.fn(async () => {
        throw new Error('csrf failed');
      }),
    });
    getTokenProvider.mockReturnValue({
      shouldRefreshToken: vi.fn(async () => {
        throw new Error('refresh failed');
      }),
      refreshToken: vi.fn(async () => {}),
      getToken: vi.fn(async () => {
        throw new Error('token failed');
      }),
    });
    prepareUpload.mockResolvedValue({
      uploadId: 'upl-9',
      uploadTarget: {
        method: 'PUT',
        url: '/_document/uploads/upl-9',
        headers: { Authorization: 'Bearer preset', 'X-XSRF-TOKEN': 'preset', baggage: 'preset' },
      },
    });
    finalizeUpload.mockResolvedValue({ attachmentObjectId: 'att-obj-9' });
    const ctx = await import('@/core/rpc/context');
    (ctx.getCurrentRequestContext as any).mockReturnValueOnce({
      activeCompanyId: 'cmp-1',
      emptyKey: '',
      'ctx.already': 'value',
    });
    const { uploadImportCsv } = await import('./upload_csv');
    const file = new File(['x'], 'partners.csv', { type: 'text/csv' });
    await uploadImportCsv({ ownerModel: 'partner.Partner', file });
    const fetchCall = (fetch as any).mock.calls[0];
    const headers = fetchCall[1].headers as Headers;
    expect(headers.get('Authorization')).toBe('Bearer preset');
    expect(headers.get('baggage')).toBe('preset');
  });

  it('refreshes token when provider says so', async () => {
    const refreshToken = vi.fn(async () => {});
    getTokenProvider.mockReturnValue({
      shouldRefreshToken: vi.fn(async () => true),
      refreshToken,
      getToken: vi.fn(async () => 'fresh-token'),
    });
    prepareUpload.mockResolvedValue({
      uploadId: 'upl-10',
      uploadTarget: { method: 'PUT', url: '/_document/uploads/upl-10', headers: {} },
    });
    finalizeUpload.mockResolvedValue({ attachmentObjectId: 'att-obj-10' });
    const { uploadImportCsv } = await import('./upload_csv');
    const file = new File(['x'], 'partners.csv', { type: 'text/csv' });
    await uploadImportCsv({ ownerModel: 'partner.Partner', file });
    expect(refreshToken).toHaveBeenCalled();
  });

  it('skips baggage when request context lookup fails', async () => {
    const ctx = await import('@/core/rpc/context');
    (ctx.getCurrentRequestContext as any).mockImplementationOnce(() => {
      throw new Error('ctx unavailable');
    });
    prepareUpload.mockResolvedValue({
      uploadId: 'upl-11',
      uploadTarget: { method: 'PUT', url: '/_document/uploads/upl-11', headers: {} },
    });
    finalizeUpload.mockResolvedValue({ attachmentObjectId: 'att-obj-11' });
    const { uploadImportCsv } = await import('./upload_csv');
    const file = new File(['x'], 'partners.csv', { type: 'text/csv' });
    await uploadImportCsv({ ownerModel: 'partner.Partner', file });
    const headers = (fetch as any).mock.calls[0][1].headers as Headers;
    expect(headers.has('baggage')).toBe(false);
  });

  it('fails when upload HTTP response is not ok', async () => {
    prepareUpload.mockResolvedValue({
      uploadId: 'upl-6',
      uploadTarget: { method: 'PUT', url: 'https://example/upload', headers: {} },
    });
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => ({ ok: false, status: 500 }) as Response),
    );
    const { uploadImportCsv } = await import('./upload_csv');
    const file = new File(['x'], 'partners.csv', { type: 'text/csv' });
    await expect(uploadImportCsv({ ownerModel: 'partner.Partner', file })).rejects.toThrow('upload failed with HTTP 500');
  });
});
