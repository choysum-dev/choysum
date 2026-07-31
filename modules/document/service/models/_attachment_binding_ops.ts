// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { normalizeOptionalString } from '@/core/service/utils/normalization';
import { createTranslate } from '@/core/service/i18n';
import BaseModel from '@/core/service/orm/model/model';
import { MetadataStorage } from '@/core/service/orm/metadata/storage';
import type { FieldMetadata } from '@/core/service/orm/metadata/field';
import { toDate } from '@/core/service/utils/datetime';
import {
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
import { DocumentErrCode, GrpcCode, throwDocumentError } from '../error';
import type AttachmentBinding from './attachment_binding';
import type AttachmentContent from './attachment_object';
import type AttachmentMutationLedger from './attachment_mutation_ledger';
import type StoredContent from './stored_content';
import { requireText, requireUserId, requireCompanyId } from './_normalizers';
import { assertOwnerReadAuthorization, assertOwnerWriteAuthorization } from './_owner_authorization';
import {
  normalizeBindReq,
  normalizeUnbindReq,
  normalizeBatchDescribeReq,
  normalizeResolveDownloadContentReq,
  normalizePrincipalCompanyIds,
  normalizeDownloadDisposition,
  resolveDownloadSemantics,
  buildDescriptor,
  buildPayloadReadTicket,
  parseBindResp,
  parseUnbindResp,
  assertCompanyMatch,
} from './_attachment_binding_codec';
import { DEFAULT_GLOBAL_MAX_UPLOAD_BYTES } from './_upload';

const { _t } = createTranslate('document');

function resolveOwnerFieldMetadata(ownerModel: string, fieldName: string): FieldMetadata | undefined {
  const ModelCtor = BaseModel.resolveModelConstructor(ownerModel);
  if (!ModelCtor) {
    return undefined;
  }
  const modelMeta = MetadataStorage.instance.getModelMetadata(ModelCtor);
  if (!modelMeta) {
    return undefined;
  }
  return modelMeta.fields.get(fieldName);
}

function resolveEffectiveMaxUploadBytes(maxUploadBytes: number): number {
  return Math.min(maxUploadBytes, DEFAULT_GLOBAL_MAX_UPLOAD_BYTES);
}

/**
 * Validates finalized attachment content against per-field upload/dimension limits (PR-P2-F3).
 * Dimension probe is skipped when ImageWidth/ImageHeight are absent (future work).
 */
export function validateAttachmentContentFieldLimits(
  ownerModel: string,
  fieldName: string,
  attachmentContent: AttachmentContent
): void {
  const fieldMeta = resolveOwnerFieldMetadata(ownerModel, fieldName);
  if (!fieldMeta) {
    return;
  }

  const hasByteLimit = typeof fieldMeta.maxUploadBytes === 'number' && fieldMeta.maxUploadBytes > 0;
  const hasWidthLimit = fieldMeta.type === 'image' && typeof fieldMeta.maxWidth === 'number' && fieldMeta.maxWidth > 0;
  const hasHeightLimit = fieldMeta.type === 'image' && typeof fieldMeta.maxHeight === 'number' && fieldMeta.maxHeight > 0;
  if (!hasByteLimit && !hasWidthLimit && !hasHeightLimit) {
    return;
  }

  if (hasByteLimit) {
    const effectiveMaxBytes = resolveEffectiveMaxUploadBytes(fieldMeta.maxUploadBytes as number);
    const sizeBytes = Number((attachmentContent as any).SizeBytes ?? 0);
    if (Number.isFinite(sizeBytes) && sizeBytes > effectiveMaxBytes) {
      throwDocumentError(
        DocumentErrCode.INVALID_ARGUMENT,
        _t('upload exceeds field maxUploadBytes', { scope: 'service/models/_attachment_binding_ops' }),
        GrpcCode.InvalidArgument,
        {
          ownerModel,
          fieldName,
          sizeBytes: String(sizeBytes),
          maxUploadBytes: String(effectiveMaxBytes),
        }
      );
    }
  }

  if (fieldMeta.type === 'image' && (hasWidthLimit || hasHeightLimit)) {
    const imageWidth = (attachmentContent as any).ImageWidth;
    const imageHeight = (attachmentContent as any).ImageHeight;
    const width = typeof imageWidth === 'number' && Number.isFinite(imageWidth) ? Math.trunc(imageWidth) : undefined;
    const height = typeof imageHeight === 'number' && Number.isFinite(imageHeight) ? Math.trunc(imageHeight) : undefined;
    // Skip dimension check when probe data is missing; byte validation still applies above.
    if (width === undefined && height === undefined) {
      return;
    }
    if (hasWidthLimit && width !== undefined && width > (fieldMeta.maxWidth as number)) {
      throwDocumentError(
        DocumentErrCode.INVALID_ARGUMENT,
        _t('upload exceeds field maxWidth', { scope: 'service/models/_attachment_binding_ops' }),
        GrpcCode.InvalidArgument,
        {
          ownerModel,
          fieldName,
          imageWidth: String(width),
          maxWidth: String(fieldMeta.maxWidth),
        }
      );
    }
    if (hasHeightLimit && height !== undefined && height > (fieldMeta.maxHeight as number)) {
      throwDocumentError(
        DocumentErrCode.INVALID_ARGUMENT,
        _t('upload exceeds field maxHeight', { scope: 'service/models/_attachment_binding_ops' }),
        GrpcCode.InvalidArgument,
        {
          ownerModel,
          fieldName,
          imageHeight: String(height),
          maxHeight: String(fieldMeta.maxHeight),
        }
      );
    }
  }
}

// ---------------------------------------------------------------------------
// Model ops contract — passed by the AttachmentBinding class when delegating.
// ---------------------------------------------------------------------------

export interface BindingModelOps {
  readonly userId: unknown;
  readonly companyId: unknown;
  readonly companyIds: string[];
  Search(condition: unknown, options?: unknown): Promise<unknown[]>;
  Create(values: unknown, fields?: unknown): Promise<unknown>;
  UpdateById(id: string, values: unknown, fields?: unknown): Promise<void>;
}

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

function assertPrincipalParityWithRuntimeContext(ops: BindingModelOps, principal: PrincipalContext, stage: 'resolve_download_content'): void {
  const runtimeUserId = requireUserId(ops.userId);
  if (runtimeUserId !== principal.userId) {
    throwDocumentError(
      DocumentErrCode.PERMISSION_DENIED,
      _t('principal userId does not match runtime identity', { scope: 'service/models/_attachment_binding_ops' }),
      GrpcCode.PermissionDenied,
      {
        stage,
        reason: 'issuer_mismatch',
      }
    );
  }

  const runtimeCompanyId = requireCompanyId(ops.companyId, stage);
  if (runtimeCompanyId !== principal.activeCompanyId) {
    throwDocumentError(
      DocumentErrCode.PERMISSION_DENIED,
      _t('principal activeCompanyId does not match runtime context', { scope: 'service/models/_attachment_binding_ops' }),
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
      _t('Active attachment content not found in company scope', { scope: 'service/models/_attachment_binding_ops' }),
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
      _t('Active attachment content not found', { scope: 'service/models/_attachment_binding_ops' }),
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
      _t('Active stored content not found', { scope: 'service/models/_attachment_binding_ops' }),
      GrpcCode.NotFound,
      { storedContentId }
    );
  }
  return record;
}

async function findActiveBinding(
  ops: BindingModelOps,
  ownerModel: string,
  ownerRecordId: string,
  fieldName: string,
  companyId: string
): Promise<AttachmentBinding | null> {
  const rows = await ops.Search(
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
      _t('database execute bridge is unavailable', { scope: 'service/models/_attachment_binding_ops' }),
      GrpcCode.Unimplemented,
      {
        stage: 'binding_cleanup',
      }
    );
  }

  await execute('delete from document_attachment_binding where id = ? and company_id = ?', JSON.stringify([bindingId, companyId]));
}

async function purgeConflictingUnboundBindings(
  ops: BindingModelOps,
  ownerModel: string,
  ownerRecordId: string,
  fieldName: string,
  companyId: string,
  keepBindingId: string
): Promise<void> {
  const rows = await ops.Search(
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

async function mustLoadBinding(ops: BindingModelOps, bindingId: string, companyId: string): Promise<AttachmentBinding> {
  const rows = await ops.Search(
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
      _t('Attachment binding not found', { scope: 'service/models/_attachment_binding_ops' }),
      GrpcCode.NotFound,
      { attachmentBindingId: bindingId, companyId }
    );
  }

  return record as AttachmentBinding;
}

async function mustLoadActiveBindingById(ops: BindingModelOps, bindingId: string): Promise<AttachmentBinding> {
  const rows = await ops.Search(
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
      _t('Active attachment binding not found', { scope: 'service/models/_attachment_binding_ops' }),
      GrpcCode.NotFound,
      { attachmentBindingId: bindingId }
    );
  }
  return record as AttachmentBinding;
}

async function patchBindingPresentation(
  ops: BindingModelOps,
  binding: AttachmentBinding,
  displayFileName: string | undefined,
  downloadDisposition: DownloadDisposition
): Promise<AttachmentBinding> {
  const bindingId = requireText(binding.Id, 'attachmentBindingId');
  const currentDisplayFileName = normalizeOptionalString(binding.DisplayFileName);
  const currentDisposition = normalizeDownloadDisposition(binding.DownloadDisposition);

  if (currentDisplayFileName === displayFileName && currentDisposition === downloadDisposition) {
    return binding;
  }

  await ops.UpdateById(
    bindingId,
    {
      DisplayFileName: displayFileName,
      DownloadDisposition: downloadDisposition,
    } as any,
    ['Id', 'DisplayFileName', 'DownloadDisposition'] as any
  );

  return mustLoadBinding(ops, bindingId, requireText(binding.CompanyId, 'companyId'));
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
      _t('bind mutationId exists but previous attempt did not succeed', { scope: 'service/models/_attachment_binding_ops' }),
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
      _t('bind replay snapshot is invalid', { scope: 'service/models/_attachment_binding_ops' }),
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
      _t('unbind mutationId exists but previous attempt did not succeed', { scope: 'service/models/_attachment_binding_ops' }),
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
      _t('unbind replay snapshot is invalid', { scope: 'service/models/_attachment_binding_ops' }),
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

export async function bindAttachment(ops: BindingModelOps, req: BindReq): Promise<BindResp> {
  const normalized = normalizeBindReq(req);
  const companyId = requireCompanyId(ops.companyId, 'bind');
  const userId = requireUserId(ops.userId);

  await assertOwnerWriteAuthorization({
    stage: 'bind',
    ownerModel: normalized.ownerModel,
    ownerRecordId: normalized.ownerRecordId,
    fieldName: normalized.fieldName,
    operation: 'update',
    companyId,
    companyIds: ops.companyIds,
    userId,
  });

  const replay = await tryReplayBindMutation(normalized.mutationId, companyId);
  if (replay) {
    return replay;
  }

  const attachmentContent = await mustLoadActiveAttachmentContent(normalized.attachmentContentId, companyId);
  validateAttachmentContentFieldLimits(normalized.ownerModel, normalized.fieldName, attachmentContent);
  const existing = await findActiveBinding(ops, normalized.ownerModel, normalized.ownerRecordId, normalized.fieldName, companyId);

  let bindingRecord: AttachmentBinding;
  if (existing && requireText(existing.AttachmentContentId, 'attachmentContentId') === normalized.attachmentContentId) {
    bindingRecord = await patchBindingPresentation(ops, existing, normalized.displayFileName, normalized.downloadDisposition);
  } else {
    if (existing) {
      await purgeConflictingUnboundBindings(
        ops,
        normalized.ownerModel,
        normalized.ownerRecordId,
        normalized.fieldName,
        companyId,
        requireText(existing.Id, 'attachmentBindingId')
      );
      await ops.UpdateById(
        requireText(existing.Id, 'attachmentBindingId'),
        {
          Status: 'unbound',
          UnboundAt: new Date(),
        } as any,
        ['Id', 'Status', 'UnboundAt'] as any
      );
    }

    bindingRecord = (await ops.Create(
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
    (req as unknown as Record<string, unknown>) ?? {},
    response as unknown as Record<string, unknown>
  );
  return response;
}

export async function unbindAttachment(ops: BindingModelOps, req: UnbindReq): Promise<UnbindResp> {
  const normalized = normalizeUnbindReq(req);
  const companyId = requireCompanyId(ops.companyId, 'unbind');
  const userId = requireUserId(ops.userId);

  const binding = await mustLoadBinding(ops, normalized.attachmentBindingId, companyId);
  await assertOwnerWriteAuthorization({
    stage: 'unbind',
    ownerModel: requireText(binding.OwnerModel, 'ownerModel'),
    ownerRecordId: requireText(binding.OwnerRecordId, 'ownerRecordId'),
    fieldName: requireText(binding.FieldName, 'fieldName'),
    operation: 'update',
    companyId,
    companyIds: ops.companyIds,
    userId,
  });

  const replay = await tryReplayUnbindMutation(normalized.mutationId, companyId);
  if (replay) {
    return replay;
  }

  const unboundAt = binding.Status === 'unbound' ? (toDate(binding.UnboundAt) ?? new Date()) : new Date();

  if (binding.Status !== 'unbound') {
    await purgeConflictingUnboundBindings(
      ops,
      requireText(binding.OwnerModel, 'ownerModel'),
      requireText(binding.OwnerRecordId, 'ownerRecordId'),
      requireText(binding.FieldName, 'fieldName'),
      companyId,
      normalized.attachmentBindingId
    );
    await ops.UpdateById(
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
    (req as unknown as Record<string, unknown>) ?? {},
    response as unknown as Record<string, unknown>
  );
  return response;
}

export async function batchDescribeAttachments(ops: BindingModelOps, req: BatchDescribeReq): Promise<BatchDescribeResp> {
  const AttachmentContentModel = getAttachmentContentModel();

  const normalized = normalizeBatchDescribeReq(req);
  if (normalized.attachmentBindingIds.length === 0) {
    return { items: [] };
  }

  const userId = requireUserId(ops.userId);
  const companyId = requireCompanyId(ops.companyId, 'descriptor');

  const bindings = await ops.Search(
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
    const bindingId = normalizeOptionalString((binding as any).Id);
    const attachmentContentId = normalizeOptionalString((binding as any).AttachmentContentId);
    if (!bindingId || !attachmentContentId) {
      continue;
    }
    bindingById.set(bindingId, binding as AttachmentBinding);
    attachmentContentIds.push(attachmentContentId);
  }

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

  const attachmentContentById = new Map<string, AttachmentContent>();
  for (const attachmentContent of attachmentContents) {
    const attachmentContentId = normalizeOptionalString((attachmentContent as any).Id);
    if (!attachmentContentId) {
      continue;
    }
    attachmentContentById.set(attachmentContentId, attachmentContent as AttachmentContent);
  }

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
      companyIds: ops.companyIds,
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

export async function resolveDownloadContent(ops: BindingModelOps, req: ResolveDownloadContentReq): Promise<ResolveDownloadContentResp> {
  const normalized = normalizeResolveDownloadContentReq(req);
  assertPrincipalParityWithRuntimeContext(ops, normalized.principal, 'resolve_download_content');

  const binding = await mustLoadActiveBindingById(ops, normalized.attachmentBindingId);
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

export async function buildDescriptorForBinding(ops: BindingModelOps, bindingId: string): Promise<AttachmentDescriptor> {
  const userId = requireUserId(ops.userId);
  const companyId = requireCompanyId(ops.companyId, 'descriptor');

  const normalizedBindingId = requireText(bindingId, 'attachmentBindingId');
  const binding = await mustLoadActiveBindingById(ops, normalizedBindingId);
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
    companyIds: ops.companyIds,
    userId,
  });

  return buildDescriptor(binding, attachmentContent);
}
