// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Role from '@/auth/service/models/role';
import User from '@/auth/service/models/user';
import UserRole from '@/auth/service/models/user_role';
import { withContext as withModelContext } from '@/core/service/api/context';
import { ChoysumError } from '@/core/service/error';
import MetaModelData from '@/meta/service/models/model_data';
import ExportTemplate from '@/web/service/models/export_template';

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
      'web.ExportTemplate:read',
      'web.ExportTemplate:write',
      'web.ExportTemplate:create',
      'web.ExportTemplate:delete',
      'ExportTemplate:read',
      'ExportTemplate:write',
      'ExportTemplate:create',
      'ExportTemplate:delete',
      'meta.MetaModelData:read',
      'MetaModelData:read',
      'auth.User:read',
      'auth.User:write',
      'auth.User:create',
      'User:read',
      'User:write',
      'User:create',
      'auth.Role:read',
      'Role:read',
      'auth.UserRole:read',
      'auth.UserRole:write',
      'auth.UserRole:create',
      'UserRole:read',
      'UserRole:write',
      'UserRole:create',
      'auth.RoleRecordRule:read',
      'RoleRecordRule:read',
    ],
    fieldRuleMode: 'skip',
  };
  jsCtx.identity = { userId: uid('bootstrap') };
  delete (jsCtx as any)[Symbol.for('choysum.ctx.override')];
  delete (jsCtx as any)[Symbol.for('choysum.ctx.frozen')];
  delete (jsCtx as any)[RR_CACHE_KEY];
  delete (jsCtx as any)[FR_CACHE_KEY];
}

function uid(prefix: string): string {
  const xid = (globalThis as any).$choysum?.xid?.New?.();
  const u = typeof xid === 'string' && xid.trim() ? xid.trim() : String(Date.now());
  return `${prefix}_${u}`;
}

function setIdentity(userId?: string): void {
  const jsCtx = ensureRequestContext();
  if (!jsCtx.identity) jsCtx.identity = {};
  if (userId) jsCtx.identity.userId = userId;
  else delete jsCtx.identity.userId;
}

function setReq(patch: Record<string, any>): void {
  const jsCtx = ensureRequestContext();
  if (!jsCtx.req) jsCtx.req = {};
  Object.assign(jsCtx.req, patch);
}

function disableAllowlist(): void {
  setReq({ recordRuleMode: '', recordRuleAllow: [] });
}

function toErr(err: any): { domain?: string; code?: string } | null {
  if (!err) return null;
  if (err instanceof ChoysumError) return err as any;
  const visited = new Set<any>();
  const queue: any[] = [err];
  while (queue.length) {
    const cur = queue.shift();
    if (!cur || visited.has(cur)) continue;
    visited.add(cur);
    if (cur instanceof ChoysumError) return cur as any;
    if (typeof cur === 'object') {
      if (typeof cur.domain === 'string' || typeof cur.code === 'string') {
        return { domain: cur.domain, code: cur.code };
      }
      if (cur.cause) queue.push(cur.cause);
      if (cur.error) queue.push(cur.error);
    }
  }
  return null;
}

function collectErrorCodes(err: any): string[] {
  const codes: string[] = [];
  const visited = new Set<any>();
  const queue: any[] = [err];
  while (queue.length) {
    const cur = queue.shift();
    if (!cur || visited.has(cur)) continue;
    visited.add(cur);
    if (typeof cur.code === 'string' && cur.code) codes.push(cur.code);
    if (cur.cause) queue.push(cur.cause);
    if (cur.error) queue.push(cur.error);
  }
  return codes;
}

async function expectCode(fn: () => Promise<any>, code: string, messageHint?: string): Promise<void> {
  let caught: any;
  try {
    await fn();
  } catch (e) {
    caught = e;
  }
  if (!caught) {
    throw new Error(`expected error ${code}, got nothing`);
  }
  const oe = toErr(caught);
  const codes = collectErrorCodes(caught);
  const hasCode = oe?.code === code || codes.includes(code);
  if (!hasCode) {
    throw new Error(`expected error ${code}, got codes=${codes.join(',') || '(none)'} msg=${String((caught as any)?.message || caught)}`);
  }
  if (messageHint) {
    const msg = String((caught as any)?.message || '');
    if (!msg.includes(messageHint)) {
      throw new Error(`expected message hint=${messageHint}, got ${msg}`);
    }
  }
}

async function resolveRoleByCode(code: string): Promise<string> {
  const rows = await Role.Search({ And: [['Code', '=', code]] } as any, { fields: ['Id'], limit: 1 } as any);
  const id = String((rows as any)?.[0]?.Id || '').trim();
  if (!id) throw new Error(`bootstrap role not found: ${code}`);
  return id;
}

async function resolveAdminCompanyId(): Promise<string> {
  const rows = await User.Search({ And: [['Username', '=', 'admin']] } as any, { fields: ['CompanyId'], limit: 1 } as any);
  const companyId = String((rows as any)?.[0]?.CompanyId || '').trim();
  if (!companyId) throw new Error('bootstrap admin company not found');
  return companyId;
}

async function createBaseUser(companyId: string): Promise<string> {
  return await withModelContext(
    { activeCompanyId: companyId, enabledCompanyIds: [companyId] } as any,
    async () => {
      const created = await User.Create(
        {
          Username: uid('et_u'),
          PasswordHash: 'test',
          FirstName: 'ET',
          LastName: 'User',
          CompanyId: companyId,
          CompanyIds: [companyId],
          IsActive: true,
        } as any,
        ['Id'] as any
      );
      const userId = String((created as any).Id || '').trim();
      const roleId = await resolveRoleByCode('base.user');
      await UserRole.Create(
        {
          UserId: { Id: userId } as any,
          RoleId: { Id: roleId } as any,
          CompanyId: null as any,
        } as any,
        ['Id'] as any
      );
      return userId;
    },
    { merge: false }
  );
}

test('ExportTemplate CRUD + Fields round-trip', async () => {
  resetRequestContext();
  const actor = uid('et_actor');
  setIdentity(actor);
  const fields = ['Name', 'CompanyId/Code'];

  const created = await ExportTemplate.Create(
    {
      Name: uid('tpl'),
      Application: 'partner',
      ModelName: 'Partner',
      Fields: fields,
      ImportCompatible: true,
    } as any,
    ['Id', 'UserId', 'Fields', 'ImportCompatible'] as any
  );
  expect(String((created as any).UserId)).toBe(actor);
  expect((created as any).Fields).toEqual(fields);
  expect((created as any).ImportCompatible).toBe(true);

  const again = await ExportTemplate.Browse(String((created as any).Id), ['Fields', 'ImportCompatible'] as any);
  expect((again as any).Fields).toEqual(fields);
  expect((again as any).ImportCompatible).toBe(true);

  await ExportTemplate.DeleteById(String((created as any).Id));
});

test('ExportTemplate rejects foreign UserId on Create', async () => {
  resetRequestContext();
  const actor = uid('et_owner');
  const other = uid('et_victim');
  setIdentity(actor);
  await expectCode(
    async () =>
      ExportTemplate.Create(
        {
          Name: uid('steal'),
          Application: 'partner',
          ModelName: 'Partner',
          Fields: ['Name'],
          UserId: other,
        } as any,
        ['Id'] as any
      ),
    'validation_failed',
    'another user'
  );
});

test('ExportTemplate shared visibility follows UserFilter pattern', async () => {
  resetRequestContext();
  const actor = uid('et_actor');
  setIdentity(actor);
  const shared = await ExportTemplate.Create(
    {
      Name: uid('shared_tpl'),
      Application: 'partner',
      ModelName: 'Partner',
      Fields: ['Name'],
      UserId: null,
    } as any,
    ['Id', 'UserId'] as any
  );
  const other = uid('et_other');
  setIdentity(other);
  const visible = await ExportTemplate.Search(
    {
      And: [
        ['Application', '=', 'partner'],
        ['ModelName', '=', 'Partner'],
        {
          Or: [
            ['UserId', '=', other],
            ['UserId', '=', null],
          ],
        },
      ],
    } as any,
    { fields: ['Id'] } as any
  );
  const ids = new Set((visible || []).map((r: any) => String(r.Id)));
  expect(ids.has(String((shared as any).Id))).toBe(true);
  setIdentity(actor);
  await ExportTemplate.DeleteById(String((shared as any).Id));
});

test('SF11: shared write/delete only for creator via Record rules', async () => {
  resetRequestContext();
  const companyId = await resolveAdminCompanyId();
  const creator = await createBaseUser(companyId);
  const stranger = await createBaseUser(companyId);

  setIdentity(creator);
  const shared = await ExportTemplate.Create(
    {
      Name: uid('sf11'),
      Application: 'partner',
      ModelName: 'Partner',
      Fields: ['Name'],
      UserId: null,
    } as any,
    ['Id', 'CreatedUid'] as any
  );

  disableAllowlist();
  delete (ensureRequestContext() as any)[RR_CACHE_KEY];

  setIdentity(stranger);
  await expectCode(
    async () => ExportTemplate.UpdateById(String((shared as any).Id), { Name: uid('hijack') } as any, ['Id'] as any),
    'record_rule_violation',
    'violates record rule'
  );
  await expectCode(
    async () => ExportTemplate.DeleteById(String((shared as any).Id)),
    'record_rule_violation',
    'violates record rule'
  );

  setIdentity(creator);
  await ExportTemplate.UpdateById(String((shared as any).Id), { Name: uid('ok') } as any, ['Id'] as any);
  await ExportTemplate.DeleteById(String((shared as any).Id));
});

test('web bootstrap seeds ExportTemplate authz packs (RMA/RFR/RR)', async () => {
  resetRequestContext();
  const expected: Array<{ name: string; model: string }> = [
    { name: 'rma_base_user_web_export_template_search', model: 'RoleMethodAccess' },
    { name: 'rma_base_user_web_export_template_browse', model: 'RoleMethodAccess' },
    { name: 'rma_base_user_web_export_template_create', model: 'RoleMethodAccess' },
    { name: 'rma_base_user_web_export_template_update', model: 'RoleMethodAccess' },
    { name: 'rma_base_user_web_export_template_update_by_id', model: 'RoleMethodAccess' },
    { name: 'rma_base_user_web_export_template_delete', model: 'RoleMethodAccess' },
    { name: 'rma_base_user_web_export_template_delete_by_id', model: 'RoleMethodAccess' },
    { name: 'rfr_base_user_web_export_template_rw', model: 'RoleFieldRule' },
    { name: 'rrr_base_user_web_export_template_rc', model: 'RoleRecordRule' },
    { name: 'rrr_base_user_web_export_template_wd_private', model: 'RoleRecordRule' },
    { name: 'rrr_base_user_web_export_template_wd_shared', model: 'RoleRecordRule' },
  ];
  for (const { name, model } of expected) {
    const rows = await MetaModelData.Search(
      {
        And: [
          ['Module', '=', 'web'],
          ['Name', '=', name],
        ],
      } as any,
      { fields: ['Id', 'Application', 'ModelName'], limit: 1 } as any
    );
    if (!Array.isArray(rows) || rows.length !== 1) {
      throw new Error(`missing web.${name}`);
    }
    expect(String((rows as any)[0].Application)).toBe('auth');
    expect(String((rows as any)[0].ModelName)).toBe(model);
  }
});
