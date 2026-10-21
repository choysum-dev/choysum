// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { setTokenProvider } from '@/core/web/rpc';
import {
  downloadAttachmentFromUrl,
  fetchAuthenticatedPreviewAsDataUrl,
  isInternalDocumentBindingContentUrl,
  stripUrlQueryToken,
} from './attachmentPreviewAuth';

function resetTokenProvider() {
  setTokenProvider({
    getToken: async () => null,
    refreshToken: async () => false,
    shouldRefreshToken: async () => false,
  });
}

beforeEach(() => {
  resetTokenProvider();
});

afterEach(() => {
  resetTokenProvider();
  delete (globalThis as any).fetch;
});

test('isInternalDocumentBindingContentUrl: accepts binding content paths', () => {
  expect(isInternalDocumentBindingContentUrl('/_document/bindings/b1/content')).toBe(true);
  expect(isInternalDocumentBindingContentUrl('/_document/bindings/b1/content?token=x')).toBe(true);
  expect(isInternalDocumentBindingContentUrl('data:image/png;base64,aa')).toBe(false);
  expect(isInternalDocumentBindingContentUrl('/_document/uploads/u1')).toBe(false);
});

test('stripUrlQueryToken: removes token query param from relative urls', () => {
  expect(stripUrlQueryToken('/_document/bindings/b1/content?token=abc&x=1')).toBe(
    '/_document/bindings/b1/content?x=1'
  );
  expect(stripUrlQueryToken('/_document/bindings/b1/content?token=abc')).toBe('/_document/bindings/b1/content');
});

test('fetchAuthenticatedPreviewAsDataUrl: loads with Bearer and returns data url', async () => {
  setTokenProvider({
    getToken: async () => 'preview-token',
    refreshToken: async () => true,
    shouldRefreshToken: async () => false,
  });

  const png = new Uint8Array([0x89, 0x50, 0x4e, 0x47]);
  (globalThis as any).fetch = async (input: RequestInfo, init?: RequestInit) => {
    expect(String(input)).toBe('/_document/bindings/b1/content');
    expect((init?.headers as Record<string, string>)?.Authorization).toBe('Bearer preview-token');
    return {
      ok: true,
      headers: {
        get: (name: string) => (name.toLowerCase() === 'content-type' ? 'image/png' : null),
      },
      blob: async () => new Blob([png], { type: 'image/png' }),
    };
  };

  const dataUrl = await fetchAuthenticatedPreviewAsDataUrl('/_document/bindings/b1/content?token=stale');
  expect(dataUrl).toMatch(/^data:image\/png;base64,/);
});

test('fetchAuthenticatedPreviewAsDataUrl: returns undefined on non-ok response', async () => {
  setTokenProvider({
    getToken: async () => 'preview-token',
    refreshToken: async () => true,
    shouldRefreshToken: async () => false,
  });
  (globalThis as any).fetch = async () => ({
    ok: false,
    headers: { get: () => 'application/json' },
    blob: async () => new Blob([]),
  });

  expect(await fetchAuthenticatedPreviewAsDataUrl('/_document/bindings/b1/content')).toBeUndefined();
});

test('downloadAttachmentFromUrl: fetches internal binding and triggers blob download', async () => {
  setTokenProvider({
    getToken: async () => 'dl-token',
    refreshToken: async () => true,
    shouldRefreshToken: async () => false,
  });

  const png = new Uint8Array([0x89, 0x50, 0x4e, 0x47]);
  (globalThis as any).fetch = async (input: RequestInfo, init?: RequestInit) => {
    expect(String(input)).toBe('/_document/bindings/b1/content');
    expect((init?.headers as Record<string, string>)?.Authorization).toBe('Bearer dl-token');
    return {
      ok: true,
      headers: { get: () => 'image/png' },
      blob: async () => new Blob([png], { type: 'image/png' }),
    };
  };

  const clicks: Array<{ href: string; download: string }> = [];
  const originalCreateElement = document.createElement.bind(document);
  const originalCreateObjectURL = URL.createObjectURL.bind(URL);
  const originalRevokeObjectURL = URL.revokeObjectURL.bind(URL);
  URL.createObjectURL = (() => 'blob:mock-url') as typeof URL.createObjectURL;
  URL.revokeObjectURL = (() => undefined) as typeof URL.revokeObjectURL;
  document.createElement = ((tagName: string) => {
    const el = originalCreateElement(tagName);
    if (tagName.toLowerCase() === 'a') {
      el.click = () => {
        clicks.push({ href: el.getAttribute('href') || '', download: el.getAttribute('download') || '' });
      };
    }
    return el;
  }) as typeof document.createElement;

  try {
    expect(await downloadAttachmentFromUrl('/_document/bindings/b1/content?token=stale', 'photo.png')).toBe(true);
    expect(clicks).toEqual([{ href: 'blob:mock-url', download: 'photo.png' }]);
  } finally {
    document.createElement = originalCreateElement;
    URL.createObjectURL = originalCreateObjectURL;
    URL.revokeObjectURL = originalRevokeObjectURL;
  }
});
