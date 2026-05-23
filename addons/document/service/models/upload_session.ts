// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { UploadOperation, UploadSessionStatus, UploadedPayloadRef } from '../contracts';

const DEFAULT_UPLOAD_SESSION_TTL_SECONDS = 900;
const DEFAULT_GC_BATCH_SIZE = 200;

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

/**
 * AttachmentUploadSession tracks staged uploads before payloads become active content.
 */
@Model('AttachmentUploadSession', { application: 'document', companyScoped: true })
export default class AttachmentUploadSession extends BaseModel {
  /**
   * Owner model that will receive the uploaded attachment.
   */
  @Field({ type: 'varchar', column: { size: 120, notNull: true, index: true } })
  OwnerModel: string;

  /**
   * Optional owner record targeted by the upload.
   */
  @Field({ type: 'char', column: { size: 20, index: true } })
  OwnerRecordId?: string;

  /**
   * Owner field that will bind the uploaded content.
   */
  @Field({ type: 'varchar', column: { size: 120, notNull: true, index: true } })
  FieldName: string;

  /**
   * Owner-side mutation intent for the upload session.
   */
  @Field({
    type: 'selection',
    selection: [
      { value: 'create', label: 'create' },
      { value: 'update', label: 'update' },
    ],
    column: { size: 16, notNull: true, index: true },
  })
  Operation: UploadOperation;

  /**
   * User who prepared the upload session.
   */
  @Field({
    type: 'ManyToOneRef',
    targetModel: 'auth.User',
    column: { size: 20, notNull: true, index: true, uniqueIndex: 'uidx_document_upload_business_request' },
  })
  IssuerUserId: string;

  /**
   * Idempotency key supplied by the caller.
   */
  @Field({ type: 'varchar', column: { size: 100, notNull: true, index: true, uniqueIndex: 'uidx_document_upload_business_request' } })
  BusinessRequestId: string;

  /**
   * Suggested file name captured during prepare.
   */
  @Field({ type: 'varchar', column: { size: 255 } })
  ProposedFileName?: string;

  /**
   * Suggested content type captured during prepare.
   */
  @Field({ type: 'varchar', column: { size: 255 } })
  ProposedContentType?: string;

  /**
   * Suggested payload size captured during prepare.
   */
  @Field({ type: 'bigint' })
  ProposedSizeBytes?: number;

  /**
   * Expected payload checksum supplied during prepare.
   */
  @Field({ type: 'char', column: { size: 64, index: true } })
  ChecksumSha256?: string;

  /**
   * Uploaded payload size captured after PUT authorization completes.
   */
  @Field({ type: 'bigint' })
  UploadedSizeBytes?: number;

  /**
   * Uploaded payload checksum captured after PUT authorization completes.
   */
  @Field({ type: 'char', column: { size: 64, index: true } })
  UploadedChecksumSha256?: string;

  /**
   * Uploaded payload content type captured after PUT authorization completes.
   */
  @Field({ type: 'varchar', column: { size: 255 } })
  UploadedContentType?: string;

  /**
   * Reference to the staged payload stored by the backing content provider.
   */
  @Field({ type: 'jsonobject', column: { notNull: false } })
  UploadedPayloadRef?: UploadedPayloadRef;

  /**
   * Maximum payload size allowed for the session.
   */
  @Field({ type: 'bigint', column: { notNull: true } })
  MaxUploadBytes: number;

  /**
   * Checksum algorithm required for authorized upload requests.
   */
  @Field({ type: 'selection', selection: [{ value: 'sha256', label: 'sha256' }], column: { size: 16, notNull: true, default: () => 'sha256' } })
  RequiredChecksumAlgorithm: 'sha256';

  /**
   * Optional MIME allowlist enforced during upload authorization.
   */
  @Field({ type: 'jsonobject' })
  AllowedMimeTypes?: string[];

  /**
   * Expiration timestamp for the session.
   */
  @Field({ type: 'datetime', column: { notNull: true, index: true } })
  ExpiresAt: Date;

  /**
   * Current lifecycle state of the upload session.
   */
  @Field({
    type: 'selection',
    selection: [
      { value: 'prepared', label: 'prepared' },
      { value: 'uploaded', label: 'uploaded' },
      { value: 'finalized', label: 'finalized' },
      { value: 'expired', label: 'expired' },
    ],
    column: { size: 16, notNull: true, default: () => 'prepared', index: true },
  })
  Status: UploadSessionStatus;

  /**
   * Attachment content record created when the upload is finalized.
   */
  @Field({ type: 'ManyToOneRef', targetModel: 'document.AttachmentContent', column: { size: 20, index: true } })
  AttachmentContentId?: string;

  /**
   * Company that owns the upload session.
   */
  @Field({
    type: 'ManyToOneRef',
    targetModel: 'base.Company',
    column: { size: 20, notNull: true, index: true, uniqueIndex: 'uidx_document_upload_business_request' },
  })
  CompanyId: string;

  /**
   * Expires pending sessions and purges finalized or expired rows past retention.
   */
  public static async garbageCollectExpired(nowISO?: string): Promise<{ expiredCount: number; purgedCount: number }> {
    const now = parseNowInput(nowISO);
    const batch = resolvePositiveIntEnv(['CHOYSUM_DOCUMENT_GC_BATCH_SIZE'], DEFAULT_GC_BATCH_SIZE);
    const uploadSessionTTLSeconds = resolvePositiveIntEnv(
      ['CHOYSUM_DOCUMENT_ATTACHMENT_UPLOAD_SESSION_TTL_SECONDS', 'CHOYSUM_DOCUMENT_UPLOAD_SESSION_TTL_SECONDS'],
      DEFAULT_UPLOAD_SESSION_TTL_SECONDS
    );

    let expiredCount = 0;
    for (;;) {
      const sessions = await this.Search(
        {
          And: [
            ['Status', 'in', ['prepared', 'uploaded'] as any],
            ['ExpiresAt', '<', now],
          ],
        } as any,
        { limit: batch, fields: ['Id'] as any } as any
      );
      if (!sessions.length) break;
      for (const session of sessions) {
        const sessionId = String((session as any)?.Id || '').trim();
        if (!sessionId) continue;
        await this.UpdateById(sessionId, { Status: 'expired' } as any, ['Id'] as any);
        expiredCount += 1;
      }
      if (sessions.length < batch) break;
    }

    const cutoff = new Date(now.getTime() - uploadSessionTTLSeconds * 1000);
    let purgedCount = 0;
    for (;;) {
      const rows = await this.Search(
        {
          And: [
            ['Status', 'in', ['finalized', 'expired'] as any],
            ['UpdatedAt', '<', cutoff],
          ],
        } as any,
        { limit: batch, fields: ['Id'] as any } as any
      );
      if (!rows.length) break;
      for (const row of rows) {
        const sessionId = String((row as any)?.Id || '').trim();
        if (!sessionId) continue;
        await this.DeleteById(sessionId as any);
        purgedCount += 1;
      }
      if (rows.length < batch) break;
    }

    return {
      expiredCount,
      purgedCount,
    };
  }
}
