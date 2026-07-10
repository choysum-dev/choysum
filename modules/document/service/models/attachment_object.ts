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
} from '../contracts';
import {
  prepareUpload,
  finalizeUpload,
  authorizeUploadPut,
  commitUploadPut,
  runGarbageCollection,
  createUploadSessionInternal,
  finalizeUploadInternal,
  type UploadModelOps,
} from './_attachment_upload_ops';

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

  // -----------------------------------------------------------------------
  // Public API — delegates to extracted ops
  // -----------------------------------------------------------------------

  /**
   * Prepares an upload session and returns a client upload target.
   */
  public static async PrepareUpload(req: PrepareUploadReq): Promise<PrepareUploadResp> {
    return prepareUpload(this as unknown as UploadModelOps, req);
  }

  /**
   * Finalizes a prepared upload session into active attachment content.
   */
  public static async FinalizeUpload(req: FinalizeUploadReq): Promise<FinalizeUploadResp> {
    return finalizeUpload(this as unknown as UploadModelOps, req);
  }

  /**
   * Issues a direct upload ticket after validating the upload session and principal.
   */
  public static async AuthorizeUploadPut(req: AuthorizeUploadPutReq): Promise<AuthorizeUploadPutResp> {
    return authorizeUploadPut(this as unknown as UploadModelOps, req);
  }

  /**
   * Commits uploaded payload metadata back into the upload session lifecycle.
   */
  public static async CommitUploadPut(req: CommitUploadPutReq): Promise<CommitUploadPutResp> {
    return commitUploadPut(this as unknown as UploadModelOps, req);
  }

  /**
   * Performs retention cleanup for upload sessions, mutation ledgers, and unbound content.
   */
  public static async RunGarbageCollection(nowISO?: string): Promise<Record<string, unknown>> {
    return runGarbageCollection(this as unknown as UploadModelOps, nowISO);
  }

  // -----------------------------------------------------------------------
  // Protected seams — used by internal workflows and tests
  // -----------------------------------------------------------------------

  protected static async createUploadSessionInternal(req: PrepareUploadReq): Promise<string> {
    return createUploadSessionInternal(this as unknown as UploadModelOps, req);
  }

  protected static async finalizeUploadInternal(uploadId: string): Promise<FinalizeUploadResp> {
    return finalizeUploadInternal(this as unknown as UploadModelOps, uploadId);
  }
}
