// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { normalizeOptionalString, asRecord, normalizeOptionalNonNegativeInt } from '@/core/service/utils/normalization';
import { toDate } from '@/core/service/utils/date';
import { GrpcCode } from '../error';
import {
  AttachmentBindingStatus,
  DownloadDisposition,
  BindReq,
  BindResp,
  UnbindReq,
  UnbindResp,
  AttachmentDescriptor,
  BatchDescribeReq,
  BatchDescribeResp,
  ResolveDownloadContentReq,
  ResolveDownloadContentResp,
  PrincipalContext,
} from '../contracts';
import { newDocumentError, DocumentErrCode } from '../error';
import AttachmentContent from './attachment_object';
import AttachmentMutationLedger from './attachment_mutation_ledger';
import StoredContent from './stored_content';
import { assertOwnerReadAuthorization, assertOwnerWriteAuthorization } from './_owner_authorization';
import { requireText, requireUserId, requireCompanyId } from './_helpers';

type NormalizedBindReq = {
  attachmentContentId: string;
  ownerModel: string;
  ownerRecordId: string;
  fieldName: string;
  displayFileName?: string;
  downloadDisposition: DownloadDisposition;
  mutationId: string;
};

type NormalizedUnbindReq = {
  attachmentBindingId: string;
  mutationId: string;
};

type NormalizedBatchDescribeReq = {
  attachmentBindingIds: string[];
};

type NormalizedResolveDownloadContentReq = {
  attachmentBindingId: string;
  principal: PrincipalContext;
};

type ResolvedDownloadSemantics = {
  fileName: string;
  mimeType: string;
  sizeBytes: number;
  checksumSha256?: string;
  downloadDisposition: DownloadDisposition;
  etag?: string;
};

/**
 * AttachmentBinding links finalized content to owner records and fields.
 */
@Model('AttachmentBinding', { application: 'document', companyScoped: true })
export default class AttachmentBinding extends BaseModel {
  /**
   * Owner model that holds the attachment binding.
   */
  @Field({ type: 'varchar', column: { size: 120, notNull: true, index: true, uniqueIndex: 'uidx_document_binding_owner_field_status' } })
  OwnerModel: string;

  /**
   * Owner record that holds the attachment binding.
   */
  @Field({ type: 'char', column: { size: 20, notNull: true, index: true, uniqueIndex: 'uidx_document_binding_owner_field_status' } })
  OwnerRecordId: string;

  /**
   * Owner field that exposes the attachment binding.
   */
  @Field({ type: 'varchar', column: { size: 120, notNull: true, index: true, uniqueIndex: 'uidx_document_binding_owner_field_status' } })
  FieldName: string;

  /**
   * Attachment content row currently bound to the owner field.
   */
  @Field({ type: 'ManyToOneRef', targetModel: 'document.AttachmentContent', column: { size: 20, notNull: true, index: true } })
  AttachmentContentId: string;

  /**
   * Caller-visible file name presented in download descriptors.
   */
  @Field({ type: 'varchar', column: { size: 255 } })
  DisplayFileName?: string;

  /**
   * Preferred disposition used when the binding is downloaded.
   */
  @Field({
    type: 'selection',
    selection: [
      { value: 'inline', label: 'inline' },
      { value: 'attachment', label: 'attachment' },
    ],
    column: { size: 20, notNull: true, default: () => 'attachment' },
  })
  DownloadDisposition: DownloadDisposition;

  /**
   * Lifecycle state of the binding row.
   */
  @Field({
    type: 'selection',
    selection: [
      { value: 'active', label: 'active' },
      { value: 'unbound', label: 'unbound' },
    ],
    column: {
      size: 20,
      notNull: true,
      default: () => 'active',
      index: true,
      uniqueIndex: 'uidx_document_binding_owner_field_status',
    },
  })
  Status: AttachmentBindingStatus;

  /**
   * Timestamp captured when the binding becomes unbound.
   */
  @Field({ type: 'datetime', column: { index: true } })
  UnboundAt?: Date;

  /**
   * Company that owns the attachment binding.
   */
  @Field({ type: 'ManyToOneRef', targetModel: 'base.Company', column: { size: 20, notNull: true, index: true } })
  CompanyId: string;

  /**
   * Binds active attachment content to an owner field with mutation replay support.
   */
  public static async Bind(req: BindReq): Promise<BindResp> {
    const normalized = this.normalizeBindReq(req);
    const companyId = requireCompanyId(this.companyId, 'bind');
    const userId = requireUserId(this.userId);

    await assertOwnerWriteAuthorization({
      stage: 'bind',
      ownerModel: normalized.ownerModel,
      ownerRecordId: normalized.ownerRecordId,
      fieldName: normalized.fieldName,
      operation: 'update',
      companyId,
      companyIds: this.companyIds,
      userId,
    });

    const replay = await this.tryReplayBindMutation(normalized.mutationId, companyId);
    if (replay) {
      return replay;
    }

    const attachmentContent = await this.mustLoadActiveAttachmentContent(normalized.attachmentContentId, companyId);
    const existing = await this.findActiveBinding(normalized.ownerModel, normalized.ownerRecordId, normalized.fieldName, companyId);

    let bindingRecord: AttachmentBinding;
    if (existing && requireText(existing.AttachmentContentId, 'attachmentContentId') === normalized.attachmentContentId) {
      bindingRecord = await this.patchBindingPresentation(existing, normalized.displayFileName, normalized.downloadDisposition);
    } else {
      if (existing) {
        await this.purgeConflictingUnboundBindings(
          normalized.ownerModel,
          normalized.ownerRecordId,
          normalized.fieldName,
          companyId,
          requireText(existing.Id, 'attachmentBindingId')
        );
        await this.UpdateById(
          requireText(existing.Id, 'attachmentBindingId'),
          {
            Status: 'unbound',
            UnboundAt: new Date(),
          } as any,
          ['Id'] as any
        );
      }

      bindingRecord = await this.Create(
        {
          OwnerModel: normalized.ownerModel,
          OwnerRecordId: normalized.ownerRecordId,
          FieldName: normalized.fieldName,
          AttachmentContentId: normalized.attachmentContentId,
          DisplayFileName: normalized.displayFileName,
          DownloadDisposition: normalized.downloadDisposition,
          Status: 'active',
          CompanyId: companyId,
        } as any,
        ['Id', 'AttachmentContentId', 'DisplayFileName', 'DownloadDisposition', 'Status'] as any
      );
    }

    const descriptor = this.buildDescriptor(bindingRecord, attachmentContent);
    const response: BindResp = {
      attachmentBindingId: requireText(bindingRecord.Id, 'attachmentBindingId'),
      status: 'active',
      descriptor,
    };

    await this.recordMutationSuccess(
      'bind',
      normalized.mutationId,
      companyId,
      (req as unknown as Record<string, unknown>) ?? {},
      response as unknown as Record<string, unknown>
    );
    return response;
  }

  /**
   * Moves an attachment binding into the unbound state.
   */
  public static async Unbind(req: UnbindReq): Promise<UnbindResp> {
    const normalized = this.normalizeUnbindReq(req);
    const companyId = requireCompanyId(this.companyId, 'unbind');
    const userId = requireUserId(this.userId);

    const binding = await this.mustLoadBinding(normalized.attachmentBindingId, companyId);
    await assertOwnerWriteAuthorization({
      stage: 'unbind',
      ownerModel: requireText(binding.OwnerModel, 'ownerModel'),
      ownerRecordId: requireText(binding.OwnerRecordId, 'ownerRecordId'),
      fieldName: requireText(binding.FieldName, 'fieldName'),
      operation: 'update',
      companyId,
      companyIds: this.companyIds,
      userId,
    });

    const replay = await this.tryReplayUnbindMutation(normalized.mutationId, companyId);
    if (replay) {
      return replay;
    }

    const unboundAt = binding.Status === 'unbound' ? (toDate(binding.UnboundAt) ?? new Date()) : new Date();

    if (binding.Status !== 'unbound') {
      await this.purgeConflictingUnboundBindings(
        requireText(binding.OwnerModel, 'ownerModel'),
        requireText(binding.OwnerRecordId, 'ownerRecordId'),
        requireText(binding.FieldName, 'fieldName'),
        companyId,
        normalized.attachmentBindingId
      );
      await this.UpdateById(
        normalized.attachmentBindingId,
        {
          Status: 'unbound',
          UnboundAt: unboundAt,
        } as any,
        ['Id', 'UnboundAt', 'Status'] as any
      );
    }

    const response: UnbindResp = {
      attachmentBindingId: normalized.attachmentBindingId,
      status: 'unbound',
      gcEligibleAfter: unboundAt.toISOString(),
    };

    await this.recordMutationSuccess(
      'unbind',
      normalized.mutationId,
      companyId,
      (req as unknown as Record<string, unknown>) ?? {},
      response as unknown as Record<string, unknown>
    );
    return response;
  }

  /**
   * Resolves descriptors for a batch of active attachment bindings.
   */
  public static async BatchDescribe(req: BatchDescribeReq): Promise<BatchDescribeResp> {
    const normalized = this.normalizeBatchDescribeReq(req);
    if (normalized.attachmentBindingIds.length === 0) {
      return { items: [] };
    }

    const userId = requireUserId(this.userId);
    const companyId = requireCompanyId(this.companyId, 'descriptor');

    const bindings = await this.Search(
      {
        And: [
          ['Id', 'in', normalized.attachmentBindingIds],
          ['CompanyId', '=', companyId],
          ['Status', '=', 'active'],
        ],
      } as any,
      { limit: normalized.attachmentBindingIds.length } as any
    );

    const bindingById = new Map<string, AttachmentBinding>();
    const attachmentContentIds: string[] = [];
    for (const binding of bindings) {
      const bindingId = normalizeOptionalString(binding.Id);
      const attachmentContentId = normalizeOptionalString(binding.AttachmentContentId);
      if (!bindingId || !attachmentContentId) {
        continue;
      }
      bindingById.set(bindingId, binding);
      attachmentContentIds.push(attachmentContentId);
    }

    if (attachmentContentIds.length === 0) {
      return { items: [] };
    }

    const dedupedAttachmentContentIds = Array.from(new Set(attachmentContentIds));
    const attachmentContents = await AttachmentContent.Search(
      {
        And: [
          ['Id', 'in', dedupedAttachmentContentIds],
          ['CompanyId', '=', companyId],
          ['Status', '=', 'active'],
        ],
      } as any,
      { limit: dedupedAttachmentContentIds.length } as any
    );

    const attachmentContentById = new Map<string, AttachmentContent>();
    for (const attachmentContent of attachmentContents) {
      const attachmentContentId = normalizeOptionalString(attachmentContent.Id);
      if (!attachmentContentId) {
        continue;
      }
      attachmentContentById.set(attachmentContentId, attachmentContent);
    }

    const items: BatchDescribeResp['items'] = [];
    for (const bindingId of normalized.attachmentBindingIds) {
      const binding = bindingById.get(bindingId);
      if (!binding) {
        continue;
      }

      const attachmentContentId = requireText(binding.AttachmentContentId, 'attachmentContentId');
      const attachmentContent = attachmentContentById.get(attachmentContentId);
      if (!attachmentContent) {
        continue;
      }

      await assertOwnerReadAuthorization({
        stage: 'descriptor',
        ownerModel: requireText(binding.OwnerModel, 'ownerModel'),
        ownerRecordId: requireText(binding.OwnerRecordId, 'ownerRecordId'),
        fieldName: requireText(binding.FieldName, 'fieldName'),
        companyId,
        companyIds: this.companyIds,
        userId,
      });

      const descriptor = this.buildDescriptor(binding, attachmentContent);
      const displayName = normalizeOptionalString(binding.DisplayFileName) ?? descriptor.fileName;
      const isImage = descriptor.mimeType.toLowerCase().startsWith('image/');

      items.push({
        attachmentBindingId: bindingId,
        descriptor,
        displayName,
        previewUrl: isImage ? descriptor.downloadUrl : undefined,
        ownerModel: requireText(binding.OwnerModel, 'ownerModel'),
        ownerRecordId: requireText(binding.OwnerRecordId, 'ownerRecordId'),
        fieldName: requireText(binding.FieldName, 'fieldName'),
      });
    }

    return { items };
  }

  /**
   * Resolves download semantics and a payload read ticket for a binding.
   */
  public static async ResolveDownloadContent(req: ResolveDownloadContentReq): Promise<ResolveDownloadContentResp> {
    const normalized = this.normalizeResolveDownloadContentReq(req);
    this.assertPrincipalParityWithRuntimeContext(normalized.principal, 'resolve_download_content');

    const binding = await this.mustLoadActiveBindingById(normalized.attachmentBindingId);
    const bindingCompanyId = requireText(binding.CompanyId, 'companyId');
    this.assertCompanyMatch(bindingCompanyId, normalized.principal.activeCompanyId, 'resolve_download_content', {
      resource: 'attachmentBinding',
      attachmentBindingId: normalized.attachmentBindingId,
    });

    const attachmentContentId = requireText(binding.AttachmentContentId, 'attachmentContentId');
    const attachmentContent = await this.mustLoadActiveAttachmentContentById(attachmentContentId);
    this.assertCompanyMatch(requireText(attachmentContent.CompanyId, 'companyId'), normalized.principal.activeCompanyId, 'resolve_download_content', {
      resource: 'attachmentContent',
      attachmentContentId,
    });

    await assertOwnerReadAuthorization({
      stage: 'resolve_download_content',
      ownerModel: requireText(binding.OwnerModel, 'ownerModel'),
      ownerRecordId: requireText(binding.OwnerRecordId, 'ownerRecordId'),
      fieldName: requireText(binding.FieldName, 'fieldName'),
      companyId: bindingCompanyId,
      companyIds: this.normalizePrincipalCompanyIds(normalized.principal, bindingCompanyId),
      userId: normalized.principal.userId,
    });

    const storedContentId = requireText((attachmentContent as any)?.StoredContentId, 'storedContentId');
    const storedContent = await this.mustLoadActiveStoredContentById(storedContentId);
    this.assertCompanyMatch(requireText((storedContent as any)?.CompanyId, 'companyId'), normalized.principal.activeCompanyId, 'resolve_download_content', {
      resource: 'storedContent',
      storedContentId,
    });

    const semantics = this.resolveDownloadSemantics(binding, attachmentContent);

    return {
      attachmentBindingId: normalized.attachmentBindingId,
      payloadReadTicket: this.buildPayloadReadTicket(normalized.attachmentBindingId, attachmentContentId, storedContentId),
      mimeType: semantics.mimeType,
      sizeBytes: semantics.sizeBytes,
      checksumSha256: semantics.checksumSha256,
      fileName: semantics.fileName,
      downloadDisposition: semantics.downloadDisposition,
      etag: semantics.etag,
    };
  }

  protected static async buildDescriptorInternal(bindingId: string): Promise<AttachmentDescriptor> {
    const userId = requireUserId(this.userId);
    const companyId = requireCompanyId(this.companyId, 'descriptor');

    const normalizedBindingId = requireText(bindingId, 'attachmentBindingId');
    const binding = await this.mustLoadActiveBindingById(normalizedBindingId);
    this.assertCompanyMatch(requireText(binding.CompanyId, 'companyId'), companyId, 'descriptor', {
      resource: 'attachmentBinding',
      attachmentBindingId: normalizedBindingId,
    });

    const attachmentContentId = requireText(binding.AttachmentContentId, 'attachmentContentId');
    const attachmentContent = await this.mustLoadActiveAttachmentContentById(attachmentContentId);
    this.assertCompanyMatch(requireText(attachmentContent.CompanyId, 'companyId'), companyId, 'descriptor', {
      resource: 'attachmentContent',
      attachmentContentId,
    });

    await assertOwnerReadAuthorization({
      stage: 'descriptor',
      ownerModel: requireText(binding.OwnerModel, 'ownerModel'),
      ownerRecordId: requireText(binding.OwnerRecordId, 'ownerRecordId'),
      fieldName: requireText(binding.FieldName, 'fieldName'),
      companyId,
      companyIds: this.companyIds,
      userId,
    });

    return this.buildDescriptor(binding, attachmentContent);
  }

  private static normalizeBindReq(req: BindReq | undefined | null): NormalizedBindReq {
    return {
      attachmentContentId: requireText(req?.attachmentObjectId, 'attachmentObjectId'),
      ownerModel: requireText(req?.ownerModel, 'ownerModel'),
      ownerRecordId: requireText(req?.ownerRecordId, 'ownerRecordId'),
      fieldName: requireText(req?.fieldName, 'fieldName'),
      displayFileName: normalizeOptionalString(req?.displayFileName),
      downloadDisposition: this.normalizeDownloadDisposition(req?.downloadDisposition),
      mutationId: requireText(req?.mutationId, 'mutationId'),
    };
  }

  private static normalizeUnbindReq(req: UnbindReq | undefined | null): NormalizedUnbindReq {
    return {
      attachmentBindingId: requireText(req?.attachmentBindingId, 'attachmentBindingId'),
      mutationId: requireText(req?.mutationId, 'mutationId'),
    };
  }

  private static normalizeBatchDescribeReq(req: BatchDescribeReq | undefined | null): NormalizedBatchDescribeReq {
    const rawIds = req?.attachmentBindingIds;
    if (rawIds === undefined || rawIds === null) {
      return { attachmentBindingIds: [] };
    }
    if (!Array.isArray(rawIds)) {
      throw newDocumentError({
        code: DocumentErrCode.INVALID_ARGUMENT,
        message: 'attachmentBindingIds must be an array',
      })
        .withGrpcCode(GrpcCode.InvalidArgument)
        .withMetadata({ field: 'attachmentBindingIds' });
    }

    const deduped: string[] = [];
    const seen = new Set<string>();
    for (const rawId of rawIds) {
      const bindingId = requireText(rawId, 'attachmentBindingId');
      if (seen.has(bindingId)) {
        continue;
      }
      seen.add(bindingId);
      deduped.push(bindingId);
    }
    return { attachmentBindingIds: deduped };
  }

  private static normalizeResolveDownloadContentReq(req: ResolveDownloadContentReq | undefined | null): NormalizedResolveDownloadContentReq {
    return {
      attachmentBindingId: requireText(req?.attachmentBindingId, 'attachmentBindingId'),
      principal: this.normalizePrincipal(req?.principal),
    };
  }

  private static normalizePrincipal(raw: unknown): PrincipalContext {
    const principal = asRecord(raw);
    return {
      userId: requireText(principal?.userId, 'principal.userId'),
      activeCompanyId: requireText(principal?.activeCompanyId, 'principal.activeCompanyId'),
      enabledCompanyIds: Array.isArray(principal?.enabledCompanyIds)
        ? (principal?.enabledCompanyIds as unknown[]).map(item => normalizeOptionalString(item)).filter((item): item is string => Boolean(item))
        : undefined,
    };
  }

  private static normalizePrincipalCompanyIds(principal: PrincipalContext, activeCompanyId: string): string[] {
    const values = Array.isArray(principal.enabledCompanyIds)
      ? principal.enabledCompanyIds.map(item => normalizeOptionalString(item)).filter((item): item is string => Boolean(item))
      : [];

    const normalizedActiveCompanyId = normalizeOptionalString(activeCompanyId);
    if (normalizedActiveCompanyId && !values.includes(normalizedActiveCompanyId)) {
      values.unshift(normalizedActiveCompanyId);
    }
    return Array.from(new Set(values));
  }

  private static assertPrincipalParityWithRuntimeContext(principal: PrincipalContext, stage: 'resolve_download_content'): void {
    const runtimeUserId = normalizeOptionalString(this.userId);
    if (runtimeUserId && runtimeUserId !== principal.userId) {
      throw newDocumentError({
        code: DocumentErrCode.PERMISSION_DENIED,
        message: 'principal userId does not match runtime identity',
      })
        .withGrpcCode(GrpcCode.PermissionDenied)
        .withMetadata({ stage, reason: 'issuer_mismatch' });
    }

    const runtimeCompanyId = normalizeOptionalString(this.companyId);
    if (runtimeCompanyId && runtimeCompanyId !== principal.activeCompanyId) {
      throw newDocumentError({
        code: DocumentErrCode.PERMISSION_DENIED,
        message: 'principal activeCompanyId does not match runtime context',
      })
        .withGrpcCode(GrpcCode.PermissionDenied)
        .withMetadata({ stage, reason: 'company_mismatch' });
    }
  }

  private static normalizeDownloadDisposition(value: unknown): DownloadDisposition {
    const disposition = normalizeOptionalString(value);
    if (disposition === undefined) return 'attachment';
    if (disposition === 'inline' || disposition === 'attachment') return disposition;
    throw newDocumentError({
      code: DocumentErrCode.INVALID_ARGUMENT,
      message: 'downloadDisposition must be inline or attachment',
    })
      .withGrpcCode(GrpcCode.InvalidArgument)
      .withMetadata({ downloadDisposition: disposition });
  }

  private static resolveDownloadSemantics(binding: AttachmentBinding, attachmentContent: AttachmentContent): ResolvedDownloadSemantics {
    const mimeType = normalizeOptionalString(attachmentContent.MimeType) ?? 'application/octet-stream';
    const sizeBytes = normalizeOptionalNonNegativeInt(attachmentContent.SizeBytes) ?? 0;
    const checksumSha256 = this.normalizeChecksum(attachmentContent.ChecksumSha256);

    return {
      fileName: this.buildFileName(binding, attachmentContent),
      mimeType,
      sizeBytes,
      checksumSha256,
      downloadDisposition: this.resolveDownloadDispositionForResponse(binding.DownloadDisposition, mimeType),
      etag: this.buildEtag(checksumSha256),
    };
  }

  private static resolveDownloadDispositionForResponse(value: unknown, mimeType: string): DownloadDisposition {
    const requested = this.normalizeDownloadDisposition(value);
    if (requested === 'inline' && this.inlineMimeAllowed(mimeType)) {
      return 'inline';
    }
    return 'attachment';
  }

  private static inlineMimeAllowed(mimeType: string): boolean {
    const normalized = mimeType.toLowerCase();
    return (
      normalized === 'image/png' ||
      normalized === 'image/jpeg' ||
      normalized === 'image/webp' ||
      normalized === 'application/pdf' ||
      normalized === 'text/plain'
    );
  }

  private static buildEtag(checksumSha256: string | undefined): string | undefined {
    if (!checksumSha256) return undefined;
    return `"sha256:${checksumSha256}"`;
  }

  private static async mustLoadActiveAttachmentContent(attachmentContentId: string, companyId: string): Promise<AttachmentContent> {
    const rows = await AttachmentContent.Search(
      {
        And: [
          ['Id', '=', attachmentContentId],
          ['CompanyId', '=', companyId],
          ['Status', '=', 'active'],
        ],
      } as any,
      { limit: 1 } as any
    );

    const record = rows[0] as AttachmentContent | undefined;
    if (!record) {
      throw newDocumentError({
        code: DocumentErrCode.NOT_FOUND,
        message: 'Active attachment content not found in company scope',
      })
        .withGrpcCode(GrpcCode.NotFound)
        .withMetadata({ attachmentContentId, companyId });
    }

    return record;
  }

  private static async mustLoadActiveAttachmentContentById(attachmentContentId: string): Promise<AttachmentContent> {
    const rows = await AttachmentContent.Search(
      {
        And: [
          ['Id', '=', attachmentContentId],
          ['Status', '=', 'active'],
        ],
      } as any,
      { limit: 1 } as any
    );

    const record = rows[0] as AttachmentContent | undefined;
    if (!record) {
      throw newDocumentError({
        code: DocumentErrCode.NOT_FOUND,
        message: 'Active attachment content not found',
      })
        .withGrpcCode(GrpcCode.NotFound)
        .withMetadata({ attachmentContentId });
    }
    return record;
  }

  private static async mustLoadActiveStoredContentById(storedContentId: string): Promise<StoredContent> {
    const rows = await StoredContent.Search(
      {
        And: [
          ['Id', '=', storedContentId],
          ['Status', '=', 'active'],
        ],
      } as any,
      { limit: 1 } as any
    );

    const record = rows[0] as StoredContent | undefined;
    if (!record) {
      throw newDocumentError({
        code: DocumentErrCode.NOT_FOUND,
        message: 'Active stored content not found',
      })
        .withGrpcCode(GrpcCode.NotFound)
        .withMetadata({ storedContentId });
    }
    return record;
  }

  private static async findActiveBinding(ownerModel: string, ownerRecordId: string, fieldName: string, companyId: string): Promise<AttachmentBinding | null> {
    const rows = await this.Search(
      {
        And: [
          ['OwnerModel', '=', ownerModel],
          ['OwnerRecordId', '=', ownerRecordId],
          ['FieldName', '=', fieldName],
          ['CompanyId', '=', companyId],
          ['Status', '=', 'active'],
        ],
      } as any,
      { limit: 1 } as any
    );
    return rows[0] ?? null;
  }

  private static async purgeConflictingUnboundBindings(
    ownerModel: string,
    ownerRecordId: string,
    fieldName: string,
    companyId: string,
    keepBindingId: string
  ): Promise<void> {
    const rows = await this.Search(
      {
        And: [
          ['OwnerModel', '=', ownerModel],
          ['OwnerRecordId', '=', ownerRecordId],
          ['FieldName', '=', fieldName],
          ['CompanyId', '=', companyId],
          ['Status', '=', 'unbound'],
        ],
      } as any,
      { limit: 100 } as any
    );

    for (const row of rows) {
      const staleBindingId = normalizeOptionalString((row as any)?.Id);
      if (!staleBindingId || staleBindingId === keepBindingId) {
        continue;
      }
      await this.hardDeleteBindingById(staleBindingId, companyId);
    }
  }

  private static async hardDeleteBindingById(bindingId: string, companyId: string): Promise<void> {
    const db = (globalThis as any)?.$choysum?.db;
    const execute = typeof db?.execute === 'function' ? db.execute.bind(db) : undefined;
    if (!execute) {
      throw newDocumentError({
        code: DocumentErrCode.SKELETON_NOT_IMPLEMENTED,
        message: 'database execute bridge is unavailable',
      })
        .withGrpcCode(GrpcCode.Unimplemented)
        .withMetadata({ stage: 'binding_cleanup' });
    }

    await execute('delete from document_attachment_binding where id = ? and company_id = ?', JSON.stringify([bindingId, companyId]));
  }

  private static async mustLoadBinding(bindingId: string, companyId: string): Promise<AttachmentBinding> {
    const rows = await this.Search(
      {
        And: [
          ['Id', '=', bindingId],
          ['CompanyId', '=', companyId],
        ],
      } as any,
      { limit: 1 } as any
    );

    const record = rows[0];
    if (!record) {
      throw newDocumentError({
        code: DocumentErrCode.NOT_FOUND,
        message: 'Attachment binding not found',
      })
        .withGrpcCode(GrpcCode.NotFound)
        .withMetadata({ attachmentBindingId: bindingId, companyId });
    }

    return record;
  }

  private static async mustLoadActiveBinding(bindingId: string, companyId: string): Promise<AttachmentBinding> {
    const rows = await this.Search(
      {
        And: [
          ['Id', '=', bindingId],
          ['CompanyId', '=', companyId],
          ['Status', '=', 'active'],
        ],
      } as any,
      { limit: 1 } as any
    );

    const record = rows[0];
    if (!record) {
      throw newDocumentError({
        code: DocumentErrCode.NOT_FOUND,
        message: 'Active attachment binding not found',
      })
        .withGrpcCode(GrpcCode.NotFound)
        .withMetadata({ attachmentBindingId: bindingId, companyId });
    }

    return record;
  }

  private static async mustLoadActiveBindingById(bindingId: string): Promise<AttachmentBinding> {
    const rows = await this.Search(
      {
        And: [
          ['Id', '=', bindingId],
          ['Status', '=', 'active'],
        ],
      } as any,
      { limit: 1 } as any
    );

    const record = rows[0];
    if (!record) {
      throw newDocumentError({
        code: DocumentErrCode.NOT_FOUND,
        message: 'Active attachment binding not found',
      })
        .withGrpcCode(GrpcCode.NotFound)
        .withMetadata({ attachmentBindingId: bindingId });
    }
    return record;
  }

  private static assertCompanyMatch(
    actualCompanyId: string,
    expectedCompanyId: string,
    stage: 'bind' | 'unbind' | 'descriptor' | 'resolve_download_content',
    metadata: Record<string, unknown>
  ): void {
    if (actualCompanyId === expectedCompanyId) return;
    throw newDocumentError({
      code: DocumentErrCode.PERMISSION_DENIED,
      message: 'Attachment resource company scope mismatch',
    })
      .withGrpcCode(GrpcCode.PermissionDenied)
      .withMetadata({
        stage,
        ...metadata,
        expectedCompanyId,
        actualCompanyId,
      });
  }

  private static buildPayloadReadTicket(attachmentBindingId: string, attachmentContentId: string, storedContentId: string): string {
    return JSON.stringify({
      attachmentBindingId,
      attachmentContentId,
      storedContentId,
    });
  }

  private static async patchBindingPresentation(
    binding: AttachmentBinding,
    displayFileName: string | undefined,
    downloadDisposition: DownloadDisposition
  ): Promise<AttachmentBinding> {
    const bindingId = requireText(binding.Id, 'attachmentBindingId');
    const currentDisplayFileName = normalizeOptionalString(binding.DisplayFileName);
    const currentDisposition = this.normalizeDownloadDisposition(binding.DownloadDisposition);

    if (currentDisplayFileName === displayFileName && currentDisposition === downloadDisposition) {
      return binding;
    }

    await this.UpdateById(
      bindingId,
      {
        DisplayFileName: displayFileName,
        DownloadDisposition: downloadDisposition,
      } as any,
      ['Id', 'DisplayFileName', 'DownloadDisposition'] as any
    );

    return this.mustLoadBinding(bindingId, requireText(binding.CompanyId, 'companyId'));
  }

  private static buildDescriptor(binding: AttachmentBinding, attachmentContent: AttachmentContent): AttachmentDescriptor {
    const bindingId = requireText(binding.Id, 'attachmentBindingId');
    const semantics = this.resolveDownloadSemantics(binding, attachmentContent);
    return {
      id: bindingId,
      fileName: semantics.fileName,
      mimeType: semantics.mimeType,
      sizeBytes: semantics.sizeBytes,
      checksumSha256: semantics.checksumSha256 ?? '',
      downloadUrl: `/_document/bindings/${bindingId}/content`,
    };
  }

  private static buildFileName(binding: AttachmentBinding, attachmentContent: AttachmentContent): string {
    const displayFileName = normalizeOptionalString(binding.DisplayFileName);
    if (displayFileName) {
      return displayFileName;
    }

    const mime = normalizeOptionalString(attachmentContent.MimeType) ?? '';
    const suffix = this.mimeSuffix(mime);
    const bindingId = requireText(binding.Id, 'attachmentBindingId');
    return `attachment-${bindingId}${suffix}`;
  }

  private static mimeSuffix(mimeType: string): string {
    const normalized = mimeType.toLowerCase();
    switch (normalized) {
      case 'image/png':
        return '.png';
      case 'image/jpeg':
        return '.jpg';
      case 'image/webp':
        return '.webp';
      case 'application/pdf':
        return '.pdf';
      case 'text/plain':
        return '.txt';
      default:
        return '';
    }
  }

  private static async tryReplayBindMutation(mutationId: string, companyId: string): Promise<BindResp | null> {
    const row = await this.findLedgerRow('bind', mutationId, companyId);
    if (!row) return null;

    if (row.Status !== 'succeeded') {
      throw newDocumentError({
        code: DocumentErrCode.FAILED_PRECONDITION,
        message: 'bind mutationId exists but previous attempt did not succeed',
      })
        .withGrpcCode(GrpcCode.FailedPrecondition)
        .withMetadata({ mutationId, action: 'bind', status: String(row.Status || '') });
    }

    const snapshot = this.parseBindResp(row.ResponseJson);
    if (!snapshot) {
      throw newDocumentError({
        code: DocumentErrCode.FAILED_PRECONDITION,
        message: 'bind replay snapshot is invalid',
      })
        .withGrpcCode(GrpcCode.FailedPrecondition)
        .withMetadata({ mutationId, action: 'bind' });
    }
    return snapshot;
  }

  private static async tryReplayUnbindMutation(mutationId: string, companyId: string): Promise<UnbindResp | null> {
    const row = await this.findLedgerRow('unbind', mutationId, companyId);
    if (!row) return null;

    if (row.Status !== 'succeeded') {
      throw newDocumentError({
        code: DocumentErrCode.FAILED_PRECONDITION,
        message: 'unbind mutationId exists but previous attempt did not succeed',
      })
        .withGrpcCode(GrpcCode.FailedPrecondition)
        .withMetadata({ mutationId, action: 'unbind', status: String(row.Status || '') });
    }

    const snapshot = this.parseUnbindResp(row.ResponseJson);
    if (!snapshot) {
      throw newDocumentError({
        code: DocumentErrCode.FAILED_PRECONDITION,
        message: 'unbind replay snapshot is invalid',
      })
        .withGrpcCode(GrpcCode.FailedPrecondition)
        .withMetadata({ mutationId, action: 'unbind' });
    }
    return snapshot;
  }

  private static async findLedgerRow(action: 'bind' | 'unbind', mutationId: string, companyId: string): Promise<AttachmentMutationLedger | null> {
    const rows = await AttachmentMutationLedger.Search(
      {
        And: [
          ['Action', '=', action],
          ['MutationId', '=', mutationId],
          ['CompanyId', '=', companyId],
        ],
      } as any,
      { limit: 1 } as any
    );
    return rows[0] ?? null;
  }

  private static async recordMutationSuccess(
    action: 'bind' | 'unbind',
    mutationId: string,
    companyId: string,
    requestJson: Record<string, unknown>,
    responseJson: Record<string, unknown>
  ): Promise<void> {
    try {
      await AttachmentMutationLedger.Create(
        {
          Action: action,
          MutationId: mutationId,
          RequestJson: requestJson,
          ResponseJson: responseJson,
          Status: 'succeeded',
          CompanyId: companyId,
        } as any,
        ['Id'] as any
      );
    } catch (err) {
      const replayed = action === 'bind' ? await this.tryReplayBindMutation(mutationId, companyId) : await this.tryReplayUnbindMutation(mutationId, companyId);
      if (replayed) return;
      throw err;
    }
  }

  private static parseBindResp(value: unknown): BindResp | null {
    const record = asRecord(value);
    if (!record) return null;

    const attachmentBindingId = normalizeOptionalString(record.attachmentBindingId);
    const status = normalizeOptionalString(record.status);
    const descriptorRaw = asRecord(record.descriptor);
    if (!attachmentBindingId || status !== 'active' || !descriptorRaw) return null;

    const descriptorId = normalizeOptionalString(descriptorRaw.id);
    const fileName = normalizeOptionalString(descriptorRaw.fileName);
    const mimeType = normalizeOptionalString(descriptorRaw.mimeType);
    const checksumSha256 = normalizeOptionalString(descriptorRaw.checksumSha256);
    const sizeBytes = normalizeOptionalNonNegativeInt(descriptorRaw.sizeBytes);
    if (!descriptorId || !fileName || !mimeType || !checksumSha256 || sizeBytes === undefined) return null;

    return {
      attachmentBindingId,
      status: 'active',
      descriptor: {
        id: descriptorId,
        fileName,
        mimeType,
        sizeBytes,
        checksumSha256,
        downloadUrl: normalizeOptionalString(descriptorRaw.downloadUrl),
        downloadUrlExpiresAt: normalizeOptionalString(descriptorRaw.downloadUrlExpiresAt),
      },
    };
  }

  private static parseUnbindResp(value: unknown): UnbindResp | null {
    const record = asRecord(value);
    if (!record) return null;

    const attachmentBindingId = normalizeOptionalString(record.attachmentBindingId);
    const status = normalizeOptionalString(record.status);
    const gcEligibleAfter = normalizeOptionalString(record.gcEligibleAfter);
    if (!attachmentBindingId || status !== 'unbound') return null;

    return {
      attachmentBindingId,
      status: 'unbound',
      gcEligibleAfter,
    };
  }

  private static normalizeChecksum(value: unknown): string | undefined {
    const text = normalizeOptionalString(value);
    if (!text) return undefined;
    const normalized = text.toLowerCase();
    if (!/^[a-f0-9]{64}$/.test(normalized)) return undefined;
    return normalized;
  }

  private static newSkeletonNotImplementedError(method: string) {
    return newDocumentError({
      code: DocumentErrCode.SKELETON_NOT_IMPLEMENTED,
      message: 'Document binding skeleton is mounted but not implemented yet',
    })
      .withGrpcCode(GrpcCode.Unimplemented)
      .withMetadata({ method });
  }
}
