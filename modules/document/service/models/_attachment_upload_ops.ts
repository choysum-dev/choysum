// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  normalizeOptionalString,
  asRecord,
  normalizeOptionalNonNegativeInt,
  normalizeChecksumSha256,
  normalizeContentType,
} from '@/core/service/utils/normalization';
import { parseISODate } from '@/core/service/utils/date';
import { GrpcCode } from '../error';
import {
  PrepareUploadReq,
  PrepareUploadResp,
  FinalizeUploadReq,
  FinalizeUploadResp,
  UploadedPayloadRef,
  AuthorizeUploadPutReq,
  AuthorizeUploadPutResp,
  CommitUploadPutReq,
  CommitUploadPutResp,
  PrincipalContext,
} from '../contracts';
import { DocumentErrCode, throwDocumentError } from '../error';
import type AttachmentContent from './attachment_object';
import type AttachmentUploadSession from './upload_session';
import type AttachmentMutationLedger from './attachment_mutation_ledger';
import type StoredContent from './stored_content';
import { assertOwnerWriteAuthorization } from './_owner_authorization';
import { requireText, requireUserId, requireCompanyId, mustLoadOne } from './_normalizers';
import { garbageCollectUnboundObjects } from './_attachment_gc';
import { isMimeTypeAllowed } from '@/core/service/utils/mime';
import {
  DEFAULT_UPLOAD_SESSION_TTL_SECONDS,
  DEFAULT_MAX_UPLOAD_BYTES,
  EMPTY_SHA256,
  normalizePrepareUploadReq,
  normalizeAuthorizeUploadPutReq,
  normalizeCommitUploadPutReq,
  assertUploadSessionPrincipal,
  assertFinalizeIdentity,
  assertPrepareReplayConsistency,
} from './_upload';
import {
  throwUploadSessionExpired,
  throwUploadSessionFinalized,
  normalizeAllowedMimeTypes,
  buildPayloadWriteTicket,
  buildUploadedPayloadRefFromPayloadId,
  normalizeUploadedPayloadRef,
  isSessionExpired,
  buildPrepareUploadResp,
  buildFinalizeResp,
} from './_attachment_upload_helpers';

// ---------------------------------------------------------------------------
// Model ops contract
// ---------------------------------------------------------------------------

export interface UploadModelOps {
  readonly userId: unknown;
  readonly companyId: unknown;
  readonly companyIds: string[];
  Search(condition: unknown, options?: unknown): Promise<unknown[]>;
  Create(values: unknown, fields?: unknown): Promise<unknown>;
  UpdateById(id: string, values: unknown, fields?: unknown): Promise<void>;
}

// ---------------------------------------------------------------------------
// Lazy imports to avoid circular dependency at module-init time
// ---------------------------------------------------------------------------

function getAttachmentUploadSessionModel(): typeof AttachmentUploadSession {
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  return require('./upload_session').default as typeof AttachmentUploadSession;
}

function getAttachmentMutationLedgerModel(): typeof AttachmentMutationLedger {
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  return require('./attachment_mutation_ledger').default as typeof AttachmentMutationLedger;
}

function getStoredContentModel(): typeof StoredContent {
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  return require('./stored_content').default as typeof StoredContent;
}

// ---------------------------------------------------------------------------
// Data-access helpers — some need model ops, others hit external models
// ---------------------------------------------------------------------------

async function findUploadSessionByBusinessRequestId(
  businessRequestId: string,
  companyId: string,
  issuerUserId: string
): Promise<AttachmentUploadSession | null> {
  const AttachmentUploadSessionModel = getAttachmentUploadSessionModel();
  const rows = await AttachmentUploadSessionModel.Search(
    {
      And: [
        ['BusinessRequestId', '=', businessRequestId],
        ['CompanyId', '=', companyId],
        ['IssuerUserId', '=', issuerUserId],
      ],
    } as any,
    { limit: 1 } as any
  );
  return rows[0] ?? null;
}

async function mustLoadUploadSession(uploadId: string): Promise<AttachmentUploadSession> {
  const AttachmentUploadSessionModel = getAttachmentUploadSessionModel();
  return mustLoadOne<AttachmentUploadSession>(
    (condition, opts) => AttachmentUploadSessionModel.Search(condition, opts as any),
    ['Id', '=', uploadId],
    'Upload session not found',
    { uploadId }
  );
}

async function assertUploadSessionOwnerWriteAuthorization(
  session: AttachmentUploadSession,
  principal: PrincipalContext,
  stage: 'authorize_upload_put' | 'commit_upload_put'
): Promise<void> {
  await assertOwnerWriteAuthorization({
    stage,
    ownerModel: requireText(session.OwnerModel, 'ownerModel'),
    ownerRecordId: normalizeOptionalString(session.OwnerRecordId),
    fieldName: requireText(session.FieldName, 'fieldName'),
    operation: requireText(session.Operation, 'operation') === 'create' ? 'create' : 'update',
    companyId: requireText(session.CompanyId, 'companyId'),
    companyIds: principal.enabledCompanyIds,
    userId: principal.userId,
  });
}

async function markUploadSessionExpired(uploadId: string, session: AttachmentUploadSession): Promise<void> {
  if (session.Status === 'expired') return;
  const AttachmentUploadSessionModel = getAttachmentUploadSessionModel();
  await AttachmentUploadSessionModel.UpdateById(uploadId, { Status: 'expired' } as any, ['Id'] as any);
}

async function resolveStoredContentIdForFinalize(uploadedPayloadRef: UploadedPayloadRef | undefined, uploadId: string, companyId: string): Promise<string> {
  if (!uploadedPayloadRef) {
    throwDocumentError(DocumentErrCode.FAILED_PRECONDITION, 'Upload session is missing uploaded payload reference', GrpcCode.FailedPrecondition, {
      stage: 'finalize',
      uploadId,
    });
  }

  const storedContentId = requireText(uploadedPayloadRef.storedContentId, 'storedContentId');
  const StoredContentModel = getStoredContentModel();
  const stored = await StoredContentModel.mustLoadByID(storedContentId);
  const normalizedCompanyID = requireText((stored as any)?.CompanyId, 'storedContent.companyId');
  if (normalizedCompanyID !== companyId) {
    throwDocumentError(DocumentErrCode.PERMISSION_DENIED, 'Stored content company does not match upload session company', GrpcCode.PermissionDenied, {
      stage: 'finalize',
      uploadId,
      storedContentId,
    });
  }

  const storedStatus = normalizeOptionalString((stored as any)?.Status);
  if (storedStatus !== 'active') {
    throwDocumentError(DocumentErrCode.FAILED_PRECONDITION, 'Stored content must be active before finalize', GrpcCode.FailedPrecondition, {
      stage: 'finalize',
      uploadId,
      storedContentId,
      status: String(storedStatus || ''),
    });
  }

  return storedContentId;
}

async function mustLoadAttachmentContent(ops: UploadModelOps, attachmentContentId: string): Promise<AttachmentContent> {
  return mustLoadOne<AttachmentContent>(
    (condition, opts) => ops.Search(condition, opts as any) as Promise<AttachmentContent[]>,
    ['Id', '=', attachmentContentId],
    'Attachment content not found',
    { attachmentContentId }
  );
}

// ---------------------------------------------------------------------------
// Exported ops — called by the AttachmentContent model class
// ---------------------------------------------------------------------------

export async function prepareUpload(ops: UploadModelOps, req: PrepareUploadReq): Promise<PrepareUploadResp> {
  const normalized = normalizePrepareUploadReq(req);
  const companyId = requireCompanyId(ops.companyId, 'prepare');
  const issuerUserId = requireUserId(ops.userId);

  await assertOwnerWriteAuthorization({
    stage: 'prepare',
    ownerModel: normalized.ownerModel,
    ownerRecordId: normalized.ownerRecordId,
    fieldName: normalized.fieldName,
    operation: normalized.operation,
    companyId,
    companyIds: ops.companyIds,
    userId: issuerUserId,
  });

  const existing = await findUploadSessionByBusinessRequestId(normalized.businessRequestId, companyId, issuerUserId);
  if (existing) {
    assertPrepareReplayConsistency(existing, normalized, companyId, issuerUserId);
    return buildPrepareUploadResp(existing.Id, existing);
  }

  const uploadId = await createUploadSessionInternal(ops, req);
  const created = await mustLoadUploadSession(uploadId);
  return buildPrepareUploadResp(uploadId, created);
}

export async function finalizeUpload(ops: UploadModelOps, req: FinalizeUploadReq): Promise<FinalizeUploadResp> {
  const uploadId = requireText(req?.uploadId, 'uploadId');
  const businessRequestId = requireText(req?.businessRequestId, 'businessRequestId');

  const session = await mustLoadUploadSession(uploadId);
  assertFinalizeIdentity(session, ops.userId, ops.companyId, ops.companyIds);

  if (session.BusinessRequestId !== businessRequestId) {
    throwDocumentError(
      DocumentErrCode.IDEMPOTENCY_KEY_REUSED,
      'FinalizeUpload businessRequestId does not match the existing upload session',
      GrpcCode.FailedPrecondition,
      { uploadId, businessRequestId, expectedBusinessRequestId: String(session.BusinessRequestId || '') }
    );
  }

  await assertOwnerWriteAuthorization({
    stage: 'finalize',
    ownerModel: requireText(session.OwnerModel, 'ownerModel'),
    ownerRecordId: normalizeOptionalString(session.OwnerRecordId),
    fieldName: requireText(session.FieldName, 'fieldName'),
    operation: requireText(session.Operation, 'operation') === 'create' ? 'create' : 'update',
    companyId: requireText(session.CompanyId, 'companyId'),
    companyIds: ops.companyIds,
    userId: requireText(session.IssuerUserId, 'issuerUserId'),
  });

  return finalizeUploadInternal(ops, uploadId);
}

export async function authorizeUploadPut(ops: UploadModelOps, req: AuthorizeUploadPutReq): Promise<AuthorizeUploadPutResp> {
  const normalized = normalizeAuthorizeUploadPutReq(req);
  const session = await mustLoadUploadSession(normalized.uploadId);

  assertUploadSessionPrincipal(session, normalized.principal, 'authorize_upload_put');
  await assertUploadSessionOwnerWriteAuthorization(session, normalized.principal, 'authorize_upload_put');

  if (session.Status === 'finalized') {
    throwUploadSessionFinalized(normalized.uploadId);
  }

  if (session.Status === 'expired' || isSessionExpired(session)) {
    await markUploadSessionExpired(normalized.uploadId, session);
    throwUploadSessionExpired(normalized.uploadId);
  }

  const maxUploadBytes = normalizeOptionalNonNegativeInt(session.MaxUploadBytes) ?? DEFAULT_MAX_UPLOAD_BYTES;
  if ((normalized.requestMeta.contentLength ?? 0) > maxUploadBytes) {
    throwDocumentError(DocumentErrCode.UPLOAD_TOO_LARGE, 'upload body exceeds maxUploadBytes', GrpcCode.InvalidArgument, {
      uploadId: normalized.uploadId,
      maxUploadBytes: String(maxUploadBytes),
    });
  }

  const allowedMimeTypes = normalizeAllowedMimeTypes(session.AllowedMimeTypes);
  const contentType = normalized.requestMeta.contentType ?? normalizeContentType(session.ProposedContentType);
  if (!isMimeTypeAllowed(contentType, allowedMimeTypes)) {
    throwDocumentError(DocumentErrCode.MIME_TYPE_NOT_ALLOWED, 'content type is not allowed', GrpcCode.InvalidArgument, {
      uploadId: normalized.uploadId,
      contentType: String(contentType || ''),
    });
  }

  const expectedChecksumSha256 = normalizeChecksumSha256(session.ChecksumSha256);
  if (expectedChecksumSha256 && normalized.requestMeta.checksumSha256 && normalized.requestMeta.checksumSha256 !== expectedChecksumSha256) {
    throwDocumentError(DocumentErrCode.CHECKSUM_MISMATCH, 'checksum mismatch', GrpcCode.FailedPrecondition, { uploadId: normalized.uploadId });
  }

  return {
    uploadId: normalized.uploadId,
    maxUploadBytes,
    requiredChecksumAlgorithm: 'sha256',
    expectedChecksumSha256,
    allowedMimeTypes: allowedMimeTypes.length > 0 ? allowedMimeTypes : undefined,
    payloadWriteTicket: buildPayloadWriteTicket(session, normalized.principal),
  };
}

export async function commitUploadPut(ops: UploadModelOps, req: CommitUploadPutReq): Promise<CommitUploadPutResp> {
  const normalized = normalizeCommitUploadPutReq(req);
  const session = await mustLoadUploadSession(normalized.uploadId);

  assertUploadSessionPrincipal(session, normalized.principal, 'commit_upload_put');
  await assertUploadSessionOwnerWriteAuthorization(session, normalized.principal, 'commit_upload_put');

  if (session.Status === 'finalized') {
    throwUploadSessionFinalized(normalized.uploadId);
  }

  if (session.Status === 'expired' || isSessionExpired(session)) {
    await markUploadSessionExpired(normalized.uploadId, session);
    throwUploadSessionExpired(normalized.uploadId);
  }

  if (session.Status === 'uploaded') {
    return {
      uploadId: normalized.uploadId,
      attachmentUploadSessionStatus: 'uploaded',
    };
  }

  if (session.Status !== 'prepared') {
    throwDocumentError(DocumentErrCode.FAILED_PRECONDITION, 'Upload session must be prepared before commit', GrpcCode.FailedPrecondition, {
      uploadId: normalized.uploadId,
      status: String(session.Status || ''),
    });
  }

  const maxUploadBytes = normalizeOptionalNonNegativeInt(session.MaxUploadBytes) ?? DEFAULT_MAX_UPLOAD_BYTES;
  if (normalized.payloadReceipt.sizeBytes > maxUploadBytes) {
    throwDocumentError(DocumentErrCode.UPLOAD_TOO_LARGE, 'upload body exceeds maxUploadBytes', GrpcCode.InvalidArgument, {
      uploadId: normalized.uploadId,
      maxUploadBytes: String(maxUploadBytes),
    });
  }

  const allowedMimeTypes = normalizeAllowedMimeTypes(session.AllowedMimeTypes);
  const contentType = normalized.payloadReceipt.contentType ?? normalizeContentType(session.ProposedContentType);
  if (!isMimeTypeAllowed(contentType, allowedMimeTypes)) {
    throwDocumentError(DocumentErrCode.MIME_TYPE_NOT_ALLOWED, 'content type is not allowed', GrpcCode.InvalidArgument, {
      uploadId: normalized.uploadId,
      contentType: String(contentType || ''),
    });
  }

  const expectedChecksumSha256 = normalizeChecksumSha256(session.ChecksumSha256);
  if (expectedChecksumSha256 && normalized.payloadReceipt.checksumSha256 !== expectedChecksumSha256) {
    throwDocumentError(DocumentErrCode.CHECKSUM_MISMATCH, 'checksum mismatch', GrpcCode.FailedPrecondition, { uploadId: normalized.uploadId });
  }

  const AttachmentUploadSessionModel = getAttachmentUploadSessionModel();
  await AttachmentUploadSessionModel.UpdateById(
    normalized.uploadId,
    {
      Status: 'uploaded',
      UploadedSizeBytes: normalized.payloadReceipt.sizeBytes,
      UploadedChecksumSha256: normalized.payloadReceipt.checksumSha256,
      UploadedContentType: contentType,
      UploadedPayloadRef: buildUploadedPayloadRefFromPayloadId(normalized.payloadReceipt.payloadId),
    } as any,
    ['Id'] as any
  );

  return {
    uploadId: normalized.uploadId,
    attachmentUploadSessionStatus: 'uploaded',
  };
}

export async function runGarbageCollection(ops: UploadModelOps, nowISO?: string): Promise<Record<string, unknown>> {
  const now = parseISODate(nowISO);
  const nowAt = now.toISOString();

  const AttachmentUploadSessionModel = getAttachmentUploadSessionModel();
  const AttachmentMutationLedgerModel = getAttachmentMutationLedgerModel();
  const uploadSession = await AttachmentUploadSessionModel.garbageCollectExpired(nowAt);
  const mutationLedger = await AttachmentMutationLedgerModel.garbageCollectRetention(nowAt);
  const objects = await garbageCollectUnboundObjects(ops, nowAt);

  return {
    now: nowAt,
    uploadSession,
    mutationLedger,
    objects,
  };
}

export async function createUploadSessionInternal(ops: UploadModelOps, req: PrepareUploadReq): Promise<string> {
  const normalized = normalizePrepareUploadReq(req);
  const companyId = requireCompanyId(ops.companyId, 'prepare');
  const issuerUserId = requireUserId(ops.userId);

  const now = Date.now();
  const expiresAt = new Date(now + DEFAULT_UPLOAD_SESSION_TTL_SECONDS * 1000);

  const AttachmentUploadSessionModel = getAttachmentUploadSessionModel();
  const created = await AttachmentUploadSessionModel.Create(
    {
      OwnerModel: normalized.ownerModel,
      OwnerRecordId: normalized.ownerRecordId,
      FieldName: normalized.fieldName,
      Operation: normalized.operation,
      IssuerUserId: issuerUserId,
      BusinessRequestId: normalized.businessRequestId,
      ProposedFileName: normalized.proposedFileName,
      ProposedContentType: normalized.proposedContentType,
      ProposedSizeBytes: normalized.proposedSizeBytes,
      ChecksumSha256: normalized.checksumSha256,
      MaxUploadBytes: DEFAULT_MAX_UPLOAD_BYTES,
      RequiredChecksumAlgorithm: 'sha256',
      ExpiresAt: expiresAt,
      Status: 'prepared',
      CompanyId: companyId,
    } as any,
    ['Id'] as any
  );

  return requireText((created as any)?.Id, 'uploadId');
}

export async function finalizeUploadInternal(ops: UploadModelOps, uploadId: string): Promise<FinalizeUploadResp> {
  const normalizedUploadId = requireText(uploadId, 'uploadId');
  const session = await mustLoadUploadSession(normalizedUploadId);
  assertFinalizeIdentity(session, ops.userId, ops.companyId, ops.companyIds);

  if (session.Status === 'finalized') {
    const finalizedContentId = requireText(session.AttachmentContentId, 'attachmentContentId');
    const finalizedContent = await mustLoadAttachmentContent(ops, finalizedContentId);
    return buildFinalizeResp(finalizedContent);
  }

  if (session.Status === 'expired' || isSessionExpired(session)) {
    await markUploadSessionExpired(normalizedUploadId, session);
    throwUploadSessionExpired(normalizedUploadId);
  }

  if (session.Status !== 'uploaded') {
    throwDocumentError(DocumentErrCode.FAILED_PRECONDITION, 'Upload session must be uploaded before finalize', GrpcCode.FailedPrecondition, {
      uploadId: normalizedUploadId,
      status: String(session.Status || ''),
    });
  }

  const sizeBytes = normalizeOptionalNonNegativeInt(session.UploadedSizeBytes) ?? 0;
  const checksumSha256 = normalizeChecksumSha256(session.UploadedChecksumSha256) ?? normalizeChecksumSha256(session.ChecksumSha256) ?? EMPTY_SHA256;
  const mimeType = normalizeOptionalString(session.UploadedContentType) ?? normalizeOptionalString(session.ProposedContentType) ?? 'application/octet-stream';

  const uploadedPayloadRef = normalizeUploadedPayloadRef(session.UploadedPayloadRef);
  const companyId = requireText(session.CompanyId, 'companyId');
  const storedContentId = await resolveStoredContentIdForFinalize(uploadedPayloadRef, normalizedUploadId, companyId);

  const created = await ops.Create(
    {
      StoredContentId: storedContentId,
      SizeBytes: sizeBytes,
      MimeType: mimeType,
      ChecksumSha256: checksumSha256,
      Status: 'active',
      CompanyId: companyId,
    } as any,
    ['Id', 'StoredContentId', 'SizeBytes', 'MimeType', 'ChecksumSha256', 'Status', 'ImageWidth', 'ImageHeight', 'ImageFormat'] as any
  );

  const attachmentContentId = requireText((created as any)?.Id, 'attachmentContentId');
  const AttachmentUploadSessionModel = getAttachmentUploadSessionModel();
  await AttachmentUploadSessionModel.UpdateById(
    normalizedUploadId,
    {
      Status: 'finalized',
      AttachmentContentId: attachmentContentId,
    } as any,
    ['Id'] as any
  );

  return buildFinalizeResp(created as AttachmentContent);
}
