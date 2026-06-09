// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { GrpcCode } from '../error';
import {
  PrepareUploadReq,
  PrepareUploadResp,
  FinalizeUploadReq,
  FinalizeUploadResp,
  AttachmentContentStatus,
  UploadedPayloadRef,
  AuthorizeUploadPutReq,
  AuthorizeUploadPutResp,
  CommitUploadPutReq,
  CommitUploadPutResp,
  PrincipalContext,
} from '../contracts';
import { newDocumentError, DocumentErrCode } from '../error';
import AttachmentUploadSession from './upload_session';
import AttachmentMutationLedger from './attachment_mutation_ledger';
import StoredContent from './stored_content';
import { assertOwnerWriteAuthorization } from './_owner_authorization';

const DEFAULT_UPLOAD_SESSION_TTL_SECONDS = 900;
const DEFAULT_MAX_UPLOAD_BYTES = 20 * 1024 * 1024;
const DEFAULT_GC_BATCH_SIZE = 200;
const DEFAULT_UNBOUND_OBJECT_GRACE_SECONDS = 24 * 60 * 60;
const DEFAULT_CLEANUP_MAX_ATTEMPTS = 8;
const DEFAULT_CLEANUP_RETRY_BASE_SECONDS = 30;
const EMPTY_SHA256 = 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855';

type CleanupStateValue = 'retrying' | 'failed' | 'deleted';

type CleanupState = {
  state?: CleanupStateValue;
  attempts?: number;
  nextRetryAt?: string;
  lastError?: string;
  at?: string;
};

function backendEnv(): Record<string, unknown> {
  const env = (globalThis as any)?.__choysumBackendEnv ?? (import.meta as any)?.env;
  if (!env || typeof env !== 'object') return {};
  return env as Record<string, unknown>;
}

function resolvePositiveIntEnv(keys: string[], fallback: number): number {
  const env = backendEnv();
  for (const key of keys) {
    const raw = env[key];
    const parsed = typeof raw === 'number' ? raw : typeof raw === 'string' ? Number.parseInt(raw, 10) : Number.NaN;
    if (Number.isFinite(parsed) && parsed > 0) {
      return Math.floor(parsed);
    }
  }
  return fallback;
}

function parseNowInput(nowISO?: string): Date {
  if (!nowISO) return new Date();
  const parsed = new Date(nowISO);
  if (Number.isNaN(parsed.getTime())) return new Date();
  return parsed;
}

function asRecord(input: unknown): Record<string, unknown> | undefined {
  if (!input || typeof input !== 'object' || Array.isArray(input)) {
    return undefined;
  }
  return input as Record<string, unknown>;
}

type NormalizedPrepareUploadReq = {
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

type NormalizedAuthorizeUploadPutReq = {
  uploadId: string;
  principal: PrincipalContext;
  requestMeta: {
    contentType?: string;
    contentLength?: number;
    checksumSha256?: string;
  };
};

type NormalizedCommitUploadPutReq = {
  uploadId: string;
  principal: PrincipalContext;
  payloadReceipt: {
    payloadId: string;
    sizeBytes: number;
    checksumSha256: string;
    contentType?: string;
  };
};

/**
 * AttachmentContent stores finalized payload metadata and drives the upload workflow.
 */
@Model('AttachmentContent', { application: 'document', companyScoped: true })
export default class AttachmentContent extends BaseModel {
  /**
   * Stored payload row that backs the attachment content.
   */
  @Field({
    type: 'ManyToOneRef',
    targetModel: 'document.StoredContent',
    column: { size: 20, notNull: true, index: true },
  })
  StoredContentId: string;

  /**
   * Persisted payload size in bytes.
   */
  @Field({ type: 'bigint', column: { notNull: true, index: 'idx_document_object_checksum_size_company' } })
  SizeBytes: number;

  /**
   * Persisted MIME type for the payload.
   */
  @Field({ type: 'varchar', column: { size: 255, notNull: true, index: true } })
  MimeType: string;

  /**
   * SHA-256 checksum for the payload bytes.
   */
  @Field({ type: 'char', column: { size: 64, notNull: true, index: 'idx_document_object_checksum_size_company' } })
  ChecksumSha256: string;

  /**
   * Persisted image width when the payload is an image.
   */
  @Field({ type: 'int', column: { index: true } })
  ImageWidth?: number;

  /**
   * Persisted image height when the payload is an image.
   */
  @Field({ type: 'int', column: { index: true } })
  ImageHeight?: number;

  /**
   * Persisted image format when the payload is an image.
   */
  @Field({ type: 'varchar', column: { size: 32, index: true } })
  ImageFormat?: string;

  /**
   * Provider-specific metadata retained alongside the payload.
   */
  @Field({ type: 'jsonobject' })
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
    column: { size: 16, notNull: true, default: () => 'staging', index: true },
  })
  Status: AttachmentContentStatus;

  /**
   * Company that owns the attachment content.
   */
  @Field({
    type: 'ManyToOneRef',
    targetModel: 'base.Company',
    column: {
      size: 20,
      notNull: true,
      index: 'idx_document_object_checksum_size_company',
    },
  })
  CompanyId: string;

  /**
   * Prepares an upload session and returns a client upload target.
   */
  public static async PrepareUpload(req: PrepareUploadReq): Promise<PrepareUploadResp> {
    const normalized = this.normalizePrepareUploadReq(req);
    const companyId = this.requireCompanyId('prepare');
    const issuerUserId = this.requireUserId();

    await assertOwnerWriteAuthorization({
      stage: 'prepare',
      ownerModel: normalized.ownerModel,
      ownerRecordId: normalized.ownerRecordId,
      fieldName: normalized.fieldName,
      operation: normalized.operation,
      companyId,
      companyIds: this.companyIds,
      userId: issuerUserId,
    });

    const existing = await this.findUploadSessionByBusinessRequestId(normalized.businessRequestId, companyId, issuerUserId);
    if (existing) {
      this.assertPrepareReplayConsistency(existing, normalized, companyId, issuerUserId);
      return this.buildPrepareUploadResp(existing.Id, existing);
    }

    const uploadId = await this.createUploadSessionInternal(req);
    const created = await this.mustLoadUploadSession(uploadId);
    return this.buildPrepareUploadResp(uploadId, created);
  }

  /**
   * Finalizes a prepared upload session into active attachment content.
   */
  public static async FinalizeUpload(req: FinalizeUploadReq): Promise<FinalizeUploadResp> {
    const uploadId = this.requireText(req?.uploadId, 'uploadId');
    const businessRequestId = this.requireText(req?.businessRequestId, 'businessRequestId');

    const session = await this.mustLoadUploadSession(uploadId);
    this.assertFinalizeIdentity(session);

    if (session.BusinessRequestId !== businessRequestId) {
      throw newDocumentError({
        code: DocumentErrCode.IDEMPOTENCY_KEY_REUSED,
        message: 'FinalizeUpload businessRequestId does not match the existing upload session',
      })
        .withGrpcCode(GrpcCode.FailedPrecondition)
        .withMetadata({ uploadId, businessRequestId, expectedBusinessRequestId: String(session.BusinessRequestId || '') });
    }

    await assertOwnerWriteAuthorization({
      stage: 'finalize',
      ownerModel: this.requireText(session.OwnerModel, 'ownerModel'),
      ownerRecordId: this.normalizeOptionalText(session.OwnerRecordId),
      fieldName: this.requireText(session.FieldName, 'fieldName'),
      operation: this.requireText(session.Operation, 'operation') === 'create' ? 'create' : 'update',
      companyId: this.requireText(session.CompanyId, 'companyId'),
      companyIds: this.companyIds,
      userId: this.requireText(session.IssuerUserId, 'issuerUserId'),
    });

    return this.finalizeUploadInternal(uploadId);
  }

  /**
   * Issues a direct upload ticket after validating the upload session and principal.
   */
  public static async AuthorizeUploadPut(req: AuthorizeUploadPutReq): Promise<AuthorizeUploadPutResp> {
    const normalized = this.normalizeAuthorizeUploadPutReq(req);
    const session = await this.mustLoadUploadSession(normalized.uploadId);

    this.assertUploadSessionPrincipal(session, normalized.principal, 'authorize_upload_put');
    await this.assertUploadSessionOwnerWriteAuthorization(session, normalized.principal, 'authorize_upload_put');

    if (session.Status === 'finalized') {
      this.throwUploadSessionFinalized(normalized.uploadId);
    }

    if (session.Status === 'expired' || this.isSessionExpired(session)) {
      await this.markUploadSessionExpired(normalized.uploadId, session);
      this.throwUploadSessionExpired(normalized.uploadId);
    }

    const maxUploadBytes = this.normalizeOptionalNonNegativeInt(session.MaxUploadBytes) ?? DEFAULT_MAX_UPLOAD_BYTES;
    if ((normalized.requestMeta.contentLength ?? 0) > maxUploadBytes) {
      throw newDocumentError({
        code: DocumentErrCode.UPLOAD_TOO_LARGE,
        message: 'upload body exceeds maxUploadBytes',
      })
        .withGrpcCode(GrpcCode.InvalidArgument)
        .withMetadata({ uploadId: normalized.uploadId, maxUploadBytes: String(maxUploadBytes) });
    }

    const allowedMimeTypes = this.normalizeAllowedMimeTypes(session.AllowedMimeTypes);
    const contentType = normalized.requestMeta.contentType ?? this.normalizeContentType(session.ProposedContentType);
    if (!this.isMimeTypeAllowed(contentType, allowedMimeTypes)) {
      throw newDocumentError({
        code: DocumentErrCode.MIME_TYPE_NOT_ALLOWED,
        message: 'content type is not allowed',
      })
        .withGrpcCode(GrpcCode.InvalidArgument)
        .withMetadata({ uploadId: normalized.uploadId, contentType: String(contentType || '') });
    }

    const expectedChecksumSha256 = this.normalizeChecksum(session.ChecksumSha256);
    if (expectedChecksumSha256 && normalized.requestMeta.checksumSha256 && normalized.requestMeta.checksumSha256 !== expectedChecksumSha256) {
      throw newDocumentError({
        code: DocumentErrCode.CHECKSUM_MISMATCH,
        message: 'checksum mismatch',
      })
        .withGrpcCode(GrpcCode.FailedPrecondition)
        .withMetadata({ uploadId: normalized.uploadId });
    }

    return {
      uploadId: normalized.uploadId,
      maxUploadBytes,
      requiredChecksumAlgorithm: 'sha256',
      expectedChecksumSha256,
      allowedMimeTypes: allowedMimeTypes.length > 0 ? allowedMimeTypes : undefined,
      payloadWriteTicket: this.buildPayloadWriteTicket(session, normalized.principal),
    };
  }

  /**
   * Commits uploaded payload metadata back into the upload session lifecycle.
   */
  public static async CommitUploadPut(req: CommitUploadPutReq): Promise<CommitUploadPutResp> {
    const normalized = this.normalizeCommitUploadPutReq(req);
    const session = await this.mustLoadUploadSession(normalized.uploadId);

    this.assertUploadSessionPrincipal(session, normalized.principal, 'commit_upload_put');
    await this.assertUploadSessionOwnerWriteAuthorization(session, normalized.principal, 'commit_upload_put');

    if (session.Status === 'finalized') {
      this.throwUploadSessionFinalized(normalized.uploadId);
    }

    if (session.Status === 'expired' || this.isSessionExpired(session)) {
      await this.markUploadSessionExpired(normalized.uploadId, session);
      this.throwUploadSessionExpired(normalized.uploadId);
    }

    if (session.Status === 'uploaded') {
      return {
        uploadId: normalized.uploadId,
        attachmentUploadSessionStatus: 'uploaded',
      };
    }

    if (session.Status !== 'prepared') {
      throw newDocumentError({
        code: DocumentErrCode.FAILED_PRECONDITION,
        message: 'Upload session must be prepared before commit',
      })
        .withGrpcCode(GrpcCode.FailedPrecondition)
        .withMetadata({ uploadId: normalized.uploadId, status: String(session.Status || '') });
    }

    const maxUploadBytes = this.normalizeOptionalNonNegativeInt(session.MaxUploadBytes) ?? DEFAULT_MAX_UPLOAD_BYTES;
    if (normalized.payloadReceipt.sizeBytes > maxUploadBytes) {
      throw newDocumentError({
        code: DocumentErrCode.UPLOAD_TOO_LARGE,
        message: 'upload body exceeds maxUploadBytes',
      })
        .withGrpcCode(GrpcCode.InvalidArgument)
        .withMetadata({ uploadId: normalized.uploadId, maxUploadBytes: String(maxUploadBytes) });
    }

    const allowedMimeTypes = this.normalizeAllowedMimeTypes(session.AllowedMimeTypes);
    const contentType = normalized.payloadReceipt.contentType ?? this.normalizeContentType(session.ProposedContentType);
    if (!this.isMimeTypeAllowed(contentType, allowedMimeTypes)) {
      throw newDocumentError({
        code: DocumentErrCode.MIME_TYPE_NOT_ALLOWED,
        message: 'content type is not allowed',
      })
        .withGrpcCode(GrpcCode.InvalidArgument)
        .withMetadata({ uploadId: normalized.uploadId, contentType: String(contentType || '') });
    }

    const expectedChecksumSha256 = this.normalizeChecksum(session.ChecksumSha256);
    if (expectedChecksumSha256 && normalized.payloadReceipt.checksumSha256 !== expectedChecksumSha256) {
      throw newDocumentError({
        code: DocumentErrCode.CHECKSUM_MISMATCH,
        message: 'checksum mismatch',
      })
        .withGrpcCode(GrpcCode.FailedPrecondition)
        .withMetadata({ uploadId: normalized.uploadId });
    }

    await AttachmentUploadSession.UpdateById(
      normalized.uploadId,
      {
        Status: 'uploaded',
        UploadedSizeBytes: normalized.payloadReceipt.sizeBytes,
        UploadedChecksumSha256: normalized.payloadReceipt.checksumSha256,
        UploadedContentType: contentType,
        UploadedPayloadRef: this.buildUploadedPayloadRefFromPayloadId(normalized.payloadReceipt.payloadId),
      } as any,
      ['Id'] as any
    );

    return {
      uploadId: normalized.uploadId,
      attachmentUploadSessionStatus: 'uploaded',
    };
  }

  /**
   * Performs retention cleanup for upload sessions, mutation ledgers, and unbound content.
   */
  public static async RunGarbageCollection(nowISO?: string): Promise<Record<string, unknown>> {
    const now = parseNowInput(nowISO);
    const nowAt = now.toISOString();

    const uploadSession = await AttachmentUploadSession.garbageCollectExpired(nowAt);
    const mutationLedger = await AttachmentMutationLedger.garbageCollectRetention(nowAt);
    const objects = await this.garbageCollectUnboundObjects(nowAt);

    return {
      now: nowAt,
      uploadSession,
      mutationLedger,
      objects,
    };
  }

  /**
   * Deletes active content that no longer has bindings after the grace period elapses.
   */
  public static async garbageCollectUnboundObjects(
    nowISO?: string
  ): Promise<{ scannedCount: number; deletedCount: number; retriedCount: number; failedCount: number; skippedCount: number }> {
    const now = parseNowInput(nowISO);
    const nowAt = now.toISOString();
    const batch = resolvePositiveIntEnv(['CHOYSUM_DOCUMENT_GC_BATCH_SIZE'], DEFAULT_GC_BATCH_SIZE);
    const unboundGraceSeconds = resolvePositiveIntEnv(
      ['CHOYSUM_DOCUMENT_ATTACHMENT_UNBOUND_OBJECT_GRACE_SECONDS', 'CHOYSUM_DOCUMENT_UNBOUND_OBJECT_GRACE_SECONDS'],
      DEFAULT_UNBOUND_OBJECT_GRACE_SECONDS
    );
    const maxAttempts = resolvePositiveIntEnv(['CHOYSUM_DOCUMENT_CLEANUP_MAX_ATTEMPTS'], DEFAULT_CLEANUP_MAX_ATTEMPTS);
    const retryBaseSeconds = resolvePositiveIntEnv(['CHOYSUM_DOCUMENT_CLEANUP_RETRY_BASE_SECONDS'], DEFAULT_CLEANUP_RETRY_BASE_SECONDS);
    const graceCutoff = new Date(now.getTime() - unboundGraceSeconds * 1000);

    const AttachmentBinding = (await import('./attachment_binding')).default;

    let scannedCount = 0;
    let deletedCount = 0;
    let retriedCount = 0;
    let failedCount = 0;
    let skippedCount = 0;
    let offset = 0;

    for (;;) {
      const candidates = await this.Search(
        {
          And: [
            ['Status', '=', 'active'],
            ['UpdatedAt', '<', graceCutoff],
          ],
        } as any,
        {
          limit: batch,
          offset,
          orderBy: { field: 'UpdatedAt', order: 'asc' } as any,
        } as any
      );
      if (!candidates.length) break;

      for (const candidate of candidates) {
        scannedCount += 1;
        const contentId = this.normalizeOptionalText((candidate as any)?.Id);
        if (!contentId) {
          skippedCount += 1;
          continue;
        }

        const activeBindings = await AttachmentBinding.Search(
          {
            And: [
              ['AttachmentContentId', '=', contentId],
              ['Status', '=', 'active'],
            ],
          } as any,
          { limit: 1, fields: ['Id'] as any } as any
        );
        if (activeBindings.length > 0) {
          skippedCount += 1;
          continue;
        }

        const metadata = asRecord((candidate as any)?.MetadataJson);
        const cleanup = this.readCleanupState(metadata);
        const attempts = Math.max(0, Math.trunc(Number(cleanup.attempts || 0)));
        const state = this.normalizeOptionalText(cleanup.state)?.toLowerCase();
        if (state === 'failed' && attempts >= maxAttempts) {
          skippedCount += 1;
          continue;
        }

        const nextRetryAt = this.toDate(cleanup.nextRetryAt);
        if (nextRetryAt && nextRetryAt.getTime() > now.getTime()) {
          skippedCount += 1;
          continue;
        }

        const nextAttempt = attempts + 1;
        try {
          const storedContentId = this.normalizeOptionalText((candidate as any)?.StoredContentId);
          if (!storedContentId) {
            throw new Error('attachment content missing storedContentId');
          }

          const documentBridge = (globalThis as any)?.$choysum?.document;
          const deleteStoredContent =
            typeof documentBridge?.deleteStoredContent === 'function' ? documentBridge.deleteStoredContent.bind(documentBridge) : undefined;
          if (!deleteStoredContent) {
            throw new Error('document.deleteStoredContent bridge is unavailable');
          }
          await deleteStoredContent({ storedContentId });

          await this.UpdateById(
            contentId,
            {
              Status: 'deleted',
              MetadataJson: this.writeCleanupState(metadata, {
                state: 'deleted',
                attempts: nextAttempt,
                at: nowAt,
              }),
            } as any,
            ['Id'] as any
          );

          deletedCount += 1;
        } catch (error) {
          const message = String((error as any)?.message || error || 'attachment cleanup failed');
          const terminal = nextAttempt >= maxAttempts;
          const nextRetryAtISO = terminal
            ? undefined
            : new Date(now.getTime() + this.computeRetryBackoffSeconds(nextAttempt, retryBaseSeconds) * 1000).toISOString();

          await this.UpdateById(
            contentId,
            {
              MetadataJson: this.writeCleanupState(metadata, {
                state: terminal ? 'failed' : 'retrying',
                attempts: nextAttempt,
                nextRetryAt: nextRetryAtISO,
                lastError: message.slice(0, 1024),
                at: nowAt,
              }),
            } as any,
            ['Id'] as any
          );

          if (terminal) {
            failedCount += 1;
          } else {
            retriedCount += 1;
          }
        }
      }

      offset += candidates.length;

      if (candidates.length < batch) break;
    }

    return {
      scannedCount,
      deletedCount,
      retriedCount,
      failedCount,
      skippedCount,
    };
  }

  protected static async createUploadSessionInternal(req: PrepareUploadReq): Promise<string> {
    const normalized = this.normalizePrepareUploadReq(req);
    const companyId = this.requireCompanyId('prepare');
    const issuerUserId = this.requireUserId();

    const now = Date.now();
    const expiresAt = new Date(now + DEFAULT_UPLOAD_SESSION_TTL_SECONDS * 1000);

    const created = await AttachmentUploadSession.Create(
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

    return this.requireText((created as any)?.Id, 'uploadId');
  }

  protected static async finalizeUploadInternal(uploadId: string): Promise<FinalizeUploadResp> {
    const normalizedUploadId = this.requireText(uploadId, 'uploadId');
    const session = await this.mustLoadUploadSession(normalizedUploadId);
    this.assertFinalizeIdentity(session);

    if (session.Status === 'finalized') {
      const finalizedContentId = this.requireText(session.AttachmentContentId, 'attachmentContentId');
      const finalizedContent = await this.mustLoadAttachmentContent(finalizedContentId);
      return this.buildFinalizeResp(finalizedContent);
    }

    if (session.Status === 'expired' || this.isSessionExpired(session)) {
      await this.markUploadSessionExpired(normalizedUploadId, session);
      this.throwUploadSessionExpired(normalizedUploadId);
    }

    if (session.Status !== 'uploaded') {
      throw newDocumentError({
        code: DocumentErrCode.FAILED_PRECONDITION,
        message: 'Upload session must be uploaded before finalize',
      })
        .withGrpcCode(GrpcCode.FailedPrecondition)
        .withMetadata({ uploadId: normalizedUploadId, status: String(session.Status || '') });
    }

    const sizeBytes = this.normalizeOptionalNonNegativeInt(session.UploadedSizeBytes) ?? 0;
    const checksumSha256 = this.normalizeChecksum(session.UploadedChecksumSha256) ?? this.normalizeChecksum(session.ChecksumSha256) ?? EMPTY_SHA256;
    const mimeType =
      this.normalizeOptionalText(session.UploadedContentType) ?? this.normalizeOptionalText(session.ProposedContentType) ?? 'application/octet-stream';

    const uploadedPayloadRef = this.normalizeUploadedPayloadRef(session.UploadedPayloadRef);
    const companyId = this.requireText(session.CompanyId, 'companyId');
    const storedContentId = await this.resolveStoredContentIdForFinalize(uploadedPayloadRef, normalizedUploadId, companyId);

    const created = await this.Create(
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

    const attachmentContentId = this.requireText((created as any)?.Id, 'attachmentContentId');
    await AttachmentUploadSession.UpdateById(
      normalizedUploadId,
      {
        Status: 'finalized',
        AttachmentContentId: attachmentContentId,
      } as any,
      ['Id'] as any
    );

    return this.buildFinalizeResp(created as AttachmentContent);
  }

  private static computeRetryBackoffSeconds(attempts: number, baseSeconds: number): number {
    const exponent = Math.max(0, Math.min(10, attempts - 1));
    const backoff = baseSeconds * 2 ** exponent;
    return Math.min(backoff, 6 * 60 * 60);
  }

  private static readCleanupState(metadata: Record<string, unknown> | undefined): CleanupState {
    const cleanup = asRecord(metadata?.cleanup);
    if (!cleanup) {
      return {};
    }

    const state = this.normalizeOptionalText(cleanup.state) as CleanupStateValue | undefined;
    const attempts = this.normalizeOptionalNonNegativeInt(cleanup.attempts);
    const nextRetryAt = this.normalizeOptionalText(cleanup.nextRetryAt);
    const lastError = this.normalizeOptionalText(cleanup.lastError);
    const at = this.normalizeOptionalText(cleanup.at);

    return {
      state,
      attempts,
      nextRetryAt,
      lastError,
      at,
    };
  }

  private static writeCleanupState(metadata: Record<string, unknown> | undefined, state: CleanupState): Record<string, unknown> {
    const nextMetadata: Record<string, unknown> = {
      ...(metadata || {}),
      cleanup: {
        state: state.state,
        attempts: state.attempts,
        nextRetryAt: state.nextRetryAt,
        lastError: state.lastError,
        at: state.at,
      },
    };
    return nextMetadata;
  }

  private static normalizePrepareUploadReq(req: PrepareUploadReq | undefined | null): NormalizedPrepareUploadReq {
    const ownerModel = this.requireText(req?.ownerModel, 'ownerModel');
    const fieldName = this.requireText(req?.fieldName, 'fieldName');
    const operation = this.requireText(req?.operation, 'operation');
    const businessRequestId = this.requireText(req?.businessRequestId, 'businessRequestId');

    if (operation !== 'create' && operation !== 'update') {
      throw newDocumentError({
        code: DocumentErrCode.INVALID_ARGUMENT,
        message: 'operation must be create or update',
      })
        .withGrpcCode(GrpcCode.InvalidArgument)
        .withMetadata({ operation });
    }

    const ownerRecordId = this.normalizeOptionalText(req?.ownerRecordId);
    if (operation === 'update' && !ownerRecordId) {
      throw newDocumentError({
        code: DocumentErrCode.INVALID_ARGUMENT,
        message: 'ownerRecordId is required when operation=update',
      }).withGrpcCode(GrpcCode.InvalidArgument);
    }

    return {
      ownerModel,
      fieldName,
      operation,
      ownerRecordId,
      businessRequestId,
      proposedFileName: this.normalizeOptionalText(req?.proposedFileName ?? req?.originalFileName),
      proposedContentType: this.normalizeOptionalText(req?.proposedContentType ?? req?.clientContentType),
      proposedSizeBytes: this.normalizeOptionalNonNegativeInt(req?.proposedSizeBytes),
      checksumSha256: this.normalizeChecksum(req?.checksumSha256),
    };
  }

  private static async findUploadSessionByBusinessRequestId(
    businessRequestId: string,
    companyId: string,
    issuerUserId: string
  ): Promise<AttachmentUploadSession | null> {
    const rows = await AttachmentUploadSession.Search(
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

  private static async mustLoadUploadSession(uploadId: string): Promise<AttachmentUploadSession> {
    const rows = await AttachmentUploadSession.Search(['Id', '=', uploadId] as any, { limit: 1 } as any);
    const session = rows[0];
    if (!session) {
      throw newDocumentError({
        code: DocumentErrCode.NOT_FOUND,
        message: 'Upload session not found',
      })
        .withGrpcCode(GrpcCode.NotFound)
        .withMetadata({ uploadId });
    }
    return session;
  }

  private static assertPrepareReplayConsistency(
    existing: AttachmentUploadSession,
    req: NormalizedPrepareUploadReq,
    companyId: string,
    issuerUserId: string
  ): void {
    const mismatches: string[] = [];

    if (this.requireText(existing.CompanyId, 'companyId') !== companyId) mismatches.push('companyId');
    if (this.requireText(existing.IssuerUserId, 'issuerUserId') !== issuerUserId) mismatches.push('issuerUserId');
    if (this.requireText(existing.OwnerModel, 'ownerModel') !== req.ownerModel) mismatches.push('ownerModel');
    if ((this.normalizeOptionalText(existing.OwnerRecordId) ?? '') !== (req.ownerRecordId ?? '')) mismatches.push('ownerRecordId');
    if (this.requireText(existing.FieldName, 'fieldName') !== req.fieldName) mismatches.push('fieldName');
    if (this.requireText(existing.Operation, 'operation') !== req.operation) mismatches.push('operation');

    if (mismatches.length > 0) {
      throw newDocumentError({
        code: DocumentErrCode.IDEMPOTENCY_KEY_REUSED,
        message: 'businessRequestId was already used with a different upload context',
      })
        .withGrpcCode(GrpcCode.FailedPrecondition)
        .withMetadata({ businessRequestId: req.businessRequestId, mismatches: mismatches.join(',') });
    }
  }

  private static assertFinalizeIdentity(session: AttachmentUploadSession): void {
    const principal = {
      userId: this.requireUserId(),
      activeCompanyId: this.requireCompanyId('finalize'),
      enabledCompanyIds: this.companyIds,
    };
    this.assertUploadSessionPrincipal(session, principal, 'finalize');
  }

  private static assertUploadSessionPrincipal(
    session: AttachmentUploadSession,
    principal: PrincipalContext,
    stage: 'finalize' | 'authorize_upload_put' | 'commit_upload_put'
  ): void {
    const sessionCompanyId = this.requireText(session.CompanyId, 'companyId');
    const sessionUserId = this.requireText(session.IssuerUserId, 'issuerUserId');

    if (sessionCompanyId !== principal.activeCompanyId || sessionUserId !== principal.userId) {
      throw newDocumentError({
        code: DocumentErrCode.PERMISSION_DENIED,
        message: 'upload session caller does not match upload session identity',
      })
        .withGrpcCode(GrpcCode.PermissionDenied)
        .withMetadata({ stage, uploadId: this.requireText(session.Id, 'uploadId') });
    }
  }

  private static async assertUploadSessionOwnerWriteAuthorization(
    session: AttachmentUploadSession,
    principal: PrincipalContext,
    stage: 'authorize_upload_put' | 'commit_upload_put'
  ): Promise<void> {
    await assertOwnerWriteAuthorization({
      stage,
      ownerModel: this.requireText(session.OwnerModel, 'ownerModel'),
      ownerRecordId: this.normalizeOptionalText(session.OwnerRecordId),
      fieldName: this.requireText(session.FieldName, 'fieldName'),
      operation: this.requireText(session.Operation, 'operation') === 'create' ? 'create' : 'update',
      companyId: this.requireText(session.CompanyId, 'companyId'),
      companyIds: principal.enabledCompanyIds,
      userId: principal.userId,
    });
  }

  private static async markUploadSessionExpired(uploadId: string, session: AttachmentUploadSession): Promise<void> {
    if (session.Status === 'expired') return;
    await AttachmentUploadSession.UpdateById(uploadId, { Status: 'expired' } as any, ['Id'] as any);
  }

  private static throwUploadSessionExpired(uploadId: string): never {
    throw newDocumentError({
      code: DocumentErrCode.UPLOAD_SESSION_EXPIRED,
      message: 'Upload session has expired',
    })
      .withGrpcCode(GrpcCode.FailedPrecondition)
      .withMetadata({ uploadId });
  }

  private static throwUploadSessionFinalized(uploadId: string): never {
    throw newDocumentError({
      code: DocumentErrCode.UPLOAD_SESSION_FINALIZED,
      message: 'Upload session has already been finalized',
    })
      .withGrpcCode(GrpcCode.FailedPrecondition)
      .withMetadata({ uploadId });
  }

  private static normalizeAuthorizeUploadPutReq(req: AuthorizeUploadPutReq | undefined | null): NormalizedAuthorizeUploadPutReq {
    const uploadId = this.requireText(req?.uploadId, 'uploadId');
    const principal = this.normalizePrincipal(req?.principal);
    const requestMeta = asRecord(req?.requestMeta);

    return {
      uploadId,
      principal,
      requestMeta: {
        contentType: this.normalizeContentType(requestMeta?.contentType),
        contentLength:
          requestMeta && Object.prototype.hasOwnProperty.call(requestMeta, 'contentLength')
            ? this.parseRequiredNonNegativeInt(requestMeta.contentLength, 'requestMeta.contentLength')
            : undefined,
        checksumSha256: this.normalizeChecksum(requestMeta?.checksumSha256),
      },
    };
  }

  private static normalizeCommitUploadPutReq(req: CommitUploadPutReq | undefined | null): NormalizedCommitUploadPutReq {
    const uploadId = this.requireText(req?.uploadId, 'uploadId');
    const principal = this.normalizePrincipal(req?.principal);
    const payloadReceipt = asRecord(req?.payloadReceipt);

    return {
      uploadId,
      principal,
      payloadReceipt: {
        payloadId: this.normalizePayloadReceiptID(payloadReceipt?.payloadId),
        sizeBytes: this.parseRequiredNonNegativeInt(payloadReceipt?.sizeBytes, 'payloadReceipt.sizeBytes'),
        checksumSha256: this.requireText(this.normalizeChecksum(payloadReceipt?.checksumSha256), 'payloadReceipt.checksumSha256'),
        contentType: this.normalizeContentType(payloadReceipt?.contentType),
      },
    };
  }

  private static normalizePayloadReceiptID(value: unknown): string {
    const payloadId = this.requireText(value, 'payloadReceipt.payloadId');
    if (this.isDisallowedInlinePayloadID(payloadId)) {
      throw newDocumentError({
        code: DocumentErrCode.INVALID_ARGUMENT,
        message: 'payloadReceipt.payloadId must be an opaque handle, inline byte payload is forbidden',
      })
        .withGrpcCode(GrpcCode.InvalidArgument)
        .withMetadata({ field: 'payloadReceipt.payloadId' });
    }
    return payloadId;
  }

  private static isDisallowedInlinePayloadID(payloadId: string): boolean {
    const text = this.normalizeOptionalText(payloadId);
    if (!text) return false;
    const normalized = text.toLowerCase();
    return normalized.startsWith('inline_base64:') || normalized.startsWith('data:');
  }

  private static normalizePrincipal(raw: unknown): PrincipalContext {
    const principal = asRecord(raw);
    return {
      userId: this.requireText(principal?.userId, 'principal.userId'),
      activeCompanyId: this.requireText(principal?.activeCompanyId, 'principal.activeCompanyId'),
      enabledCompanyIds: Array.isArray(principal?.enabledCompanyIds)
        ? (principal?.enabledCompanyIds as unknown[]).map(item => this.normalizeOptionalText(item)).filter((item): item is string => Boolean(item))
        : undefined,
    };
  }

  private static parseRequiredNonNegativeInt(value: unknown, fieldName: string): number {
    const raw = this.normalizeOptionalText(value);
    if (raw === undefined) {
      throw newDocumentError({
        code: DocumentErrCode.INVALID_ARGUMENT,
        message: `${fieldName} is required`,
      })
        .withGrpcCode(GrpcCode.InvalidArgument)
        .withMetadata({ field: fieldName });
    }

    const num = Number(raw);
    if (!Number.isFinite(num) || num < 0) {
      throw newDocumentError({
        code: DocumentErrCode.INVALID_ARGUMENT,
        message: `${fieldName} must be a non-negative integer`,
      })
        .withGrpcCode(GrpcCode.InvalidArgument)
        .withMetadata({ field: fieldName });
    }

    return Math.trunc(num);
  }

  private static normalizeAllowedMimeTypes(raw: unknown): string[] {
    if (Array.isArray(raw)) {
      return raw.map(item => this.normalizeContentType(item)).filter((item): item is string => Boolean(item));
    }

    if (raw && typeof raw === 'object') {
      // jsonobject columns can materialize as {} when value is unset.
      // Treat non-array object payloads as "no allow-list".
      return [];
    }

    const text = this.normalizeOptionalText(raw);
    if (!text) {
      return [];
    }

    try {
      const parsed = JSON.parse(text);
      if (Array.isArray(parsed)) {
        return parsed.map(item => this.normalizeContentType(item)).filter((item): item is string => Boolean(item));
      }
      if (typeof parsed === 'string') {
        const normalized = this.normalizeContentType(parsed);
        return normalized ? [normalized] : [];
      }
      return [];
    } catch {
      // Fallback to single content type token.
    }

    const normalized = this.normalizeContentType(text);
    return normalized ? [normalized] : [];
  }

  private static normalizeContentType(value: unknown): string | undefined {
    const text = this.normalizeOptionalText(value);
    if (!text) return undefined;
    const semicolon = text.indexOf(';');
    const token = semicolon >= 0 ? text.slice(0, semicolon) : text;
    const normalized = token.trim().toLowerCase();
    return normalized || undefined;
  }

  private static isMimeTypeAllowed(contentType: string | undefined, allowedMimeTypes: string[]): boolean {
    if (allowedMimeTypes.length === 0) {
      return true;
    }
    if (!contentType) {
      return false;
    }
    return allowedMimeTypes.includes(contentType);
  }

  private static buildPayloadWriteTicket(session: AttachmentUploadSession, principal: PrincipalContext): string {
    const payload = {
      uploadId: this.requireText(session.Id, 'uploadId'),
      companyId: this.requireText(session.CompanyId, 'companyId'),
      userId: this.requireText(session.IssuerUserId, 'issuerUserId'),
      activeCompanyId: principal.activeCompanyId,
      status: this.requireText(session.Status, 'status'),
      expiresAt: this.toDate(session.ExpiresAt)?.toISOString(),
    };
    return JSON.stringify(payload);
  }

  private static async resolveStoredContentIdForFinalize(
    uploadedPayloadRef: UploadedPayloadRef | undefined,
    uploadId: string,
    companyId: string
  ): Promise<string> {
    if (!uploadedPayloadRef) {
      throw newDocumentError({
        code: DocumentErrCode.FAILED_PRECONDITION,
        message: 'Upload session is missing uploaded payload reference',
      })
        .withGrpcCode(GrpcCode.FailedPrecondition)
        .withMetadata({ stage: 'finalize', uploadId });
    }

    const storedContentId = this.requireText(uploadedPayloadRef.storedContentId, 'storedContentId');
    const stored = await StoredContent.mustLoadByID(storedContentId);
    const normalizedCompanyID = this.requireText((stored as any)?.CompanyId, 'storedContent.companyId');
    if (normalizedCompanyID !== companyId) {
      throw newDocumentError({
        code: DocumentErrCode.PERMISSION_DENIED,
        message: 'Stored content company does not match upload session company',
      })
        .withGrpcCode(GrpcCode.PermissionDenied)
        .withMetadata({ stage: 'finalize', uploadId, storedContentId });
    }

    const storedStatus = this.normalizeOptionalText((stored as any)?.Status);
    if (storedStatus !== 'active') {
      throw newDocumentError({
        code: DocumentErrCode.FAILED_PRECONDITION,
        message: 'Stored content must be active before finalize',
      })
        .withGrpcCode(GrpcCode.FailedPrecondition)
        .withMetadata({ stage: 'finalize', uploadId, storedContentId, status: String(storedStatus || '') });
    }

    return storedContentId;
  }

  private static buildUploadedPayloadRefFromPayloadId(payloadId: string): UploadedPayloadRef {
    const text = this.requireText(payloadId, 'payloadReceipt.payloadId');
    if (this.isDisallowedInlinePayloadID(text)) {
      throw newDocumentError({
        code: DocumentErrCode.INVALID_ARGUMENT,
        message: 'payloadReceipt.payloadId must be an opaque handle, inline byte payload is forbidden',
      })
        .withGrpcCode(GrpcCode.InvalidArgument)
        .withMetadata({ field: 'payloadReceipt.payloadId' });
    }

    const parsedStoredContentID = this.parseStoredContentPayloadRef(text);
    if (parsedStoredContentID) {
      return {
        kind: 'stored_content',
        storedContentId: parsedStoredContentID,
      };
    }

    throw newDocumentError({
      code: DocumentErrCode.INVALID_ARGUMENT,
      message: 'payloadReceipt.payloadId format is unsupported',
    })
      .withGrpcCode(GrpcCode.InvalidArgument)
      .withMetadata({ field: 'payloadReceipt.payloadId' });
  }

  private static normalizeUploadedPayloadRef(raw: unknown): UploadedPayloadRef | undefined {
    if (raw === undefined || raw === null) return undefined;

    if (typeof raw === 'string') {
      const text = this.normalizeOptionalText(raw);
      if (!text) {
        return undefined;
      }

      try {
        const parsed = JSON.parse(text) as Record<string, unknown>;
        return this.normalizeUploadedPayloadRef(parsed);
      } catch {
        return this.buildUploadedPayloadRefFromPayloadId(text);
      }
    }

    const record = asRecord(raw);
    if (!record) return undefined;

    const kind = this.normalizeOptionalText(record.kind)?.toLowerCase();
    if (kind === 'stored_content') {
      const storedContentId = this.normalizeOptionalText(record.storedContentId);
      if (!storedContentId) return undefined;
      return {
        kind: 'stored_content',
        storedContentId,
      };
    }

    const payloadId = this.normalizeOptionalText(record.payloadId);
    if (payloadId) {
      return this.buildUploadedPayloadRefFromPayloadId(payloadId);
    }

    return undefined;
  }

  private static parseStoredContentPayloadRef(payloadId: string): string | undefined {
    const text = this.normalizeOptionalText(payloadId);
    if (!text) return undefined;

    const normalized = text.toLowerCase();
    if (!normalized.startsWith('sc:')) {
      return undefined;
    }

    const storedContentId = this.normalizeOptionalText(text.slice(3));
    return storedContentId || undefined;
  }

  private static isSessionExpired(session: AttachmentUploadSession): boolean {
    const expiresAt = this.toDate(session.ExpiresAt);
    if (!expiresAt) return false;
    return expiresAt.getTime() <= Date.now();
  }

  private static async mustLoadAttachmentContent(attachmentContentId: string): Promise<AttachmentContent> {
    const rows = await this.Search(['Id', '=', attachmentContentId] as any, { limit: 1 } as any);
    const attachmentContent = rows[0] as AttachmentContent | undefined;
    if (!attachmentContent) {
      throw newDocumentError({
        code: DocumentErrCode.NOT_FOUND,
        message: 'Attachment content not found',
      })
        .withGrpcCode(GrpcCode.NotFound)
        .withMetadata({ attachmentContentId });
    }
    return attachmentContent;
  }

  private static buildPrepareUploadResp(uploadId: string, session: AttachmentUploadSession): PrepareUploadResp {
    const expiresAt = this.toDate(session.ExpiresAt);
    const maxUploadBytes = this.normalizeOptionalNonNegativeInt(session.MaxUploadBytes) ?? DEFAULT_MAX_UPLOAD_BYTES;
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

  private static buildFinalizeResp(obj: AttachmentContent): FinalizeUploadResp {
    const imageWidth = this.normalizeOptionalNonNegativeInt(obj.ImageWidth);
    const imageHeight = this.normalizeOptionalNonNegativeInt(obj.ImageHeight);
    const imageFormat = this.normalizeOptionalText(obj.ImageFormat);

    return {
      attachmentObjectId: this.requireText(obj.Id, 'attachmentObjectId'),
      status: 'active',
      mimeType: this.normalizeOptionalText(obj.MimeType) ?? 'application/octet-stream',
      sizeBytes: this.normalizeOptionalNonNegativeInt(obj.SizeBytes) ?? 0,
      checksumSha256: this.normalizeChecksum(obj.ChecksumSha256) ?? EMPTY_SHA256,
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

  private static requireUserId(): string {
    const userId = this.normalizeOptionalText(this.userId);
    if (!userId) {
      throw newDocumentError({
        code: DocumentErrCode.UNAUTHENTICATED,
        message: 'Authentication is required',
      }).withGrpcCode(GrpcCode.Unauthenticated);
    }
    return userId;
  }

  private static requireCompanyId(stage: 'prepare' | 'finalize'): string {
    const companyId = this.normalizeOptionalText(this.companyId);
    if (!companyId) {
      throw newDocumentError({
        code: DocumentErrCode.PERMISSION_DENIED,
        message: 'activeCompanyId is required for document operations',
      })
        .withGrpcCode(GrpcCode.PermissionDenied)
        .withMetadata({ stage });
    }
    return companyId;
  }

  private static requireText(value: unknown, fieldName: string): string {
    const text = this.normalizeOptionalText(value);
    if (!text) {
      throw newDocumentError({
        code: DocumentErrCode.INVALID_ARGUMENT,
        message: `${fieldName} is required`,
      })
        .withGrpcCode(GrpcCode.InvalidArgument)
        .withMetadata({ field: fieldName });
    }
    return text;
  }

  private static normalizeOptionalText(value: unknown): string | undefined {
    const text = String(value ?? '').trim();
    return text === '' ? undefined : text;
  }

  private static normalizeOptionalNonNegativeInt(value: unknown): number | undefined {
    if (value === undefined || value === null || value === '') return undefined;
    const num = Number(value);
    if (!Number.isFinite(num)) return undefined;
    if (num < 0) return undefined;
    return Math.trunc(num);
  }

  private static normalizeChecksum(value: unknown): string | undefined {
    const text = this.normalizeOptionalText(value);
    if (!text) return undefined;
    const normalized = text.toLowerCase();
    if (!/^[a-f0-9]{64}$/.test(normalized)) {
      throw newDocumentError({
        code: DocumentErrCode.INVALID_ARGUMENT,
        message: 'checksumSha256 must be a 64-character hex string',
      }).withGrpcCode(GrpcCode.InvalidArgument);
    }
    return normalized;
  }

  private static toDate(value: unknown): Date | undefined {
    if (value instanceof Date) return Number.isNaN(value.getTime()) ? undefined : value;
    const parsed = new Date(String(value ?? ''));
    return Number.isNaN(parsed.getTime()) ? undefined : parsed;
  }

  private static newSkeletonNotImplementedError(method: string) {
    return newDocumentError({
      code: DocumentErrCode.SKELETON_NOT_IMPLEMENTED,
      message: 'Document control-plane skeleton is mounted but not implemented yet',
    })
      .withGrpcCode(GrpcCode.Unimplemented)
      .withMetadata({ method });
  }
}
