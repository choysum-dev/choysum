// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { withContext } from '@/core/service/api/context';
import { ChoysumError } from '@/core/service/error';
import AttachmentObject from '../models/attachment_object';
import UploadSession from '../models/upload_session';
import StoredContent from '../models/stored_content';
import { ensureAuthUserOwnerRecordRuleGrants, disableRepositoryRecordRuleForDocumentTests } from './_owner_auth_test_fixtures';

const RR_CACHE_KEY = Symbol.for('choysum.recordrule.cache');
const FR_CACHE_KEY = Symbol.for('choysum.fieldrule.cache');

const TEST_COMPANY_ID = 'cmp_document_test';
const TEST_USER_ID = 'usr_document_test';
const EMPTY_SHA256 = 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855';

function uid(prefix: string): string {
  const xid = (globalThis as any).$choysum?.xid?.New?.();
  const token = typeof xid === 'string' && xid.trim() ? xid.trim() : `${Date.now()}${Math.random()}`;
  return `${prefix}_${token}`;
}

function ensureRequestContext(): any {
  const root: any = (globalThis as any).$choysum ?? {};
  if (!root.request) root.request = {};
  if (!root.request.context) root.request.context = {};

  const jsCtx = root.request.context;
  if (!jsCtx.ctx) jsCtx.ctx = {};
  if (!jsCtx.req) jsCtx.req = {};
  if (!jsCtx.identity) jsCtx.identity = {};

  (globalThis as any).$choysum = root;
  return jsCtx;
}

function resetRequestContext(): void {
  const jsCtx = ensureRequestContext();
  jsCtx.ctx = {};
  jsCtx.req = {
    depth: 0,
    fieldRuleMode: 'skip',
  };
  jsCtx.identity = {
    userId: TEST_USER_ID,
  };

  delete (jsCtx as any)[Symbol.for('choysum.ctx.override')];
  delete (jsCtx as any)[Symbol.for('choysum.ctx.frozen')];
  delete (jsCtx as any)[RR_CACHE_KEY];
  delete (jsCtx as any)[FR_CACHE_KEY];
  disableRepositoryRecordRuleForDocumentTests();
}

async function withDocumentScope<T>(fn: () => Promise<T>): Promise<T> {
  return withContext(
    {
      activeCompanyId: TEST_COMPANY_ID,
      enabledCompanyIds: [TEST_COMPANY_ID],
    } as any,
    async () => {
      await ensureAuthUserOwnerRecordRuleGrants();
      return fn();
    },
    { merge: false }
  );
}

async function withScope<T>(companyId: string, userId: string, fn: () => Promise<T>): Promise<T> {
  return withContext(
    {
      activeCompanyId: companyId,
      enabledCompanyIds: [companyId],
    } as any,
    async () => {
      const jsCtx = ensureRequestContext();
      jsCtx.ctx = {};
      jsCtx.req = {
        depth: 0,
        fieldRuleMode: 'skip',
      };
      jsCtx.identity = {
        userId,
      };
      await ensureAuthUserOwnerRecordRuleGrants();
      return fn();
    },
    { merge: false }
  );
}

function buildPrincipalContext(
  companyId = TEST_COMPANY_ID,
  userId = TEST_USER_ID
): {
  userId: string;
  activeCompanyId: string;
  enabledCompanyIds: string[];
} {
  return {
    userId,
    activeCompanyId: companyId,
    enabledCompanyIds: [companyId],
  };
}

async function markSessionUploaded(uploadId: string, contentType = 'text/plain', companyId = TEST_COMPANY_ID): Promise<void> {
  const storedContentId = await createStoredContentRecord({
    companyId,
    backend: 'db',
    dbPayload: '',
  });

  await UploadSession.UpdateById(
    uploadId,
    {
      Status: 'uploaded',
      UploadedSizeBytes: 0,
      UploadedChecksumSha256: EMPTY_SHA256,
      UploadedContentType: contentType,
      UploadedPayloadRef: {
        kind: 'stored_content',
        storedContentId,
      },
    } as any,
    ['Id'] as any
  );
}

function futureNowISO(daysAhead = 2): string {
  return new Date(Date.now() + daysAhead * 24 * 60 * 60 * 1000).toISOString();
}

async function createActiveAttachmentObjectRecord(input: {
  companyId: string;
  backend: 'db' | 's3';
  sizeBytes: number;
  mimeType: string;
  checksumSha256?: string;
  dbPayload?: string;
  bucket?: string;
  documentKey?: string;
}): Promise<string> {
  const checksumSha256 = input.checksumSha256 ?? EMPTY_SHA256;

  const storedContentId = await createStoredContentRecord({
    companyId: input.companyId,
    backend: input.backend,
    dbPayload: input.dbPayload,
    bucket: input.bucket,
    documentKey: input.documentKey,
  });

  const created = await AttachmentObject.Create(
    {
      StoredContentId: storedContentId,
      SizeBytes: input.sizeBytes,
      MimeType: input.mimeType,
      ChecksumSha256: checksumSha256,
      Status: 'active',
      CompanyId: input.companyId,
    } as any,
    ['Id'] as any
  );

  const objectId = String((created as any)?.Id || '').trim();
  if (!objectId) {
    throw new Error('failed to create attachment content fixture');
  }

  return objectId;
}

async function createStoredContentRecord(input: {
  companyId: string;
  backend: 'db' | 's3';
  dbPayload?: string;
  bucket?: string;
  documentKey?: string;
}): Promise<string> {
  const storedPayload: Record<string, unknown> = {
    Provider: input.backend,
    Status: 'active',
    CompanyId: input.companyId,
  };
  if (input.backend === 's3') {
    storedPayload.LocatorJson = {
      bucket: input.bucket ?? 'choysum-attachments-test',
      key: input.documentKey ?? `s3/test/${uid('obj')}`,
    };
  } else if (input.dbPayload !== undefined && input.dbPayload !== '') {
    storedPayload.BlobData = input.dbPayload;
  }

  const stored = await StoredContent.Create(storedPayload as any, ['Id'] as any);
  const storedContentId = String((stored as any)?.Id || '').trim();
  if (!storedContentId) {
    throw new Error('failed to create stored content fixture');
  }

  return storedContentId;
}

async function loadAttachmentObjectById(objectId: string): Promise<any> {
  const rows = await AttachmentObject.Search(
    ['Id', '=', objectId] as any,
    {
      limit: 1,
      fields: ['Id', 'Status', 'StoredContentId', 'MetadataJson'] as any,
    } as any
  );
  expect(rows.length).toBe(1);

  const objectRecord = rows[0] as any;
  const storedContentId = String(objectRecord?.StoredContentId || '').trim();
  expect(storedContentId).toBeTruthy();

  const storedRows = await StoredContent.Search(
    ['Id', '=', storedContentId] as any,
    {
      limit: 1,
      fields: ['Id', 'Provider', 'BlobData', 'LocatorJson', 'Status'] as any,
    } as any
  );
  expect(storedRows.length).toBe(1);

  const stored = storedRows[0] as any;
  return {
    ...objectRecord,
    Backend: stored?.Provider,
    BlobData: stored?.BlobData,
    LocatorJson: stored?.LocatorJson,
    StoredContentStatus: stored?.Status,
  };
}

function cleanupStateOf(record: any): Record<string, any> {
  const metadata = (record?.MetadataJson as Record<string, any> | undefined) ?? {};
  const cleanup = metadata.cleanup;
  if (!cleanup || typeof cleanup !== 'object' || Array.isArray(cleanup)) {
    return {};
  }
  return cleanup as Record<string, any>;
}

test('document.attachment_object: PrepareUpload rejects unauthenticated and creates no upload session', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const businessRequestId = uid('biz_prepare_unauth');
    const jsCtx = ensureRequestContext();
    jsCtx.identity = {};

    try {
      await AttachmentObject.PrepareUpload({
        ownerModel: 'auth.User',
        ownerRecordId: uid('owner_no_auth'),
        fieldName: 'Avatar',
        operation: 'update',
        businessRequestId,
      });
      throw new Error('expected unauthenticated error');
    } catch (err) {
      expect(err instanceof ChoysumError).toBe(true);
      const oe = err as ChoysumError;
      expect(oe.domain).toBe('document');
      expect(oe.code).toBe('UNAUTHENTICATED');
    }

    const sessions = await UploadSession.Search(['BusinessRequestId', '=', businessRequestId] as any, { limit: 1 } as any);
    expect(sessions.length).toBe(0);
  });
});

test('document.attachment_object: PrepareUpload rejects when owner write authorization denies', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const businessRequestId = uid('biz_prepare_owner_denied');
    try {
      await AttachmentObject.PrepareUpload({
        ownerModel: 'unknown.Model',
        ownerRecordId: uid('owner_unknown_model'),
        fieldName: 'Avatar',
        operation: 'update',
        businessRequestId,
      });
      throw new Error('expected owner authorization denied error');
    } catch (err) {
      expect(err instanceof ChoysumError).toBe(true);
      const oe = err as ChoysumError;
      expect(oe.domain).toBe('document');
      expect(oe.code).toBe('PERMISSION_DENIED');
      expect(oe.metadata?.stage).toBe('prepare');
    }

    const sessions = await UploadSession.Search(['BusinessRequestId', '=', businessRequestId] as any, { limit: 1 } as any);
    expect(sessions.length).toBe(0);
  });
});

test('document.attachment_object: PrepareUpload replays same uploadId for same businessRequestId', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const req = {
      ownerModel: 'auth.User',
      ownerRecordId: uid('owner'),
      fieldName: 'Avatar',
      operation: 'update' as const,
      businessRequestId: uid('biz_prepare_replay'),
      proposedFileName: 'avatar.png',
      proposedContentType: 'image/png',
      proposedSizeBytes: 12,
    };

    const first = await AttachmentObject.PrepareUpload(req);
    const replay = await AttachmentObject.PrepareUpload(req);

    expect(first.uploadId).toBeTruthy();
    expect(replay.uploadId).toBe(first.uploadId);
    expect(first.uploadTarget.url).toBe(`/_document/uploads/${first.uploadId}`);
    expect(first.uploadTarget.method).toBe('PUT');
  });
});

test('document.attachment_object: PrepareUpload idempotency key is isolated by company and issuer', async () => {
  resetRequestContext();
  const businessRequestId = uid('biz_prepare_scope');

  const req = {
    ownerModel: 'auth.User',
    ownerRecordId: uid('owner_scope'),
    fieldName: 'Avatar',
    operation: 'update' as const,
    businessRequestId,
  };

  const first = await withScope('cmp_scope_a', 'usr_scope_a', async () => AttachmentObject.PrepareUpload(req));
  const second = await withScope('cmp_scope_b', 'usr_scope_b', async () => AttachmentObject.PrepareUpload(req));

  expect(first.uploadId).toBeTruthy();
  expect(second.uploadId).toBeTruthy();
  expect(second.uploadId).not.toBe(first.uploadId);
});

test('document.attachment_object: AuthorizeUploadPut rejects caller mismatch', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const prepared = await AttachmentObject.PrepareUpload({
      ownerModel: 'auth.User',
      ownerRecordId: uid('owner_authorize_mismatch'),
      fieldName: 'Avatar',
      operation: 'update',
      businessRequestId: uid('biz_authorize_mismatch'),
    });

    try {
      await AttachmentObject.AuthorizeUploadPut({
        uploadId: prepared.uploadId,
        principal: buildPrincipalContext(TEST_COMPANY_ID, uid('usr_mismatch')),
      });
      throw new Error('expected principal mismatch error');
    } catch (err) {
      expect(err instanceof ChoysumError).toBe(true);
      const oe = err as ChoysumError;
      expect(oe.domain).toBe('document');
      expect(oe.code).toBe('PERMISSION_DENIED');
      expect(oe.metadata?.stage).toBe('authorize_upload_put');
    }
  });
});

test('document.attachment_object: AuthorizeUploadPut enforces max upload size', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const prepared = await AttachmentObject.PrepareUpload({
      ownerModel: 'auth.User',
      ownerRecordId: uid('owner_authorize_size'),
      fieldName: 'IdentityDocument',
      operation: 'update',
      businessRequestId: uid('biz_authorize_size'),
      proposedContentType: 'text/plain',
    });

    await UploadSession.UpdateById(
      prepared.uploadId,
      {
        MaxUploadBytes: 4,
      } as any,
      ['Id'] as any
    );

    try {
      await AttachmentObject.AuthorizeUploadPut({
        uploadId: prepared.uploadId,
        principal: buildPrincipalContext(),
        requestMeta: {
          contentType: 'text/plain',
          contentLength: 8,
        },
      });
      throw new Error('expected max upload size error');
    } catch (err) {
      expect(err instanceof ChoysumError).toBe(true);
      const oe = err as ChoysumError;
      expect(oe.domain).toBe('document');
      expect(oe.code).toBe('UPLOAD_TOO_LARGE');
    }
  });
});

test('document.attachment_object: AuthorizeUploadPut validates mime and checksum constraints', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const expectedChecksum = 'a'.repeat(64);
    const prepared = await AttachmentObject.PrepareUpload({
      ownerModel: 'auth.User',
      ownerRecordId: uid('owner_authorize_constraints'),
      fieldName: 'IdentityDocument',
      operation: 'update',
      businessRequestId: uid('biz_authorize_constraints'),
      checksumSha256: expectedChecksum,
      proposedContentType: 'image/png',
    });

    await UploadSession.UpdateById(
      prepared.uploadId,
      {
        AllowedMimeTypes: ['image/png'],
      } as any,
      ['Id'] as any
    );

    try {
      await AttachmentObject.AuthorizeUploadPut({
        uploadId: prepared.uploadId,
        principal: buildPrincipalContext(),
        requestMeta: {
          contentType: 'text/plain',
          contentLength: 1,
          checksumSha256: expectedChecksum,
        },
      });
      throw new Error('expected mime type deny error');
    } catch (err) {
      expect(err instanceof ChoysumError).toBe(true);
      const oe = err as ChoysumError;
      expect(oe.domain).toBe('document');
      expect(oe.code).toBe('MIME_TYPE_NOT_ALLOWED');
    }

    try {
      await AttachmentObject.AuthorizeUploadPut({
        uploadId: prepared.uploadId,
        principal: buildPrincipalContext(),
        requestMeta: {
          contentType: 'image/png',
          contentLength: 1,
          checksumSha256: 'b'.repeat(64),
        },
      });
      throw new Error('expected checksum mismatch error');
    } catch (err) {
      expect(err instanceof ChoysumError).toBe(true);
      const oe = err as ChoysumError;
      expect(oe.domain).toBe('document');
      expect(oe.code).toBe('CHECKSUM_MISMATCH');
    }
  });
});

test('document.attachment_object: AuthorizeUploadPut treats object AllowedMimeTypes as unset allow-list', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const prepared = await AttachmentObject.PrepareUpload({
      ownerModel: 'auth.User',
      ownerRecordId: uid('owner_authorize_object_allowlist'),
      fieldName: 'Avatar',
      operation: 'update',
      businessRequestId: uid('biz_authorize_object_allowlist'),
      proposedContentType: 'image/jpeg',
    });

    await UploadSession.UpdateById(
      prepared.uploadId,
      {
        AllowedMimeTypes: {} as any,
      } as any,
      ['Id'] as any
    );

    const authorized = await AttachmentObject.AuthorizeUploadPut({
      uploadId: prepared.uploadId,
      principal: buildPrincipalContext(),
      requestMeta: {
        contentType: 'image/jpeg',
        contentLength: 1,
      },
    });

    expect(authorized.uploadId).toBe(prepared.uploadId);
    expect(typeof authorized.payloadWriteTicket).toBe('string');
    expect(authorized.payloadWriteTicket.length).toBeGreaterThan(0);
  });
});

test('document.attachment_object: CommitUploadPut persists uploaded state and supports duplicate commit', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const expectedChecksum = 'c'.repeat(64);
    const prepared = await AttachmentObject.PrepareUpload({
      ownerModel: 'auth.User',
      ownerRecordId: uid('owner_commit_ok'),
      fieldName: 'IdentityDocument',
      operation: 'update',
      businessRequestId: uid('biz_commit_ok'),
      checksumSha256: expectedChecksum,
      proposedContentType: 'text/plain',
    });

    await UploadSession.UpdateById(
      prepared.uploadId,
      {
        AllowedMimeTypes: ['text/plain'],
      } as any,
      ['Id'] as any
    );

    const first = await AttachmentObject.CommitUploadPut({
      uploadId: prepared.uploadId,
      principal: buildPrincipalContext(),
      payloadReceipt: {
        payloadId: `sc:${await createStoredContentRecord({ companyId: TEST_COMPANY_ID, backend: 'db', dbPayload: 'abc' })}`,
        sizeBytes: 3,
        checksumSha256: expectedChecksum,
        contentType: 'text/plain',
      },
    });

    expect(first.uploadId).toBe(prepared.uploadId);
    expect(first.attachmentUploadSessionStatus).toBe('uploaded');

    const replay = await AttachmentObject.CommitUploadPut({
      uploadId: prepared.uploadId,
      principal: buildPrincipalContext(),
      payloadReceipt: {
        payloadId: String(
          (await UploadSession.Search(['Id', '=', prepared.uploadId] as any, { limit: 1 } as any))[0]?.UploadedPayloadRef?.storedContentId
            ? `sc:${(await UploadSession.Search(['Id', '=', prepared.uploadId] as any, { limit: 1 } as any))[0]?.UploadedPayloadRef?.storedContentId}`
            : ''
        ),
        sizeBytes: 3,
        checksumSha256: expectedChecksum,
        contentType: 'text/plain',
      },
    });

    expect(replay.uploadId).toBe(prepared.uploadId);
    expect(replay.attachmentUploadSessionStatus).toBe('uploaded');

    const rows = await UploadSession.Search(['Id', '=', prepared.uploadId] as any, { limit: 1 } as any);
    expect(rows.length).toBe(1);
    const saved = rows[0] as any;
    expect(String(saved.Status || '')).toBe('uploaded');
    expect(String(saved.UploadedChecksumSha256 || '')).toBe(expectedChecksum);
    expect(Number(saved.UploadedSizeBytes || 0)).toBe(3);
    expect(String(saved.UploadedContentType || '')).toBe('text/plain');
    expect(saved.UploadedPayloadRef).toEqual({
      kind: 'stored_content',
      storedContentId: String(saved.UploadedPayloadRef?.storedContentId || ''),
    });
  });
});

test('document.attachment_object: CommitUploadPut rejects expired upload session', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const prepared = await AttachmentObject.PrepareUpload({
      ownerModel: 'auth.User',
      ownerRecordId: uid('owner_commit_expired'),
      fieldName: 'IdentityDocument',
      operation: 'update',
      businessRequestId: uid('biz_commit_expired'),
      proposedContentType: 'text/plain',
    });

    await UploadSession.UpdateById(
      prepared.uploadId,
      {
        ExpiresAt: new Date(Date.now() - 60 * 1000),
      } as any,
      ['Id'] as any
    );

    try {
      const storedContentId = await createStoredContentRecord({ companyId: TEST_COMPANY_ID, backend: 'db', dbPayload: 'abc' });
      await AttachmentObject.CommitUploadPut({
        uploadId: prepared.uploadId,
        principal: buildPrincipalContext(),
        payloadReceipt: {
          payloadId: `sc:${storedContentId}`,
          sizeBytes: 3,
          checksumSha256: EMPTY_SHA256,
          contentType: 'text/plain',
        },
      });
      throw new Error('expected expired session error');
    } catch (err) {
      expect(err instanceof ChoysumError).toBe(true);
      const oe = err as ChoysumError;
      expect(oe.domain).toBe('document');
      expect(oe.code).toBe('UPLOAD_SESSION_EXPIRED');
    }
  });
});

test('document.attachment_object: CommitUploadPut rejects inline byte payload ids', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const prepared = await AttachmentObject.PrepareUpload({
      ownerModel: 'auth.User',
      ownerRecordId: uid('owner_commit_inline_payload'),
      fieldName: 'IdentityDocument',
      operation: 'update',
      businessRequestId: uid('biz_commit_inline_payload'),
      proposedContentType: 'application/octet-stream',
    });

    try {
      await AttachmentObject.CommitUploadPut({
        uploadId: prepared.uploadId,
        principal: buildPrincipalContext(),
        payloadReceipt: {
          payloadId: 'inline_base64:AAECAw==',
          sizeBytes: 3,
          checksumSha256: EMPTY_SHA256,
          contentType: 'application/octet-stream',
        },
      });
      throw new Error('expected inline payload id rejection');
    } catch (err) {
      expect(err instanceof ChoysumError).toBe(true);
      const oe = err as ChoysumError;
      expect(oe.domain).toBe('document');
      expect(oe.code).toBe('INVALID_ARGUMENT');
      expect(oe.metadata?.field).toBe('payloadReceipt.payloadId');
    }
  });
});

test('document.attachment_object: CommitUploadPut rejects data-url payload ids', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const prepared = await AttachmentObject.PrepareUpload({
      ownerModel: 'auth.User',
      ownerRecordId: uid('owner_commit_data_url_payload'),
      fieldName: 'IdentityDocument',
      operation: 'update',
      businessRequestId: uid('biz_commit_data_url_payload'),
      proposedContentType: 'application/octet-stream',
    });

    try {
      await AttachmentObject.CommitUploadPut({
        uploadId: prepared.uploadId,
        principal: buildPrincipalContext(),
        payloadReceipt: {
          payloadId: 'data:application/octet-stream;base64,AAECAw==',
          sizeBytes: 3,
          checksumSha256: EMPTY_SHA256,
          contentType: 'application/octet-stream',
        },
      });
      throw new Error('expected data-url payload id rejection');
    } catch (err) {
      expect(err instanceof ChoysumError).toBe(true);
      const oe = err as ChoysumError;
      expect(oe.domain).toBe('document');
      expect(oe.code).toBe('INVALID_ARGUMENT');
      expect(oe.metadata?.field).toBe('payloadReceipt.payloadId');
    }
  });
});

test('document.attachment_object: FinalizeUpload creates active object and supports replay', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const businessRequestId = uid('biz_finalize');
    const prepared = await AttachmentObject.PrepareUpload({
      ownerModel: 'auth.User',
      ownerRecordId: uid('owner_finalize'),
      fieldName: 'IdentityDocument',
      operation: 'update',
      businessRequestId,
      proposedFileName: 'doc.txt',
      proposedContentType: 'text/plain',
      proposedSizeBytes: 0,
    });

    await markSessionUploaded(prepared.uploadId, 'text/plain');

    const finalized = await AttachmentObject.FinalizeUpload({
      uploadId: prepared.uploadId,
      businessRequestId,
    });

    expect(finalized.status).toBe('active');
    expect(finalized.attachmentObjectId).toBeTruthy();
    expect(finalized.mimeType).toBe('text/plain');

    const replay = await AttachmentObject.FinalizeUpload({
      uploadId: prepared.uploadId,
      businessRequestId,
    });
    expect(replay.attachmentObjectId).toBe(finalized.attachmentObjectId);
  });
});

test('document.attachment_object: FinalizeUpload accepts stored_content payload handle and reuses payload row', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const businessRequestId = uid('biz_finalize_reuse_stored_content');
    const prepared = await AttachmentObject.PrepareUpload({
      ownerModel: 'auth.User',
      ownerRecordId: uid('owner_finalize_reuse_stored_content'),
      fieldName: 'IdentityDocument',
      operation: 'update',
      businessRequestId,
      proposedFileName: 'doc.bin',
      proposedContentType: 'application/octet-stream',
      proposedSizeBytes: 5,
    });

    const precreatedStoredContent = await StoredContent.Create(
      {
        Provider: 'db',
        BlobData: 'hello',
        Status: 'active',
        CompanyId: TEST_COMPANY_ID,
      } as any,
      ['Id'] as any
    );
    const storedContentId = String((precreatedStoredContent as any)?.Id || '').trim();
    expect(storedContentId).toBeTruthy();

    await UploadSession.UpdateById(
      prepared.uploadId,
      {
        Status: 'uploaded',
        UploadedSizeBytes: 5,
        UploadedChecksumSha256: EMPTY_SHA256,
        UploadedContentType: 'application/octet-stream',
        UploadedPayloadRef: {
          kind: 'stored_content',
          storedContentId,
        },
      } as any,
      ['Id'] as any
    );

    const finalized = await AttachmentObject.FinalizeUpload({
      uploadId: prepared.uploadId,
      businessRequestId,
    });

    const reloaded = await loadAttachmentObjectById(finalized.attachmentObjectId);
    expect(String(reloaded.StoredContentId || '')).toBe(storedContentId);
    expect(String(reloaded.BlobData || '')).toBe('hello');
  });
});

test('document.attachment_object: FinalizeUpload rejects uploaded session without payload handle', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const businessRequestId = uid('biz_finalize_missing_payload_ref');
    const prepared = await AttachmentObject.PrepareUpload({
      ownerModel: 'auth.User',
      ownerRecordId: uid('owner_finalize_missing_payload_ref'),
      fieldName: 'IdentityDocument',
      operation: 'update',
      businessRequestId,
      proposedFileName: 'doc.bin',
      proposedContentType: 'application/octet-stream',
      proposedSizeBytes: 5,
    });

    await UploadSession.UpdateById(
      prepared.uploadId,
      {
        Status: 'uploaded',
        UploadedSizeBytes: 5,
        UploadedChecksumSha256: EMPTY_SHA256,
        UploadedContentType: 'application/octet-stream',
      } as any,
      ['Id'] as any
    );

    try {
      await AttachmentObject.FinalizeUpload({
        uploadId: prepared.uploadId,
        businessRequestId,
      });
      throw new Error('expected missing uploaded payload reference rejection');
    } catch (err) {
      expect(err instanceof ChoysumError).toBe(true);
      const oe = err as ChoysumError;
      expect(oe.domain).toBe('document');
      expect(oe.code).toBe('FAILED_PRECONDITION');
    }
  });
});

test('document.attachment_object: CommitUploadPut rejects legacy s3 payload ids', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const prepared = await AttachmentObject.PrepareUpload({
      ownerModel: 'auth.User',
      ownerRecordId: uid('owner_commit_legacy_s3_payload'),
      fieldName: 'IdentityDocument',
      operation: 'update',
      businessRequestId: uid('biz_commit_legacy_s3_payload'),
      proposedContentType: 'application/octet-stream',
    });

    try {
      await AttachmentObject.CommitUploadPut({
        uploadId: prepared.uploadId,
        principal: buildPrincipalContext(),
        payloadReceipt: {
          payloadId: `s3://choysum-attachments-test/staging/${prepared.uploadId}/payload`,
          sizeBytes: 3,
          checksumSha256: EMPTY_SHA256,
          contentType: 'application/octet-stream',
        },
      });
      throw new Error('expected legacy s3 payload id rejection');
    } catch (err) {
      expect(err instanceof ChoysumError).toBe(true);
      const oe = err as ChoysumError;
      expect(oe.domain).toBe('document');
      expect(oe.code).toBe('INVALID_ARGUMENT');
      expect(oe.metadata?.field).toBe('payloadReceipt.payloadId');
    }
  });
});

test('document.attachment_object: FinalizeUpload reuses stored_content payload row when provider=s3', async () => {
  resetRequestContext();

  const envKey = '__choysumBackendEnv';
  const root = globalThis as any;
  const previousEnv = root[envKey];
  root[envKey] = {
    ...(previousEnv && typeof previousEnv === 'object' ? previousEnv : {}),
    CHOYSUM_DOCUMENT_ATTACHMENT_BACKEND: 's3',
  };

  try {
    await withDocumentScope(async () => {
      const businessRequestId = uid('biz_finalize_s3');
      const prepared = await AttachmentObject.PrepareUpload({
        ownerModel: 'auth.User',
        ownerRecordId: uid('owner_finalize_s3'),
        fieldName: 'IdentityDocument',
        operation: 'update',
        businessRequestId,
        proposedFileName: 'doc.txt',
        proposedContentType: 'text/plain',
        proposedSizeBytes: 0,
      });

      const stagingRef = {
        backend: 's3',
        bucket: 'choysum-attachments-test',
        key: `staging/${prepared.uploadId}/manual`,
      };
      const storedContentId = await createStoredContentRecord({
        companyId: TEST_COMPANY_ID,
        backend: 's3',
        bucket: stagingRef.bucket,
        documentKey: stagingRef.key,
      });
      await UploadSession.UpdateById(
        prepared.uploadId,
        {
          Status: 'uploaded',
          UploadedSizeBytes: 0,
          UploadedChecksumSha256: EMPTY_SHA256,
          UploadedContentType: 'text/plain',
          UploadedPayloadRef: {
            kind: 'stored_content',
            storedContentId,
          },
        } as any,
        ['Id'] as any
      );

      const finalized = await AttachmentObject.FinalizeUpload({
        uploadId: prepared.uploadId,
        businessRequestId,
      });

      const reloaded = await loadAttachmentObjectById(finalized.attachmentObjectId);
      expect(String(reloaded.Backend || '')).toBe('s3');
      expect(reloaded.LocatorJson).toMatchObject({ bucket: stagingRef.bucket, key: stagingRef.key });
      expect(reloaded.BlobData === null || reloaded.BlobData === undefined || reloaded.BlobData === '').toBe(true);
    });
  } finally {
    if (previousEnv === undefined) {
      delete root[envKey];
    } else {
      root[envKey] = previousEnv;
    }
  }
});

test('document.attachment_object: FinalizeUpload rejects when upload session has not been uploaded', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const businessRequestId = uid('biz_finalize_without_upload');
    const prepared = await AttachmentObject.PrepareUpload({
      ownerModel: 'auth.User',
      ownerRecordId: uid('owner_finalize_without_upload'),
      fieldName: 'Avatar',
      operation: 'update',
      businessRequestId,
    });

    try {
      await AttachmentObject.FinalizeUpload({
        uploadId: prepared.uploadId,
        businessRequestId,
      });
      throw new Error('expected finalize precondition error');
    } catch (err) {
      expect(err instanceof ChoysumError).toBe(true);
      const oe = err as ChoysumError;
      expect(oe.domain).toBe('document');
      expect(oe.code).toBe('FAILED_PRECONDITION');
    }
  });
});

test('document.attachment_object: FinalizeUpload rejects when company context drifts', async () => {
  resetRequestContext();

  const businessRequestId = uid('biz_finalize_company_mismatch');
  const prepared = await withScope('cmp_finalize_a', 'usr_finalize_a', async () => {
    const result = await AttachmentObject.PrepareUpload({
      ownerModel: 'auth.User',
      ownerRecordId: uid('owner_finalize_company_mismatch'),
      fieldName: 'Avatar',
      operation: 'update',
      businessRequestId,
    });
    await markSessionUploaded(result.uploadId, 'application/octet-stream', 'cmp_finalize_a');
    return result;
  });

  await withContext(
    {
      activeCompanyId: 'cmp_finalize_b',
      enabledCompanyIds: ['cmp_finalize_a', 'cmp_finalize_b'],
    } as any,
    async () => {
      const jsCtx = ensureRequestContext();
      jsCtx.ctx = {};
      jsCtx.req = {
        depth: 0,
        fieldRuleMode: 'skip',
      };
      jsCtx.identity = {
        userId: 'usr_finalize_a',
      };

      try {
        await AttachmentObject.FinalizeUpload({
          uploadId: prepared.uploadId,
          businessRequestId,
        });
        throw new Error('expected finalize company mismatch error');
      } catch (err) {
        expect(err instanceof ChoysumError).toBe(true);
        const oe = err as ChoysumError;
        expect(oe.domain).toBe('document');
        expect(oe.code).toBe('PERMISSION_DENIED');
        expect(oe.metadata?.stage).toBe('finalize');
      }
    },
    { merge: false }
  );
});

test('document.attachment_object: FinalizeUpload rejects mismatched businessRequestId', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const prepared = await AttachmentObject.PrepareUpload({
      ownerModel: 'auth.User',
      ownerRecordId: uid('owner_finalize_mismatch'),
      fieldName: 'Avatar',
      operation: 'update',
      businessRequestId: uid('biz_finalize_match'),
    });

    await markSessionUploaded(prepared.uploadId, 'application/octet-stream');

    try {
      await AttachmentObject.FinalizeUpload({
        uploadId: prepared.uploadId,
        businessRequestId: uid('biz_finalize_mismatch'),
      });
      throw new Error('expected finalize mismatch error');
    } catch (err) {
      expect(err instanceof ChoysumError).toBe(true);
      const oe = err as ChoysumError;
      expect(oe.domain).toBe('document');
      expect(oe.code).toBe('IDEMPOTENCY_KEY_REUSED');
    }
  });
});

test('document.attachment_object: FinalizeUpload rejects when owner write authorization denies', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const businessRequestId = uid('biz_finalize_owner_denied');
    const created = await UploadSession.Create(
      {
        OwnerModel: 'unknown.Model',
        OwnerRecordId: uid('owner_finalize_owner_denied'),
        FieldName: 'Avatar',
        Operation: 'update',
        IssuerUserId: TEST_USER_ID,
        BusinessRequestId: businessRequestId,
        MaxUploadBytes: 1024,
        RequiredChecksumAlgorithm: 'sha256',
        ExpiresAt: new Date(Date.now() + 5 * 60 * 1000),
        Status: 'uploaded',
        UploadedSizeBytes: 0,
        UploadedChecksumSha256: EMPTY_SHA256,
        UploadedContentType: 'application/octet-stream',
        UploadedPayloadRef: {
          kind: 'stored_content',
          storedContentId: await createStoredContentRecord({ companyId: TEST_COMPANY_ID, backend: 'db', dbPayload: '' }),
        },
        CompanyId: TEST_COMPANY_ID,
      } as any,
      ['Id'] as any
    );

    const uploadId = String((created as any)?.Id || '');
    expect(uploadId).toBeTruthy();

    try {
      await AttachmentObject.FinalizeUpload({
        uploadId,
        businessRequestId,
      });
      throw new Error('expected owner authorization denied on finalize');
    } catch (err) {
      expect(err instanceof ChoysumError).toBe(true);
      const oe = err as ChoysumError;
      expect(oe.domain).toBe('document');
      expect(oe.code).toBe('PERMISSION_DENIED');
      expect(oe.metadata?.stage).toBe('finalize');
    }
  });
});

test('document.attachment_object: RunGarbageCollection deletes unbound db object and clears blob data', async () => {
  resetRequestContext();
  const companyId = uid('cmp_gc_db');
  const userId = uid('usr_gc_db');

  await withScope(companyId, userId, async () => {
    const objectId = await createActiveAttachmentObjectRecord({
      companyId,
      backend: 'db',
      sizeBytes: 5,
      mimeType: 'text/plain',
      checksumSha256: EMPTY_SHA256,
      dbPayload: 'hello',
    });
    expect(objectId).toBeTruthy();

    const gcResult = await AttachmentObject.RunGarbageCollection(futureNowISO());
    expect(Number((gcResult as any)?.objects?.deletedCount || 0)).toBeGreaterThanOrEqual(1);

    const reloaded = await loadAttachmentObjectById(objectId);
    expect(String(reloaded.Status || '')).toBe('deleted');
    expect(reloaded.BlobData === null || reloaded.BlobData === undefined || reloaded.BlobData === '').toBe(true);
    expect(String(reloaded.StoredContentStatus || '')).toBe('deleted');

    const cleanup = cleanupStateOf(reloaded);
    expect(String(cleanup.state || '')).toBe('deleted');
    expect(Number(cleanup.attempts || 0)).toBeGreaterThanOrEqual(1);
  });
});

test('document.attachment_object: RunGarbageCollection deletes unbound s3 stored content through document bridge', async () => {
  resetRequestContext();

  const root: any = (globalThis as any).$choysum ?? {};
  const hasStorage = Object.prototype.hasOwnProperty.call(root, 'document');
  const previousStorage = root.document;
  const deleteCalls: Array<{ storedContentId: string }> = [];

  const baseStorage = previousStorage && typeof previousStorage === 'object' ? previousStorage : {};
  root.document = {
    ...baseStorage,
    deleteStoredContent: async (payload: { storedContentId?: string }) => {
      deleteCalls.push({
        storedContentId: String(payload?.storedContentId || ''),
      });
    },
  };
  (globalThis as any).$choysum = root;

  try {
    const companyId = uid('cmp_gc_s3');
    const userId = uid('usr_gc_s3');

    await withScope(companyId, userId, async () => {
      const bucket = 'choysum-attachments-test';
      const documentKey = `s3/gc/${uid('obj')}`;
      const objectId = await createActiveAttachmentObjectRecord({
        companyId,
        backend: 's3',
        sizeBytes: 0,
        mimeType: 'application/octet-stream',
        checksumSha256: EMPTY_SHA256,
        bucket,
        documentKey,
      });
      expect(objectId).toBeTruthy();

      const gcResult = await AttachmentObject.RunGarbageCollection(futureNowISO());
      expect(Number((gcResult as any)?.objects?.deletedCount || 0)).toBeGreaterThanOrEqual(1);

      const reloaded = await loadAttachmentObjectById(objectId);
      expect(String(reloaded.Status || '')).toBe('deleted');

      const cleanup = cleanupStateOf(reloaded);
      expect(String(cleanup.state || '')).toBe('deleted');
      expect(Number(cleanup.attempts || 0)).toBeGreaterThanOrEqual(1);

      expect(deleteCalls).toHaveLength(1);
      expect(deleteCalls[0]).toEqual({ storedContentId: String(reloaded.StoredContentId || '') });
    });
  } finally {
    if (hasStorage) {
      root.document = previousStorage;
    } else {
      delete root.document;
    }
    (globalThis as any).$choysum = root;
  }
});
