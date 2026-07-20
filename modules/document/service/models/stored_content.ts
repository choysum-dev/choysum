// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { AttachmentBackend } from '../contracts';
import { _t, _lt } from '../i18n';
import { mustLoadOne } from './_query_loaders';

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
    size: 16,
    notNull: true,
    index: true,
    string: _lt('Provider', { scope: 'document.model.StoredContent.fields' }),
  })
  Provider: AttachmentBackend;

  /**
   * Backend-specific locator metadata for external payload stores.
   */
  @Field({
    type: 'jsonobject',
    notNull: false,
    string: _lt('Locator JSON', { scope: 'document.model.StoredContent.fields' }),
  })
  LocatorJson?: Record<string, unknown>;

  /**
   * Inline blob storage used when the provider is database-backed.
   */
  @Field({
    type: 'binary',
    notNull: false,
    string: _lt('Blob Data', { scope: 'document.model.StoredContent.fields' }),
  })
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
    size: 16,
    notNull: true,
    default: () => 'active',
    index: true,
    string: _lt('Status', { scope: 'document.model.StoredContent.fields' }),
  })
  Status: StoredContentStatus;

  /**
   * Company that owns the stored payload.
   */
  @Field({
    type: 'ManyToOneRef',
    relation: { targetModel: 'base.Company' },
    size: 20,
    notNull: true,
    index: true,
    string: _lt('Company', { scope: 'document.model.StoredContent.fields' }),
  })
  CompanyId: string;

  /**
   * Loads a stored payload row or raises a document-domain not-found error.
   */
  public static async mustLoadByID(storedContentId: string): Promise<StoredContent> {
    return mustLoadOne<StoredContent>(
      (condition, opts) => this.Search(condition, opts as any),
      ['Id', '=', storedContentId],
      _t('Stored content not found', { scope: 'service/models/stored_content' }),
      {
        storedContentId,
      }
    );
  }
}
