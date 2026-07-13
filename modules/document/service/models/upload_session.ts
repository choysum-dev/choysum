// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { parseISODate } from '@/core/service/utils/datetime';
import { getBackendEnvPositiveInt } from '@/core/service/runtime/env/backend_env';
import { UploadOperation, UploadSessionStatus, UploadedPayloadRef } from '../contracts';
import { resolveGcBatchSize } from './_gc_config';
import { paginateBatch } from '@/core/service/utils/pagination';
import { DEFAULT_UPLOAD_SESSION_TTL_SECONDS } from './_upload';

/**
 * AttachmentUploadSession tracks staged uploads before payloads become active content.
 */
@Model('AttachmentUploadSession', { application: 'document', companyScoped: true })
export default class AttachmentUploadSession extends BaseModel {
  /**
   * Owner model that will receive the uploaded attachment.
   */
  @Field({ type: 'varchar', size: 120, notNull: true, index: true})
  OwnerModel: string;

  /**
   * Optional owner record targeted by the upload.
   */
  @Field({ type: 'char', size: 20, index: true})
  OwnerRecordId?: string;

  /**
   * Owner field that will bind the uploaded content.
   */
  @Field({ type: 'varchar', size: 120, notNull: true, index: true})
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
    size: 16, notNull: true, index: true,
  })
  Operation: UploadOperation;

  /**
   * User who prepared the upload session.
   */
  @Field({
    type: 'ManyToOneRef',
    targetModel: 'auth.User',
    size: 20, notNull: true, index: true, uniqueIndex: 'uidx_document_upload_business_request',
  })
  IssuerUserId: string;

  /**
   * Idempotency key supplied by the caller.
   */
  @Field({ type: 'varchar', size: 100, notNull: true, index: true, uniqueIndex: 'uidx_document_upload_business_request'})
  BusinessRequestId: string;

  /**
   * Suggested file name captured during prepare.
   */
  @Field({ type: 'varchar', size: 255})
  ProposedFileName?: string;

  /**
   * Suggested content type captured during prepare.
   */
  @Field({ type: 'varchar', size: 255})
  ProposedContentType?: string;

  /**
   * Suggested payload size captured during prepare.
   */
  @Field({ type: 'bigint' })
  ProposedSizeBytes?: number;

  /**
   * Expected payload checksum supplied during prepare.
   */
  @Field({ type: 'char', size: 64, index: true})
  ChecksumSha256?: string;

  /**
   * Uploaded payload size captured after PUT authorization completes.
   */
  @Field({ type: 'bigint' })
  UploadedSizeBytes?: number;

  /**
   * Uploaded payload checksum captured after PUT authorization completes.
   */
  @Field({ type: 'char', size: 64, index: true})
  UploadedChecksumSha256?: string;

  /**
   * Uploaded payload content type captured after PUT authorization completes.
   */
  @Field({ type: 'varchar', size: 255})
  UploadedContentType?: string;

  /**
   * Reference to the staged payload stored by the backing content provider.
   */
  @Field({ type: 'jsonobject', notNull: false})
  UploadedPayloadRef?: UploadedPayloadRef;

  /**
   * Maximum payload size allowed for the session.
   */
  @Field({ type: 'bigint', notNull: true})
  MaxUploadBytes: number;

  /**
   * Checksum algorithm required for authorized upload requests.
   */
  @Field({ type: 'selection', selection: [{ value: 'sha256', label: 'sha256' }], size: 16, notNull: true, default: () => 'sha256'})
  RequiredChecksumAlgorithm: 'sha256';

  /**
   * Optional MIME allowlist enforced during upload authorization.
   */
  @Field({ type: 'jsonobject' })
  AllowedMimeTypes?: string[];

  /**
   * Expiration timestamp for the session.
   */
  @Field({ type: 'datetime', notNull: true, index: true})
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
    size: 16, notNull: true, default: () => 'prepared', index: true,
  })
  Status: UploadSessionStatus;

  /**
   * Attachment content record created when the upload is finalized.
   */
  @Field({ type: 'ManyToOneRef', targetModel: 'document.AttachmentContent', size: 20, index: true})
  AttachmentContentId?: string;

  /**
   * Company that owns the upload session.
   */
  @Field({
    type: 'ManyToOneRef',
    targetModel: 'base.Company',
    size: 20, notNull: true, index: true, uniqueIndex: 'uidx_document_upload_business_request',
  })
  CompanyId: string;

  /**
   * Expires pending sessions and purges finalized or expired rows past retention.
   */
  public static async garbageCollectExpired(nowISO?: string): Promise<{ expiredCount: number; purgedCount: number }> {
    const now = parseISODate(nowISO);
    const batch = resolveGcBatchSize();
    const uploadSessionTTLSeconds = getBackendEnvPositiveInt(
      ['CHOYSUM_DOCUMENT_ATTACHMENT_UPLOAD_SESSION_TTL_SECONDS', 'CHOYSUM_DOCUMENT_UPLOAD_SESSION_TTL_SECONDS'],
      DEFAULT_UPLOAD_SESSION_TTL_SECONDS
    );

    const self = this;
    const expiredCount = await paginateBatch(
      (condition, opts) => self.Search(condition, opts as any) as Promise<unknown[]>,
      {
        And: [
          ['Status', 'in', ['prepared', 'uploaded'] as any],
          ['ExpiresAt', '<', now],
        ],
      },
      async session => {
        const sessionId = String((session as any)?.Id || '').trim();
        if (!sessionId) return;
        await self.UpdateById(sessionId, { Status: 'expired' } as any, ['Id', 'Status'] as any);
      },
      { batch, fields: ['Id'] }
    );

    const cutoff = new Date(now.getTime() - uploadSessionTTLSeconds * 1000);
    const purgedCount = await paginateBatch(
      (condition, opts) => self.Search(condition, opts as any) as Promise<unknown[]>,
      {
        And: [
          ['Status', 'in', ['finalized', 'expired'] as any],
          ['UpdatedAt', '<', cutoff],
        ],
      },
      async row => {
        const sessionId = String((row as any)?.Id || '').trim();
        if (!sessionId) return;
        await self.DeleteById(sessionId as any);
      },
      { batch, fields: ['Id'] }
    );

    return {
      expiredCount,
      purgedCount,
    };
  }
}
