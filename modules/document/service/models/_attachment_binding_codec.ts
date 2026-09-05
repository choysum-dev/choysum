// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  normalizeOptionalString,
  asRecord,
  normalizeOptionalNonNegativeInt,
  normalizeChecksumSha256,
  parseDownloadDisposition,
} from '@/core/service/utils/normalization';
import { createTranslate } from '@/core/service/i18n';
import {
  DownloadDisposition,
  BindReq,
  BindResp,
  UnbindReq,
  UnbindResp,
  AttachmentDescriptor,
  BatchDescribeReq,
  ResolveDownloadContentReq,
  PrincipalContext,
} from '../contracts';
import { DocumentErrCode, GrpcCode, throwDocumentError } from '../error';
import type AttachmentBinding from './attachment_binding';
import type AttachmentContent from './attachment_object';
import { requireText, assertPrincipal, normalizeCompanyIdList } from './_document_bridge';
import { inlineMimeAllowed, mimeSuffix } from '@/core/service/utils/mime';

const { _t } = createTranslate('document');

// ---------------------------------------------------------------------------
// Internal normalised-request shapes
// ---------------------------------------------------------------------------

export type NormalizedBindReq = {
  attachmentContentId: string;
  ownerModel: string;
  ownerRecordId: string;
  fieldName: string;
  displayFileName?: string;
  downloadDisposition: DownloadDisposition;
  mutationId: string;
};

export type NormalizedUnbindReq = {
  attachmentBindingId: string;
  mutationId: string;
};

export type NormalizedBatchDescribeReq = {
  attachmentBindingIds: string[];
};

export type NormalizedResolveDownloadContentReq = {
  attachmentBindingId: string;
  principal: PrincipalContext;
};

export type ResolvedDownloadSemantics = {
  fileName: string;
  mimeType: string;
  sizeBytes: number;
  checksumSha256?: string;
  downloadDisposition: DownloadDisposition;
  etag?: string;
};

// ---------------------------------------------------------------------------
// Request assertions
// ---------------------------------------------------------------------------

export function assertBindReq(req: BindReq | undefined | null): NormalizedBindReq {
  return {
    attachmentContentId: requireText(req?.attachmentObjectId, 'attachmentObjectId'),
    ownerModel: requireText(req?.ownerModel, 'ownerModel'),
    ownerRecordId: requireText(req?.ownerRecordId, 'ownerRecordId'),
    fieldName: requireText(req?.fieldName, 'fieldName'),
    displayFileName: normalizeOptionalString(req?.displayFileName),
    downloadDisposition: assertDownloadDisposition(req?.downloadDisposition),
    mutationId: requireText(req?.mutationId, 'mutationId'),
  };
}

export function assertUnbindReq(req: UnbindReq | undefined | null): NormalizedUnbindReq {
  return {
    attachmentBindingId: requireText(req?.attachmentBindingId, 'attachmentBindingId'),
    mutationId: requireText(req?.mutationId, 'mutationId'),
  };
}

export function assertBatchDescribeReq(req: BatchDescribeReq | undefined | null): NormalizedBatchDescribeReq {
  const rawIds = req?.attachmentBindingIds;
  if (rawIds === undefined || rawIds === null) {
    return { attachmentBindingIds: [] };
  }
  if (!Array.isArray(rawIds)) {
    throwDocumentError(
      DocumentErrCode.INVALID_ARGUMENT,
      _t('attachmentBindingIds must be an array', { scope: 'service/models/_attachment_binding_codec' }),
      GrpcCode.InvalidArgument,
      {
        field: 'attachmentBindingIds',
      }
    );
  }

  const MAX_BATCH_SIZE = 200;
  if (rawIds.length > MAX_BATCH_SIZE) {
    throwDocumentError(
      DocumentErrCode.INVALID_ARGUMENT,
      _t('attachmentBindingIds exceeds maximum batch size of %s', { scope: 'service/models/_attachment_binding_codec' }, MAX_BATCH_SIZE),
      GrpcCode.InvalidArgument,
      {
        field: 'attachmentBindingIds',
      }
    );
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

export function assertResolveDownloadContentReq(req: ResolveDownloadContentReq | undefined | null): NormalizedResolveDownloadContentReq {
  return {
    attachmentBindingId: requireText(req?.attachmentBindingId, 'attachmentBindingId'),
    principal: assertPrincipal(req?.principal),
  };
}

// ---------------------------------------------------------------------------
// Domain normalizers
// ---------------------------------------------------------------------------

export function normalizePrincipalCompanyIds(principal: PrincipalContext, activeCompanyId: string): string[] {
  return normalizeCompanyIdList(principal.enabledCompanyIds, activeCompanyId);
}

export function assertDownloadDisposition(value: unknown): DownloadDisposition {
  const parsed = parseDownloadDisposition(value);
  if (!parsed.ok) {
    throwDocumentError(
      DocumentErrCode.INVALID_ARGUMENT,
      _t('downloadDisposition must be inline or attachment', { scope: 'service/models/_attachment_binding_codec' }),
      GrpcCode.InvalidArgument,
      {
        downloadDisposition: parsed.raw,
      }
    );
  }
  return parsed.value ?? 'attachment';
}

// ---------------------------------------------------------------------------
// Response builders
// ---------------------------------------------------------------------------

export function resolveDownloadSemantics(binding: AttachmentBinding, attachmentContent: AttachmentContent): ResolvedDownloadSemantics {
  const mimeType = normalizeOptionalString(attachmentContent.MimeType) ?? 'application/octet-stream';
  const sizeBytes = normalizeOptionalNonNegativeInt(attachmentContent.SizeBytes) ?? 0;
  const checksumSha256 = normalizeChecksumSha256(attachmentContent.ChecksumSha256);

  return {
    fileName: buildFileName(binding, attachmentContent),
    mimeType,
    sizeBytes,
    checksumSha256,
    downloadDisposition: resolveDownloadDispositionForResponse(binding.DownloadDisposition, mimeType),
    etag: buildEtag(checksumSha256),
  };
}

export function resolveDownloadDispositionForResponse(value: unknown, mimeType: string): DownloadDisposition {
  const requested = assertDownloadDisposition(value);
  if (requested === 'inline' && inlineMimeAllowed(mimeType)) {
    return 'inline';
  }
  return 'attachment';
}

export function buildEtag(checksumSha256: string | undefined): string | undefined {
  if (!checksumSha256) return undefined;
  return `"sha256:${checksumSha256}"`;
}

export function buildFileName(binding: AttachmentBinding, attachmentContent: AttachmentContent): string {
  const displayFileName = normalizeOptionalString(binding.DisplayFileName);
  if (displayFileName) {
    return displayFileName;
  }

  const mime = normalizeOptionalString(attachmentContent.MimeType) ?? '';
  const suffix = mimeSuffix(mime);
  const bindingId = requireText(binding.Id, 'attachmentBindingId');
  return `attachment-${bindingId}${suffix}`;
}

export function buildPayloadReadTicket(attachmentBindingId: string, attachmentContentId: string, storedContentId: string): string {
  return JSON.stringify({
    attachmentBindingId,
    attachmentContentId,
    storedContentId,
  });
}

export function buildDescriptor(binding: AttachmentBinding, attachmentContent: AttachmentContent): AttachmentDescriptor {
  const bindingId = requireText(binding.Id, 'attachmentBindingId');
  const semantics = resolveDownloadSemantics(binding, attachmentContent);
  return {
    id: bindingId,
    fileName: semantics.fileName,
    mimeType: semantics.mimeType,
    sizeBytes: semantics.sizeBytes,
    checksumSha256: semantics.checksumSha256 ?? '',
    downloadUrl: `/_document/bindings/${bindingId}/content`,
  };
}

// ---------------------------------------------------------------------------
// Response parsers
// ---------------------------------------------------------------------------

export function parseBindResp(value: unknown): BindResp | null {
  const record = asRecord(value);
  if (!record) return null;

  const attachmentBindingId = normalizeOptionalString(record.attachmentBindingId);
  const status = normalizeOptionalString(record.status);
  const descriptorRaw = asRecord(record.descriptor);
  if (!attachmentBindingId || status !== 'active' || !descriptorRaw) return null;

  const descriptorId = normalizeOptionalString(descriptorRaw.id);
  const fileName = normalizeOptionalString(descriptorRaw.fileName);
  const mimeType = normalizeOptionalString(descriptorRaw.mimeType);
  const checksumSha256 = normalizeOptionalString(descriptorRaw.checksumSha256) ?? '';
  const sizeBytes = normalizeOptionalNonNegativeInt(descriptorRaw.sizeBytes);
  if (!descriptorId || !fileName || !mimeType || sizeBytes === undefined) return null;

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

export function parseUnbindResp(value: unknown): UnbindResp | null {
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

// ---------------------------------------------------------------------------
// Assert helpers
// ---------------------------------------------------------------------------

export function assertCompanyMatch(
  actualCompanyId: string,
  expectedCompanyId: string,
  stage: 'bind' | 'unbind' | 'descriptor' | 'resolve_download_content',
  metadata: Record<string, unknown>
): void {
  if (actualCompanyId === expectedCompanyId) return;
  throwDocumentError(
    DocumentErrCode.PERMISSION_DENIED,
    _t('Attachment resource company scope mismatch', { scope: 'service/models/_attachment_binding_codec' }),
    GrpcCode.PermissionDenied,
    {
      stage,
      ...metadata,
      expectedCompanyId,
      actualCompanyId,
    }
  );
}
