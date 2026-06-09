// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  ATTACHMENT_UPLOAD_SESSION_STATUS_VALUES,
  ATTACHMENT_UPLOAD_SESSION_TRANSITIONS,
  COMMIT_UPLOAD_IDEMPOTENCY_RULES,
  DOCUMENT_PERMISSION_DENIED_REASON_VALUES,
  DOCUMENT_PERMISSION_DENIED_STAGE_VALUES,
  isAttachmentUploadSessionTransitionAllowed,
  type AuthorizeUploadPutReq,
  type CommitUploadPutReq,
  type ResolveDownloadContentReq,
} from './contracts';

test('document contracts: upload session status transition contract is frozen', () => {
  expect(ATTACHMENT_UPLOAD_SESSION_STATUS_VALUES).toEqual(['prepared', 'uploaded', 'finalized', 'expired']);

  expect(ATTACHMENT_UPLOAD_SESSION_TRANSITIONS.prepared).toEqual(['uploaded', 'expired']);
  expect(ATTACHMENT_UPLOAD_SESSION_TRANSITIONS.uploaded).toEqual(['finalized', 'expired']);
  expect(ATTACHMENT_UPLOAD_SESSION_TRANSITIONS.finalized).toEqual([]);
  expect(ATTACHMENT_UPLOAD_SESSION_TRANSITIONS.expired).toEqual([]);

  expect(isAttachmentUploadSessionTransitionAllowed('prepared', 'uploaded')).toBe(true);
  expect(isAttachmentUploadSessionTransitionAllowed('prepared', 'finalized')).toBe(false);
  expect(isAttachmentUploadSessionTransitionAllowed('uploaded', 'finalized')).toBe(true);
  expect(isAttachmentUploadSessionTransitionAllowed('finalized', 'uploaded')).toBe(false);

  expect(COMMIT_UPLOAD_IDEMPOTENCY_RULES.duplicateCommit).toBe('return_current_state');
  expect(COMMIT_UPLOAD_IDEMPOTENCY_RULES.concurrentCommit).toBe('first_writer_wins');
  expect(COMMIT_UPLOAD_IDEMPOTENCY_RULES.expiredSession).toBe('reject_gone');
  expect(COMMIT_UPLOAD_IDEMPOTENCY_RULES.finalizedSession).toBe('reject_conflict');
});

test('document contracts: permission denied stage/reason contract exports expected labels', () => {
  expect(DOCUMENT_PERMISSION_DENIED_STAGE_VALUES).toContain('download');
  expect(DOCUMENT_PERMISSION_DENIED_STAGE_VALUES).toContain('authorize_upload_put');
  expect(DOCUMENT_PERMISSION_DENIED_STAGE_VALUES).toContain('commit_upload_put');
  expect(DOCUMENT_PERMISSION_DENIED_STAGE_VALUES).toContain('resolve_download_content');

  expect(DOCUMENT_PERMISSION_DENIED_REASON_VALUES).toContain('binding_company_mismatch');
  expect(DOCUMENT_PERMISSION_DENIED_REASON_VALUES).toContain('attachment_company_mismatch');
  expect(DOCUMENT_PERMISSION_DENIED_REASON_VALUES).toContain('owner_record_rule_false');
  expect(DOCUMENT_PERMISSION_DENIED_REASON_VALUES).toContain('owner_field_read_deny');
  expect(DOCUMENT_PERMISSION_DENIED_REASON_VALUES).toContain('unknown');
});

test('document contracts: stage-1 request shape compile smoke', () => {
  const authorizeReq: AuthorizeUploadPutReq = {
    uploadId: 'up_001',
    principal: {
      userId: 'usr_001',
      activeCompanyId: 'cmp_001',
      enabledCompanyIds: ['cmp_001'],
    },
    requestMeta: {
      contentType: 'image/png',
      contentLength: 128,
      checksumSha256: 'a'.repeat(64),
    },
  };

  const commitReq: CommitUploadPutReq = {
    uploadId: 'up_001',
    principal: authorizeReq.principal,
    payloadReceipt: {
      payloadId: 'pld_001',
      sizeBytes: 128,
      checksumSha256: 'b'.repeat(64),
      contentType: 'image/png',
    },
  };

  const resolveReq: ResolveDownloadContentReq = {
    attachmentBindingId: 'bnd_001',
    principal: authorizeReq.principal,
  };

  expect(authorizeReq.uploadId).toBe('up_001');
  expect(commitReq.payloadReceipt.payloadId).toBe('pld_001');
  expect(resolveReq.attachmentBindingId).toBe('bnd_001');
});
