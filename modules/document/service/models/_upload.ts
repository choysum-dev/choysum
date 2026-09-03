// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  normalizeOptionalString,
  asRecord,
  normalizeOptionalNonNegativeInt,
  normalizeChecksumSha256,
  normalizeContentType,
} from '@/core/service/utils/normalization';
import { createTranslate } from '@/core/service/i18n';
import { GrpcCode } from '../error';
import { DocumentErrCode, throwDocumentError } from '../error';
import type { PrincipalContext, PrepareUploadReq, AuthorizeUploadPutReq, CommitUploadPutReq } from '../contracts';
import { requireText, requireUserId, requireCompanyId, assertPrincipal } from './_document_bridge';
import { DEFAULT_GLOBAL_MAX_UPLOAD_BYTES } from '@/core/service/orm/upload_limits';
import type AttachmentUploadSession from './upload_session';

const { _t } = createTranslate('document');

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

export const DEFAULT_UPLOAD_SESSION_TTL_SECONDS = 900;
/** @deprecated Use DEFAULT_GLOBAL_MAX_UPLOAD_BYTES from core ORM upload_limits. */
export const DEFAULT_MAX_UPLOAD_BYTES = DEFAULT_GLOBAL_MAX_UPLOAD_BYTES;
export { DEFAULT_GLOBAL_MAX_UPLOAD_BYTES };
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
  if (value === undefined || value === null) {
    throwDocumentError(
      DocumentErrCode.INVALID_ARGUMENT,
      _t('%s is required', { scope: 'service/models/_upload' }, fieldName),
      GrpcCode.InvalidArgument,
      { field: fieldName }
    );
  }
  const trimmed = typeof value === 'string' ? value.trim() : value;
  if (trimmed === '') {
    throwDocumentError(
      DocumentErrCode.INVALID_ARGUMENT,
      _t('%s is required', { scope: 'service/models/_upload' }, fieldName),
      GrpcCode.InvalidArgument,
      { field: fieldName }
    );
  }

  const num = Number(trimmed);
  if (!Number.isFinite(num) || num < 0) {
    throwDocumentError(
      DocumentErrCode.INVALID_ARGUMENT,
      _t('%s must be a non-negative integer', { scope: 'service/models/_upload' }, fieldName),
      GrpcCode.InvalidArgument,
      { field: fieldName }
    );
  }

  return Math.trunc(num);
}

export function isDisallowedInlinePayloadID(payloadId: string): boolean {
  return payloadId.startsWith('inline:') || payloadId.startsWith('inline_base64:') || payloadId.startsWith('data:') || payloadId.startsWith('s3:');
}

export function assertPayloadReceiptID(payloadId: unknown): string {
  const id = requireText(payloadId, 'payloadReceipt.payloadId');
  if (isDisallowedInlinePayloadID(id)) {
    throwDocumentError(
      DocumentErrCode.INVALID_ARGUMENT,
      _t('payloadReceipt.payloadId must be an opaque handle, inline byte payload is forbidden', {
        scope: 'service/models/_upload',
      }),
      GrpcCode.InvalidArgument,
      { field: 'payloadReceipt.payloadId' }
    );
  }
  return id;
}

// ---------------------------------------------------------------------------
// Request normalizers
// ---------------------------------------------------------------------------

export function assertPrepareUploadReq(req: PrepareUploadReq | undefined | null): NormalizedPrepareUploadReq {
  const ownerModel = requireText(req?.ownerModel, 'ownerModel');
  const fieldName = requireText(req?.fieldName, 'fieldName');
  const operation = requireText(req?.operation, 'operation');
  const businessRequestId = requireText(req?.businessRequestId, 'businessRequestId');

  if (operation !== 'create' && operation !== 'update') {
    throwDocumentError(
      DocumentErrCode.INVALID_ARGUMENT,
      _t('operation must be create or update', { scope: 'service/models/_upload' }),
      GrpcCode.InvalidArgument,
      { operation }
    );
  }

  const ownerRecordId = normalizeOptionalString(req?.ownerRecordId);
  if (operation === 'update' && !ownerRecordId) {
    throwDocumentError(
      DocumentErrCode.INVALID_ARGUMENT,
      _t('ownerRecordId is required when operation=update', { scope: 'service/models/_upload' }),
      GrpcCode.InvalidArgument
    );
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

export function assertAuthorizeUploadPutReq(req: AuthorizeUploadPutReq | undefined | null): NormalizedAuthorizeUploadPutReq {
  const uploadId = requireText(req?.uploadId, 'uploadId');
  const principal = assertPrincipal(req?.principal);
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

export function assertCommitUploadPutReq(req: CommitUploadPutReq | undefined | null): NormalizedCommitUploadPutReq {
  const uploadId = requireText(req?.uploadId, 'uploadId');
  const principal = assertPrincipal(req?.principal);
  const payloadReceipt = asRecord(req?.payloadReceipt);

  return {
    uploadId,
    principal,
    payloadReceipt: {
      payloadId: assertPayloadReceiptID(payloadReceipt?.payloadId),
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
    throwDocumentError(
      DocumentErrCode.PERMISSION_DENIED,
      _t('activeCompanyId does not match upload session owner', { scope: 'service/models/_upload' }),
      GrpcCode.PermissionDenied,
      {
        stage,
        uploadId: sessionId,
      }
    );
  }

  if (principal.userId !== sessionUserId) {
    throwDocumentError(
      DocumentErrCode.PERMISSION_DENIED,
      _t('userId does not match upload session issuer', { scope: 'service/models/_upload' }),
      GrpcCode.PermissionDenied,
      {
        stage,
        uploadId: sessionId,
      }
    );
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
      _t('businessRequestId was already used with a different upload context', { scope: 'service/models/_upload' }),
      GrpcCode.FailedPrecondition,
      { businessRequestId: req.businessRequestId, mismatches: mismatches.join(',') }
    );
  }
}
