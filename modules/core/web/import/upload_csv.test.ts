// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { clearGlobalRequestContextProvider, setGlobalRequestContextProvider } from '@/core/rpc/context';
import { setCSRFProvider as bindCSRFProvider, setTokenProvider as bindTokenProvider } from '@/core/web/rpc/providers';
import { registerStoreFactory } from '@/core/web/stores/registry';
import { uploadImportCsv, type UploadImportCsvOptions } from './upload_csv';

// QuickJS FE unit host may lack File / Headers / Blob.
(function polyfillUploadPrimitives() {
  const g = globalThis as any;
  if (typeof g.Blob !== 'function') {
    g.Blob = class Blob {
      parts: unknown[];
      type: string;
      size: number;
      constructor(parts: unknown[] = [], opts?: { type?: string }) {
        this.parts = parts;
        this.type = opts?.type || '';
        this.size = parts.reduce((n: number, p) => n + String(p ?? '').length, 0);
      }
      async arrayBuffer(): Promise<ArrayBuffer> {
        const text = this.parts.map(p => String(p ?? '')).join('');
        const bytes = new Uint8Array(text.length);
        for (let i = 0; i < text.length; i++) bytes[i] = text.charCodeAt(i) & 0xff;
        return bytes.buffer;
      }
    };
  }
  if (typeof g.File !== 'function') {
    g.File = class File extends g.Blob {
      name: string;
      constructor(parts: unknown[], name: string, opts?: { type?: string }) {
        super(parts, opts);
        this.name = name || '';
      }
    };
  }
  if (typeof g.Headers !== 'function') {
    g.Headers = class Headers {
      private map = new Map<string, string>();
      get(name: string): string | null {
        return this.map.get(String(name).toLowerCase()) ?? null;
      }
      has(name: string): boolean {
        return this.map.has(String(name).toLowerCase());
      }
      set(name: string, value: string): void {
        this.map.set(String(name).toLowerCase(), String(value));
      }
    };
  }
})();

type Call = { args: unknown[] };

function makeFn() {
  const calls: Call[] = [];
  let impl: ((...args: any[]) => any) | undefined;
  const fn = (...args: any[]) => {
    calls.push({ args });
    return impl ? impl(...args) : undefined;
  };
  return {
    fn,
    calls,
    reset() {
      calls.length = 0;
    },
    resolve(value: unknown) {
      impl = async () => value;
    },
    impl(next: (...args: any[]) => any) {
      impl = next;
    },
  };
}

type Harness = {
  prepareUpload: ReturnType<typeof makeFn>;
  finalizeUpload: ReturnType<typeof makeFn>;
  fetchFn: ReturnType<typeof makeFn>;
  setStore: (store: unknown) => void;
  setRequestContext: (ctx: unknown | (() => unknown)) => void;
  setCSRFProvider: (provider: unknown) => void;
  setTokenProvider: (provider: unknown) => void;
  upload: (options: Omit<UploadImportCsvOptions, 'fetch'>) => Promise<string>;
  clear: () => void;
};

function installHarness(): Harness {
  const prepareUpload = makeFn();
  const finalizeUpload = makeFn();
  const fetchFn = makeFn();
  fetchFn.resolve({ ok: true, status: 200 });

  let store: any = {
    PrepareUpload: prepareUpload.fn,
    FinalizeUpload: finalizeUpload.fn,
  };

  registerStoreFactory('document.AttachmentContent', () => store);
  setGlobalRequestContextProvider({ activeCompanyId: 'cmp-1', lang: 'en' });
  bindCSRFProvider(null);
  bindTokenProvider(null);

  return {
    prepareUpload,
    finalizeUpload,
    fetchFn,
    setStore(next) {
      store = next;
      registerStoreFactory('document.AttachmentContent', () => store);
    },
    setRequestContext(ctx) {
      if (typeof ctx === 'function') {
        setGlobalRequestContextProvider(ctx as () => Record<string, string>);
      } else if (ctx == null) {
        setGlobalRequestContextProvider(() => ({}));
      } else {
        setGlobalRequestContextProvider(ctx as Record<string, string>);
      }
    },
    setCSRFProvider(provider) {
      bindCSRFProvider(provider as any);
    },
    setTokenProvider(provider) {
      bindTokenProvider(provider as any);
    },
    upload(options) {
      return uploadImportCsv({ ...options, fetch: fetchFn.fn as any });
    },
    clear() {
      clearGlobalRequestContextProvider();
      bindCSRFProvider(null);
      bindTokenProvider(null);
    },
  };
}

function csvFile(content = 'Name,Code\nA,1\n', name = 'partners.csv', type = 'text/csv'): File {
  return new File([content], name, { type });
}

function withCrypto(value: unknown, run: () => Promise<void>): Promise<void> {
  const original = (globalThis as any).crypto;
  Object.defineProperty(globalThis, 'crypto', { value, configurable: true });
  return run().finally(() => {
    if (original === undefined) Reflect.deleteProperty(globalThis, 'crypto');
    else Object.defineProperty(globalThis, 'crypto', { value: original, configurable: true });
  });
}

test('uploadImportCsv: uploads CSV and returns attachment object id', async () => {
  const h = installHarness();
  try {
    h.prepareUpload.resolve({
      uploadId: 'upl-1',
      uploadTarget: { method: 'PUT', url: 'https://example/upload', headers: { 'Content-Type': 'text/csv' } },
    });
    h.finalizeUpload.resolve({ attachmentObjectId: 'att-obj-1' });

    const sourceRef = await h.upload({ ownerModel: 'partner.Partner', file: csvFile() });
    expect(sourceRef).toBe('att-obj-1');
    expect(h.prepareUpload.calls.length).toBe(1);
    expect(h.finalizeUpload.calls[0]?.args[0]).toMatchObject({ uploadId: 'upl-1' });
    expect(typeof (h.finalizeUpload.calls[0]?.args[0] as any).businessRequestId).toBe('string');
    expect(h.fetchFn.calls.length).toBe(1);
  } finally {
    h.clear();
  }
});

test('uploadImportCsv: applies internal upload auth headers', async () => {
  const h = installHarness();
  try {
    h.setCSRFProvider({ getCSRFToken: async () => 'csrf-token' });
    h.setTokenProvider({
      shouldRefreshToken: async () => false,
      getToken: async () => 'jwt-token',
    });
    h.prepareUpload.resolve({
      uploadId: 'upl-2',
      uploadTarget: { method: 'PUT', url: '/_document/uploads/upl-2', headers: {} },
    });
    h.finalizeUpload.resolve({ attachmentObjectId: 'att-obj-2' });

    await h.upload({ ownerModel: 'partner.Partner', file: csvFile('x'), businessRequestId: 'req.fixed' });

    const headers = (h.fetchFn.calls[0]?.args[1] as any).headers as Headers;
    expect(headers.get('Authorization')).toBe('Bearer jwt-token');
    expect(headers.get('X-XSRF-TOKEN')).toBe('csrf-token');
    expect(String(headers.get('baggage'))).toContain('ctx.activecompanyid');
  } finally {
    h.clear();
  }
});

test('uploadImportCsv: fails when prepare upload returns no target', async () => {
  const h = installHarness();
  try {
    h.prepareUpload.resolve({ uploadId: '' });
    await expectRejects(
      () => h.upload({ ownerModel: 'partner.Partner', file: csvFile('x') }),
      'PrepareUpload did not return upload target',
    );
  } finally {
    h.clear();
  }
});

test('uploadImportCsv: fails when upload target url is empty', async () => {
  const h = installHarness();
  try {
    h.prepareUpload.resolve({
      uploadId: 'upl-3',
      uploadTarget: { method: 'PUT', url: '', headers: {} },
    });
    await expectRejects(
      () => h.upload({ ownerModel: 'partner.Partner', file: csvFile('x') }),
      'upload target url is empty',
    );
  } finally {
    h.clear();
  }
});

test('uploadImportCsv: fails when upload uses unsupported method', async () => {
  const h = installHarness();
  try {
    h.prepareUpload.resolve({
      uploadId: 'upl-4',
      uploadTarget: { method: 'POST', url: 'https://example/upload', headers: {} },
    });
    await expectRejects(
      () => h.upload({ ownerModel: 'partner.Partner', file: csvFile('x') }),
      'unsupported upload method POST',
    );
  } finally {
    h.clear();
  }
});

test('uploadImportCsv: fails when finalize returns no attachment id', async () => {
  const h = installHarness();
  try {
    h.prepareUpload.resolve({
      uploadId: 'upl-5',
      uploadTarget: { method: 'PUT', url: 'https://example/upload', headers: {} },
    });
    h.finalizeUpload.resolve({});
    await expectRejects(
      () => h.upload({ ownerModel: 'partner.Partner', file: csvFile('x') }),
      'FinalizeUpload did not return attachmentObjectId',
    );
  } finally {
    h.clear();
  }
});

test('uploadImportCsv: fails when attachment service is unavailable', async () => {
  const h = installHarness();
  try {
    h.setStore({});
    await expectRejects(
      () => h.upload({ ownerModel: 'partner.Partner', file: csvFile('x') }),
      'document.AttachmentContent service is unavailable',
    );
  } finally {
    h.clear();
  }
});

test('uploadImportCsv: continues when sha256 digest fails', async () => {
  const h = installHarness();
  try {
    await withCrypto(
      {
        subtle: {
          digest: async () => {
            throw new Error('digest failed');
          },
        },
      },
      async () => {
        h.prepareUpload.resolve({
          uploadId: 'upl-7',
          uploadTarget: { method: 'PUT', url: 'https://example/upload', headers: {} },
        });
        h.finalizeUpload.resolve({ attachmentObjectId: 'att-obj-7' });
        const sourceRef = await h.upload({ ownerModel: 'partner.Partner', file: csvFile('x') });
        expect(sourceRef).toBe('att-obj-7');
      },
    );
  } finally {
    h.clear();
  }
});

test('uploadImportCsv: uploads without crypto.subtle and without randomUUID', async () => {
  const h = installHarness();
  try {
    await withCrypto({}, async () => {
      h.prepareUpload.resolve({
        uploadId: 'upl-8',
        uploadTarget: { method: 'PUT', url: 'https://example/upload', headers: {} },
      });
      h.finalizeUpload.resolve({ attachmentObjectId: 'att-obj-8' });
      const sourceRef = await h.upload({
        ownerModel: 'partner.Partner',
        file: csvFile('x'),
        fieldName: ' CustomField ',
      });
      expect(sourceRef).toBe('att-obj-8');
      expect((h.prepareUpload.calls[0]?.args[0] as any).fieldName).toBe('CustomField');
    });
  } finally {
    h.clear();
  }
});

test('uploadImportCsv: ignores auth provider failures and skips duplicate headers', async () => {
  const h = installHarness();
  try {
    h.setCSRFProvider({
      getCSRFToken: async () => {
        throw new Error('csrf failed');
      },
    });
    h.setTokenProvider({
      shouldRefreshToken: async () => {
        throw new Error('refresh failed');
      },
      refreshToken: async () => {},
      getToken: async () => {
        throw new Error('token failed');
      },
    });
    h.setRequestContext({
      activeCompanyId: 'cmp-1',
      emptyKey: '',
      'ctx.already': 'value',
    });
    h.prepareUpload.resolve({
      uploadId: 'upl-9',
      uploadTarget: {
        method: 'PUT',
        url: '/_document/uploads/upl-9',
        headers: { Authorization: 'Bearer preset', 'X-XSRF-TOKEN': 'preset', baggage: 'preset' },
      },
    });
    h.finalizeUpload.resolve({ attachmentObjectId: 'att-obj-9' });

    await h.upload({ ownerModel: 'partner.Partner', file: csvFile('x') });
    const headers = (h.fetchFn.calls[0]?.args[1] as any).headers as Headers;
    expect(headers.get('Authorization')).toBe('Bearer preset');
    expect(headers.get('baggage')).toBe('preset');
  } finally {
    h.clear();
  }
});

test('uploadImportCsv: refreshes token when provider says so', async () => {
  const h = installHarness();
  try {
    let refreshCalls = 0;
    h.setTokenProvider({
      shouldRefreshToken: async () => true,
      refreshToken: async () => {
        refreshCalls += 1;
      },
      getToken: async () => 'fresh-token',
    });
    h.prepareUpload.resolve({
      uploadId: 'upl-10',
      uploadTarget: { method: 'PUT', url: '/_document/uploads/upl-10', headers: {} },
    });
    h.finalizeUpload.resolve({ attachmentObjectId: 'att-obj-10' });

    await h.upload({ ownerModel: 'partner.Partner', file: csvFile('x') });
    expect(refreshCalls).toBe(1);
  } finally {
    h.clear();
  }
});

test('uploadImportCsv: skips baggage when request context lookup fails', async () => {
  const h = installHarness();
  try {
    h.setRequestContext(() => {
      throw new Error('ctx unavailable');
    });
    h.prepareUpload.resolve({
      uploadId: 'upl-11',
      uploadTarget: { method: 'PUT', url: '/_document/uploads/upl-11', headers: {} },
    });
    h.finalizeUpload.resolve({ attachmentObjectId: 'att-obj-11' });

    await h.upload({ ownerModel: 'partner.Partner', file: csvFile('x') });
    const headers = (h.fetchFn.calls[0]?.args[1] as any).headers as Headers;
    expect(headers.has('baggage')).toBe(false);
  } finally {
    h.clear();
  }
});

test('uploadImportCsv: computes sha256 checksum when crypto.subtle works', async () => {
  const h = installHarness();
  try {
    let digestCalls = 0;
    await withCrypto(
      {
        subtle: {
          digest: async () => {
            digestCalls += 1;
            return new Uint8Array([0xab, 0xcd]).buffer;
          },
        },
        randomUUID: () => 'uuid-fixed',
      },
      async () => {
        h.prepareUpload.resolve({
          uploadId: 'upl-12',
          uploadTarget: { method: 'PUT', url: 'https://example/upload', headers: { '': 'skip', ' ': 'skip2' } },
        });
        h.finalizeUpload.resolve({ attachmentObjectId: 'att-obj-12' });
        await h.upload({ ownerModel: 'partner.Partner', file: new File([''], '', { type: '' }) });
        expect(digestCalls).toBe(1);
        expect((h.prepareUpload.calls[0]?.args[0] as any).proposedFileName).toBe('import.csv');
        expect((h.prepareUpload.calls[0]?.args[0] as any).proposedContentType).toBe('text/csv');
        expect((h.prepareUpload.calls[0]?.args[0] as any).checksumSha256).toBe('abcd');
      },
    );
  } finally {
    h.clear();
  }
});

test('uploadImportCsv: uses fallback business request id and skips missing auth providers', async () => {
  const h = installHarness();
  try {
    h.setCSRFProvider(undefined);
    h.setTokenProvider(undefined);
    h.setRequestContext({});
    h.prepareUpload.resolve({
      uploadId: 'upl-13',
      uploadTarget: { method: '', url: '/_document/uploads/upl-13', headers: {} },
    });
    h.finalizeUpload.resolve({ attachmentObjectId: 'att-obj-13' });

    await h.upload({ ownerModel: 'partner.Partner', file: csvFile('x') });
    const init = h.fetchFn.calls[0]?.args[1] as any;
    expect(init.method).toBe('PUT');
    expect(init.credentials).toBe('include');
  } finally {
    h.clear();
  }
});

test('uploadImportCsv: skips baggage when all context values are empty', async () => {
  const h = installHarness();
  try {
    h.setRequestContext(null);
    h.prepareUpload.resolve({
      uploadId: 'upl-15',
      uploadTarget: { method: 'PUT', url: '/_document/uploads/upl-15', headers: { '': 'x', valid: '   ' } },
    });
    h.finalizeUpload.resolve({ attachmentObjectId: 'att-obj-15' });

    await h.upload({ ownerModel: 'partner.Partner', file: csvFile('x') });
    const headers = (h.fetchFn.calls[0]?.args[1] as any).headers as Headers;
    expect(headers.has('baggage')).toBe(false);
  } finally {
    h.clear();
  }
});

test('uploadImportCsv: uses randomUUID for business request id when available', async () => {
  const h = installHarness();
  try {
    await withCrypto(
      {
        randomUUID: () => 'req-uuid-1',
        subtle: { digest: async () => new ArrayBuffer(1) },
      },
      async () => {
        h.prepareUpload.resolve({
          uploadId: 'upl-16',
          uploadTarget: { method: 'PUT', url: 'https://example/upload', headers: undefined },
        });
        h.finalizeUpload.resolve({ attachmentObjectId: 'att-obj-16' });
        await h.upload({ ownerModel: 'partner.Partner', file: csvFile('x') });
        expect((h.prepareUpload.calls[0]?.args[0] as any).businessRequestId).toBe('import.csv.req-uuid-1');
      },
    );
  } finally {
    h.clear();
  }
});

test('uploadImportCsv: falls back when randomUUID returns empty', async () => {
  const h = installHarness();
  try {
    await withCrypto(
      {
        randomUUID: () => '',
        subtle: { digest: async () => new ArrayBuffer(1) },
      },
      async () => {
        h.prepareUpload.resolve({
          uploadId: 'upl-17',
          uploadTarget: { method: 'PUT', url: 'https://example/upload', headers: {} },
        });
        h.finalizeUpload.resolve({ attachmentObjectId: 'att-obj-17' });
        await h.upload({ ownerModel: 'partner.Partner', file: csvFile('x') });
        expect(String((h.prepareUpload.calls[0]?.args[0] as any).businessRequestId).startsWith('import.csv.')).toBe(true);
      },
    );
  } finally {
    h.clear();
  }
});

test('uploadImportCsv: ignores empty csrf and token values', async () => {
  const h = installHarness();
  try {
    h.setCSRFProvider({ getCSRFToken: async () => '   ' });
    h.setTokenProvider({ getToken: async () => '   ' });
    h.prepareUpload.resolve({
      uploadId: 'upl-14',
      uploadTarget: { method: 'PUT', url: '/_document/uploads/upl-14', headers: {} },
    });
    h.finalizeUpload.resolve({ attachmentObjectId: 'att-obj-14' });

    await h.upload({ ownerModel: 'partner.Partner', file: csvFile('x') });
    const headers = (h.fetchFn.calls[0]?.args[1] as any).headers as Headers;
    expect(headers.has('Authorization')).toBe(false);
    expect(headers.has('X-XSRF-TOKEN')).toBe(false);
  } finally {
    h.clear();
  }
});

test('uploadImportCsv: fails when upload HTTP response is not ok', async () => {
  const h = installHarness();
  try {
    h.prepareUpload.resolve({
      uploadId: 'upl-6',
      uploadTarget: { method: 'PUT', url: 'https://example/upload', headers: {} },
    });
    h.fetchFn.resolve({ ok: false, status: 500 });
    await expectRejects(
      () => h.upload({ ownerModel: 'partner.Partner', file: csvFile('x') }),
      'upload failed with HTTP 500',
    );
  } finally {
    h.clear();
  }
});
