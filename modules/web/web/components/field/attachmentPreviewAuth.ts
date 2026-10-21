// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getTokenProvider } from '@/core/web/rpc/providers';
import { normalizeOptionalString } from '@/core/service/utils/normalization';

/** True for same-app document binding content URLs that need session auth for <img>. */
export function isInternalDocumentBindingContentUrl(rawUrl: string | undefined): boolean {
  const url = normalizeOptionalString(rawUrl);
  if (!url) return false;
  if (url.startsWith('data:') || url.startsWith('blob:')) return false;

  if (url.startsWith('/_document/bindings/')) {
    return url.includes('/content');
  }

  const locationHref = typeof globalThis !== 'undefined' ? globalThis.location?.href : undefined;
  const locationOrigin = typeof globalThis !== 'undefined' ? globalThis.location?.origin : undefined;
  if (!locationHref || !locationOrigin) return false;

  try {
    const parsed = new URL(url, locationHref);
    return parsed.origin === locationOrigin && parsed.pathname.startsWith('/_document/bindings/') && parsed.pathname.includes('/content');
  } catch {
    return false;
  }
}

/** Drop query token so Authorization header is the sole credential. */
export function stripUrlQueryToken(rawUrl: string): string {
  const url = normalizeOptionalString(rawUrl);
  if (!url) return '';

  const hashIdx = url.indexOf('#');
  const hash = hashIdx >= 0 ? url.slice(hashIdx) : '';
  const beforeHash = hashIdx >= 0 ? url.slice(0, hashIdx) : url;
  const qIdx = beforeHash.indexOf('?');
  if (qIdx < 0) return url;

  const path = beforeHash.slice(0, qIdx);
  const query = beforeHash.slice(qIdx + 1);
  const kept = query
    .split('&')
    .map(part => part.trim())
    .filter(Boolean)
    .filter(part => {
      const eq = part.indexOf('=');
      const key = decodeURIComponent((eq >= 0 ? part.slice(0, eq) : part).replace(/\+/g, ' '));
      return key !== 'token';
    });

  return kept.length ? `${path}?${kept.join('&')}${hash}` : `${path}${hash}`;
}

async function resolveBearerToken(): Promise<string | undefined> {
  const tokenProvider = getTokenProvider();
  if (!tokenProvider) return undefined;

  try {
    const needRefresh = await tokenProvider.shouldRefreshToken?.();
    if (needRefresh) {
      await tokenProvider.refreshToken();
    }
  } catch {
    // Keep best-effort behavior consistent with RPC auth interceptor.
  }

  try {
    return normalizeOptionalString(await tokenProvider.getToken());
  } catch {
    return undefined;
  }
}

async function blobToDataUrl(blob: Blob): Promise<string | undefined> {
  try {
    if (typeof blob.arrayBuffer === 'function' && typeof btoa === 'function') {
      const bytes = new Uint8Array(await blob.arrayBuffer());
      let binary = '';
      for (let i = 0; i < bytes.length; i += 1) {
        binary += String.fromCharCode(bytes[i]!);
      }
      const mime = normalizeOptionalString(blob.type) || 'application/octet-stream';
      return `data:${mime};base64,${btoa(binary)}`;
    }
  } catch {
    // Fall through to FileReader when arrayBuffer/btoa path is unavailable.
  }

  if (typeof FileReader === 'undefined') return undefined;
  return await new Promise(resolve => {
    const reader = new FileReader();
    reader.onload = () => {
      resolve(typeof reader.result === 'string' ? reader.result : undefined);
    };
    reader.onerror = () => resolve(undefined);
    reader.onabort = () => resolve(undefined);
    try {
      reader.readAsDataURL(blob);
    } catch {
      resolve(undefined);
    }
  });
}

async function fetchAuthenticatedAttachmentBlob(rawUrl: string): Promise<Blob | undefined> {
  if (typeof fetch !== 'function') return undefined;
  if (!isInternalDocumentBindingContentUrl(rawUrl)) return undefined;

  const token = await resolveBearerToken();
  if (!token) return undefined;

  const requestUrl = stripUrlQueryToken(rawUrl);
  if (!requestUrl) return undefined;

  try {
    const response = await fetch(requestUrl, {
      method: 'GET',
      headers: {
        Authorization: `Bearer ${token}`,
      },
      credentials: 'same-origin',
    });
    if (!response.ok) return undefined;
    const blob = await response.blob();
    if (!blob || blob.size <= 0) return undefined;
    return blob;
  } catch {
    return undefined;
  }
}

function triggerBlobDownload(blob: Blob, fileName?: string): boolean {
  if (typeof document === 'undefined' || typeof URL === 'undefined') return false;
  if (typeof URL.createObjectURL !== 'function') return false;

  const objectUrl = URL.createObjectURL(blob);
  try {
    const anchor = document.createElement('a');
    anchor.href = objectUrl;
    const name = normalizeOptionalString(fileName);
    if (name) anchor.download = name;
    else anchor.download = 'download';
    anchor.rel = 'noopener';
    anchor.style.display = 'none';
    document.body.appendChild(anchor);
    anchor.click();
    anchor.remove();
    return true;
  } catch {
    return false;
  } finally {
    try {
      URL.revokeObjectURL(objectUrl);
    } catch {
      // ignore revoke failures
    }
  }
}

function triggerHrefDownload(rawUrl: string, fileName?: string): boolean {
  if (typeof document === 'undefined') return false;
  const href = normalizeOptionalString(rawUrl);
  if (!href) return false;

  try {
    const anchor = document.createElement('a');
    anchor.href = href;
    const name = normalizeOptionalString(fileName);
    if (name) anchor.download = name;
    else anchor.download = '';
    anchor.rel = 'noopener';
    anchor.style.display = 'none';
    document.body.appendChild(anchor);
    anchor.click();
    anchor.remove();
    return true;
  } catch {
    return false;
  }
}

/**
 * Load an internal document binding content URL with Bearer auth and return a
 * data: URL so <img> works under CSP img-src ('self' data:) without cookies.
 */
export async function fetchAuthenticatedPreviewAsDataUrl(rawUrl: string): Promise<string | undefined> {
  const blob = await fetchAuthenticatedAttachmentBlob(rawUrl);
  if (!blob) return undefined;
  const contentType = normalizeOptionalString(blob.type)?.toLowerCase() || '';
  if (contentType && !contentType.startsWith('image/') && !contentType.startsWith('application/octet-stream')) {
    return undefined;
  }
  return await blobToDataUrl(blob);
}

/**
 * Download attachment bytes. Internal document binding URLs use Bearer auth +
 * blob save so Content-Disposition: inline cannot open a preview tab.
 */
export async function downloadAttachmentFromUrl(rawUrl: string, fileName?: string): Promise<boolean> {
  const href = normalizeOptionalString(rawUrl);
  if (!href) return false;

  if (isInternalDocumentBindingContentUrl(href)) {
    const blob = await fetchAuthenticatedAttachmentBlob(href);
    if (!blob) return false;
    return triggerBlobDownload(blob, fileName);
  }

  return triggerHrefDownload(href, fileName);
}
