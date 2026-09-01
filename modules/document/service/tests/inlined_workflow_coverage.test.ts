// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { withContext } from '@/core/service/api/context';
import { ChoysumError } from '@/core/service/error';
import RoleRecordRule from '@/auth/service/models/role_record_rule';
import MetaModel from '@/meta/service/models/model';
import { withPermissionGraphBypass } from '@/auth/service/models/user/_authz_shared';
import AttachmentObject from '../models/attachment_object';
import UploadSession from '../models/upload_session';
import StoredContent from '../models/stored_content';
import {
  assertOwnerReadAuthorization,
  assertOwnerWriteAuthorization,
  documentProbeOwnerRecordForTest,
} from '../models/_owner_authorization';
import {
  ensureAuthUserOwnerFieldRuleGrants,
  ensureAuthUserOwnerRecordRuleGrants,
  disableRepositoryFieldRuleForDocumentTests,
  disableRepositoryRecordRuleForDocumentTests,
} from './_owner_auth_test_fixtures';

const RR_CACHE_KEY = Symbol.for('choysum.recordrule.cache');
const FR_CACHE_KEY = Symbol.for('choysum.fieldrule.cache');

const TEST_COMPANY_ID = 'cmp_document_test';
const TEST_USER_ID = 'usr_document_test';
const EMPTY_SHA256 = 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855';

class AttachmentContentUploadProxy extends AttachmentObject {
  public static CreateUploadSession(req: Parameters<typeof AttachmentObject.PrepareUpload>[0]): Promise<string> {
    return (AttachmentObject as any).createUploadSessionInternal(req);
  }

  public static FinalizeUploadInternal(uploadId: string): Promise<Awaited<ReturnType<typeof AttachmentObject.FinalizeUpload>>> {
    return (AttachmentObject as any).finalizeUploadInternal(uploadId);
  }
}

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
  jsCtx.req = { depth: 0, fieldRuleMode: 'skip' };
  jsCtx.identity = { userId: TEST_USER_ID };
  delete (jsCtx as any)[RR_CACHE_KEY];
  delete (jsCtx as any)[FR_CACHE_KEY];
  disableRepositoryRecordRuleForDocumentTests();
  disableRepositoryFieldRuleForDocumentTests();
}

async function withDocumentScope<T>(fn: () => Promise<T>): Promise<T> {
  return withContext(
    { activeCompanyId: TEST_COMPANY_ID, enabledCompanyIds: [TEST_COMPANY_ID] } as any,
    async () => {
      await ensureAuthUserOwnerRecordRuleGrants();
      await ensureAuthUserOwnerFieldRuleGrants();
      return fn();
    },
    { merge: false }
  );
}

function buildPrincipal(userId = TEST_USER_ID, companyId = TEST_COMPANY_ID) {
  return { userId, activeCompanyId: companyId, enabledCompanyIds: [companyId] };
}

async function createStoredContent(companyId: string, status: 'active' | 'deleted' = 'active'): Promise<string> {
  const row = await StoredContent.Create(
    { Provider: 'db', Status: status, CompanyId: companyId, BlobData: 'x' } as any,
    ['Id'] as any
  );
  return String((row as any).Id);
}

async function prepareUploadedSession(ownerRecordId: string, fieldName: string): Promise<{ uploadId: string; businessRequestId: string }> {
  const businessRequestId = uid('biz_cov');
  const prepared = await AttachmentObject.PrepareUpload({
    ownerModel: 'auth.User',
    ownerRecordId,
    fieldName,
    operation: 'update',
    businessRequestId,
    proposedContentType: 'text/plain',
  });
  const storedContentId = await createStoredContent(TEST_COMPANY_ID);
  await UploadSession.UpdateById(
    prepared.uploadId,
    {
      Status: 'uploaded',
      UploadedSizeBytes: 1,
      UploadedChecksumSha256: EMPTY_SHA256,
      UploadedContentType: 'text/plain',
      UploadedPayloadRef: { kind: 'stored_content', storedContentId },
    } as any,
    ['Id'] as any
  );
  return { uploadId: prepared.uploadId, businessRequestId };
}

async function markUploaded(uploadId: string, companyId = TEST_COMPANY_ID): Promise<void> {
  const storedContentId = await createStoredContent(companyId);
  await UploadSession.UpdateById(
    uploadId,
    {
      Status: 'uploaded',
      UploadedSizeBytes: 1,
      UploadedChecksumSha256: EMPTY_SHA256,
      UploadedContentType: 'text/plain',
      UploadedPayloadRef: { kind: 'stored_content', storedContentId },
    } as any,
    ['Id'] as any
  );
}

test('inlined upload coverage: protected seams and finalize reuse partial content', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const ownerRecordId = uid('owner_cov_proxy');
    const uploadId = await AttachmentContentUploadProxy.CreateUploadSession({
      ownerModel: 'auth.User',
      ownerRecordId,
      fieldName: 'Avatar',
      operation: 'update',
      businessRequestId: uid('biz_cov_proxy'),
    });
    expect(uploadId).toBeTruthy();

    await markUploaded(uploadId);
    const first = await AttachmentContentUploadProxy.FinalizeUploadInternal(uploadId);
    const replay = await AttachmentContentUploadProxy.FinalizeUploadInternal(uploadId);
    expect(replay.attachmentObjectId).toBe(first.attachmentObjectId);
  });
});

test('inlined upload coverage: finalize rejects stored content company and inactive status', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const ownerRecordId = uid('owner_cov_finalize');
    const bizA = uid('biz_cov_finalize_a');
    const prepared = await AttachmentObject.PrepareUpload({
      ownerModel: 'auth.User',
      ownerRecordId,
      fieldName: 'Avatar',
      operation: 'update',
      businessRequestId: bizA,
    });

    const wrongCompanyStored = await withContext(
      { activeCompanyId: TEST_COMPANY_ID, enabledCompanyIds: [TEST_COMPANY_ID, 'cmp_cov_other'] } as any,
      () => createStoredContent('cmp_cov_other')
    );
    await UploadSession.UpdateById(
      prepared.uploadId,
      {
        Status: 'uploaded',
        UploadedPayloadRef: { kind: 'stored_content', storedContentId: wrongCompanyStored },
        UploadedSizeBytes: 1,
        UploadedChecksumSha256: EMPTY_SHA256,
        UploadedContentType: 'text/plain',
      } as any,
      ['Id'] as any
    );
    try {
      await AttachmentObject.FinalizeUpload({ uploadId: prepared.uploadId, businessRequestId: bizA });
      throw new Error('expected company mismatch');
    } catch (err) {
      expect(['PERMISSION_DENIED', 'NOT_FOUND']).toContain((err as ChoysumError).code);
    }

    const bizB = uid('biz_cov_finalize_b');
    const prepared2 = await AttachmentObject.PrepareUpload({
      ownerModel: 'auth.User',
      ownerRecordId,
      fieldName: 'Avatar',
      operation: 'update',
      businessRequestId: bizB,
    });
    const inactiveStored = await createStoredContent(TEST_COMPANY_ID, 'deleted');
    await UploadSession.UpdateById(
      prepared2.uploadId,
      {
        Status: 'uploaded',
        UploadedPayloadRef: { kind: 'stored_content', storedContentId: inactiveStored },
        UploadedSizeBytes: 1,
        UploadedChecksumSha256: EMPTY_SHA256,
        UploadedContentType: 'text/plain',
      } as any,
      ['Id'] as any
    );
    try {
      await AttachmentObject.FinalizeUpload({ uploadId: prepared2.uploadId, businessRequestId: bizB });
      throw new Error('expected inactive stored content');
    } catch (err) {
      expect((err as ChoysumError).code).toBe('FAILED_PRECONDITION');
    }
  });
});

test('inlined upload coverage: authorize and commit guard rails', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const ownerRecordId = uid('owner_cov_auth');
    const uploaded = await prepareUploadedSession(ownerRecordId, 'Avatar');
    await AttachmentObject.FinalizeUpload({ uploadId: uploaded.uploadId, businessRequestId: uploaded.businessRequestId });

    try {
      await AttachmentObject.AuthorizeUploadPut({ uploadId: uploaded.uploadId, principal: buildPrincipal() });
      throw new Error('expected finalized authorize reject');
    } catch (err) {
      expect((err as ChoysumError).code).toBe('UPLOAD_SESSION_FINALIZED');
    }

    const expiredPrepared = await AttachmentObject.PrepareUpload({
      ownerModel: 'auth.User',
      ownerRecordId,
      fieldName: 'Avatar',
      operation: 'update',
      businessRequestId: uid('biz_cov_expired'),
    });
    await UploadSession.UpdateById(
      expiredPrepared.uploadId,
      { ExpiresAt: new Date(Date.now() - 60_000), Status: 'prepared' } as any,
      ['Id'] as any
    );
    try {
      await AttachmentObject.AuthorizeUploadPut({ uploadId: expiredPrepared.uploadId, principal: buildPrincipal() });
      throw new Error('expected expired authorize reject');
    } catch (err) {
      expect((err as ChoysumError).code).toBe('UPLOAD_SESSION_EXPIRED');
    }

    const commitBiz = uid('biz_cov_commit_ready');
    const commitReady = await AttachmentObject.PrepareUpload({
      ownerModel: 'auth.User',
      ownerRecordId,
      fieldName: 'Doc',
      operation: 'update',
      businessRequestId: commitBiz,
      proposedContentType: 'text/plain',
    });
    await markUploaded(commitReady.uploadId);
    await AttachmentObject.FinalizeUpload({ uploadId: commitReady.uploadId, businessRequestId: commitBiz });

    try {
      await AttachmentObject.CommitUploadPut({
        uploadId: commitReady.uploadId,
        principal: buildPrincipal(),
        payloadReceipt: { payloadId: 'stored_content:abc', sizeBytes: 1, checksumSha256: EMPTY_SHA256, contentType: 'text/plain' },
      });
      throw new Error('expected commit on finalized reject');
    } catch (err) {
      expect((err as ChoysumError).code).toBe('UPLOAD_SESSION_FINALIZED');
    }

    const badStatusBiz = uid('biz_cov_bad_status');
    const badStatus = await AttachmentObject.PrepareUpload({
      ownerModel: 'auth.User',
      ownerRecordId,
      fieldName: 'Doc2',
      operation: 'update',
      businessRequestId: badStatusBiz,
    });
    await UploadSession.UpdateById(badStatus.uploadId, { Status: 'finalized' } as any, ['Id'] as any);
    try {
      await AttachmentObject.CommitUploadPut({
        uploadId: badStatus.uploadId,
        principal: buildPrincipal(),
        payloadReceipt: { payloadId: 'stored_content:abc', sizeBytes: 1, checksumSha256: EMPTY_SHA256, contentType: 'text/plain' },
      });
      throw new Error('expected commit bad status');
    } catch (err) {
      expect((err as ChoysumError).code).toBe('UPLOAD_SESSION_FINALIZED');
    }

    const limitsBiz = uid('biz_cov_limits');
    const limitsPrepared = await AttachmentObject.PrepareUpload({
      ownerModel: 'auth.User',
      ownerRecordId,
      fieldName: 'Doc3',
      operation: 'update',
      businessRequestId: limitsBiz,
      proposedContentType: 'text/plain',
    });
    await UploadSession.UpdateById(limitsPrepared.uploadId, { MaxUploadBytes: 2, AllowedMimeTypes: ['image/png'] } as any, ['Id'] as any);

    try {
      await AttachmentObject.CommitUploadPut({
        uploadId: limitsPrepared.uploadId,
        principal: buildPrincipal(),
        payloadReceipt: { payloadId: 'stored_content:abc', sizeBytes: 99, checksumSha256: EMPTY_SHA256, contentType: 'text/plain' },
      });
      throw new Error('expected commit size');
    } catch (err) {
      expect((err as ChoysumError).code).toBe('UPLOAD_TOO_LARGE');
    }

    try {
      await AttachmentObject.CommitUploadPut({
        uploadId: limitsPrepared.uploadId,
        principal: buildPrincipal(),
        payloadReceipt: { payloadId: 'stored_content:abc', sizeBytes: 1, checksumSha256: EMPTY_SHA256, contentType: 'text/plain' },
      });
      throw new Error('expected commit mime');
    } catch (err) {
      expect((err as ChoysumError).code).toBe('MIME_TYPE_NOT_ALLOWED');
    }

    await UploadSession.UpdateById(
      limitsPrepared.uploadId,
      { ChecksumSha256: 'b'.repeat(64), AllowedMimeTypes: ['text/plain'] } as any,
      ['Id'] as any
    );
    try {
      await AttachmentObject.CommitUploadPut({
        uploadId: limitsPrepared.uploadId,
        principal: buildPrincipal(),
        payloadReceipt: { payloadId: 'stored_content:abc', sizeBytes: 1, checksumSha256: EMPTY_SHA256, contentType: 'text/plain' },
      });
      throw new Error('expected commit checksum');
    } catch (err) {
      expect((err as ChoysumError).code).toBe('CHECKSUM_MISMATCH');
    }
  });
});

test('inlined upload coverage: finalize expires stale uploaded session', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const ownerRecordId = uid('owner_cov_expire_finalize');
    const expireBiz = uid('biz_cov_expire_finalize');
    const prepared = await AttachmentObject.PrepareUpload({
      ownerModel: 'auth.User',
      ownerRecordId,
      fieldName: 'Avatar',
      operation: 'update',
      businessRequestId: expireBiz,
    });
    const storedContentId = await createStoredContent(TEST_COMPANY_ID);
    await UploadSession.UpdateById(
      prepared.uploadId,
      {
        Status: 'uploaded',
        ExpiresAt: new Date(Date.now() - 60_000),
        UploadedPayloadRef: { kind: 'stored_content', storedContentId },
        UploadedSizeBytes: 1,
        UploadedChecksumSha256: EMPTY_SHA256,
        UploadedContentType: 'text/plain',
      } as any,
      ['Id'] as any
    );
    try {
      await AttachmentObject.FinalizeUpload({ uploadId: prepared.uploadId, businessRequestId: expireBiz });
      throw new Error('expected expired finalize');
    } catch (err) {
      expect((err as ChoysumError).code).toBe('UPLOAD_SESSION_EXPIRED');
    }
  });
});

test('inlined owner auth coverage: probe and expr scope denials', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    expect(await documentProbeOwnerRecordForTest('bind', 'auth.User', TEST_USER_ID)).toBe(true);
    expect(await documentProbeOwnerRecordForTest('bind', 'auth.User', uid('missing_owner'))).toBe(false);
    expect(
      await documentProbeOwnerRecordForTest('bind', 'auth.User', TEST_USER_ID, ['Id', '=', uid('expr_miss')] as any)
    ).toBe(false);

    let grantRuleId = '';
    let originalCondition: unknown = { And: [] };
    await withPermissionGraphBypass(async () => {
      const modelRows = await MetaModel.Search({ And: [['Application', '=', 'auth'], ['Name', '=', 'User']] } as any, {
        fields: ['Id'],
        limit: 1,
      } as any);
      const modelId = String((modelRows[0] as any)?.Id || '').trim();
      const existing = await RoleRecordRule.Search(
        {
          And: [
            ['RoleId', 'is', null],
            ['Kind', '=', 'grant'],
            ['MetaModelId', '=', modelId],
            ['MetaApplicationId', 'is', null],
            ['PermRead', '=', true],
            ['PermWrite', '=', true],
          ],
        } as any,
        { fields: ['Id', 'Condition'], limit: 8 } as any
      );
      const grant = (existing || [])[0] as any;
      grantRuleId = String(grant?.Id || '').trim();
      originalCondition = grant?.Condition ?? { And: [] };
      await RoleRecordRule.UpdateById(
        grantRuleId,
        { Condition: ['Id', '=', uid('rr_expr_block')] as any } as any,
        ['Id'] as any
      );
    });
    delete (ensureRequestContext() as any)[RR_CACHE_KEY];

    try {
      await assertOwnerWriteAuthorization({
        stage: 'bind',
        ownerModel: 'auth.User',
        ownerRecordId: TEST_USER_ID,
        fieldName: 'Avatar',
        operation: 'update',
        companyId: TEST_COMPANY_ID,
        companyIds: [TEST_COMPANY_ID],
        userId: TEST_USER_ID,
      });
      throw new Error('expected expr write scope deny');
    } catch (err) {
      expect((err as ChoysumError).code).toBe('PERMISSION_DENIED');
    }

    try {
      await assertOwnerReadAuthorization({
        stage: 'descriptor',
        ownerModel: 'auth.User',
        ownerRecordId: TEST_USER_ID,
        fieldName: 'Avatar',
        companyId: TEST_COMPANY_ID,
        companyIds: [TEST_COMPANY_ID],
        userId: TEST_USER_ID,
      });
      throw new Error('expected expr read scope deny');
    } catch (err) {
      expect((err as ChoysumError).code).toBe('PERMISSION_DENIED');
    }

    if (grantRuleId) {
      await withPermissionGraphBypass(async () => {
        await RoleRecordRule.UpdateById(grantRuleId, { Condition: originalCondition as any } as any, ['Id'] as any);
      });
      delete (ensureRequestContext() as any)[RR_CACHE_KEY];
    }
  });
});
