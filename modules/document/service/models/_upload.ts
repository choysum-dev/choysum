// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  normalizeOptionalString,
  asRecord,
  normalizeOptionalNonNegativeInt,
  normalizeChecksumSha256,
  normalizeContentType,
} from '@/core/service/utils/normalization';
import { GrpcCode } from '../error';
import { DocumentErrCode, throwDocumentError } from '../error';
import type { PrincipalContext, PrepareUploadReq, AuthorizeUploadPutReq, CommitUploadPutReq } from '../contracts';
import { requireText, requireUserId, requireCompanyId, normalizePrincipal } from './_normalizers';
import type AttachmentUploadSession from './upload_session';

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

export const DEFAULT_UPLOAD_SESSION_TTL_SECONDS = 900;
export const DEFAULT_MAX_UPLOAD_BYTES = 20 * 1024 * 1024;
export const EMPTY_SHA256 = 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855';

// ---------------------------------------------------------------------------
// Normalized request types
// ---------------------------------------------------------------------------

export type NormalizedPrepareUploadReq = {
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

export type NormalizedAuthorizeUploadPutReq = {
  uploadId: string;
  principal: PrincipalContext;
  requestMeta: {
    contentType?: string;
    contentLength?: number;
    checksumSha256?: string;
  };
};

export type NormalizedCommitUploadPutReq = {
  uploadId: string;
  principal: PrincipalContext;
  payloadReceipt: {
    payloadId: string;
    sizeBytes: number;
    checksumSha256: string;
    contentType?: string;
  };
};

// ---------------------------------------------------------------------------
// Pure normalizers
// ---------------------------------------------------------------------------

function parseRequiredNonNegativeInt(value: unknown, fieldName: string): number {
  const raw = normalizeOptionalString(value);
  if (raw === undefined) {
    throwDocumentError(DocumentErrCode.INVALID_ARGUMENT, `${fieldName} is required`, GrpcCode.InvalidArgument, { field: fieldName });
  }

  const num = Number(raw);
  if (!Number.isFinite(num) || num < 0) {
    throwDocumentError(DocumentErrCode.INVALID_ARGUMENT, `${fieldName} must be a non-negative integer`, GrpcCode.InvalidArgument, { field: fieldName });
  }

  return Math.trunc(num);
}

export function isDisallowedInlinePayloadID(payloadId: string): boolean {
  return payloadId.startsWith('inline:') || payloadId.startsWith('inline_base64:') || payloadId.startsWith('data:') || payloadId.startsWith('s3:');
}

export function normalizePayloadReceiptID(payloadId: unknown): string {
  const id = requireText(payloadId, 'payloadReceipt.payloadId');
  if (isDisallowedInlinePayloadID(id)) {
    throwDocumentError(
      DocumentErrCode.INVALID_ARGUMENT,
      'payloadReceipt.payloadId must be an opaque handle, inline byte payload is forbidden',
      GrpcCode.InvalidArgument,
      { field: 'payloadReceipt.payloadId' }
    );
  }
  return id;
}

// ---------------------------------------------------------------------------
// Request normalizers
// ---------------------------------------------------------------------------

export function normalizePrepareUploadReq(req: PrepareUploadReq | undefined | null): NormalizedPrepareUploadReq {
  const ownerModel = requireText(req?.ownerModel, 'ownerModel');
  const fieldName = requireText(req?.fieldName, 'fieldName');
  const operation = requireText(req?.operation, 'operation');
  const businessRequestId = requireText(req?.businessRequestId, 'businessRequestId');

  if (operation !== 'create' && operation !== 'update') {
    throwDocumentError(DocumentErrCode.INVALID_ARGUMENT, 'operation must be create or update', GrpcCode.InvalidArgument, { operation });
  }

  const ownerRecordId = normalizeOptionalString(req?.ownerRecordId);
  if (operation === 'update' && !ownerRecordId) {
    throwDocumentError(DocumentErrCode.INVALID_ARGUMENT, 'ownerRecordId is required when operation=update', GrpcCode.InvalidArgument);
  }

  return {
    ownerModel,
    fieldName,
    operation,
    ownerRecordId,
    businessRequestId,
    proposedFileName: normalizeOptionalString(req?.proposedFileName ?? req?.originalFileName),
    proposedContentType: normalizeOptionalString(req?.proposedContentType ?? req?.clientContentType),
    proposedSizeBytes: normalizeOptionalNonNegativeInt(req?.proposedSizeBytes),
    checksumSha256: normalizeChecksumSha256(req?.checksumSha256),
  };
}

export function normalizeAuthorizeUploadPutReq(req: AuthorizeUploadPutReq | undefined | null): NormalizedAuthorizeUploadPutReq {
  const uploadId = requireText(req?.uploadId, 'uploadId');
  const principal = normalizePrincipal(req?.principal);
  const requestMeta = asRecord(req?.requestMeta);

  return {
    uploadId,
    principal,
    requestMeta: {
      contentType: normalizeContentType(requestMeta?.contentType),
      contentLength:
        requestMeta && Object.prototype.hasOwnProperty.call(requestMeta, 'contentLength')
          ? parseRequiredNonNegativeInt(requestMeta.contentLength, 'requestMeta.contentLength')
          : undefined,
      checksumSha256: normalizeChecksumSha256(requestMeta?.checksumSha256),
    },
  };
}

export function normalizeCommitUploadPutReq(req: CommitUploadPutReq | undefined | null): NormalizedCommitUploadPutReq {
  const uploadId = requireText(req?.uploadId, 'uploadId');
  const principal = normalizePrincipal(req?.principal);
  const payloadReceipt = asRecord(req?.payloadReceipt);

  return {
    uploadId,
    principal,
    payloadReceipt: {
      payloadId: normalizePayloadReceiptID(payloadReceipt?.payloadId),
      sizeBytes: parseRequiredNonNegativeInt(payloadReceipt?.sizeBytes, 'payloadReceipt.sizeBytes'),
      checksumSha256: requireText(normalizeChecksumSha256(payloadReceipt?.checksumSha256), 'payloadReceipt.checksumSha256'),
      contentType: normalizeContentType(payloadReceipt?.contentType),
    },
  };
}

// ---------------------------------------------------------------------------
// Assert helpers
// ---------------------------------------------------------------------------

export function assertUploadSessionPrincipal(session: AttachmentUploadSession, principal: PrincipalContext, stage: string): void {
  const sessionCompanyId = requireText((session as any).CompanyId, 'companyId');
  const sessionUserId = requireText((session as any).IssuerUserId, 'issuerUserId');
  const sessionId = requireText((session as any).Id, 'uploadId');

  if (principal.activeCompanyId !== sessionCompanyId) {
    throwDocumentError(DocumentErrCode.PERMISSION_DENIED, 'activeCompanyId does not match upload session owner', GrpcCode.PermissionDenied, {
      stage,
      uploadId: sessionId,
    });
  }

  if (principal.userId !== sessionUserId) {
    throwDocumentError(DocumentErrCode.PERMISSION_DENIED, 'userId does not match upload session issuer', GrpcCode.PermissionDenied, {
      stage,
      uploadId: sessionId,
    });
  }
}

export function assertFinalizeIdentity(
  session: AttachmentUploadSession,
  runtimeUserId: unknown,
  runtimeCompanyId: unknown,
  runtimeCompanyIds: string[]
): PrincipalContext {
  const principal: PrincipalContext = {
    userId: requireUserId(runtimeUserId),
    activeCompanyId: requireCompanyId(runtimeCompanyId, 'finalize'),
    enabledCompanyIds: runtimeCompanyIds,
  };
  assertUploadSessionPrincipal(session, principal, 'finalize');
  return principal;
}

export function assertPrepareReplayConsistency(
  existing: AttachmentUploadSession,
  req: NormalizedPrepareUploadReq,
  companyId: string,
  issuerUserId: string
): void {
  const mismatches: string[] = [];

  if (requireText((existing as any).CompanyId, 'companyId') !== companyId) mismatches.push('companyId');
  if (requireText((existing as any).IssuerUserId, 'issuerUserId') !== issuerUserId) mismatches.push('issuerUserId');
  if (requireText((existing as any).OwnerModel, 'ownerModel') !== req.ownerModel) mismatches.push('ownerModel');
  if ((normalizeOptionalString((existing as any).OwnerRecordId) ?? '') !== (req.ownerRecordId ?? '')) mismatches.push('ownerRecordId');
  if (requireText((existing as any).FieldName, 'fieldName') !== req.fieldName) mismatches.push('fieldName');
  if (requireText((existing as any).Operation, 'operation') !== req.operation) mismatches.push('operation');

  if (mismatches.length > 0) {
    throwDocumentError(
      DocumentErrCode.IDEMPOTENCY_KEY_REUSED,
      'businessRequestId was already used with a different upload context',
      GrpcCode.FailedPrecondition,
      { businessRequestId: req.businessRequestId, mismatches: mismatches.join(',') }
    );
  }
}
