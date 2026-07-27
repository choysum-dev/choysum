// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { withContext } from '@/core/service/api/context';
import { ChoysumError } from '@/core/service/error';
import Role from '@/auth/service/models/role';
import RoleFieldRule from '@/auth/service/models/role_field_rule';
import User from '@/auth/service/models/user';
import UserRole from '@/auth/service/models/user_role';
import IrField from '@/meta/service/models/ir_field';
import IrModel from '@/meta/service/models/ir_model';
import AttachmentBinding from '../models/attachment_binding';
import AttachmentObject from '../models/attachment_object';
import UploadSession from '../models/upload_session';
import StoredContent from '../models/stored_content';
import { normalizeBatchDescribeReq } from '../models/_attachment_binding_codec';
import {
  ensureAuthUserOwnerRecordRuleGrants,
  ensureAuthUserOwnerFieldRuleGrants,
  disableRepositoryRecordRuleForDocumentTests,
  disableRepositoryFieldRuleForDocumentTests,
} from './_owner_auth_test_fixtures';

const RR_CACHE_KEY = Symbol.for('choysum.recordrule.cache');
const FR_CACHE_KEY = Symbol.for('choysum.fieldrule.cache');

const TEST_COMPANY_ID = 'cmp_document_test';
const TEST_USER_ID = 'usr_document_test';
const EMPTY_SHA256 = 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855';

class AttachmentBindingDescriptorTestProxy extends AttachmentBinding {
  public static async BuildDescriptor(bindingId: string) {
    return (AttachmentBinding as any).buildDescriptorInternal(bindingId);
  }
}

function uid(prefix: string): string {
  const xid = (globalThis as any).$choysum?.xid?.New?.();
  const token = typeof xid === 'string' && xid.trim() ? xid.trim() : `${Date.now()}${Math.random()}`;
  return `${prefix}_${token}`;
}

function shortUid(prefix: string, maxLen = 40): string {
  const value = uid(prefix);
  if (value.length <= maxLen) return value;
  return value.slice(0, maxLen);
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
  disableRepositoryFieldRuleForDocumentTests();
}

async function withDocumentScope<T>(fn: () => Promise<T>): Promise<T> {
  return withContext(
    {
      activeCompanyId: TEST_COMPANY_ID,
      enabledCompanyIds: [TEST_COMPANY_ID],
    } as any,
    async () => {
      await ensureAuthUserOwnerRecordRuleGrants();
      await ensureAuthUserOwnerFieldRuleGrants();
      return fn();
    },
    { merge: false }
  );
}

async function withScope<T>(companyId: string, enabledCompanyIds: string[], userId: string, fn: () => Promise<T>): Promise<T> {
  return withContext(
    {
      activeCompanyId: companyId,
      enabledCompanyIds,
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
      await ensureAuthUserOwnerFieldRuleGrants();
      return fn();
    },
    { merge: false }
  );
}

async function markSessionUploaded(uploadId: string, companyId = TEST_COMPANY_ID): Promise<void> {
  const storedContent = await StoredContent.Create(
    {
      Provider: 'db',
      Status: 'active',
      CompanyId: companyId,
    } as any,
    ['Id'] as any
  );
  const storedContentId = String((storedContent as any)?.Id || '').trim();
  if (!storedContentId) {
    throw new Error('failed to create stored content fixture');
  }

  await UploadSession.UpdateById(
    uploadId,
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
}

async function createActiveContentRecord(input: { backend: 'db' | 's3'; mimeType: string; sizeBytes: number; checksumSha256?: string }): Promise<string> {
  const checksumSha256 = input.checksumSha256 ?? EMPTY_SHA256;

  const storedContentPayload: Record<string, unknown> = {
    Provider: input.backend,
    Status: 'active',
    CompanyId: TEST_COMPANY_ID,
  };
  if (input.backend === 's3') {
    storedContentPayload.LocatorJson = {
      bucket: 'choysum-attachments-test',
      key: `s3/test/${uid('obj')}`,
    };
  }

  const createdStored = await StoredContent.Create(storedContentPayload as any, ['Id'] as any);
  const storedContentId = String((createdStored as any)?.Id || '').trim();
  if (!storedContentId) {
    throw new Error('failed to create stored content fixture');
  }

  const created = await AttachmentObject.Create(
    {
      StoredContentId: storedContentId,
      SizeBytes: input.sizeBytes,
      MimeType: input.mimeType,
      ChecksumSha256: checksumSha256,
      Status: 'active',
      CompanyId: TEST_COMPANY_ID,
    } as any,
    ['Id'] as any
  );
  const attachmentContentId = String((created as any)?.Id || '').trim();
  if (!attachmentContentId) {
    throw new Error('failed to create attachment content fixture');
  }
  return attachmentContentId;
}

async function createActiveObject(ownerRecordId: string, fieldName: string, companyId = TEST_COMPANY_ID): Promise<string> {
  const businessRequestId = uid('biz_for_binding');
  const prepared = await AttachmentObject.PrepareUpload({
    ownerModel: 'auth.User',
    ownerRecordId,
    fieldName,
    operation: 'update',
    businessRequestId,
    proposedContentType: 'text/plain',
    proposedSizeBytes: 0,
  });

  await markSessionUploaded(prepared.uploadId, companyId);

  const finalized = await AttachmentObject.FinalizeUpload({
    uploadId: prepared.uploadId,
    businessRequestId,
  });
  return finalized.attachmentObjectId;
}

async function createActiveObjectForBackend(ownerRecordId: string, fieldName: string, backend: 'db' | 's3'): Promise<string> {
  if (backend === 'db') {
    return createActiveObject(ownerRecordId, fieldName);
  }
  return createActiveContentRecord({
    backend: 's3',
    mimeType: 'application/octet-stream',
    sizeBytes: 0,
  });
}

async function mustFindAdminUserId(): Promise<string> {
  const rows = await User.Search(['Username', '=', 'admin'] as any, { fields: ['Id'], limit: 1 } as any);
  const userId = String((rows[0] as any)?.Id || '').trim();
  if (!userId) {
    throw new Error('admin user not found for descriptor field-rule deny fixture');
  }
  return userId;
}

async function mustFindAuthUserFieldScope(fieldName: string): Promise<{ modelId: string; fieldId: string }> {
  const modelRows = await IrModel.Search(
    {
      And: [
        ['Application', '=', 'auth'],
        ['Name', '=', 'User'],
      ],
    } as any,
    { fields: ['Id'], limit: 1 } as any
  );
  const modelId = String((modelRows[0] as any)?.Id || '').trim();
  if (!modelId) {
    throw new Error('meta model auth.User not found for descriptor field-rule deny fixture');
  }

  const fieldRows = await IrField.Search(
    {
      And: [
        ['ModelId', '=', modelId],
        ['Name', '=', fieldName],
      ],
    } as any,
    { fields: ['Id'], limit: 1 } as any
  );
  const fieldId = String((fieldRows[0] as any)?.Id || '').trim();
  if (!fieldId) {
    throw new Error(`meta field auth.User.${fieldName} not found for descriptor field-rule deny fixture`);
  }

  return { modelId, fieldId };
}

test('document.attachment_binding: Bind replays first success snapshot by mutationId', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const ownerRecordId = uid('owner_bind');
    const fieldName = 'AttachmentField';
    const attachmentObjectId = await createActiveObject(ownerRecordId, fieldName);
    const mutationId = uid('mutation_bind');

    const first = await AttachmentBinding.Bind({
      attachmentObjectId,
      ownerModel: 'auth.User',
      ownerRecordId,
      fieldName,
      displayFileName: 'contract.txt',
      downloadDisposition: 'attachment',
      mutationId,
    });

    const replay = await AttachmentBinding.Bind({
      attachmentObjectId,
      ownerModel: 'auth.User',
      ownerRecordId,
      fieldName,
      displayFileName: 'contract.txt',
      downloadDisposition: 'attachment',
      mutationId,
    });

    expect(first.status).toBe('active');
    expect(replay.attachmentBindingId).toBe(first.attachmentBindingId);
    expect(replay.descriptor.id).toBe(first.descriptor.id);
    expect(replay.descriptor.downloadUrl).toBe(`/_document/bindings/${first.attachmentBindingId}/content`);
  });
});

test('document.attachment_binding: Bind rejects unauthenticated and does not create binding', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const ownerRecordId = uid('owner_bind_unauth');
    const fieldName = 'AttachmentField';
    const attachmentObjectId = await createActiveObject(ownerRecordId, fieldName);

    const jsCtx = ensureRequestContext();
    jsCtx.identity = {};

    try {
      await AttachmentBinding.Bind({
        attachmentObjectId,
        ownerModel: 'auth.User',
        ownerRecordId,
        fieldName,
        mutationId: uid('mutation_bind_unauth'),
      });
      throw new Error('expected bind unauthenticated error');
    } catch (err) {
      expect(err instanceof ChoysumError).toBe(true);
      const oe = err as ChoysumError;
      expect(oe.domain).toBe('document');
      expect(oe.code).toBe('UNAUTHENTICATED');
    }

    const rows = await AttachmentBinding.Search(
      {
        And: [
          ['OwnerModel', '=', 'auth.User'],
          ['OwnerRecordId', '=', ownerRecordId],
          ['FieldName', '=', fieldName],
          ['Status', '=', 'active'],
        ],
      } as any,
      { limit: 1 } as any
    );
    expect(rows.length).toBe(0);
  });
});

test('document.attachment_binding: Bind rejects when owner write authorization denies', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const ownerRecordId = uid('owner_bind_owner_denied');
    const fieldName = 'AttachmentField';
    const attachmentObjectId = await createActiveObject(ownerRecordId, fieldName);

    try {
      await AttachmentBinding.Bind({
        attachmentObjectId,
        ownerModel: 'unknown.Model',
        ownerRecordId,
        fieldName,
        mutationId: uid('mutation_bind_owner_denied'),
      });
      throw new Error('expected owner authorization denied on bind');
    } catch (err) {
      expect(err instanceof ChoysumError).toBe(true);
      const oe = err as ChoysumError;
      expect(oe.domain).toBe('document');
      expect(oe.code).toBe('PERMISSION_DENIED');
      expect(oe.metadata?.stage).toBe('bind');
    }
  });
});

test('document.attachment_binding: Unbind replays first success snapshot by mutationId', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const ownerRecordId = uid('owner_unbind');
    const fieldName = 'AttachmentField';
    const attachmentObjectId = await createActiveObject(ownerRecordId, fieldName);

    const bound = await AttachmentBinding.Bind({
      attachmentObjectId,
      ownerModel: 'auth.User',
      ownerRecordId,
      fieldName,
      mutationId: uid('mutation_bind_for_unbind'),
    });

    const mutationId = uid('mutation_unbind');
    const first = await AttachmentBinding.Unbind({
      attachmentBindingId: bound.attachmentBindingId,
      mutationId,
      reason: 'clear',
    });

    const replay = await AttachmentBinding.Unbind({
      attachmentBindingId: bound.attachmentBindingId,
      mutationId,
      reason: 'clear',
    });

    expect(first.status).toBe('unbound');
    expect(replay.attachmentBindingId).toBe(first.attachmentBindingId);
    expect(replay.gcEligibleAfter).toBe(first.gcEligibleAfter);
  });
});

test('document.attachment_binding: Unbind handles stale unbound row for same owner field', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const ownerRecordId = uid('owner_unbind_conflict_cleanup');
    const fieldName = 'AttachmentField';

    const firstObjectId = await createActiveObject(ownerRecordId, fieldName);
    const firstBind = await AttachmentBinding.Bind({
      attachmentObjectId: firstObjectId,
      ownerModel: 'auth.User',
      ownerRecordId,
      fieldName,
      mutationId: uid('mutation_bind_conflict_cleanup_1'),
    });
    await AttachmentBinding.Unbind({
      attachmentBindingId: firstBind.attachmentBindingId,
      mutationId: uid('mutation_unbind_conflict_cleanup_1'),
      reason: 'clear',
    });

    const secondObjectId = await createActiveObject(ownerRecordId, fieldName);
    const secondBind = await AttachmentBinding.Bind({
      attachmentObjectId: secondObjectId,
      ownerModel: 'auth.User',
      ownerRecordId,
      fieldName,
      mutationId: uid('mutation_bind_conflict_cleanup_2'),
    });

    const secondUnbind = await AttachmentBinding.Unbind({
      attachmentBindingId: secondBind.attachmentBindingId,
      mutationId: uid('mutation_unbind_conflict_cleanup_2'),
      reason: 'clear',
    });

    expect(secondUnbind.status).toBe('unbound');

    const unboundRows = await AttachmentBinding.Search(
      {
        And: [
          ['OwnerModel', '=', 'auth.User'],
          ['OwnerRecordId', '=', ownerRecordId],
          ['FieldName', '=', fieldName],
          ['Status', '=', 'unbound'],
        ],
      } as any,
      { limit: 10 } as any
    );
    expect(unboundRows.length).toBe(1);
    expect(String((unboundRows[0] as any)?.Id || '')).toBe(secondBind.attachmentBindingId);
  });
});

test('document.attachment_binding: Bind handles stale unbound row when replacing active binding', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const ownerRecordId = uid('owner_bind_conflict_cleanup');
    const fieldName = 'AttachmentField';

    const firstObjectId = await createActiveObject(ownerRecordId, fieldName);
    const firstBind = await AttachmentBinding.Bind({
      attachmentObjectId: firstObjectId,
      ownerModel: 'auth.User',
      ownerRecordId,
      fieldName,
      mutationId: uid('mutation_bind_replace_conflict_1'),
    });
    await AttachmentBinding.Unbind({
      attachmentBindingId: firstBind.attachmentBindingId,
      mutationId: uid('mutation_unbind_replace_conflict_1'),
      reason: 'clear',
    });

    const secondObjectId = await createActiveObject(ownerRecordId, fieldName);
    const secondBind = await AttachmentBinding.Bind({
      attachmentObjectId: secondObjectId,
      ownerModel: 'auth.User',
      ownerRecordId,
      fieldName,
      mutationId: uid('mutation_bind_replace_conflict_2'),
    });

    const thirdObjectId = await createActiveObject(ownerRecordId, fieldName);
    const thirdBind = await AttachmentBinding.Bind({
      attachmentObjectId: thirdObjectId,
      ownerModel: 'auth.User',
      ownerRecordId,
      fieldName,
      mutationId: uid('mutation_bind_replace_conflict_3'),
    });

    expect(thirdBind.status).toBe('active');
    expect(thirdBind.attachmentBindingId).not.toBe(secondBind.attachmentBindingId);

    const activeRows = await AttachmentBinding.Search(
      {
        And: [
          ['OwnerModel', '=', 'auth.User'],
          ['OwnerRecordId', '=', ownerRecordId],
          ['FieldName', '=', fieldName],
          ['Status', '=', 'active'],
        ],
      } as any,
      { limit: 10 } as any
    );
    expect(activeRows.length).toBe(1);
    expect(String((activeRows[0] as any)?.Id || '')).toBe(thirdBind.attachmentBindingId);

    const unboundRows = await AttachmentBinding.Search(
      {
        And: [
          ['OwnerModel', '=', 'auth.User'],
          ['OwnerRecordId', '=', ownerRecordId],
          ['FieldName', '=', fieldName],
          ['Status', '=', 'unbound'],
        ],
      } as any,
      { limit: 10 } as any
    );
    expect(unboundRows.length).toBe(1);
    expect(String((unboundRows[0] as any)?.Id || '')).toBe(secondBind.attachmentBindingId);
  });
});

test('document.attachment_binding: Unbind rejects when owner write authorization denies', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const ownerRecordId = uid('owner_unbind_owner_denied');
    const fieldName = 'AttachmentField';
    const attachmentObjectId = await createActiveObject(ownerRecordId, fieldName);

    const created = await AttachmentBinding.Create(
      {
        OwnerModel: 'unknown.Model',
        OwnerRecordId: ownerRecordId,
        FieldName: fieldName,
        AttachmentContentId: attachmentObjectId,
        DownloadDisposition: 'attachment',
        Status: 'active',
        CompanyId: TEST_COMPANY_ID,
      } as any,
      ['Id', 'Status'] as any
    );

    const attachmentBindingId = String((created as any)?.Id || '');
    expect(attachmentBindingId).toBeTruthy();

    try {
      await AttachmentBinding.Unbind({
        attachmentBindingId,
        mutationId: uid('mutation_unbind_owner_denied'),
      });
      throw new Error('expected owner authorization denied on unbind');
    } catch (err) {
      expect(err instanceof ChoysumError).toBe(true);
      const oe = err as ChoysumError;
      expect(oe.domain).toBe('document');
      expect(oe.code).toBe('PERMISSION_DENIED');
      expect(oe.metadata?.stage).toBe('unbind');
    }

    const reloaded = await AttachmentBinding.Search(['Id', '=', attachmentBindingId] as any, { limit: 1 } as any);
    expect(reloaded.length).toBe(1);
    expect(String((reloaded[0] as any)?.Status || '')).toBe('active');
  });
});

test('document.attachment_binding: descriptor read interface allows same-company authenticated caller', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    for (const backend of ['db', 's3'] as const) {
      const ownerRecordId = uid(`owner_descriptor_allow_${backend}`);
      const fieldName = 'AttachmentField';
      const attachmentObjectId = await createActiveObjectForBackend(ownerRecordId, fieldName, backend);

      const bound = await AttachmentBinding.Bind({
        attachmentObjectId,
        ownerModel: 'auth.User',
        ownerRecordId,
        fieldName,
        displayFileName: 'descriptor.txt',
        mutationId: uid(`mutation_bind_descriptor_allow_${backend}`),
      });

      const descriptor = await AttachmentBindingDescriptorTestProxy.BuildDescriptor(bound.attachmentBindingId);
      expect(descriptor.id).toBe(bound.attachmentBindingId);
      expect(descriptor.fileName).toBe('descriptor.txt');
      expect(descriptor.downloadUrl).toBe(`/_document/bindings/${bound.attachmentBindingId}/content`);
    }
  });
});

test('document.attachment_binding: descriptor read interface denies unauthenticated caller', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const ownerRecordId = uid('owner_descriptor_unauth');
    const fieldName = 'AttachmentField';
    const attachmentObjectId = await createActiveObject(ownerRecordId, fieldName);

    const bound = await AttachmentBinding.Bind({
      attachmentObjectId,
      ownerModel: 'auth.User',
      ownerRecordId,
      fieldName,
      mutationId: uid('mutation_bind_descriptor_unauth'),
    });

    const jsCtx = ensureRequestContext();
    jsCtx.identity = {};

    try {
      await AttachmentBindingDescriptorTestProxy.BuildDescriptor(bound.attachmentBindingId);
      throw new Error('expected unauthenticated descriptor error');
    } catch (err) {
      expect(err instanceof ChoysumError).toBe(true);
      const oe = err as ChoysumError;
      expect(oe.domain).toBe('document');
      expect(oe.code).toBe('UNAUTHENTICATED');
    }
  });
});

test('document.attachment_binding: descriptor read interface denies activeCompany mismatch (download parity)', async () => {
  resetRequestContext();

  const ownerRecordId = uid('owner_descriptor_company_mismatch');
  const fieldName = 'AttachmentField';
  const attachmentObjectId = await withScope('cmp_desc_a', ['cmp_desc_a'], 'usr_desc_a', async () => {
    return createActiveObject(ownerRecordId, fieldName, 'cmp_desc_a');
  });

  const bound = await withScope('cmp_desc_a', ['cmp_desc_a'], 'usr_desc_a', async () => {
    return AttachmentBinding.Bind({
      attachmentObjectId,
      ownerModel: 'auth.User',
      ownerRecordId,
      fieldName,
      mutationId: uid('mutation_bind_descriptor_company_mismatch'),
    });
  });

  await withScope('cmp_desc_b', ['cmp_desc_a', 'cmp_desc_b'], 'usr_desc_a', async () => {
    try {
      await AttachmentBindingDescriptorTestProxy.BuildDescriptor(bound.attachmentBindingId);
      throw new Error('expected company mismatch descriptor error');
    } catch (err) {
      expect(err instanceof ChoysumError).toBe(true);
      const oe = err as ChoysumError;
      expect(oe.domain).toBe('document');
      expect(oe.code).toBe('PERMISSION_DENIED');
      expect(oe.metadata?.stage).toBe('descriptor');
    }
  });
});

test('document.attachment_binding: descriptor read interface denies owner record-rule false', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    for (const backend of ['db', 's3'] as const) {
      const ownerRecordId = uid(`owner_descriptor_rr_false_${backend}`);
      const fieldName = 'AttachmentField';
      const attachmentObjectId = await createActiveObjectForBackend(ownerRecordId, fieldName, backend);

      const created = await AttachmentBinding.Create(
        {
          OwnerModel: 'unknown.Model',
          OwnerRecordId: ownerRecordId,
          FieldName: fieldName,
          AttachmentContentId: attachmentObjectId,
          DownloadDisposition: 'attachment',
          Status: 'active',
          CompanyId: TEST_COMPANY_ID,
        } as any,
        ['Id'] as any
      );
      const attachmentBindingId = String((created as any)?.Id || '').trim();
      expect(attachmentBindingId).toBeTruthy();

      try {
        await AttachmentBindingDescriptorTestProxy.BuildDescriptor(attachmentBindingId);
        throw new Error(`expected owner record-rule false error on descriptor read (backend=${backend})`);
      } catch (err) {
        expect(err instanceof ChoysumError).toBe(true);
        const oe = err as ChoysumError;
        expect(oe.domain).toBe('document');
        expect(oe.code).toBe('PERMISSION_DENIED');
        expect(oe.metadata?.stage).toBe('descriptor');
      }
    }
  });
});

test('document.attachment_binding: descriptor read interface denies owner field deny-read', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const adminUserId = await mustFindAdminUserId();
    const deniedFieldName = 'PasswordHash';
    const deniedFieldScope = await mustFindAuthUserFieldScope(deniedFieldName);

    const createdRole = await Role.Create(
      {
        Name: shortUid('document_descriptor_deny_role', 90),
        Code: shortUid('document.descriptor.deny', 45),
        Description: 'document descriptor field deny fixture',
        IsActive: true,
        IsSystem: false,
      } as any,
      ['Id'] as any
    );
    const roleId = String((createdRole as any)?.Id || '').trim();
    expect(roleId).toBeTruthy();

    const createdUserRole = await UserRole.Create(
      {
        UserId: { Id: adminUserId } as any,
        RoleId: { Id: roleId } as any,
        CompanyId: TEST_COMPANY_ID,
      } as any,
      ['Id'] as any
    );
    const userRoleId = String((createdUserRole as any)?.Id || '').trim();
    expect(userRoleId).toBeTruthy();

    const createdRule = await RoleFieldRule.Create(
      {
        RoleId: { Id: roleId } as any,
        IrApplicationId: null,
        IrModelId: deniedFieldScope.modelId,
        IrFieldId: deniedFieldScope.fieldId,
        PermRead: 'deny',
      } as any,
      ['Id'] as any
    );
    const ruleId = String((createdRule as any)?.Id || '').trim();
    expect(ruleId).toBeTruthy();

    try {
      for (const backend of ['db', 's3'] as const) {
        const ownerRecordId = uid(`owner_descriptor_field_deny_${backend}`);
        const attachmentObjectId = await withScope(TEST_COMPANY_ID, [TEST_COMPANY_ID], adminUserId, async () => {
          return createActiveObjectForBackend(ownerRecordId, 'AttachmentField', backend);
        });

        const createdBinding = await withScope(TEST_COMPANY_ID, [TEST_COMPANY_ID], adminUserId, async () => {
          return AttachmentBinding.Create(
            {
              OwnerModel: 'auth.User',
              OwnerRecordId: ownerRecordId,
              FieldName: deniedFieldName,
              AttachmentContentId: attachmentObjectId,
              DownloadDisposition: 'attachment',
              Status: 'active',
              CompanyId: TEST_COMPANY_ID,
            } as any,
            ['Id'] as any
          );
        });
        const attachmentBindingId = String((createdBinding as any)?.Id || '').trim();
        expect(attachmentBindingId).toBeTruthy();

        await withScope(TEST_COMPANY_ID, [TEST_COMPANY_ID], adminUserId, async () => {
          try {
            await AttachmentBindingDescriptorTestProxy.BuildDescriptor(attachmentBindingId);
            throw new Error(`expected owner field deny-read error on descriptor read (backend=${backend})`);
          } catch (err) {
            expect(err instanceof ChoysumError).toBe(true);
            const oe = err as ChoysumError;
            expect(oe.domain).toBe('document');
            expect(oe.code).toBe('PERMISSION_DENIED');
            expect(oe.metadata?.stage).toBe('descriptor');
          }
        });
      }
    } finally {
      await withScope(TEST_COMPANY_ID, [TEST_COMPANY_ID], adminUserId, async () => {
        await RoleFieldRule.DeleteById(ruleId as any);
        await UserRole.DeleteById(userRoleId as any);
        await Role.DeleteById(roleId as any);
      });
    }
  });
});

test('document.attachment_binding: ResolveDownloadContent returns read ticket and inline semantics for visible binding', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const ownerRecordId = uid('owner_resolve_allow');
    const fieldName = 'AttachmentField';

    const attachmentContentId = await createActiveContentRecord({
      backend: 's3',
      mimeType: 'image/png',
      sizeBytes: 32,
    });

    const created = await AttachmentBinding.Create(
      {
        OwnerModel: 'auth.User',
        OwnerRecordId: ownerRecordId,
        FieldName: fieldName,
        AttachmentContentId: attachmentContentId,
        DisplayFileName: 'cover.png',
        DownloadDisposition: 'inline',
        Status: 'active',
        CompanyId: TEST_COMPANY_ID,
      } as any,
      ['Id'] as any
    );
    const attachmentBindingId = String((created as any)?.Id || '').trim();
    expect(attachmentBindingId).toBeTruthy();

    const resolved = await AttachmentBinding.ResolveDownloadContent({
      attachmentBindingId,
      principal: {
        userId: TEST_USER_ID,
        activeCompanyId: TEST_COMPANY_ID,
        enabledCompanyIds: [TEST_COMPANY_ID],
      },
    });

    expect(resolved.attachmentBindingId).toBe(attachmentBindingId);
    expect(resolved.mimeType).toBe('image/png');
    expect(resolved.sizeBytes).toBe(32);
    expect(resolved.checksumSha256).toBe(EMPTY_SHA256);
    expect(resolved.fileName).toBe('cover.png');
    expect(resolved.downloadDisposition).toBe('inline');
    expect(resolved.etag).toBe(`"sha256:${EMPTY_SHA256}"`);

    const readTicket = JSON.parse(resolved.payloadReadTicket) as Record<string, unknown>;
    expect(readTicket.attachmentBindingId).toBe(attachmentBindingId);
    expect(readTicket.attachmentContentId).toBe(attachmentContentId);
    expect(String(readTicket.storedContentId || '')).toBeTruthy();
  });
});

test('document.attachment_binding: ResolveDownloadContent downgrades inline to attachment for non-inline mime', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const ownerRecordId = uid('owner_resolve_attachment_only');
    const fieldName = 'AttachmentField';
    const checksumSha256 = 'a'.repeat(64);

    const attachmentContentId = await createActiveContentRecord({
      backend: 'db',
      mimeType: 'application/octet-stream',
      sizeBytes: 11,
      checksumSha256,
    });

    const created = await AttachmentBinding.Create(
      {
        OwnerModel: 'auth.User',
        OwnerRecordId: ownerRecordId,
        FieldName: fieldName,
        AttachmentContentId: attachmentContentId,
        DownloadDisposition: 'inline',
        Status: 'active',
        CompanyId: TEST_COMPANY_ID,
      } as any,
      ['Id'] as any
    );
    const attachmentBindingId = String((created as any)?.Id || '').trim();
    expect(attachmentBindingId).toBeTruthy();

    const resolved = await AttachmentBinding.ResolveDownloadContent({
      attachmentBindingId,
      principal: {
        userId: TEST_USER_ID,
        activeCompanyId: TEST_COMPANY_ID,
        enabledCompanyIds: [TEST_COMPANY_ID],
      },
    });

    expect(resolved.mimeType).toBe('application/octet-stream');
    expect(resolved.downloadDisposition).toBe('attachment');
    expect(resolved.fileName).toBe(`attachment-${attachmentBindingId}`);
    expect(resolved.etag).toBe(`"sha256:${checksumSha256}"`);
  });
});

test('document.attachment_binding: ResolveDownloadContent rejects principal issuer mismatch', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const ownerRecordId = uid('owner_resolve_issuer_mismatch');
    const fieldName = 'AttachmentField';
    const attachmentContentId = await createActiveContentRecord({
      backend: 'db',
      mimeType: 'text/plain',
      sizeBytes: 3,
    });

    const created = await AttachmentBinding.Create(
      {
        OwnerModel: 'auth.User',
        OwnerRecordId: ownerRecordId,
        FieldName: fieldName,
        AttachmentContentId: attachmentContentId,
        DownloadDisposition: 'attachment',
        Status: 'active',
        CompanyId: TEST_COMPANY_ID,
      } as any,
      ['Id'] as any
    );
    const attachmentBindingId = String((created as any)?.Id || '').trim();
    expect(attachmentBindingId).toBeTruthy();

    try {
      await AttachmentBinding.ResolveDownloadContent({
        attachmentBindingId,
        principal: {
          userId: uid('other_user'),
          activeCompanyId: TEST_COMPANY_ID,
          enabledCompanyIds: [TEST_COMPANY_ID],
        },
      });
      throw new Error('expected issuer mismatch to be denied');
    } catch (err) {
      expect(err instanceof ChoysumError).toBe(true);
      const oe = err as ChoysumError;
      expect(oe.domain).toBe('document');
      expect(oe.code).toBe('PERMISSION_DENIED');
      expect(oe.metadata?.stage).toBe('resolve_download_content');
      expect(oe.metadata?.reason).toBe('issuer_mismatch');
    }
  });
});

test('document.attachment_binding: ResolveDownloadContent rejects activeCompany mismatch', async () => {
  resetRequestContext();

  const ownerRecordId = uid('owner_resolve_company_mismatch');
  const fieldName = 'AttachmentField';

  const attachmentObjectId = await withScope('cmp_resolve_a', ['cmp_resolve_a'], 'usr_resolve_a', async () => {
    return createActiveObject(ownerRecordId, fieldName, 'cmp_resolve_a');
  });

  const bound = await withScope('cmp_resolve_a', ['cmp_resolve_a'], 'usr_resolve_a', async () => {
    return AttachmentBinding.Bind({
      attachmentObjectId,
      ownerModel: 'auth.User',
      ownerRecordId,
      fieldName,
      mutationId: uid('mutation_resolve_company_mismatch'),
    });
  });

  await withScope('cmp_resolve_b', ['cmp_resolve_a', 'cmp_resolve_b'], 'usr_resolve_a', async () => {
    try {
      await AttachmentBinding.ResolveDownloadContent({
        attachmentBindingId: bound.attachmentBindingId,
        principal: {
          userId: 'usr_resolve_a',
          activeCompanyId: 'cmp_resolve_b',
          enabledCompanyIds: ['cmp_resolve_a', 'cmp_resolve_b'],
        },
      });
      throw new Error('expected company mismatch to be denied');
    } catch (err) {
      expect(err instanceof ChoysumError).toBe(true);
      const oe = err as ChoysumError;
      expect(oe.domain).toBe('document');
      expect(oe.code).toBe('PERMISSION_DENIED');
      expect(oe.metadata?.stage).toBe('resolve_download_content');
    }
  });
});

test('document.attachment_binding: ResolveDownloadContent keeps stable etag for same attachment', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const ownerRecordId = uid('owner_resolve_etag_stable');
    const fieldName = 'AttachmentField';
    const checksumSha256 = 'b'.repeat(64);

    const attachmentContentId = await createActiveContentRecord({
      backend: 'db',
      mimeType: 'text/plain',
      sizeBytes: 7,
      checksumSha256,
    });

    const created = await AttachmentBinding.Create(
      {
        OwnerModel: 'auth.User',
        OwnerRecordId: ownerRecordId,
        FieldName: fieldName,
        AttachmentContentId: attachmentContentId,
        DisplayFileName: 'etag.txt',
        DownloadDisposition: 'attachment',
        Status: 'active',
        CompanyId: TEST_COMPANY_ID,
      } as any,
      ['Id'] as any
    );
    const attachmentBindingId = String((created as any)?.Id || '').trim();
    expect(attachmentBindingId).toBeTruthy();

    const first = await AttachmentBinding.ResolveDownloadContent({
      attachmentBindingId,
      principal: {
        userId: TEST_USER_ID,
        activeCompanyId: TEST_COMPANY_ID,
        enabledCompanyIds: [TEST_COMPANY_ID],
      },
    });
    const second = await AttachmentBinding.ResolveDownloadContent({
      attachmentBindingId,
      principal: {
        userId: TEST_USER_ID,
        activeCompanyId: TEST_COMPANY_ID,
        enabledCompanyIds: [TEST_COMPANY_ID],
      },
    });

    expect(first.etag).toBe(`"sha256:${checksumSha256}"`);
    expect(second.etag).toBe(first.etag);
    expect(second.payloadReadTicket).toBe(first.payloadReadTicket);
  });
});

test('document.attachment_binding: BatchDescribe returns descriptors in request order for visible active bindings', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const ownerRecordId = uid('owner_batch_describe_allow');
    const fieldName = 'AttachmentField';

    const imageObjectId = await createActiveContentRecord({
      backend: 'db',
      mimeType: 'image/png',
      sizeBytes: 8,
    });
    const textObjectId = await createActiveContentRecord({
      backend: 'db',
      mimeType: 'text/plain',
      sizeBytes: 12,
    });
    expect(imageObjectId).toBeTruthy();
    expect(textObjectId).toBeTruthy();

    const createdImageBinding = await AttachmentBinding.Create(
      {
        OwnerModel: 'auth.User',
        OwnerRecordId: ownerRecordId,
        FieldName: fieldName,
        AttachmentContentId: imageObjectId,
        DisplayFileName: 'cover.png',
        DownloadDisposition: 'inline',
        Status: 'active',
        CompanyId: TEST_COMPANY_ID,
      } as any,
      ['Id'] as any
    );
    const createdTextBinding = await AttachmentBinding.Create(
      {
        OwnerModel: 'auth.User',
        OwnerRecordId: ownerRecordId,
        FieldName: `${fieldName}Second`,
        AttachmentContentId: textObjectId,
        DisplayFileName: 'notes.txt',
        DownloadDisposition: 'attachment',
        Status: 'active',
        CompanyId: TEST_COMPANY_ID,
      } as any,
      ['Id'] as any
    );

    const imageBindingId = String((createdImageBinding as any)?.Id || '').trim();
    const textBindingId = String((createdTextBinding as any)?.Id || '').trim();
    expect(imageBindingId).toBeTruthy();
    expect(textBindingId).toBeTruthy();

    const response = await AttachmentBinding.BatchDescribe({
      attachmentBindingIds: [textBindingId, uid('missing_binding'), imageBindingId, textBindingId],
    });

    expect(response.items.map(item => item.attachmentBindingId)).toEqual([textBindingId, imageBindingId]);

    const textItem = response.items[0];
    const imageItem = response.items[1];

    expect(textItem.displayName).toBe('notes.txt');
    expect(textItem.descriptor.fileName).toBe('notes.txt');
    expect(textItem.descriptor.mimeType).toBe('text/plain');
    expect(textItem.previewUrl).toBeUndefined();

    expect(imageItem.displayName).toBe('cover.png');
    expect(imageItem.descriptor.fileName).toBe('cover.png');
    expect(imageItem.descriptor.mimeType).toBe('image/png');
    expect(imageItem.previewUrl).toBe(`/_document/bindings/${imageBindingId}/content`);
  });
});

test('document.attachment_binding: BatchDescribe rejects unauthenticated caller', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const ownerRecordId = uid('owner_batch_describe_unauth');
    const fieldName = 'AttachmentField';
    const attachmentObjectId = await createActiveObject(ownerRecordId, fieldName);

    const created = await AttachmentBinding.Create(
      {
        OwnerModel: 'auth.User',
        OwnerRecordId: ownerRecordId,
        FieldName: fieldName,
        AttachmentContentId: attachmentObjectId,
        DisplayFileName: 'unauth.txt',
        DownloadDisposition: 'attachment',
        Status: 'active',
        CompanyId: TEST_COMPANY_ID,
      } as any,
      ['Id'] as any
    );
    const attachmentBindingId = String((created as any)?.Id || '').trim();
    expect(attachmentBindingId).toBeTruthy();

    const jsCtx = ensureRequestContext();
    jsCtx.identity = {};

    try {
      await AttachmentBinding.BatchDescribe({ attachmentBindingIds: [attachmentBindingId] });
      throw new Error('expected batch describe unauthenticated error');
    } catch (err) {
      expect(err instanceof ChoysumError).toBe(true);
      const oe = err as ChoysumError;
      expect(oe.domain).toBe('document');
      expect(oe.code).toBe('UNAUTHENTICATED');
    }
  });
});

test('document.attachment_binding: BatchDescribe denies owner record-rule false', async () => {
  resetRequestContext();
  await withDocumentScope(async () => {
    const ownerRecordId = uid('owner_batch_describe_rr_false');
    const fieldName = 'AttachmentField';
    const attachmentObjectId = await createActiveObject(ownerRecordId, fieldName);

    const created = await AttachmentBinding.Create(
      {
        OwnerModel: 'unknown.Model',
        OwnerRecordId: ownerRecordId,
        FieldName: fieldName,
        AttachmentContentId: attachmentObjectId,
        DownloadDisposition: 'attachment',
        Status: 'active',
        CompanyId: TEST_COMPANY_ID,
      } as any,
      ['Id'] as any
    );
    const attachmentBindingId = String((created as any)?.Id || '').trim();
    expect(attachmentBindingId).toBeTruthy();

    try {
      await AttachmentBinding.BatchDescribe({ attachmentBindingIds: [attachmentBindingId] });
      throw new Error('expected owner record-rule false error on batch describe');
    } catch (err) {
      expect(err instanceof ChoysumError).toBe(true);
      const oe = err as ChoysumError;
      expect(oe.domain).toBe('document');
      expect(oe.code).toBe('PERMISSION_DENIED');
      expect(oe.metadata?.stage).toBe('descriptor');
    }
  });
});

test('document.attachment_binding: BatchDescribe rejects when attachmentBindingIds exceeds max batch size', () => {
  const oversizedIds = Array.from({ length: 201 }, (_, i) => `bid_${i}`);
  let caught: ChoysumError | undefined;
  try {
    normalizeBatchDescribeReq({ attachmentBindingIds: oversizedIds } as any);
  } catch (err) {
    caught = err as ChoysumError;
  }
  expect(caught).toBeDefined();
  expect(caught!.domain).toBe('document');
  expect(caught!.code).toBe('INVALID_ARGUMENT');
  expect(caught!.metadata?.field).toBe('attachmentBindingIds');
});
