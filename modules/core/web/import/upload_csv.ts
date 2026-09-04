// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getCurrentRequestContext } from '@/core/rpc/context';
import { getCSRFProvider, getTokenProvider } from '@/core/web/rpc/providers';
import { createStoreByModel } from '@/core/web/stores/registry';
import { normalizeOptionalString } from '@/core/service/utils/normalization';

type PrepareUploadReq = {
  ownerModel: string;
  fieldName: string;
  operation: 'create' | 'update';
  ownerRecordId?: string;
  businessRequestId: string;
  proposedFileName?: string;
  proposedContentType?: string;
  proposedSizeBytes?: number;
  checksumSha256?: string;
};

type PrepareUploadResp = {
  uploadId?: string;
  uploadTarget?: { method?: string; url?: string; headers?: Record<string, string> };
};

type FinalizeUploadResp = {
  attachmentObjectId?: string;
};

type AttachmentContentServiceLike = {
  PrepareUpload(req: PrepareUploadReq): Promise<PrepareUploadResp>;
  FinalizeUpload(req: { uploadId: string; businessRequestId: string }): Promise<FinalizeUploadResp>;
};

export type UploadImportCsvOptions = {
  ownerModel: string;
  fieldName?: string;
  file: File;
  businessRequestId?: string;
};

function newBusinessRequestId(prefix: string): string {
  const base =
    (typeof globalThis !== 'undefined' && typeof globalThis.crypto !== 'undefined' && typeof globalThis.crypto.randomUUID === 'function'
      ? globalThis.crypto.randomUUID()
      : `${Date.now()}_${Math.random().toString(36).slice(2)}`) || `${Date.now()}`;
  return `${prefix}.${base}`;
}

async function sha256Hex(blob: Blob): Promise<string | undefined> {
  const subtle = globalThis.crypto?.subtle;
  if (!subtle || typeof subtle.digest !== 'function') return undefined;
  try {
    const bytes = await blob.arrayBuffer();
    const digest = await subtle.digest('SHA-256', bytes);
    return Array.from(new Uint8Array(digest))
      .map(v => v.toString(16).padStart(2, '0'))
      .join('');
  } catch {
    return undefined;
  }
}

function resolveAttachmentContentService(): AttachmentContentServiceLike {
  const service = createStoreByModel('document.AttachmentContent') as unknown as AttachmentContentServiceLike;
  if (!service || typeof service.PrepareUpload !== 'function' || typeof service.FinalizeUpload !== 'function') {
    throw new Error('document.AttachmentContent service is unavailable');
  }
  return service;
}

async function applyInternalUploadAuthHeaders(headers: Headers): Promise<void> {
  if (!headers.has('x-xsrf-token')) {
    const csrfProvider = getCSRFProvider();
    if (csrfProvider) {
      try {
        const csrfToken = normalizeOptionalString(await csrfProvider.getCSRFToken());
        if (csrfToken) headers.set('X-XSRF-TOKEN', csrfToken);
      } catch {
        // best effort
      }
    }
  }

  if (!headers.has('authorization')) {
    const tokenProvider = getTokenProvider();
    if (tokenProvider) {
      try {
        const needRefresh = await tokenProvider.shouldRefreshToken?.();
        if (needRefresh) await tokenProvider.refreshToken();
      } catch {
        // best effort
      }
      try {
        const token = normalizeOptionalString(await tokenProvider.getToken());
        if (token) headers.set('Authorization', `Bearer ${token}`);
      } catch {
        // best effort
      }
    }
  }

  if (!headers.has('baggage')) {
    try {
      const ctx = getCurrentRequestContext();
      const pairs: string[] = [];
      for (const [key, value] of Object.entries(ctx || {})) {
        const normalizedKey = key.trim().toLowerCase();
        const normalizedValue = normalizeOptionalString(value);
        if (!normalizedKey || !normalizedValue) continue;
        const baggageKey = normalizedKey.startsWith('ctx.') ? normalizedKey : `ctx.${normalizedKey}`;
        pairs.push(`${encodeURIComponent(baggageKey)}=${encodeURIComponent(normalizedValue)}`);
      }
      if (pairs.length > 0) headers.set('baggage', pairs.join(','));
    } catch {
      // best effort
    }
  }
}

async function uploadToTarget(fieldName: string, target: NonNullable<PrepareUploadResp['uploadTarget']>, body: Blob): Promise<void> {
  const url = normalizeOptionalString(target?.url);
  if (!url) {
    throw new Error(`${fieldName}: upload target url is empty`);
  }
  const method = (normalizeOptionalString(target?.method) || 'PUT').toUpperCase();
  if (method !== 'PUT') {
    throw new Error(`${fieldName}: unsupported upload method ${method}`);
  }

  const headers = new Headers();
  for (const [k, v] of Object.entries(target?.headers || {})) {
    const key = normalizeOptionalString(k);
    const value = normalizeOptionalString(v);
    if (key && value) headers.set(key, value);
  }

  const internal = url.startsWith('/_document/uploads/');
  if (internal) {
    await applyInternalUploadAuthHeaders(headers);
  }

  const response = await fetch(url, {
    method,
    headers,
    body,
    credentials: internal ? 'include' : 'same-origin',
  });
  if (!response.ok) {
    throw new Error(`${fieldName}: upload failed with HTTP ${response.status}`);
  }
}

/**
 * Uploads a CSV file through the document staging flow and returns attachment content id for ImportHub sourceRef.
 */
export async function uploadImportCsv(options: UploadImportCsvOptions): Promise<string> {
  const service = resolveAttachmentContentService();
  const fieldName = normalizeOptionalString(options.fieldName) || 'ImportSource';
  const businessRequestId = normalizeOptionalString(options.businessRequestId) || newBusinessRequestId('import.csv');
  const file = options.file;
  const contentType = normalizeOptionalString(file.type) || 'text/csv';
  const checksumSha256 = await sha256Hex(file);

  const prepared = await service.PrepareUpload({
    ownerModel: options.ownerModel,
    fieldName,
    operation: 'create',
    businessRequestId,
    proposedFileName: normalizeOptionalString(file.name) || 'import.csv',
    proposedContentType: contentType,
    proposedSizeBytes: file.size,
    checksumSha256,
  });

  const uploadId = normalizeOptionalString(prepared.uploadId);
  if (!uploadId || !prepared.uploadTarget) {
    throw new Error('PrepareUpload did not return upload target');
  }

  await uploadToTarget(fieldName, prepared.uploadTarget, file);

  const finalized = await service.FinalizeUpload({ uploadId, businessRequestId });
  const sourceRef = normalizeOptionalString(finalized.attachmentObjectId);
  if (!sourceRef) {
    throw new Error('FinalizeUpload did not return attachmentObjectId');
  }
  return sourceRef;
}
