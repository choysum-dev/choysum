// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * BaseModel CreatedUid / UpdatedUid / DeletedUid stamp matrix.
 */

import UoMCategory from '@/base/service/models/uom_category';

import { uid } from './_helpers';

const RR_CACHE_KEY = Symbol.for('choysum.recordrule.cache');
const FR_CACHE_KEY = Symbol.for('choysum.fieldrule.cache');

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
    recordRuleMode: 'allowlist',
    recordRuleAllow: [
      'base.UoMCategory:read',
      'base.UoMCategory:write',
      'base.UoMCategory:create',
      'base.UoMCategory:delete',
      'UoMCategory:read',
      'UoMCategory:write',
      'UoMCategory:create',
      'UoMCategory:delete',
    ],
    fieldRuleMode: 'skip',
  };
  jsCtx.identity = {};
  delete (jsCtx as any)[Symbol.for('choysum.ctx.override')];
  delete (jsCtx as any)[Symbol.for('choysum.ctx.frozen')];
  delete (jsCtx as any)[RR_CACHE_KEY];
  delete (jsCtx as any)[FR_CACHE_KEY];
}

function setIdentity(userId?: string): void {
  const jsCtx = ensureRequestContext();
  if (!jsCtx.identity) jsCtx.identity = {};
  if (userId) jsCtx.identity.userId = userId;
  else delete jsCtx.identity.userId;
}

test('base.audit_uid: Create stamps CreatedUid/UpdatedUid; soft delete stamps DeletedUid', async () => {
  resetRequestContext();
  const actor = uid('audit_actor').slice(0, 20);
  setIdentity(actor);

  const created = await UoMCategory.Create(
    {
      Name: uid('AuditUidCategory'),
      Code: uid('AUC').slice(-16),
      IsActive: true,
    } as any,
    ['Id', 'CreatedUid', 'UpdatedUid', 'DeletedUid', 'DeletedAt'] as any
  );

  expect(String((created as any).CreatedUid)).toBe(actor);
  expect(String((created as any).UpdatedUid)).toBe(actor);
  expect((created as any).DeletedUid == null || (created as any).DeletedUid === '').toBe(true);

  const id = String((created as any).Id);
  const updater = uid('audit_upd').slice(0, 20);
  setIdentity(updater);
  const updated = await UoMCategory.UpdateById(
    id,
    { Name: uid('AuditUidCategory2'), CreatedUid: uid('hijack').slice(0, 20) } as any,
    ['Id', 'Name', 'CreatedUid', 'UpdatedUid'] as any
  );
  expect(String((updated as any).CreatedUid)).toBe(actor);
  expect(String((updated as any).UpdatedUid)).toBe(updater);

  await UoMCategory.DeleteById(id);
  const deleted = await UoMCategory.Browse(id, ['Id', 'DeletedAt', 'DeletedUid', 'UpdatedUid'] as any, {
    withDeleted: true,
  } as any);
  expect((deleted as any).DeletedAt).toBeTruthy();
  expect(String((deleted as any).DeletedUid)).toBe(updater);
  expect(String((deleted as any).UpdatedUid)).toBe(updater);

  // Restore clears DeletedUid and refreshes UpdatedUid.
  const restorer = uid('audit_res').slice(0, 20);
  setIdentity(restorer);
  const restored = await UoMCategory.Update(
    ['Id', '=', id] as any,
    { DeletedAt: null } as any,
    ['Id', 'DeletedAt', 'DeletedUid', 'UpdatedUid'] as any,
    { withDeleted: true }
  );
  expect(restored.length).toBe(1);
  expect((restored[0] as any).DeletedAt == null || (restored[0] as any).DeletedAt === '').toBe(true);
  expect((restored[0] as any).DeletedUid == null || (restored[0] as any).DeletedUid === '').toBe(true);
  expect(String((restored[0] as any).UpdatedUid)).toBe(restorer);

  await UoMCategory.DeleteById(id);
});

test('base.audit_uid: no actor leaves uid null while timestamps still write', async () => {
  resetRequestContext();
  setIdentity(undefined);

  const created = await UoMCategory.Create(
    {
      Name: uid('AuditUidNoActor'),
      Code: uid('AUNA').slice(-16),
      IsActive: true,
    } as any,
    ['Id', 'CreatedUid', 'UpdatedUid', 'CreatedAt', 'UpdatedAt'] as any
  );
  expect((created as any).CreatedUid == null || (created as any).CreatedUid === '').toBe(true);
  expect((created as any).UpdatedUid == null || (created as any).UpdatedUid === '').toBe(true);
  expect((created as any).CreatedAt).toBeTruthy();
  expect((created as any).UpdatedAt).toBeTruthy();
  await UoMCategory.DeleteById(String((created as any).Id));
});
