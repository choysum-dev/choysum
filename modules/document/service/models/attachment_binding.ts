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
import { _lt } from '../i18n';
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
@Model('AttachmentBinding', { application: 'document', companyField: 'CompanyId' })
export default class AttachmentBinding extends BaseModel {
  /**
   * Owner model that holds the attachment binding.
   */
  @Field({
    type: 'varchar',
    size: 120,
    notNull: true,
    index: true,
    uniqueIndex: 'uidx_document_binding_owner_field_status',
    string: _lt('Owner Model', { scope: 'document.model.AttachmentBinding.fields' }),
  })
  OwnerModel: string;

  /**
   * Owner record that holds the attachment binding.
   */
  @Field({
    type: 'char',
    size: 20,
    notNull: true,
    index: true,
    uniqueIndex: 'uidx_document_binding_owner_field_status',
    string: _lt('Owner Record', { scope: 'document.model.AttachmentBinding.fields' }),
  })
  OwnerRecordId: string;

  /**
   * Owner field that exposes the attachment binding.
   */
  @Field({
    type: 'varchar',
    size: 120,
    notNull: true,
    index: true,
    uniqueIndex: 'uidx_document_binding_owner_field_status',
    string: _lt('Field Name', { scope: 'document.model.AttachmentBinding.fields' }),
  })
  FieldName: string;

  /**
   * Attachment content row currently bound to the owner field.
   */
  @Field({
    type: 'ManyToOneRef',
    relation: { targetModel: 'document.AttachmentContent' },
    size: 20,
    checkCompany: true,
    notNull: true,
    index: true,
    string: _lt('Attachment Content', { scope: 'document.model.AttachmentBinding.fields' }),
  })
  AttachmentContentId: string;

  /**
   * Caller-visible file name presented in download descriptors.
   */
  @Field({
    type: 'varchar',
    size: 255,
    string: _lt('Display File Name', { scope: 'document.model.AttachmentBinding.fields' }),
  })
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
    size: 20,
    notNull: true,
    default: () => 'attachment',
    string: _lt('Download Disposition', { scope: 'document.model.AttachmentBinding.fields' }),
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
    size: 20,
    notNull: true,
    default: () => 'active',
    index: true,
    uniqueIndex: 'uidx_document_binding_owner_field_status',
    string: _lt('Status', { scope: 'document.model.AttachmentBinding.fields' }),
  })
  Status: AttachmentBindingStatus;

  /**
   * Timestamp captured when the binding becomes unbound.
   */
  @Field({
    type: 'datetime',
    index: true,
    string: _lt('Unbound At', { scope: 'document.model.AttachmentBinding.fields' }),
  })
  UnboundAt?: Date;

  /**
   * Company that owns the attachment binding.
   */
  @Field({
    type: 'ManyToOneRef',
    relation: { targetModel: 'base.Company' },
    size: 20,
    notNull: true,
    index: true,
    string: _lt('Company', { scope: 'document.model.AttachmentBinding.fields' }),
  })
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
