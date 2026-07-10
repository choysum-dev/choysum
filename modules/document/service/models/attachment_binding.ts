// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
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
} from '../contracts';
import {
  bindAttachment,
  unbindAttachment,
  batchDescribeAttachments,
  resolveDownloadContent,
  buildDescriptorForBinding,
  type BindingModelOps,
} from './_attachment_binding_ops';

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

  // -----------------------------------------------------------------------
  // Public API — delegates to extracted ops
  // -----------------------------------------------------------------------

  /**
   * Binds active attachment content to an owner field with mutation replay support.
   */
  public static async Bind(req: BindReq): Promise<BindResp> {
    return bindAttachment(this as unknown as BindingModelOps, req);
  }

  /**
   * Moves an attachment binding into the unbound state.
   */
  public static async Unbind(req: UnbindReq): Promise<UnbindResp> {
    return unbindAttachment(this as unknown as BindingModelOps, req);
  }

  /**
   * Resolves descriptors for a batch of active attachment bindings.
   */
  public static async BatchDescribe(req: BatchDescribeReq): Promise<BatchDescribeResp> {
    return batchDescribeAttachments(this as unknown as BindingModelOps, req);
  }

  /**
   * Resolves download semantics and a payload read ticket for a binding.
   */
  public static async ResolveDownloadContent(req: ResolveDownloadContentReq): Promise<ResolveDownloadContentResp> {
    return resolveDownloadContent(this as unknown as BindingModelOps, req);
  }

  /**
   * Builds an attachment descriptor for an active binding (internal / test seam).
   */
  protected static async buildDescriptorInternal(bindingId: string): Promise<AttachmentDescriptor> {
    return buildDescriptorForBinding(this as unknown as BindingModelOps, bindingId);
  }
}
