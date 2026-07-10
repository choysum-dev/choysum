// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { normalizeOptionalString, asRecord } from '@/core/service/utils/normalization';
import { toDate } from '@/core/service/utils/date';
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
  NormalizedBindReq,
  NormalizedUnbindReq,
  NormalizedBatchDescribeReq,
  NormalizedResolveDownloadContentReq,
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
} from './_attachment_binding_helpers';

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
  const runtimeUserId = normalizeOptionalString(ops.userId);
  if (runtimeUserId && runtimeUserId !== principal.userId) {
    throwDocumentError(DocumentErrCode.PERMISSION_DENIED, 'principal userId does not match runtime identity', GrpcCode.PermissionDenied, {
      stage,
      reason: 'issuer_mismatch',
    });
  }

  const runtimeCompanyId = normalizeOptionalString(ops.companyId);
  if (runtimeCompanyId && runtimeCompanyId !== principal.activeCompanyId) {
    throwDocumentError(DocumentErrCode.PERMISSION_DENIED, 'principal activeCompanyId does not match runtime context', GrpcCode.PermissionDenied, {
      stage,
      reason: 'company_mismatch',
    });
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
    throwDocumentError(DocumentErrCode.NOT_FOUND, 'Active attachment content not found in company scope', GrpcCode.NotFound, {
      attachmentContentId,
      companyId,
    });
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
    throwDocumentError(DocumentErrCode.NOT_FOUND, 'Active attachment content not found', GrpcCode.NotFound, { attachmentContentId });
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
    throwDocumentError(DocumentErrCode.NOT_FOUND, 'Active stored content not found', GrpcCode.NotFound, { storedContentId });
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
    throwDocumentError(DocumentErrCode.SKELETON_NOT_IMPLEMENTED, 'database execute bridge is unavailable', GrpcCode.Unimplemented, {
      stage: 'binding_cleanup',
    });
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
    throwDocumentError(DocumentErrCode.NOT_FOUND, 'Attachment binding not found', GrpcCode.NotFound, { attachmentBindingId: bindingId, companyId });
  }

  return record as AttachmentBinding;
}

async function mustLoadActiveBinding(ops: BindingModelOps, bindingId: string, companyId: string): Promise<AttachmentBinding> {
  const rows = await ops.Search(
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
    throwDocumentError(DocumentErrCode.NOT_FOUND, 'Active attachment binding not found', GrpcCode.NotFound, { attachmentBindingId: bindingId, companyId });
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
    throwDocumentError(DocumentErrCode.NOT_FOUND, 'Active attachment binding not found', GrpcCode.NotFound, { attachmentBindingId: bindingId });
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
    throwDocumentError(DocumentErrCode.FAILED_PRECONDITION, 'bind mutationId exists but previous attempt did not succeed', GrpcCode.FailedPrecondition, {
      mutationId,
      action: 'bind',
      status: String(row.Status || ''),
    });
  }

  const snapshot = parseBindResp(row.ResponseJson);
  if (!snapshot) {
    throwDocumentError(DocumentErrCode.FAILED_PRECONDITION, 'bind replay snapshot is invalid', GrpcCode.FailedPrecondition, { mutationId, action: 'bind' });
  }
  return snapshot;
}

async function tryReplayUnbindMutation(mutationId: string, companyId: string): Promise<UnbindResp | null> {
  const row = await findLedgerRow('unbind', mutationId, companyId);
  if (!row) return null;

  if (row.Status !== 'succeeded') {
    throwDocumentError(DocumentErrCode.FAILED_PRECONDITION, 'unbind mutationId exists but previous attempt did not succeed', GrpcCode.FailedPrecondition, {
      mutationId,
      action: 'unbind',
      status: String(row.Status || ''),
    });
  }

  const snapshot = parseUnbindResp(row.ResponseJson);
  if (!snapshot) {
    throwDocumentError(DocumentErrCode.FAILED_PRECONDITION, 'unbind replay snapshot is invalid', GrpcCode.FailedPrecondition, {
      mutationId,
      action: 'unbind',
    });
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
        ['Id'] as any
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
      companyIds: ops.companyIds,
      userId,
    });

    const descriptor = buildDescriptor(binding, attachmentContent);
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
