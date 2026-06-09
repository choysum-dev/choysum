// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { AttachmentBackend } from '../contracts';
import { DocumentErrCode, GrpcCode, newDocumentError } from '../error';

/**
 * Lifecycle states for stored payload content.
 */
export type StoredContentStatus = 'active' | 'deleted';

/**
 * StoredContent persists the backing payload location or inline blob for document attachments.
 */
@Model('StoredContent', { application: 'document', companyScoped: true })
export default class StoredContent extends BaseModel {
  /**
   * Storage backend that owns the payload bytes.
   */
  @Field({
    type: 'selection',
    selection: [
      { value: 'db', label: 'db' },
      { value: 's3', label: 's3' },
    ],
    column: { size: 16, notNull: true, index: true },
  })
  Provider: AttachmentBackend;

  /**
   * Backend-specific locator metadata for external payload stores.
   */
  @Field({ type: 'jsonobject', column: { notNull: false } })
  LocatorJson?: Record<string, unknown>;

  /**
   * Inline blob storage used when the provider is database-backed.
   */
  @Field({ type: 'binary', column: { notNull: false } })
  BlobData?: string;

  /**
   * Lifecycle state of the stored payload row.
   */
  @Field({
    type: 'selection',
    selection: [
      { value: 'active', label: 'active' },
      { value: 'deleted', label: 'deleted' },
    ],
    column: { size: 16, notNull: true, default: () => 'active', index: true },
  })
  Status: StoredContentStatus;

  /**
   * Company that owns the stored payload.
   */
  @Field({ type: 'ManyToOneRef', targetModel: 'base.Company', column: { size: 20, notNull: true, index: true } })
  CompanyId: string;

  /**
   * Loads a stored payload row or raises a document-domain not-found error.
   */
  public static async mustLoadByID(storedContentId: string): Promise<StoredContent> {
    const rows = await this.Search(['Id', '=', storedContentId] as any, { limit: 1 } as any);
    const storedContent = rows[0] as StoredContent | undefined;
    if (!storedContent) {
      throw newDocumentError({
        code: DocumentErrCode.NOT_FOUND,
        message: 'Stored content not found',
      })
        .withGrpcCode(GrpcCode.NotFound)
        .withMetadata({ storedContentId });
    }
    return storedContent;
  }
}
