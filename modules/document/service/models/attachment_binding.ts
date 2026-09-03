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
  PrincipalContext,
} from '../contracts';
import { _lt } from '../i18n';
import type Company from '@/base/service/models/company';
import type AttachmentContent from './attachment_object';
import { normalizeOptionalString } from '@/core/service/utils/normalization';
import { createTranslate } from '@/core/service/i18n';
import { toDate } from '@/core/service/utils/datetime';
import { GrpcCode, DocumentErrCode, throwDocumentError } from '../error';
import type AttachmentMutationLedger from './attachment_mutation_ledger';
import type StoredContent from './stored_content';
import { requireText, requireUserId, requireCompanyId } from './_document_bridge';
import { assertOwnerReadAuthorization, assertOwnerWriteAuthorization } from './_owner_authorization';
import { assertBindReq, assertUnbindReq, assertBatchDescribeReq, assertResolveDownloadContentReq, normalizePrincipalCompanyIds, assertDownloadDisposition, resolveDownloadSemantics, buildDescriptor, buildPayloadReadTicket, parseBindResp, parseUnbindResp, assertCompanyMatch } from './_attachment_binding_codec';
import { validateAttachmentContentFieldLimits } from './_binding_field_limits';

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
  @Field<AttachmentContent>({
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
  @Field<Company>({
    type: 'ManyToOneRef',
    relation: { targetModel: 'base.Company' },
    size: 20,
    notNull: true,
    index: true,
    string: _lt('Company', { scope: 'document.model.AttachmentBinding.fields' }),
  })
  CompanyId: string;


  // -----------------------------------------------------------------------
  // Public API
  // -----------------------------------------------------------------------

  /** Binds active attachment content to an owner field with mutation replay support. */
  public static async Bind(req: BindReq): Promise<BindResp> {
    return bindAttachment(req);
  }

  /** Moves an attachment binding into the unbound state. */
  public static async Unbind(req: UnbindReq): Promise<UnbindResp> {
    return unbindAttachment(req);
  }

  /** Resolves descriptors for a batch of active attachment bindings. */
  public static async BatchDescribe(req: BatchDescribeReq): Promise<BatchDescribeResp> {
    return batchDescribeAttachments(req);
  }

  /** Resolves download semantics and a payload read ticket for a binding. */
  public static async ResolveDownloadContent(req: ResolveDownloadContentReq): Promise<ResolveDownloadContentResp> {
    return resolveDownloadContent(req);
  }

  protected static async buildDescriptorInternal(bindingId: string): Promise<AttachmentDescriptor> {
    return buildDescriptorForBinding(bindingId);
  }
}

const { _t } = createTranslate('document');

// ---------------------------------------------------------------------------
// Lazy imports to avoid circular dependency at module-init time.
// ---------------------------------------------------------------------------

function getAttachmentContentModel(): typeof AttachmentContent {
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  return require('./attachment_object').default as typeof AttachmentContent;
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
// Data-access helpers — some need the model ops, others hit external models
// ---------------------------------------------------------------------------

function assertPrincipalParityWithRuntimeContext(principal: PrincipalContext, stage: 'resolve_download_content'): void {
  const runtimeUserId = requireUserId(AttachmentBinding.userId);
  if (runtimeUserId !== principal.userId) {
    throwDocumentError(
      DocumentErrCode.PERMISSION_DENIED,
      _t('principal userId does not match runtime identity', { scope: 'service/models/attachment_binding' }),
      GrpcCode.PermissionDenied,
      {
        stage,
        reason: 'issuer_mismatch',
      }
    );
  }

  const runtimeCompanyId = requireCompanyId(AttachmentBinding.companyId, stage);
  if (runtimeCompanyId !== principal.activeCompanyId) {
    throwDocumentError(
      DocumentErrCode.PERMISSION_DENIED,
      _t('principal activeCompanyId does not match runtime context', { scope: 'service/models/attachment_binding' }),
      GrpcCode.PermissionDenied,
      {
        stage,
        reason: 'company_mismatch',
      }
    );
  }
}

async function mustLoadActiveAttachmentContent(attachmentContentId: string, companyId: string): Promise<AttachmentContent> {
  const AttachmentContentModel = getAttachmentContentModel();
  const rows = await AttachmentContentModel.Search(
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
    throwDocumentError(
      DocumentErrCode.NOT_FOUND,
      _t('Active attachment content not found in company scope', { scope: 'service/models/attachment_binding' }),
      GrpcCode.NotFound,
      {
        attachmentContentId,
        companyId,
      }
    );
  }

  return record;
}

async function mustLoadActiveAttachmentContentById(attachmentContentId: string): Promise<AttachmentContent> {
  const AttachmentContentModel = getAttachmentContentModel();
  const rows = await AttachmentContentModel.Search(
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
    throwDocumentError(
      DocumentErrCode.NOT_FOUND,
      _t('Active attachment content not found', { scope: 'service/models/attachment_binding' }),
      GrpcCode.NotFound,
      { attachmentContentId }
    );
  }
  return record;
}

async function mustLoadActiveStoredContentById(storedContentId: string): Promise<StoredContent> {
  const StoredContentModel = getStoredContentModel();
  const rows = await StoredContentModel.Search(
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
    throwDocumentError(
      DocumentErrCode.NOT_FOUND,
      _t('Active stored content not found', { scope: 'service/models/attachment_binding' }),
      GrpcCode.NotFound,
      { storedContentId }
    );
  }
  return record;
}

async function findActiveBinding(ownerModel: string,
  ownerRecordId: string,
  fieldName: string,
  companyId: string
): Promise<AttachmentBinding | null> {
  const rows = await AttachmentBinding.Search(
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
  return (rows[0] as AttachmentBinding) ?? null;
}

async function hardDeleteBindingById(bindingId: string, companyId: string): Promise<void> {
  const db = (globalThis as any)?.$choysum?.db;
  const execute = typeof db?.execute === 'function' ? db.execute.bind(db) : undefined;
  if (!execute) {
    throwDocumentError(
      DocumentErrCode.SKELETON_NOT_IMPLEMENTED,
      _t('database execute bridge is unavailable', { scope: 'service/models/attachment_binding' }),
      GrpcCode.Unimplemented,
      {
        stage: 'binding_cleanup',
      }
    );
  }

  await execute('delete from document_attachment_binding where id = ? and company_id = ?', JSON.stringify([bindingId, companyId]));
}

async function purgeConflictingUnboundBindings(ownerModel: string,
  ownerRecordId: string,
  fieldName: string,
  companyId: string,
  keepBindingId: string
): Promise<void> {
  const rows = await AttachmentBinding.Search(
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
    await hardDeleteBindingById(staleBindingId, companyId);
  }
}

async function mustLoadBinding(bindingId: string, companyId: string): Promise<AttachmentBinding> {
  const rows = await AttachmentBinding.Search(
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
    throwDocumentError(
      DocumentErrCode.NOT_FOUND,
      _t('Attachment binding not found', { scope: 'service/models/attachment_binding' }),
      GrpcCode.NotFound,
      { attachmentBindingId: bindingId, companyId }
    );
  }

  return record as AttachmentBinding;
}

async function mustLoadActiveBindingById(bindingId: string): Promise<AttachmentBinding> {
  const rows = await AttachmentBinding.Search(
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
    throwDocumentError(
      DocumentErrCode.NOT_FOUND,
      _t('Active attachment binding not found', { scope: 'service/models/attachment_binding' }),
      GrpcCode.NotFound,
      { attachmentBindingId: bindingId }
    );
  }
  return record as AttachmentBinding;
}

async function patchBindingPresentation(binding: AttachmentBinding,
  displayFileName: string | undefined,
  downloadDisposition: DownloadDisposition
): Promise<AttachmentBinding> {
  const bindingId = requireText(binding.Id, 'attachmentBindingId');
  const currentDisplayFileName = normalizeOptionalString(binding.DisplayFileName);
  const currentDisposition = assertDownloadDisposition(binding.DownloadDisposition);

  if (currentDisplayFileName === displayFileName && currentDisposition === downloadDisposition) {
    return binding;
  }

  await AttachmentBinding.UpdateById(
    bindingId,
    {
      DisplayFileName: displayFileName,
      DownloadDisposition: downloadDisposition,
    } as any,
    ['Id', 'DisplayFileName', 'DownloadDisposition'] as any
  );

  return mustLoadBinding(bindingId, requireText(binding.CompanyId, 'companyId'));
}

// ---------------------------------------------------------------------------
// Mutation-ledger helpers — operate on external models only
// ---------------------------------------------------------------------------

async function findLedgerRow(action: 'bind' | 'unbind', mutationId: string, companyId: string): Promise<AttachmentMutationLedger | null> {
  const AttachmentMutationLedgerModel = getAttachmentMutationLedgerModel();
  const rows = await AttachmentMutationLedgerModel.Search(
    {
      And: [
        ['Action', '=', action],
        ['MutationId', '=', mutationId],
        ['CompanyId', '=', companyId],
      ],
    } as any,
    { limit: 1 } as any
  );
  return (rows[0] as AttachmentMutationLedger) ?? null;
}

async function tryReplayBindMutation(mutationId: string, companyId: string): Promise<BindResp | null> {
  const row = await findLedgerRow('bind', mutationId, companyId);
  if (!row) return null;

  if (row.Status !== 'succeeded') {
    throwDocumentError(
      DocumentErrCode.FAILED_PRECONDITION,
      _t('bind mutationId exists but previous attempt did not succeed', { scope: 'service/models/attachment_binding' }),
      GrpcCode.FailedPrecondition,
      {
        mutationId,
        action: 'bind',
        status: String(row.Status || ''),
      }
    );
  }

  const snapshot = parseBindResp(row.ResponseJson);
  if (!snapshot) {
    throwDocumentError(
      DocumentErrCode.FAILED_PRECONDITION,
      _t('bind replay snapshot is invalid', { scope: 'service/models/attachment_binding' }),
      GrpcCode.FailedPrecondition,
      { mutationId, action: 'bind' }
    );
  }
  return snapshot;
}

async function tryReplayUnbindMutation(mutationId: string, companyId: string): Promise<UnbindResp | null> {
  const row = await findLedgerRow('unbind', mutationId, companyId);
  if (!row) return null;

  if (row.Status !== 'succeeded') {
    throwDocumentError(
      DocumentErrCode.FAILED_PRECONDITION,
      _t('unbind mutationId exists but previous attempt did not succeed', { scope: 'service/models/attachment_binding' }),
      GrpcCode.FailedPrecondition,
      {
        mutationId,
        action: 'unbind',
        status: String(row.Status || ''),
      }
    );
  }

  const snapshot = parseUnbindResp(row.ResponseJson);
  if (!snapshot) {
    throwDocumentError(
      DocumentErrCode.FAILED_PRECONDITION,
      _t('unbind replay snapshot is invalid', { scope: 'service/models/attachment_binding' }),
      GrpcCode.FailedPrecondition,
      {
        mutationId,
        action: 'unbind',
      }
    );
  }
  return snapshot;
}

async function recordMutationSuccess(
  action: 'bind' | 'unbind',
  mutationId: string,
  companyId: string,
  requestJson: Record<string, unknown>,
  responseJson: Record<string, unknown>
): Promise<void> {
  const AttachmentMutationLedgerModel = getAttachmentMutationLedgerModel();
  try {
    await AttachmentMutationLedgerModel.Create(
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
  } catch (_err) {
    const replayed = action === 'bind' ? await tryReplayBindMutation(mutationId, companyId) : await tryReplayUnbindMutation(mutationId, companyId);
    if (replayed) return;
    throw _err;
  }
}

// ---------------------------------------------------------------------------
// Public ops — called by the AttachmentBinding model class
// ---------------------------------------------------------------------------

async function bindAttachment(req: BindReq): Promise<BindResp> {
  const normalized = assertBindReq(req);
  const companyId = requireCompanyId(AttachmentBinding.companyId, 'bind');
  const userId = requireUserId(AttachmentBinding.userId);

  await assertOwnerWriteAuthorization({
    stage: 'bind',
    ownerModel: normalized.ownerModel,
    ownerRecordId: normalized.ownerRecordId,
    fieldName: normalized.fieldName,
    operation: 'update',
    companyId,
    companyIds: AttachmentBinding.companyIds,
    userId,
  });

  const replay = await tryReplayBindMutation(normalized.mutationId, companyId);
  if (replay) {
    return replay;
  }

  const attachmentContent = await mustLoadActiveAttachmentContent(normalized.attachmentContentId, companyId);
  validateAttachmentContentFieldLimits(normalized.ownerModel, normalized.fieldName, attachmentContent);
  const existing = await findActiveBinding(normalized.ownerModel, normalized.ownerRecordId, normalized.fieldName, companyId);

  let bindingRecord: AttachmentBinding;
  if (existing && requireText(existing.AttachmentContentId, 'attachmentContentId') === normalized.attachmentContentId) {
    bindingRecord = await patchBindingPresentation(existing, normalized.displayFileName, normalized.downloadDisposition);
  } else {
    if (existing) {
      await purgeConflictingUnboundBindings(
        normalized.ownerModel,
        normalized.ownerRecordId,
        normalized.fieldName,
        companyId,
        requireText(existing.Id, 'attachmentBindingId')
      );
      await AttachmentBinding.UpdateById(
        requireText(existing.Id, 'attachmentBindingId'),
        {
          Status: 'unbound',
          UnboundAt: new Date(),
        } as any,
        ['Id', 'Status', 'UnboundAt'] as any
      );
    }

    bindingRecord = (await AttachmentBinding.Create(
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
    )) as AttachmentBinding;
  }

  const descriptor = buildDescriptor(bindingRecord, attachmentContent);
  const response: BindResp = {
    attachmentBindingId: requireText(bindingRecord.Id, 'attachmentBindingId'),
    status: 'active',
    descriptor,
  };

  await recordMutationSuccess(
    'bind',
    normalized.mutationId,
    companyId,
    req as unknown as Record<string, unknown>,
    response as unknown as Record<string, unknown>
  );
  return response;
}

async function unbindAttachment(req: UnbindReq): Promise<UnbindResp> {
  const normalized = assertUnbindReq(req);
  const companyId = requireCompanyId(AttachmentBinding.companyId, 'unbind');
  const userId = requireUserId(AttachmentBinding.userId);

  const binding = await mustLoadBinding(normalized.attachmentBindingId, companyId);
  await assertOwnerWriteAuthorization({
    stage: 'unbind',
    ownerModel: requireText(binding.OwnerModel, 'ownerModel'),
    ownerRecordId: requireText(binding.OwnerRecordId, 'ownerRecordId'),
    fieldName: requireText(binding.FieldName, 'fieldName'),
    operation: 'update',
    companyId,
    companyIds: AttachmentBinding.companyIds,
    userId,
  });

  const replay = await tryReplayUnbindMutation(normalized.mutationId, companyId);
  if (replay) {
    return replay;
  }

  const unboundAt = binding.Status === 'unbound' ? (toDate(binding.UnboundAt) ?? new Date()) : new Date();

  if (binding.Status !== 'unbound') {
    await purgeConflictingUnboundBindings(
      requireText(binding.OwnerModel, 'ownerModel'),
      requireText(binding.OwnerRecordId, 'ownerRecordId'),
      requireText(binding.FieldName, 'fieldName'),
      companyId,
      normalized.attachmentBindingId
    );
    await AttachmentBinding.UpdateById(
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

  await recordMutationSuccess(
    'unbind',
    normalized.mutationId,
    companyId,
    req as unknown as Record<string, unknown>,
    response as unknown as Record<string, unknown>
  );
  return response;
}

function indexAttachmentContentsById(attachmentContents: unknown[]): Map<string, AttachmentContent> {
  const attachmentContentById = new Map<string, AttachmentContent>();
  for (const attachmentContent of attachmentContents) {
    const attachmentContentId = normalizeOptionalString((attachmentContent as any)?.Id);
    if (!attachmentContentId) {
      continue;
    }
    attachmentContentById.set(attachmentContentId, attachmentContent as AttachmentContent);
  }
  return attachmentContentById;
}

function collectBindingsForDescribe(bindings: unknown[]): {
  bindingById: Map<string, AttachmentBinding>;
  attachmentContentIds: string[];
} {
  const bindingById = new Map<string, AttachmentBinding>();
  const attachmentContentIds: string[] = [];
  for (const binding of bindings) {
    const bindingId = normalizeOptionalString((binding as any)?.Id);
    const attachmentContentId = normalizeOptionalString((binding as any)?.AttachmentContentId);
    if (!bindingId || !attachmentContentId) {
      continue;
    }
    bindingById.set(bindingId, binding as AttachmentBinding);
    attachmentContentIds.push(attachmentContentId);
  }
  return { bindingById, attachmentContentIds };
}

async function batchDescribeAttachments(req: BatchDescribeReq): Promise<BatchDescribeResp> {
  const AttachmentContentModel = getAttachmentContentModel();

  const normalized = assertBatchDescribeReq(req);
  if (normalized.attachmentBindingIds.length === 0) {
    return { items: [] };
  }

  const userId = requireUserId(AttachmentBinding.userId);
  const companyId = requireCompanyId(AttachmentBinding.companyId, 'descriptor');

  const bindings = await AttachmentBinding.Search(
    {
      And: [
        ['Id', 'in', normalized.attachmentBindingIds],
        ['CompanyId', '=', companyId],
        ['Status', '=', 'active'],
      ],
    } as any,
    { limit: normalized.attachmentBindingIds.length } as any
  );

  const { bindingById, attachmentContentIds } = collectBindingsForDescribe(bindings);

  if (attachmentContentIds.length === 0) {
    return { items: [] };
  }

  const dedupedAttachmentContentIds = Array.from(new Set(attachmentContentIds));
  const attachmentContents = await AttachmentContentModel.Search(
    {
      And: [
        ['Id', 'in', dedupedAttachmentContentIds],
        ['CompanyId', '=', companyId],
        ['Status', '=', 'active'],
      ],
    } as any,
    { limit: dedupedAttachmentContentIds.length } as any
  );

  const attachmentContentById = indexAttachmentContentsById(attachmentContents);

  const itemPromises = normalized.attachmentBindingIds.map(async bindingId => {
    const binding = bindingById.get(bindingId);
    if (!binding) return null;

    const attachmentContentId = requireText(binding.AttachmentContentId, 'attachmentContentId');
    const attachmentContent = attachmentContentById.get(attachmentContentId);
    if (!attachmentContent) return null;

    await assertOwnerReadAuthorization({
      stage: 'descriptor',
      ownerModel: requireText(binding.OwnerModel, 'ownerModel'),
      ownerRecordId: requireText(binding.OwnerRecordId, 'ownerRecordId'),
      fieldName: requireText(binding.FieldName, 'fieldName'),
      companyId,
      companyIds: AttachmentBinding.companyIds,
      userId,
    });

    const descriptor = buildDescriptor(binding, attachmentContent);
    const displayName = normalizeOptionalString(binding.DisplayFileName) ?? descriptor.fileName;
    const isImage = descriptor.mimeType.toLowerCase().startsWith('image/');

    return {
      attachmentBindingId: bindingId,
      descriptor,
      displayName,
      previewUrl: isImage ? descriptor.downloadUrl : undefined,
      ownerModel: requireText(binding.OwnerModel, 'ownerModel'),
      ownerRecordId: requireText(binding.OwnerRecordId, 'ownerRecordId'),
      fieldName: requireText(binding.FieldName, 'fieldName'),
    };
  });

  const resolvedItems = await Promise.all(itemPromises);
  const items: BatchDescribeResp['items'] = resolvedItems.filter((item): item is NonNullable<typeof item> => item !== null);

  return { items };
}

async function resolveDownloadContent(req: ResolveDownloadContentReq): Promise<ResolveDownloadContentResp> {
  const normalized = assertResolveDownloadContentReq(req);
  assertPrincipalParityWithRuntimeContext(normalized.principal, 'resolve_download_content');

  const binding = await mustLoadActiveBindingById(normalized.attachmentBindingId);
  const bindingCompanyId = requireText(binding.CompanyId, 'companyId');
  assertCompanyMatch(bindingCompanyId, normalized.principal.activeCompanyId, 'resolve_download_content', {
    resource: 'attachmentBinding',
    attachmentBindingId: normalized.attachmentBindingId,
  });

  const attachmentContentId = requireText(binding.AttachmentContentId, 'attachmentContentId');
  const attachmentContent = await mustLoadActiveAttachmentContentById(attachmentContentId);
  assertCompanyMatch(requireText(attachmentContent.CompanyId, 'companyId'), normalized.principal.activeCompanyId, 'resolve_download_content', {
    resource: 'attachmentContent',
    attachmentContentId,
  });

  await assertOwnerReadAuthorization({
    stage: 'resolve_download_content',
    ownerModel: requireText(binding.OwnerModel, 'ownerModel'),
    ownerRecordId: requireText(binding.OwnerRecordId, 'ownerRecordId'),
    fieldName: requireText(binding.FieldName, 'fieldName'),
    companyId: bindingCompanyId,
    companyIds: normalizePrincipalCompanyIds(normalized.principal, bindingCompanyId),
    userId: normalized.principal.userId,
  });

  const storedContentId = requireText((attachmentContent as any)?.StoredContentId, 'storedContentId');
  const storedContent = await mustLoadActiveStoredContentById(storedContentId);
  assertCompanyMatch(requireText((storedContent as any)?.CompanyId, 'companyId'), normalized.principal.activeCompanyId, 'resolve_download_content', {
    resource: 'storedContent',
    storedContentId,
  });

  const semantics = resolveDownloadSemantics(binding, attachmentContent);

  return {
    attachmentBindingId: normalized.attachmentBindingId,
    payloadReadTicket: buildPayloadReadTicket(normalized.attachmentBindingId, attachmentContentId, storedContentId),
    mimeType: semantics.mimeType,
    sizeBytes: semantics.sizeBytes,
    checksumSha256: semantics.checksumSha256,
    fileName: semantics.fileName,
    downloadDisposition: semantics.downloadDisposition,
    etag: semantics.etag,
  };
}

async function buildDescriptorForBinding(bindingId: string): Promise<AttachmentDescriptor> {
  const userId = requireUserId(AttachmentBinding.userId);
  const companyId = requireCompanyId(AttachmentBinding.companyId, 'descriptor');

  const normalizedBindingId = requireText(bindingId, 'attachmentBindingId');
  const binding = await mustLoadActiveBindingById(normalizedBindingId);
  assertCompanyMatch(requireText(binding.CompanyId, 'companyId'), companyId, 'descriptor', {
    resource: 'attachmentBinding',
    attachmentBindingId: normalizedBindingId,
  });

  const attachmentContentId = requireText(binding.AttachmentContentId, 'attachmentContentId');
  const attachmentContent = await mustLoadActiveAttachmentContentById(attachmentContentId);
  assertCompanyMatch(requireText(attachmentContent.CompanyId, 'companyId'), companyId, 'descriptor', {
    resource: 'attachmentContent',
    attachmentContentId,
  });

  await assertOwnerReadAuthorization({
    stage: 'descriptor',
    ownerModel: requireText(binding.OwnerModel, 'ownerModel'),
    ownerRecordId: requireText(binding.OwnerRecordId, 'ownerRecordId'),
    fieldName: requireText(binding.FieldName, 'fieldName'),
    companyId,
    companyIds: AttachmentBinding.companyIds,
    userId,
  });

  return buildDescriptor(binding, attachmentContent);
}

/** Test seam for binding hard-delete cleanup when db.execute is unavailable. */
export async function documentHardDeleteBindingForTest(bindingId: string, companyId: string): Promise<void> {
  return hardDeleteBindingById(bindingId, companyId);
}

/** Test seam for purge keep-id skip and hard-delete paths during conflict cleanup. */
export async function documentPurgeConflictingUnboundBindingsForTest(
  ownerModel: string,
  ownerRecordId: string,
  fieldName: string,
  companyId: string,
  keepBindingId: string
): Promise<void> {
  return purgeConflictingUnboundBindings(ownerModel, ownerRecordId, fieldName, companyId, keepBindingId);
}

/** Test seam for mutation-ledger Create failure with successful replay. */
export async function documentRecordMutationSuccessForTest(
  action: 'bind' | 'unbind',
  mutationId: string,
  companyId: string,
  requestJson: Record<string, unknown>,
  responseJson: Record<string, unknown>
): Promise<void> {
  return recordMutationSuccess(action, mutationId, companyId, requestJson, responseJson);
}

/** Test seam for BatchDescribe content indexing when rows lack Id. */
export function documentIndexAttachmentContentsByIdForTest(rows: unknown[]): Map<string, unknown> {
  return indexAttachmentContentsById(rows);
}

/** Test seam for BatchDescribe binding collection when rows lack Id/content Id. */
export function documentCollectBindingsForDescribeForTest(rows: unknown[]): {
  bindingIds: string[];
  attachmentContentIds: string[];
} {
  const collected = collectBindingsForDescribe(rows);
  return {
    bindingIds: Array.from(collected.bindingById.keys()),
    attachmentContentIds: collected.attachmentContentIds,
  };
}
