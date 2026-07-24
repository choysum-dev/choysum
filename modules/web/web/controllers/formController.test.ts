// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import { __resolveAttachmentFieldValueForTest } from './formController';
import { setTokenProvider, setCSRFProvider } from '@/core/web/rpc';
import { setGlobalRequestContextProvider, clearGlobalRequestContextProvider } from '@/core/rpc/context';

type AttachmentServiceStub = {
  PrepareUpload: ReturnType<typeof vi.fn>;
  FinalizeUpload: ReturnType<typeof vi.fn>;
  setContext?: (ctx: Record<string, string>) => void;
  withContext?: <T>(ctx: Record<string, string>, fn: () => Promise<T>) => Promise<T>;
};

function newAttachmentService(attachmentObjectId = 'ao-test', uploadUrl = 'https://example.com/upload'): AttachmentServiceStub {
  return {
    PrepareUpload: vi.fn(async () => ({
      uploadId: 'upload-1',
      uploadTarget: {
        method: 'PUT',
        url: uploadUrl,
        headers: {
          'content-type': 'application/octet-stream',
        },
      },
    })),
    FinalizeUpload: vi.fn(async () => ({
      attachmentObjectId,
    })),
  };
}

function newCtx(service: AttachmentServiceStub) {
  return {
    operation: 'create',
    ownerModel: 'demo.Asset',
    ownerRecordId: 'RID-1',
    fieldName: 'Avatar',
    service,
    store: {
      fullModelName: 'demo.Asset',
      storeId: 'demo.Asset',
      getContext: () => ({}),
    },
  } as any;
}

describe('formController attachment protocol', () => {
  const originalFetch = globalThis.fetch;
  const originalFile = (globalThis as any).File;

  const resetTokenProvider = () => {
    setTokenProvider({
      getToken: async () => null,
      refreshToken: async () => false,
      shouldRefreshToken: async () => false,
    });
  };

  const resetCSRFProvider = () => {
    setCSRFProvider({
      getCSRFToken: async () => null,
    });
  };

  beforeEach(() => {
    globalThis.fetch = vi.fn(async () => ({ ok: true, status: 200 })) as any;
    resetTokenProvider();
    resetCSRFProvider();
    clearGlobalRequestContextProvider();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    globalThis.fetch = originalFetch;
    resetTokenProvider();
    resetCSRFProvider();
    clearGlobalRequestContextProvider();
    if (originalFile === undefined) {
      delete (globalThis as any).File;
    } else {
      (globalThis as any).File = originalFile;
    }
  });

  test('accepts Blob payload and executes Prepare -> PUT -> Finalize', async () => {
    const service = newAttachmentService('ao-blob');
    const ctx = newCtx(service);
    const blob = new Blob([new Uint8Array([1, 2, 3])], { type: 'application/octet-stream' });

    const resolved = await __resolveAttachmentFieldValueForTest(blob, ctx);

    expect(resolved).toEqual({ kind: 'set', attachmentObjectId: 'ao-blob' });
    expect(service.PrepareUpload).toHaveBeenCalledTimes(1);
    expect(service.FinalizeUpload).toHaveBeenCalledTimes(1);
    expect(globalThis.fetch as any).toHaveBeenCalledTimes(1);
  });

  test('accepts File payload and keeps filename in PrepareUpload request', async () => {
    class TestFile extends Blob {
      name: string;
      constructor(parts: BlobPart[], fileName: string, options?: BlobPropertyBag) {
        super(parts, options);
        this.name = fileName;
      }
    }
    (globalThis as any).File = TestFile;

    const service = newAttachmentService('ao-file');
    const ctx = newCtx(service);
    const file = new (globalThis as any).File([new Uint8Array([7, 8, 9])], 'avatar.png', { type: 'image/png' });

    const resolved = await __resolveAttachmentFieldValueForTest(file, ctx);

    expect(resolved).toEqual({ kind: 'set', attachmentObjectId: 'ao-file' });
    expect(service.PrepareUpload).toHaveBeenCalledTimes(1);
    const prepareReq = service.PrepareUpload.mock.calls[0]?.[0] as any;
    expect(prepareReq?.proposedFileName).toBe('avatar.png');
  });

  test('supports kind=set/clear/noop envelope protocol', async () => {
    const service = newAttachmentService();
    const ctx = newCtx(service);

    await expect(__resolveAttachmentFieldValueForTest({ kind: 'set', attachmentObjectId: 'ao-kind-set' }, ctx)).resolves.toEqual({
      kind: 'set',
      attachmentObjectId: 'ao-kind-set',
    });

    await expect(__resolveAttachmentFieldValueForTest({ kind: 'clear' }, ctx)).resolves.toEqual({ kind: 'clear' });
    await expect(__resolveAttachmentFieldValueForTest({ kind: 'noop' }, ctx)).resolves.toEqual({ kind: 'omit' });

    expect(service.PrepareUpload).not.toHaveBeenCalled();
    expect(service.FinalizeUpload).not.toHaveBeenCalled();
  });

  test('rejects array payload for binary/image fields', async () => {
    const service = newAttachmentService();
    const ctx = newCtx(service);

    await expect(__resolveAttachmentFieldValueForTest([], ctx)).rejects.toThrow('[Attachment] Avatar: array payload is not supported for binary/image fields.');
  });

  test('internal upload target carries auth/context headers and include credentials', async () => {
    const service = newAttachmentService('ao-auth', '/_document/uploads/upload-1');
    const ctx = newCtx(service);
    const blob = new Blob([new Uint8Array([10, 11, 12])], { type: 'application/octet-stream' });

    const refreshToken = vi.fn(async () => true);
    setTokenProvider({
      getToken: async () => 'token-upload',
      refreshToken,
      shouldRefreshToken: async () => true,
    });
    setCSRFProvider({
      getCSRFToken: async () => 'csrf-upload',
    });
    setGlobalRequestContextProvider({ activeCompanyId: 'company_a', tz: 'Asia/Shanghai' });

    const resolved = await __resolveAttachmentFieldValueForTest(blob, ctx);

    expect(resolved).toEqual({ kind: 'set', attachmentObjectId: 'ao-auth' });
    expect(refreshToken).toHaveBeenCalledTimes(1);

    const fetchCall = (globalThis.fetch as any).mock.calls[0] as [string, RequestInit];
    const [, requestInit] = fetchCall;
    const headers = requestInit.headers as Headers;
    expect(requestInit.credentials).toBe('include');
    expect(headers.get('x-xsrf-token')).toBe('csrf-upload');
    expect(headers.get('authorization')).toBe('Bearer token-upload');
    expect(headers.get('baggage')).toContain('ctx.activecompanyid=company_a');
    expect(headers.get('baggage')).toContain('ctx.tz=Asia%2FShanghai');
  });
});
