// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Role from '@/auth/service/models/role';
import User from '@/auth/service/models/user';
import UserRole from '@/auth/service/models/user_role';
import { withContext as withModelContext } from '@/core/service/api/context';
import { ChoysumError } from '@/core/service/error';
import { createServiceByModel } from '@/core/service/rpc';
import type MetaModel from '@/meta/service/models/model';
import MetaModelData from '@/meta/service/models/model_data';
import UserFilter from '@/web/service/models/user_filter';

const MetaModelService = createServiceByModel<typeof MetaModel>('meta.MetaModel');

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
      'web.UserFilter:read',
      'web.UserFilter:write',
      'web.UserFilter:create',
      'web.UserFilter:delete',
      'UserFilter:read',
      'UserFilter:write',
      'UserFilter:create',
      'UserFilter:delete',
      'meta.MetaModel:read',
      'MetaModel:read',
      'meta.MetaModelData:read',
      'MetaModelData:read',
      'web.FieldDefault:read',
      'FieldDefault:read',
      'web.AppSetting:read',
      'AppSetting:read',
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
    if (typeof cur?.metadata?.causeCode === 'string' && cur.metadata.causeCode) {
      codes.push(cur.metadata.causeCode);
    }
    if (typeof cur?.meta?.causeCode === 'string' && cur.meta.causeCode) {
      codes.push(cur.meta.causeCode);
    }
    if (Array.isArray(cur.issues)) for (const issue of cur.issues) queue.push(issue);
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
  // messageHint is an additional assertion after the code matches (not an alternative).
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
          Username: uid('sf_u'),
          PasswordHash: 'test',
          FirstName: 'SF',
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

test('SF13: web FieldDefault and AppSetting models exist after declared service', async () => {
  resetRequestContext();
  const fd = await MetaModelService.Search(
    {
      And: [
        ['Application', '=', 'web'],
        ['Name', '=', 'FieldDefault'],
      ],
    } as any,
    { fields: ['Id'], limit: 1 } as any
  );
  const as = await MetaModelService.Search(
    {
      And: [
        ['Application', '=', 'web'],
        ['Name', '=', 'AppSetting'],
      ],
    } as any,
    { fields: ['Id'], limit: 1 } as any
  );
  expect(Array.isArray(fd) && fd.length > 0).toBe(true);
  expect(Array.isArray(as) && as.length > 0).toBe(true);
});

test('UserFilter CRUD + IsDefault + visibility', async () => {
  resetRequestContext();
  const actor = uid('sf_actor');
  setIdentity(actor);

  const nameA = uid('fav_a');
  const privateFav = await UserFilter.Create(
    {
      Name: nameA,
      Application: 'web',
      ModelName: 'UserFilter',
      Condition: { And: [['Active', '=', true]] },
      IsDefault: true,
    } as any,
    ['Id', 'UserId', 'CreatedUid', 'IsDefault'] as any
  );
  expect(String((privateFav as any).UserId)).toBe(actor);
  expect(String((privateFav as any).CreatedUid)).toBe(actor);
  expect((privateFav as any).IsDefault).toBe(true);

  const nameB = uid('fav_b');
  const second = await UserFilter.Create(
    {
      Name: nameB,
      Application: 'web',
      ModelName: 'UserFilter',
      Condition: {},
      IsDefault: true,
    } as any,
    ['Id', 'IsDefault'] as any
  );
  expect((second as any).IsDefault).toBe(true);
  // Multiple IsDefault rows are allowed; FE picks newest.
  const firstAgain = await UserFilter.Browse(String((privateFav as any).Id), ['IsDefault'] as any);
  expect((firstAgain as any).IsDefault).toBe(true);

  const sharedName = uid('shared');
  const shared = await UserFilter.Create(
    {
      Name: sharedName,
      Application: 'web',
      ModelName: 'UserFilter',
      Condition: {},
      UserId: null,
      IsDefault: false,
    } as any,
    ['Id', 'UserId', 'CreatedUid'] as any
  );
  expect((shared as any).UserId == null || (shared as any).UserId === '').toBe(true);

  const other = uid('sf_other');
  setIdentity(other);
  const visible = await UserFilter.Search(
    {
      And: [
        ['Application', '=', 'web'],
        ['ModelName', '=', 'UserFilter'],
        {
          Or: [
            ['UserId', '=', other],
            ['UserId', '=', null],
          ],
        },
      ],
    } as any,
    { fields: ['Id', 'Name', 'UserId'] } as any
  );
  const ids = new Set((visible || []).map((r: any) => String(r.Id)));
  expect(ids.has(String((shared as any).Id))).toBe(true);
  expect(ids.has(String((privateFav as any).Id))).toBe(false);

  setIdentity(actor);
  await UserFilter.DeleteById(String((privateFav as any).Id));
  await UserFilter.DeleteById(String((second as any).Id));
  await UserFilter.DeleteById(String((shared as any).Id));
});

test('UserFilter allows Create without an installed MetaModel for Application/ModelName', async () => {
  resetRequestContext();
  const actor = uid('sf_noeff');
  setIdentity(actor);
  const created = await UserFilter.Create(
    {
      Name: uid('gone'),
      Application: 'no_such_app',
      ModelName: 'NoSuchModel',
      Condition: {},
    } as any,
    ['Id', 'Application', 'ModelName'] as any
  );
  expect(String((created as any).Application)).toBe('no_such_app');
  expect(String((created as any).ModelName)).toBe('NoSuchModel');
  await UserFilter.DeleteById(String((created as any).Id));
});

test('UserFilter rejects foreign UserId on Create', async () => {
  resetRequestContext();
  const actor = uid('sf_owner');
  const other = uid('sf_victim');
  setIdentity(actor);
  await expectCode(
    async () =>
      UserFilter.Create(
        {
          Name: uid('steal'),
          Application: 'web',
          ModelName: 'UserFilter',
          Condition: {},
          UserId: other,
        } as any,
        ['Id'] as any
      ),
    'PermissionDenied',
    'another user'
  );
});

test('UserFilter private and shared IsDefault can coexist', async () => {
  resetRequestContext();
  const actor = uid('sf_bucket');
  setIdentity(actor);
  const privateName = uid('priv_def');
  const sharedName = uid('shared_def');
  const priv = await UserFilter.Create(
    {
      Name: privateName,
      Application: 'web',
      ModelName: 'UserFilter',
      Condition: {},
      IsDefault: true,
    } as any,
    ['Id', 'IsDefault', 'UserId'] as any
  );
  const shared = await UserFilter.Create(
    {
      Name: sharedName,
      Application: 'web',
      ModelName: 'UserFilter',
      Condition: {},
      UserId: null,
      IsDefault: true,
    } as any,
    ['Id', 'IsDefault', 'UserId'] as any
  );
  expect((priv as any).IsDefault).toBe(true);
  expect((shared as any).IsDefault).toBe(true);
  const privAgain = await UserFilter.Browse(String((priv as any).Id), ['IsDefault'] as any);
  expect((privAgain as any).IsDefault).toBe(true);
  await UserFilter.DeleteById(String((priv as any).Id));
  await UserFilter.DeleteById(String((shared as any).Id));
});

test('web bootstrap seeds UserFilter authz packs (RMA/RFR/RR)', async () => {
  resetRequestContext();
  const expected: Array<{ name: string; model: string }> = [
    { name: 'rma_base_user_web_user_filter_search', model: 'RoleMethodAccess' },
    { name: 'rma_base_user_web_user_filter_browse', model: 'RoleMethodAccess' },
    { name: 'rma_base_user_web_user_filter_create', model: 'RoleMethodAccess' },
    { name: 'rma_base_user_web_user_filter_update', model: 'RoleMethodAccess' },
    { name: 'rma_base_user_web_user_filter_update_by_id', model: 'RoleMethodAccess' },
    { name: 'rma_base_user_web_user_filter_delete', model: 'RoleMethodAccess' },
    { name: 'rma_base_user_web_user_filter_delete_by_id', model: 'RoleMethodAccess' },
    { name: 'rfr_base_user_web_user_filter_rw', model: 'RoleFieldRule' },
    { name: 'rrr_base_user_web_user_filter_rc', model: 'RoleRecordRule' },
    { name: 'rrr_base_user_web_user_filter_wd_private', model: 'RoleRecordRule' },
    { name: 'rrr_base_user_web_user_filter_wd_shared', model: 'RoleRecordRule' },
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

test('SF11: shared write/delete only for creator via Record rules', async () => {
  resetRequestContext();
  const companyId = await resolveAdminCompanyId();
  const creator = await createBaseUser(companyId);
  const stranger = await createBaseUser(companyId);

  setIdentity(creator);
  const shared = await UserFilter.Create(
    {
      Name: uid('sf11'),
      Application: 'web',
      ModelName: 'UserFilter',
      Condition: {},
      UserId: null,
    } as any,
    ['Id', 'CreatedUid'] as any
  );

  disableAllowlist();
  delete (ensureRequestContext() as any)[RR_CACHE_KEY];

  setIdentity(stranger);
  // Write/delete on another user's shared row: targets fail the WD record-rule expr → violation.
  await expectCode(
    async () => UserFilter.UpdateById(String((shared as any).Id), { Name: uid('hijack') } as any, ['Id'] as any),
    'record_rule_violation',
    'violates record rule'
  );
  await expectCode(
    async () => UserFilter.DeleteById(String((shared as any).Id)),
    'record_rule_violation',
    'violates record rule'
  );

  setIdentity(creator);
  await UserFilter.UpdateById(String((shared as any).Id), { Name: uid('ok') } as any, ['Id'] as any);
  const deleted = await UserFilter.DeleteById(String((shared as any).Id));
  expect(deleted).toBe(1);
});

test('UserFilter Create without identity does not enforce login in Constraint', async () => {
  // Production gRPC AuthN + Method ACL gate anonymous writes; model Constraint no longer duplicates that.
  resetRequestContext();
  setIdentity(undefined);
  const created = await UserFilter.Create(
    {
      Name: uid('anon'),
      Application: 'web',
      ModelName: 'UserFilter',
      Condition: {},
    } as any,
    ['Id', 'UserId'] as any
  );
  expect((created as any).UserId == null || (created as any).UserId === '').toBe(true);
  await UserFilter.sudo(() => UserFilter.DeleteById(String((created as any).Id)), {
    hint: 'web.UserFilter.test.cleanupAnonCreate',
  });
});

test('UserFilter allows duplicate Name in the same ownership bucket', async () => {
  resetRequestContext();
  const actor = uid('sf_dup');
  setIdentity(actor);
  const name = uid('same_name');
  const first = await UserFilter.Create(
    {
      Name: name,
      Application: 'web',
      ModelName: 'UserFilter',
      Condition: {},
    } as any,
    ['Id', 'Name'] as any
  );
  const second = await UserFilter.Create(
    {
      Name: name,
      Application: 'web',
      ModelName: 'UserFilter',
      Condition: {},
    } as any,
    ['Id', 'Name'] as any
  );
  expect(String((first as any).Id)).not.toBe(String((second as any).Id));
  expect(String((first as any).Name)).toBe(name);
  expect(String((second as any).Name)).toBe(name);
  await UserFilter.DeleteById(String((first as any).Id));
  await UserFilter.DeleteById(String((second as any).Id));
});

test('UserFilter ScopeKey scopes lists; IsDefault may repeat (Name may repeat)', async () => {
  resetRequestContext();
  const actor = uid('sf_scope');
  setIdentity(actor);
  const name = uid('scoped_name');
  const a = await UserFilter.Create(
    {
      Name: name,
      ScopeKey: '/web/partners/:id',
      Application: 'web',
      ModelName: 'UserFilter',
      Condition: {},
      IsDefault: true,
    } as any,
    ['Id', 'ScopeKey', 'IsDefault'] as any
  );
  expect((a as any).ScopeKey).toBe('/web/partners/:id');
  expect((a as any).IsDefault).toBe(true);

  const b = await UserFilter.Create(
    {
      Name: name,
      ScopeKey: '/web/companies/:id',
      Application: 'web',
      ModelName: 'UserFilter',
      Condition: {},
      IsDefault: true,
    } as any,
    ['Id', 'ScopeKey', 'IsDefault'] as any
  );
  expect((b as any).ScopeKey).toBe('/web/companies/:id');
  expect((b as any).IsDefault).toBe(true);
  const aStill = await UserFilter.Browse(String((a as any).Id), ['IsDefault'] as any);
  expect((aStill as any).IsDefault).toBe(true);

  // Same ScopeKey + Name is allowed; multiple IsDefault in one scope are allowed.
  // ScopeKey is pass-through (FE normalizes before write).
  const sameScope = await UserFilter.Create(
    {
      Name: name,
      ScopeKey: '/web/partners/:id',
      Application: 'web',
      ModelName: 'UserFilter',
      Condition: {},
      IsDefault: true,
    } as any,
    ['Id', 'ScopeKey', 'Name', 'IsDefault'] as any
  );
  expect((sameScope as any).ScopeKey).toBe('/web/partners/:id');
  expect(String((sameScope as any).Name)).toBe(name);
  expect((sameScope as any).IsDefault).toBe(true);
  const aStillDefault = await UserFilter.Browse(String((a as any).Id), ['IsDefault'] as any);
  expect((aStillDefault as any).IsDefault).toBe(true);

  await UserFilter.DeleteById(String((a as any).Id));
  await UserFilter.DeleteById(String((b as any).Id));
  await UserFilter.DeleteById(String((sameScope as any).Id));
});

test('UserFilter stores ScopeKey as written (no BE normalize)', async () => {
  resetRequestContext();
  const actor = uid('sf_scope_passthrough');
  setIdentity(actor);
  const raw = '\\web\\partners\\abc123def456ghi7\\edit?x=1#y';
  const created = await UserFilter.Create(
    {
      Name: uid('scoped_raw'),
      ScopeKey: raw,
      Application: 'web',
      ModelName: 'UserFilter',
      Condition: {},
    } as any,
    ['Id', 'ScopeKey'] as any
  );
  expect((created as any).ScopeKey).toBe(raw);
  await UserFilter.DeleteById(String((created as any).Id));
});

test('UserFilter fills Create defaults (UserId/IsDefault/Condition)', async () => {
  resetRequestContext();
  const actor = uid('sf_defaults');
  setIdentity(actor);
  const created = await UserFilter.Create(
    {
      Name: uid('fills'),
      Application: 'web',
      ModelName: 'UserFilter',
    } as any,
    ['Id', 'UserId', 'IsDefault', 'Condition', 'CreatedUid', 'ScopeKey'] as any
  );
  expect(String((created as any).UserId)).toBe(actor);
  expect((created as any).IsDefault).toBe(false);  expect((created as any).Condition || {}).toEqual({});
  expect(String((created as any).CreatedUid)).toBe(actor);
  expect((created as any).ScopeKey).toBe('');
  await UserFilter.DeleteById(String((created as any).Id));
});

test('UserFilter Update keeps CreatedUid immutable and normalizes UserId', async () => {
  resetRequestContext();
  const actor = uid('sf_upd');
  setIdentity(actor);
  const created = await UserFilter.Create(
    {
      Name: uid('upd'),
      Application: 'web',
      ModelName: 'UserFilter',
      Condition: {},
    } as any,
    ['Id', 'CreatedUid', 'UserId'] as any
  );
  const createUid = String((created as any).CreatedUid);
  const updated = await UserFilter.UpdateById(
    String((created as any).Id),
    { CreatedUid: uid('hijack_uid'), UserId: null } as any,
    ['Id', 'CreatedUid', 'UserId'] as any
  );
  expect(String((updated as any).CreatedUid)).toBe(createUid);
  expect((updated as any).UserId == null || (updated as any).UserId === '').toBe(true);

  await expectCode(
    async () =>
      UserFilter.UpdateById(String((created as any).Id), { UserId: uid('other') } as any, ['Id'] as any),
    'PermissionDenied',
    'another user'
  );
  await UserFilter.DeleteById(String((created as any).Id));
});

test('UserFilter private IsDefault update leaves other defaults intact', async () => {
  resetRequestContext();
  const actor = uid('sf_except');
  setIdentity(actor);
  const a = await UserFilter.Create(
    {
      Name: uid('def_a'),
      Application: 'web',
      ModelName: 'UserFilter',
      Condition: {},
      IsDefault: true,
    } as any,
    ['Id', 'IsDefault'] as any
  );
  const b = await UserFilter.Create(
    {
      Name: uid('def_b'),
      Application: 'web',
      ModelName: 'UserFilter',
      Condition: {},
      IsDefault: false,
    } as any,
    ['Id', 'IsDefault'] as any
  );
  await UserFilter.UpdateById(String((b as any).Id), { IsDefault: true } as any, ['Id', 'IsDefault'] as any);
  const aAgain = await UserFilter.Browse(String((a as any).Id), ['IsDefault'] as any);
  expect((aAgain as any).IsDefault).toBe(true);
  const bAgain = await UserFilter.Browse(String((b as any).Id), ['IsDefault'] as any);
  expect((bAgain as any).IsDefault).toBe(true);
  await UserFilter.DeleteById(String((a as any).Id));
  await UserFilter.DeleteById(String((b as any).Id));
});

test('UserFilter rejects Create missing Name/Application/ModelName', async () => {
  resetRequestContext();
  setIdentity(uid('sf_req'));
  await expectCode(
    async () =>
      UserFilter.Create(
        {
          Name: '',
          Application: 'web',
          ModelName: 'UserFilter',
          Condition: {},
        } as any,
        ['Id'] as any
      ),
    'InvalidArgument',
    'required'
  );
});

test('UserFilter Update reads unchanged fields from current via mergedField', async () => {
  resetRequestContext();
  const actor = uid('sf_merged');
  setIdentity(actor);
  const created = await UserFilter.Create(
    {
      Name: uid('merged'),
      Application: 'web',
      ModelName: 'UserFilter',
      Condition: { And: [['A', '=', 1]] },
    } as any,
    ['Id', 'Name', 'Condition'] as any
  );
  // Touch only IsDefault so Name/Application/ModelName resolve from current.
  const updated = await UserFilter.UpdateById(
    String((created as any).Id),
    { IsDefault: true } as any,
    ['Id', 'Name', 'Application', 'ModelName', 'IsDefault'] as any
  );
  expect(String((updated as any).Name)).toBe(String((created as any).Name));
  expect((updated as any).IsDefault).toBe(true);
  await UserFilter.DeleteById(String((created as any).Id));
});

test('UserFilter allows multiple shared IsDefault from different users', async () => {
  resetRequestContext();
  const companyId = await resolveAdminCompanyId();
  const creator = await createBaseUser(companyId);
  const stranger = await createBaseUser(companyId);

  setIdentity(creator);
  const shared = await UserFilter.Create(
    {
      Name: uid('shared_def_owner'),
      Application: 'web',
      ModelName: 'UserFilter',
      Condition: {},
      UserId: null,
      IsDefault: true,
    } as any,
    ['Id', 'CreatedUid', 'IsDefault'] as any
  );
  expect((shared as any).IsDefault).toBe(true);

  disableAllowlist();
  delete (ensureRequestContext() as any)[RR_CACHE_KEY];

  setIdentity(stranger);
  const strangerShared = await UserFilter.Create(
    {
      Name: uid('shared_def_stranger'),
      Application: 'web',
      ModelName: 'UserFilter',
      Condition: {},
      UserId: null,
      IsDefault: true,
    } as any,
    ['Id', 'IsDefault'] as any
  );
  expect((strangerShared as any).IsDefault).toBe(true);

  setIdentity(creator);
  const again = await UserFilter.Browse(String((shared as any).Id), ['IsDefault'] as any);
  expect((again as any).IsDefault).toBe(true);
  await UserFilter.DeleteById(String((shared as any).Id));
  setIdentity(stranger);
  await UserFilter.DeleteById(String((strangerShared as any).Id));
});

test('UserFilter Create with null UserId is shared', async () => {
  resetRequestContext();
  const actor = uid('sf_ws');
  setIdentity(actor);
  const created = await UserFilter.Create(
    {
      Name: uid('ws_uid'),
      Application: 'web',
      ModelName: 'UserFilter',
      Condition: {},
      UserId: null,
      IsDefault: true,
    } as any,
    ['Id', 'UserId', 'IsDefault'] as any
  );
  expect((created as any).UserId == null || (created as any).UserId === '').toBe(true);
  expect((created as any).IsDefault).toBe(true);
  await UserFilter.DeleteById(String((created as any).Id));
});

test('UserFilter creator can set multiple shared IsDefault', async () => {
  resetRequestContext();
  const actor = uid('sf_shared_ok');
  setIdentity(actor);
  const first = await UserFilter.Create(
    {
      Name: uid('shared_a'),
      Application: 'web',
      ModelName: 'UserFilter',
      Condition: {},
      UserId: null,
      IsDefault: true,
    } as any,
    ['Id', 'IsDefault', 'CreatedUid'] as any
  );
  expect((first as any).IsDefault).toBe(true);
  const second = await UserFilter.Create(
    {
      Name: uid('shared_b'),
      Application: 'web',
      ModelName: 'UserFilter',
      Condition: {},
      UserId: null,
      IsDefault: true,
    } as any,
    ['Id', 'IsDefault'] as any
  );
  expect((second as any).IsDefault).toBe(true);
  const firstAgain = await UserFilter.Browse(String((first as any).Id), ['IsDefault'] as any);
  expect((firstAgain as any).IsDefault).toBe(true);
  await UserFilter.DeleteById(String((first as any).Id));
  await UserFilter.DeleteById(String((second as any).Id));
});

test('UserFilter Update without IsDefault keeps existing defaults', async () => {
  resetRequestContext();
  const actor = uid('sf_upd_def');
  setIdentity(actor);
  const a = await UserFilter.Create(
    {
      Name: uid('upd_def_a'),
      Application: 'web',
      ModelName: 'UserFilter',
      Condition: {},
      IsDefault: true,
    } as any,
    ['Id', 'IsDefault', 'Name'] as any
  );
  const b = await UserFilter.Create(
    {
      Name: uid('upd_def_b'),
      Application: 'web',
      ModelName: 'UserFilter',
      Condition: {},
      IsDefault: false,
    } as any,
    ['Id', 'IsDefault'] as any
  );
  // Promote b via Update; a should clear. Then rename a while it is no longer default
  // and rename the still-default b without sending IsDefault (mergedField path).
  await UserFilter.UpdateById(String((b as any).Id), { IsDefault: true } as any, ['Id', 'IsDefault'] as any);
  const newName = uid('renamed_def');
  await UserFilter.UpdateById(String((b as any).Id), { Name: newName } as any, ['Id', 'Name', 'IsDefault'] as any);
  const bAgain = await UserFilter.Browse(String((b as any).Id), ['Name', 'IsDefault'] as any);
  expect(String((bAgain as any).Name)).toBe(newName);
  expect((bAgain as any).IsDefault).toBe(true);
  const aAgain = await UserFilter.Browse(String((a as any).Id), ['IsDefault'] as any);
  expect((aAgain as any).IsDefault).toBe(true);
  await UserFilter.DeleteById(String((a as any).Id));
  await UserFilter.DeleteById(String((b as any).Id));
});

test('UserFilter Update allows renaming onto another row Name', async () => {
  resetRequestContext();
  const actor = uid('sf_rename');
  setIdentity(actor);
  const nameA = uid('rename_a');
  const nameB = uid('rename_b');
  const a = await UserFilter.Create(
    {
      Name: nameA,
      Application: 'web',
      ModelName: 'UserFilter',
      Condition: {},
    } as any,
    ['Id', 'Name'] as any
  );
  const b = await UserFilter.Create(
    {
      Name: nameB,
      Application: 'web',
      ModelName: 'UserFilter',
      Condition: {},
    } as any,
    ['Id', 'Name'] as any
  );
  const renamed = await UserFilter.UpdateById(String((b as any).Id), { Name: nameA } as any, ['Id', 'Name'] as any);
  expect(String((renamed as any).Name)).toBe(nameA);
  const self = await UserFilter.UpdateById(String((a as any).Id), { Name: nameA } as any, [
    'Id',
    'Name',
  ] as any);
  expect(String((self as any).Name)).toBe(nameA);
  await UserFilter.DeleteById(String((a as any).Id));
  await UserFilter.DeleteById(String((b as any).Id));
});

test('UserFilter Create accepts explicit self UserId', async () => {
  resetRequestContext();
  const actor = uid('sf_self');
  setIdentity(actor);
  const created = await UserFilter.Create(
    {
      Name: uid('self_uid'),
      Application: 'web',
      ModelName: 'UserFilter',
      UserId: actor,
      Condition: {},
    } as any,
    ['Id', 'UserId'] as any
  );
  expect(String((created as any).UserId)).toBe(actor);
  await UserFilter.DeleteById(String((created as any).Id));
});

test('UserFilter constraint fills null IsDefault/Condition on create', async () => {
  resetRequestContext();
  const actor = uid('sf_null_defs');
  setIdentity(actor);
  const SF = UserFilter as any;
  const values: Record<string, any> = {
    Id: uid('null_defs'),
    Name: uid('null_defs_name'),
    Application: 'web',
    ModelName: 'UserFilter',
    UserId: actor,
    IsDefault: null,
    Condition: null,
  };
  await SF.validateUserFilterConstraint({}, { mode: 'create', values, current: undefined });
  expect(values.IsDefault).toBe(false);
  expect(values.Condition).toEqual({});
  expect(values.UserId).toBe(actor);
});

test('UserFilter rejects null Application/ModelName via mergedField empty trim', async () => {
  resetRequestContext();
  setIdentity(uid('sf_null_app'));
  await expectCode(
    async () =>
      UserFilter.Create(
        {
          Name: uid('null_app'),
          Application: null,
          ModelName: 'UserFilter',
          Condition: {},
        } as any,
        ['Id'] as any
      ),
    'InvalidArgument',
    'required'
  );
  await expectCode(
    async () =>
      UserFilter.Create(
        {
          Name: uid('null_model'),
          Application: 'web',
          ModelName: null,
          Condition: {},
        } as any,
        ['Id'] as any
      ),
    'InvalidArgument',
    'required'
  );
});

test('UserFilter _mergedField falls through values/self/current', () => {
  const SF = UserFilter as any;
  expect(SF._mergedField({}, { values: undefined, current: {} }, 'Name')).toBeUndefined();
  expect(SF._mergedField({ Name: 'FromSelf' }, { values: undefined, current: {} }, 'Name')).toBe('FromSelf');
  expect(SF._mergedField({}, { values: {}, current: { Name: 'FromCurrent' } }, 'Name')).toBe('FromCurrent');
  expect(SF._mergedField({ Name: 'Self' }, { values: { Name: 'Values' }, current: { Name: 'Current' } }, 'Name')).toBe(
    'Values'
  );
});


test('UserFilter allows duplicate shared Name for null UserId', async () => {
  resetRequestContext();
  const actor = uid('sf_assert_empty');
  setIdentity(actor);
  const name = uid('assert_empty_name');
  const shared = await UserFilter.Create(
    {
      Name: name,
      ScopeKey: '/assert',
      Application: 'web',
      ModelName: 'UserFilter',
      UserId: null,
      Condition: {},
    } as any,
    ['Id', 'Name'] as any
  );
  const shared2 = await UserFilter.Create(
    {
      Name: name,
      ScopeKey: '/assert',
      Application: 'web',
      ModelName: 'UserFilter',
      UserId: null,
      Condition: {},
    } as any,
    ['Id', 'Name', 'UserId'] as any
  );
  expect(String((shared as any).Id)).not.toBe(String((shared2 as any).Id));
  expect((shared2 as any).UserId == null || (shared2 as any).UserId === '').toBe(true);
  await UserFilter.DeleteById(String((shared as any).Id));
  await UserFilter.DeleteById(String((shared2 as any).Id));
});

test('UserFilter validateUserFilterConstraint covers empty create Id fallbacks', async () => {
  resetRequestContext();
  const actor = uid('sf_validate');
  setIdentity(actor);
  const SF = UserFilter as any;

  // Create with whitespace Id → trim || undefined (exceptId omitted on unique check).
  const valuesCreate: Record<string, any> = {
    Id: '   ',
    Name: uid('empty_id'),
    Application: 'web',
    ModelName: 'UserFilter',
    Condition: {},
    IsDefault: false,
  };
  await SF.validateUserFilterConstraint({}, { mode: 'create', values: valuesCreate, current: undefined });
  expect(valuesCreate.UserId).toBe(actor);
  // CreatedUid is stamped by BaseModel, not this constraint.
  expect(valuesCreate.CreatedUid).toBeUndefined();

  // Falsy Id hits `(values.Id || '')` then `trim() || undefined`.
  const valuesNullId: Record<string, any> = {
    Id: null,
    Name: uid('null_id'),
    Application: 'web',
    ModelName: 'UserFilter',
    Condition: {},
    IsDefault: false,
  };
  await SF.validateUserFilterConstraint({}, { mode: 'create', values: valuesNullId, current: undefined });
  expect(valuesNullId.UserId).toBe(actor);

  const valuesUndefId: Record<string, any> = {
    Name: uid('undef_id'),
    Application: 'web',
    ModelName: 'UserFilter',
    Condition: {},
    IsDefault: false,
  };
  await SF.validateUserFilterConstraint({}, { mode: 'create', values: valuesUndefId, current: undefined });
  expect(valuesUndefId.UserId).toBe(actor);

  // Update path no longer rewrites CreatedUid (BaseModel strips client writes).
  const valuesUpd: Record<string, any> = {
    Name: uid('cuid_fb'),
    Application: 'web',
    ModelName: 'UserFilter',
    IsDefault: false,
  };
  await SF.validateUserFilterConstraint(
    { CreatedUid: 'fromSelf', UserId: actor, Id: uid('row') },
    {
      mode: 'update',
      values: valuesUpd,
      current: { CreatedUid: 'fromSelf', UserId: actor, Application: 'web', ModelName: 'UserFilter', Name: valuesUpd.Name },
    }
  );
  expect(valuesUpd.CreatedUid).toBeUndefined();
});

test('AU9: deleting auth.User does not cascade UserFilter rows with CreatedUid', async () => {
  resetRequestContext();
  const companyId = await resolveAdminCompanyId();
  const creator = await createBaseUser(companyId);

  setIdentity(creator);
  const fav = await UserFilter.Create(
    {
      Name: uid('au9'),
      Application: 'web',
      ModelName: 'UserFilter',
      Condition: {},
      UserId: null,
    } as any,
    ['Id', 'CreatedUid'] as any
  );
  const favId = String((fav as any).Id);
  expect(String((fav as any).CreatedUid)).toBe(creator);

  // Soft-delete the creator. ManyToOneRef audit uids have no DB FK / CASCADE.
  await withModelContext(
    { activeCompanyId: companyId, enabledCompanyIds: [companyId] } as any,
    async () => {
      await User.sudo(() => User.DeleteById(creator), { hint: 'web.UserFilter.au9.deleteUser' });
    },
    { merge: false }
  );

  const still = await UserFilter.sudo(
    () => UserFilter.Browse(favId, ['Id', 'CreatedUid'] as any),
    { hint: 'web.UserFilter.au9.browse' }
  );
  expect(String((still as any).Id)).toBe(favId);
  expect(String((still as any).CreatedUid)).toBe(creator);

  await UserFilter.sudo(() => UserFilter.DeleteById(favId), { hint: 'web.UserFilter.au9.cleanup' });
});
