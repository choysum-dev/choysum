// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  normalizeOptionalString,
  asRecord,
  normalizeOptionalNonNegativeInt,
  normalizeChecksumSha256,
  normalizeContentType,
} from '@/core/service/utils/normalization';
import { toDate } from '@/core/service/utils/date';
import { GrpcCode } from '../error';
import { PrepareUploadResp, FinalizeUploadResp, UploadedPayloadRef, PrincipalContext } from '../contracts';
import { DocumentErrCode, throwDocumentError } from '../error';
import type AttachmentContent from './attachment_object';
import type AttachmentUploadSession from './upload_session';
import { requireText } from './_normalizers';
import { isDisallowedInlinePayloadID, DEFAULT_MAX_UPLOAD_BYTES, EMPTY_SHA256 } from './_upload';

// ---------------------------------------------------------------------------
// Error helpers
// ---------------------------------------------------------------------------

export function throwUploadSessionExpired(uploadId: string): never {
  throwDocumentError(DocumentErrCode.UPLOAD_SESSION_EXPIRED, 'Upload session has expired', GrpcCode.FailedPrecondition, { uploadId });
}

export function throwUploadSessionFinalized(uploadId: string): never {
  throwDocumentError(DocumentErrCode.UPLOAD_SESSION_FINALIZED, 'Upload session has already been finalized', GrpcCode.FailedPrecondition, { uploadId });
}

// ---------------------------------------------------------------------------
// Domain normalizers
// ---------------------------------------------------------------------------

export function normalizeAllowedMimeTypes(raw: unknown): string[] {
  if (Array.isArray(raw)) {
    return raw.map(item => normalizeContentType(item)).filter((item): item is string => Boolean(item));
  }

  if (raw && typeof raw === 'object') {
    return [];
  }

  const text = normalizeOptionalString(raw);
  if (!text) {
    return [];
  }

  try {
    const parsed = JSON.parse(text);
    if (Array.isArray(parsed)) {
      return parsed.map(item => normalizeContentType(item)).filter((item): item is string => Boolean(item));
    }
    if (typeof parsed === 'string') {
      const normalized = normalizeContentType(parsed);
      return normalized ? [normalized] : [];
    }
    return [];
  } catch {
    // Fallback to single content type token.
  }

  const normalized = normalizeContentType(text);
  return normalized ? [normalized] : [];
}

// ---------------------------------------------------------------------------
// Ticket builders
// ---------------------------------------------------------------------------

export function buildPayloadWriteTicket(session: AttachmentUploadSession, principal: PrincipalContext): string {
  const payload = {
    uploadId: requireText(session.Id, 'uploadId'),
    companyId: requireText(session.CompanyId, 'companyId'),
    userId: requireText(session.IssuerUserId, 'issuerUserId'),
    activeCompanyId: principal.activeCompanyId,
    status: requireText(session.Status, 'status'),
    expiresAt: toDate(session.ExpiresAt)?.toISOString(),
  };
  return JSON.stringify(payload);
}

// ---------------------------------------------------------------------------
// Payload-ref parsers
// ---------------------------------------------------------------------------

export function parseStoredContentPayloadRef(payloadId: string): string | undefined {
  const text = normalizeOptionalString(payloadId);
  if (!text) return undefined;

  const normalized = text.toLowerCase();
  if (!normalized.startsWith('sc:')) {
    return undefined;
  }

  const storedContentId = normalizeOptionalString(text.slice(3));
  return storedContentId || undefined;
}

export function buildUploadedPayloadRefFromPayloadId(payloadId: string): UploadedPayloadRef {
  const text = requireText(payloadId, 'payloadReceipt.payloadId');
  if (isDisallowedInlinePayloadID(text)) {
    throwDocumentError(
      DocumentErrCode.INVALID_ARGUMENT,
      'payloadReceipt.payloadId must be an opaque handle, inline byte payload is forbidden',
      GrpcCode.InvalidArgument,
      { field: 'payloadReceipt.payloadId' }
    );
  }

  const parsedStoredContentID = parseStoredContentPayloadRef(text);
  if (parsedStoredContentID) {
    return {
      kind: 'stored_content',
      storedContentId: parsedStoredContentID,
    };
  }

  throwDocumentError(DocumentErrCode.INVALID_ARGUMENT, 'payloadReceipt.payloadId format is unsupported', GrpcCode.InvalidArgument, {
    field: 'payloadReceipt.payloadId',
  });
}

export function normalizeUploadedPayloadRef(raw: unknown): UploadedPayloadRef | undefined {
  if (raw === undefined || raw === null) return undefined;

  if (typeof raw === 'string') {
    const text = normalizeOptionalString(raw);
    if (!text) {
      return undefined;
    }

    try {
      const parsed = JSON.parse(text) as Record<string, unknown>;
      return normalizeUploadedPayloadRef(parsed);
    } catch {
      return buildUploadedPayloadRefFromPayloadId(text);
    }
  }

  const record = asRecord(raw);
  if (!record) return undefined;

  const kind = normalizeOptionalString(record.kind)?.toLowerCase();
  if (kind === 'stored_content') {
    const storedContentId = normalizeOptionalString(record.storedContentId);
    if (!storedContentId) return undefined;
    return {
      kind: 'stored_content',
      storedContentId,
    };
  }

  const payloadId = normalizeOptionalString(record.payloadId);
  if (payloadId) {
    return buildUploadedPayloadRefFromPayloadId(payloadId);
  }

  return undefined;
}

// ---------------------------------------------------------------------------
// Session helpers
// ---------------------------------------------------------------------------

export function isSessionExpired(session: AttachmentUploadSession): boolean {
  const expiresAt = toDate(session.ExpiresAt);
  if (!expiresAt) return false;
  return expiresAt.getTime() <= Date.now();
}

// ---------------------------------------------------------------------------
// Response builders
// ---------------------------------------------------------------------------

export function buildPrepareUploadResp(uploadId: string, session: AttachmentUploadSession): PrepareUploadResp {
  const expiresAt = toDate(session.ExpiresAt);
  const maxUploadBytes = normalizeOptionalNonNegativeInt(session.MaxUploadBytes) ?? DEFAULT_MAX_UPLOAD_BYTES;
  return {
    uploadId,
    uploadTarget: {
      method: 'PUT',
      url: `/_document/uploads/${uploadId}`,
      expiresAt: expiresAt?.toISOString(),
      maxUploadBytes,
      requiredChecksumAlgorithm: 'sha256',
    },
  };
}

export function buildFinalizeResp(obj: AttachmentContent): FinalizeUploadResp {
  const imageWidth = normalizeOptionalNonNegativeInt(obj.ImageWidth);
  const imageHeight = normalizeOptionalNonNegativeInt(obj.ImageHeight);
  const imageFormat = normalizeOptionalString(obj.ImageFormat);

  return {
    attachmentObjectId: requireText(obj.Id, 'attachmentObjectId'),
    status: 'active',
    mimeType: normalizeOptionalString(obj.MimeType) ?? 'application/octet-stream',
    sizeBytes: normalizeOptionalNonNegativeInt(obj.SizeBytes) ?? 0,
    checksumSha256: normalizeChecksumSha256(obj.ChecksumSha256) ?? EMPTY_SHA256,
    imageMetadata:
      imageWidth !== undefined && imageHeight !== undefined && imageFormat
        ? {
            width: imageWidth,
            height: imageHeight,
            format: imageFormat,
          }
        : undefined,
  };
}
