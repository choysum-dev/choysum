// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import {
  PrepareUploadReq,
  PrepareUploadResp,
  FinalizeUploadReq,
  FinalizeUploadResp,
  AttachmentContentStatus,
  AuthorizeUploadPutReq,
  AuthorizeUploadPutResp,
  CommitUploadPutReq,
  CommitUploadPutResp,
  PrincipalContext,
  UploadedPayloadRef,
} from '../contracts';
import { _lt } from '../i18n';
import type Company from '@/base/service/models/company';
import type StoredContent from './stored_content';
import { normalizeOptionalString, normalizeOptionalNonNegativeInt, normalizeChecksumSha256, normalizeContentType } from '@/core/service/utils/normalization';
import { createTranslate } from '@/core/service/i18n';
import { parseISODate } from '@/core/service/utils/datetime';
import { GrpcCode, DocumentErrCode, throwDocumentError } from '../error';
import type AttachmentUploadSession from './upload_session';
import type AttachmentMutationLedger from './attachment_mutation_ledger';
import { assertOwnerWriteAuthorization } from './_owner_authorization';
import { requireText, requireUserId, requireCompanyId } from './_normalizers';
import { mustLoadOne } from './_query_loaders';
import { garbageCollectUnboundObjects } from './_attachment_gc';
import { isMimeTypeAllowed } from '@/core/service/utils/mime';
import { DEFAULT_UPLOAD_SESSION_TTL_SECONDS, DEFAULT_MAX_UPLOAD_BYTES, EMPTY_SHA256, normalizePrepareUploadReq, normalizeAuthorizeUploadPutReq, normalizeCommitUploadPutReq, assertUploadSessionPrincipal, assertFinalizeIdentity, assertPrepareReplayConsistency } from './_upload';
import { throwUploadSessionExpired, throwUploadSessionFinalized, normalizeAllowedMimeTypes, buildPayloadWriteTicket, buildUploadedPayloadRefFromPayloadId, normalizeUploadedPayloadRef, isSessionExpired, buildPrepareUploadResp, buildFinalizeResp } from './_attachment_upload_codec';

/**
 * AttachmentContent stores finalized payload metadata and drives the upload workflow.
 */
@Model('AttachmentContent', { application: 'document', companyField: 'CompanyId' })
export default class AttachmentContent extends BaseModel {
  /**
   * Stored payload row that backs the attachment content.
   */
  @Field<StoredContent>({
    type: 'ManyToOneRef',
    relation: { targetModel: 'document.StoredContent' },
    size: 20,
    checkCompany: true,
    notNull: true,
    index: true,
    string: _lt('Stored Content', { scope: 'document.model.AttachmentContent.fields' }),
  })
  StoredContentId: string;

  /**
   * Persisted payload size in bytes.
   */
  @Field({
    type: 'bigint',
    notNull: true,
    index: 'idx_document_object_checksum_size_company',
    string: _lt('Size Bytes', { scope: 'document.model.AttachmentContent.fields' }),
  })
  SizeBytes: number;

  /**
   * Persisted MIME type for the payload.
   */
  @Field({
    type: 'varchar',
    size: 255,
    notNull: true,
    index: true,
    string: _lt('MIME Type', { scope: 'document.model.AttachmentContent.fields' }),
  })
  MimeType: string;

  /**
   * SHA-256 checksum for the payload bytes.
   */
  @Field({
    type: 'char',
    size: 64,
    notNull: true,
    index: 'idx_document_object_checksum_size_company',
    string: _lt('Checksum SHA-256', { scope: 'document.model.AttachmentContent.fields' }),
  })
  ChecksumSha256: string;

  /**
   * Persisted image width when the payload is an image.
   */
  @Field({
    type: 'int',
    index: true,
    string: _lt('Image Width', { scope: 'document.model.AttachmentContent.fields' }),
  })
  ImageWidth?: number;

  /**
   * Persisted image height when the payload is an image.
   */
  @Field({
    type: 'int',
    index: true,
    string: _lt('Image Height', { scope: 'document.model.AttachmentContent.fields' }),
  })
  ImageHeight?: number;

  /**
   * Persisted image format when the payload is an image.
   */
  @Field({
    type: 'varchar',
    size: 32,
    index: true,
    string: _lt('Image Format', { scope: 'document.model.AttachmentContent.fields' }),
  })
  ImageFormat?: string;

  /**
   * Provider-specific metadata retained alongside the payload.
   */
  @Field({
    type: 'jsonobject',
    string: _lt('Metadata', { scope: 'document.model.AttachmentContent.fields' }),
  })
  MetadataJson?: Record<string, unknown>;

  /**
   * Lifecycle state of the attachment content row.
   */
  @Field({
    type: 'selection',
    selection: [
      { value: 'staging', label: 'staging' },
      { value: 'active', label: 'active' },
      { value: 'deleted', label: 'deleted' },
    ],
    size: 16,
    notNull: true,
    default: () => 'staging',
    index: true,
    string: _lt('Status', { scope: 'document.model.AttachmentContent.fields' }),
  })
  Status: AttachmentContentStatus;

  /**
   * Company that owns the attachment content.
   */
  @Field<Company>({
    type: 'ManyToOneRef',
    relation: { targetModel: 'base.Company' },
    size: 20,
    notNull: true,
    index: 'idx_document_object_checksum_size_company',
    string: _lt('Company', { scope: 'document.model.AttachmentContent.fields' }),
  })
  CompanyId: string;


  // -----------------------------------------------------------------------
  // Public API
  // -----------------------------------------------------------------------

  /** Prepares an upload session and returns a client upload target. */
  public static async PrepareUpload(req: PrepareUploadReq): Promise<PrepareUploadResp> {
    return prepareUpload(req);
  }

  /** Finalizes a prepared upload session into active attachment content. */
  public static async FinalizeUpload(req: FinalizeUploadReq): Promise<FinalizeUploadResp> {
    return finalizeUpload(req);
  }

  /** Issues a direct upload ticket after validating the upload session and principal. */
  public static async AuthorizeUploadPut(req: AuthorizeUploadPutReq): Promise<AuthorizeUploadPutResp> {
    return authorizeUploadPut(req);
  }

  /** Commits uploaded payload metadata back into the upload session lifecycle. */
  public static async CommitUploadPut(req: CommitUploadPutReq): Promise<CommitUploadPutResp> {
    return commitUploadPut(req);
  }

  /** Performs retention cleanup for upload sessions, mutation ledgers, and unbound content. */
  public static async RunGarbageCollection(nowISO?: string): Promise<Record<string, unknown>> {
    return runGarbageCollection(nowISO);
  }

  protected static async createUploadSessionInternal(req: PrepareUploadReq): Promise<string> {
    return createUploadSessionInternal(req);
  }

  protected static async finalizeUploadInternal(uploadId: string): Promise<FinalizeUploadResp> {
    return finalizeUploadInternal(uploadId);
  }
}

// ---------------------------------------------------------------------------
// Upload/binding workflow implementation
// ---------------------------------------------------------------------------

const { _t } = createTranslate('document');

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
    _t('Upload session not found', { scope: 'service/models/attachment_object' }),
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
    throwDocumentError(
      DocumentErrCode.FAILED_PRECONDITION,
      _t('Upload session is missing uploaded payload reference', { scope: 'service/models/attachment_object' }),
      GrpcCode.FailedPrecondition,
      {
        stage: 'finalize',
        uploadId,
      }
    );
  }

  const storedContentId = requireText(uploadedPayloadRef.storedContentId, 'storedContentId');
  const StoredContentModel = getStoredContentModel();
  const stored = await StoredContentModel.mustLoadByID(storedContentId);
  const normalizedCompanyID = requireText((stored as any)?.CompanyId, 'storedContent.companyId');
  if (normalizedCompanyID !== companyId) {
    throwDocumentError(
      DocumentErrCode.PERMISSION_DENIED,
      _t('Stored content company does not match upload session company', { scope: 'service/models/attachment_object' }),
      GrpcCode.PermissionDenied,
      {
        stage: 'finalize',
        uploadId,
        storedContentId,
      }
    );
  }

  const storedStatus = normalizeOptionalString((stored as any)?.Status);
  if (storedStatus !== 'active') {
    throwDocumentError(
      DocumentErrCode.FAILED_PRECONDITION,
      _t('Stored content must be active before finalize', { scope: 'service/models/attachment_object' }),
      GrpcCode.FailedPrecondition,
      {
        stage: 'finalize',
        uploadId,
        storedContentId,
        status: String(storedStatus || ''),
      }
    );
  }

  return storedContentId;
}

async function mustLoadAttachmentContent(attachmentContentId: string): Promise<AttachmentContent> {
  return mustLoadOne<AttachmentContent>(
    (condition, opts) => AttachmentContent.Search(condition, opts as any) as Promise<AttachmentContent[]>,
    ['Id', '=', attachmentContentId],
    _t('Attachment content not found', { scope: 'service/models/attachment_object' }),
    { attachmentContentId }
  );
}

// ---------------------------------------------------------------------------
// Exported ops — called by the AttachmentContent model class
// ---------------------------------------------------------------------------

async function prepareUpload(req: PrepareUploadReq): Promise<PrepareUploadResp> {
  const normalized = normalizePrepareUploadReq(req);
  const companyId = requireCompanyId(AttachmentContent.companyId, 'prepare');
  const issuerUserId = requireUserId(AttachmentContent.userId);

  await assertOwnerWriteAuthorization({
    stage: 'prepare',
    ownerModel: normalized.ownerModel,
    ownerRecordId: normalized.ownerRecordId,
    fieldName: normalized.fieldName,
    operation: normalized.operation,
    companyId,
    companyIds: AttachmentContent.companyIds,
    userId: issuerUserId,
  });

  const existing = await findUploadSessionByBusinessRequestId(normalized.businessRequestId, companyId, issuerUserId);
  if (existing) {
    assertPrepareReplayConsistency(existing, normalized, companyId, issuerUserId);
    return buildPrepareUploadResp(existing.Id, existing);
  }

  const uploadId = await createUploadSessionInternal(req);
  const created = await mustLoadUploadSession(uploadId);
  return buildPrepareUploadResp(uploadId, created);
}

async function finalizeUpload(req: FinalizeUploadReq): Promise<FinalizeUploadResp> {
  const uploadId = requireText(req?.uploadId, 'uploadId');
  const businessRequestId = requireText(req?.businessRequestId, 'businessRequestId');

  const session = await mustLoadUploadSession(uploadId);
  assertFinalizeIdentity(session, AttachmentContent.userId, AttachmentContent.companyId, AttachmentContent.companyIds);

  if (session.BusinessRequestId !== businessRequestId) {
    throwDocumentError(
      DocumentErrCode.IDEMPOTENCY_KEY_REUSED,
      _t('FinalizeUpload businessRequestId does not match the existing upload session', {
        scope: 'service/models/attachment_object',
      }),
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
    companyIds: AttachmentContent.companyIds,
    userId: requireText(session.IssuerUserId, 'issuerUserId'),
  });

  return finalizeUploadInternal(uploadId);
}

async function authorizeUploadPut(req: AuthorizeUploadPutReq): Promise<AuthorizeUploadPutResp> {
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
    throwDocumentError(
      DocumentErrCode.UPLOAD_TOO_LARGE,
      _t('upload body exceeds maxUploadBytes', { scope: 'service/models/attachment_object' }),
      GrpcCode.InvalidArgument,
      {
        uploadId: normalized.uploadId,
        maxUploadBytes: String(maxUploadBytes),
      }
    );
  }

  const allowedMimeTypes = normalizeAllowedMimeTypes(session.AllowedMimeTypes);
  const contentType = normalized.requestMeta.contentType ?? normalizeContentType(session.ProposedContentType);
  if (!isMimeTypeAllowed(contentType, allowedMimeTypes)) {
    throwDocumentError(
      DocumentErrCode.MIME_TYPE_NOT_ALLOWED,
      _t('content type is not allowed', { scope: 'service/models/attachment_object' }),
      GrpcCode.InvalidArgument,
      {
        uploadId: normalized.uploadId,
        contentType: String(contentType || ''),
      }
    );
  }

  const expectedChecksumSha256 = normalizeChecksumSha256(session.ChecksumSha256);
  if (expectedChecksumSha256 && normalized.requestMeta.checksumSha256 && normalized.requestMeta.checksumSha256 !== expectedChecksumSha256) {
    throwDocumentError(
      DocumentErrCode.CHECKSUM_MISMATCH,
      _t('checksum mismatch', { scope: 'service/models/attachment_object' }),
      GrpcCode.FailedPrecondition,
      { uploadId: normalized.uploadId }
    );
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

async function commitUploadPut(req: CommitUploadPutReq): Promise<CommitUploadPutResp> {
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
    throwDocumentError(
      DocumentErrCode.FAILED_PRECONDITION,
      _t('Upload session must be prepared before commit', { scope: 'service/models/attachment_object' }),
      GrpcCode.FailedPrecondition,
      {
        uploadId: normalized.uploadId,
        status: String(session.Status || ''),
      }
    );
  }

  const maxUploadBytes = normalizeOptionalNonNegativeInt(session.MaxUploadBytes) ?? DEFAULT_MAX_UPLOAD_BYTES;
  if (normalized.payloadReceipt.sizeBytes > maxUploadBytes) {
    throwDocumentError(
      DocumentErrCode.UPLOAD_TOO_LARGE,
      _t('upload body exceeds maxUploadBytes', { scope: 'service/models/attachment_object' }),
      GrpcCode.InvalidArgument,
      {
        uploadId: normalized.uploadId,
        maxUploadBytes: String(maxUploadBytes),
      }
    );
  }

  const allowedMimeTypes = normalizeAllowedMimeTypes(session.AllowedMimeTypes);
  const contentType = normalized.payloadReceipt.contentType ?? normalizeContentType(session.ProposedContentType);
  if (!isMimeTypeAllowed(contentType, allowedMimeTypes)) {
    throwDocumentError(
      DocumentErrCode.MIME_TYPE_NOT_ALLOWED,
      _t('content type is not allowed', { scope: 'service/models/attachment_object' }),
      GrpcCode.InvalidArgument,
      {
        uploadId: normalized.uploadId,
        contentType: String(contentType || ''),
      }
    );
  }

  const expectedChecksumSha256 = normalizeChecksumSha256(session.ChecksumSha256);
  if (expectedChecksumSha256 && normalized.payloadReceipt.checksumSha256 !== expectedChecksumSha256) {
    throwDocumentError(
      DocumentErrCode.CHECKSUM_MISMATCH,
      _t('checksum mismatch', { scope: 'service/models/attachment_object' }),
      GrpcCode.FailedPrecondition,
      { uploadId: normalized.uploadId }
    );
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
    ['Id', 'Status', 'UploadedSizeBytes', 'UploadedChecksumSha256', 'UploadedContentType', 'UploadedPayloadRef'] as any
  );

  return {
    uploadId: normalized.uploadId,
    attachmentUploadSessionStatus: 'uploaded',
  };
}

async function runGarbageCollection(nowISO?: string): Promise<Record<string, unknown>> {
  const now = parseISODate(nowISO);
  const nowAt = now.toISOString();

  const AttachmentUploadSessionModel = getAttachmentUploadSessionModel();
  const AttachmentMutationLedgerModel = getAttachmentMutationLedgerModel();
  const uploadSession = await AttachmentUploadSessionModel.garbageCollectExpired(nowAt);
  const mutationLedger = await AttachmentMutationLedgerModel.garbageCollectRetention(nowAt);
  const objects = await garbageCollectUnboundObjects(AttachmentContent, nowAt);

  return {
    now: nowAt,
    uploadSession,
    mutationLedger,
    objects,
  };
}

async function createUploadSessionInternal(req: PrepareUploadReq): Promise<string> {
  const normalized = normalizePrepareUploadReq(req);
  const companyId = requireCompanyId(AttachmentContent.companyId, 'prepare');
  const issuerUserId = requireUserId(AttachmentContent.userId);

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

async function finalizeUploadInternal(uploadId: string): Promise<FinalizeUploadResp> {
  const normalizedUploadId = requireText(uploadId, 'uploadId');
  const session = await mustLoadUploadSession(normalizedUploadId);
  assertFinalizeIdentity(session, AttachmentContent.userId, AttachmentContent.companyId, AttachmentContent.companyIds);

  if (session.Status === 'finalized') {
    const finalizedContentId = requireText(session.AttachmentContentId, 'attachmentContentId');
    const finalizedContent = await mustLoadAttachmentContent(finalizedContentId);
    return buildFinalizeResp(finalizedContent);
  }

  if (session.Status === 'expired' || isSessionExpired(session)) {
    await markUploadSessionExpired(normalizedUploadId, session);
    throwUploadSessionExpired(normalizedUploadId);
  }

  if (session.Status !== 'uploaded') {
    throwDocumentError(
      DocumentErrCode.FAILED_PRECONDITION,
      _t('Upload session must be uploaded before finalize', { scope: 'service/models/attachment_object' }),
      GrpcCode.FailedPrecondition,
      {
        uploadId: normalizedUploadId,
        status: String(session.Status || ''),
      }
    );
  }

  const sizeBytes = normalizeOptionalNonNegativeInt(session.UploadedSizeBytes) ?? 0;
  const checksumSha256 = normalizeChecksumSha256(session.UploadedChecksumSha256) ?? normalizeChecksumSha256(session.ChecksumSha256) ?? EMPTY_SHA256;
  const mimeType = normalizeOptionalString(session.UploadedContentType) ?? normalizeOptionalString(session.ProposedContentType) ?? 'application/octet-stream';

  const uploadedPayloadRef = normalizeUploadedPayloadRef(session.UploadedPayloadRef);
  const companyId = requireText(session.CompanyId, 'companyId');
  const storedContentId = await resolveStoredContentIdForFinalize(uploadedPayloadRef, normalizedUploadId, companyId);

  const created = await AttachmentContent.Create(
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
    ['Id', 'Status', 'AttachmentContentId'] as any
  );

  return buildFinalizeResp(created as AttachmentContent);
}
