// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel } from '@/core/service';
import AttachmentOwnerMixin from '../mixins/attachment_owner_model';
import AttachmentBinding from '../models/attachment_binding';
import AttachmentObject from '../models/attachment_object';
import StoredContent from '../models/stored_content';
import { withContext } from '@/core/service/api/context';
import {
  ensureAuthUserOwnerFieldRuleGrants,
  ensureAuthUserOwnerRecordRuleGrants,
  disableRepositoryFieldRuleForDocumentTests,
  disableRepositoryRecordRuleForDocumentTests,
} from './_owner_auth_test_fixtures';

const TEST_COMPANY_ID = 'cmp_document_test';
const TEST_USER_ID = 'usr_document_test';

function uid(prefix: string): string {
  const xid = (globalThis as any).$choysum?.xid?.New?.();
  return `${prefix}_${typeof xid === 'string' && xid.trim() ? xid.trim() : Date.now()}`;
}

function resetRequestContext(): void {
  const root: any = (globalThis as any).$choysum ?? {};
  if (!root.request) root.request = {};
  if (!root.request.context) root.request.context = {};
  const jsCtx = root.request.context;
  jsCtx.ctx = {};
  jsCtx.req = { depth: 0, fieldRuleMode: 'skip' };
  jsCtx.identity = { userId: TEST_USER_ID };
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

/**
 * Harness consumer for AttachmentOwnerMixin extend contract (not a persisted domain model).
 */
class AttachmentOwnerHarness extends AttachmentOwnerMixin {}

test('AttachmentOwnerMixin: AttachmentBinding and harness extend the mixin', () => {
  expect(Object.prototype.isPrototypeOf.call(AttachmentOwnerMixin, AttachmentBinding)).toBe(false);
  expect(Object.prototype.isPrototypeOf.call(BaseModel, AttachmentBinding)).toBe(true);
  expect(Object.prototype.isPrototypeOf.call(AttachmentOwnerMixin, AttachmentOwnerHarness)).toBe(true);
});

test('AttachmentOwnerMixin: harness exposes bind/unbind entry points', () => {
  expect(typeof AttachmentOwnerHarness.AttachmentBind).toBe('function');
  expect(typeof AttachmentOwnerHarness.AttachmentUnbind).toBe('function');
});

test('AttachmentOwnerMixin: delegates bind and unbind to AttachmentBinding', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const ownerRecordId = uid('owner_mixin_delegate');
    const stored = await StoredContent.Create(
      { Provider: 'db', Status: 'active', CompanyId: TEST_COMPANY_ID, BlobData: 'm' } as any,
      ['Id'] as any
    );
    const content = await AttachmentObject.Create(
      {
        StoredContentId: String((stored as any).Id),
        SizeBytes: 1,
        MimeType: 'text/plain',
        ChecksumSha256: 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
        Status: 'active',
        CompanyId: TEST_COMPANY_ID,
      } as any,
      ['Id'] as any
    );

    const bound = await AttachmentOwnerHarness.AttachmentBind({
      attachmentObjectId: String((content as any).Id),
      ownerModel: 'auth.User',
      ownerRecordId,
      fieldName: 'Avatar',
      mutationId: uid('mut_mixin_bind'),
    });
    const unbound = await AttachmentOwnerHarness.AttachmentUnbind({
      attachmentBindingId: bound.attachmentBindingId,
      mutationId: uid('mut_mixin_unbind'),
      reason: 'test',
    });
    expect(unbound.status).toBe('unbound');
  });
});
