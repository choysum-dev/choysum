// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Role from '@/auth/service/models/role';
import User from '@/auth/service/models/user';
import UserRole from '@/auth/service/models/user_role';
import { withContext as withModelContext } from '@/core/service/api/context';
import { ChoysumError } from '@/core/service/error';
import { createServiceByModel } from '@/core/service/rpc';
import MetaModelData from '@/meta/service/models/model_data';
import SavedFilter from '@/web/service/models/saved_filter';
import { resolveEffectiveModelId } from '@/web/service/models/_resolve_effective_model';

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
      'web.SavedFilter:read',
      'web.SavedFilter:write',
      'web.SavedFilter:create',
      'web.SavedFilter:delete',
      'SavedFilter:read',
      'SavedFilter:write',
      'SavedFilter:create',
      'SavedFilter:delete',
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

function metaModel(): any {
  return createServiceByModel('meta.MetaModel');
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
  const fd = await metaModel().Search(
    {
      And: [
        ['Application', '=', 'web'],
        ['Name', '=', 'FieldDefault'],
      ],
    } as any,
    { fields: ['Id'], limit: 1 } as any
  );
  const as = await metaModel().Search(
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

test('SavedFilter CRUD + IsDefault exclusivity + visibility', async () => {
  resetRequestContext();
  const actor = uid('sf_actor');
  setIdentity(actor);

  const modelId = await resolveEffectiveModelId('web', 'SavedFilter');
  expect(modelId).toBeTruthy();

  const nameA = uid('fav_a');
  const privateFav = await SavedFilter.Create(
    {
      Name: nameA,
      Application: 'web',
      ModelName: 'SavedFilter',
      Condition: { And: [['Active', '=', true]] },
      IsDefault: true,
    } as any,
    ['Id', 'UserId', 'ModelId', 'CreateUid', 'IsDefault'] as any
  );
  expect(String((privateFav as any).UserId)).toBe(actor);
  expect(String((privateFav as any).ModelId)).toBe(modelId);
  expect(String((privateFav as any).CreateUid)).toBe(actor);
  expect((privateFav as any).IsDefault).toBe(true);

  const nameB = uid('fav_b');
  const second = await SavedFilter.Create(
    {
      Name: nameB,
      Application: 'web',
      ModelName: 'SavedFilter',
      Condition: {},
      IsDefault: true,
    } as any,
    ['Id', 'IsDefault'] as any
  );
  expect((second as any).IsDefault).toBe(true);
  const firstAgain = await SavedFilter.Browse(String((privateFav as any).Id), ['IsDefault'] as any);
  expect((firstAgain as any).IsDefault).toBe(false);

  const sharedName = uid('shared');
  const shared = await SavedFilter.Create(
    {
      Name: sharedName,
      Application: 'web',
      ModelName: 'SavedFilter',
      Condition: {},
      UserId: null,
      IsDefault: false,
    } as any,
    ['Id', 'UserId', 'CreateUid'] as any
  );
  expect((shared as any).UserId == null || (shared as any).UserId === '').toBe(true);

  const other = uid('sf_other');
  setIdentity(other);
  const visible = await SavedFilter.Search(
    {
      And: [
        ['Application', '=', 'web'],
        ['ModelName', '=', 'SavedFilter'],
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
  await SavedFilter.DeleteById(String((privateFav as any).Id));
  await SavedFilter.DeleteById(String((second as any).Id));
  await SavedFilter.DeleteById(String((shared as any).Id));
});

test('SavedFilter rejects Create without effective MetaModel', async () => {
  resetRequestContext();
  const actor = uid('sf_noeff');
  setIdentity(actor);
  await expectCode(
    async () =>
      SavedFilter.Create(
        {
          Name: uid('gone'),
          Application: 'no_such_app',
          ModelName: 'NoSuchModel',
          Condition: {},
        } as any,
        ['Id'] as any
      ),
    'FailedPrecondition',
    'No effective model'
  );
});

test('SavedFilter rejects foreign UserId on Create', async () => {
  resetRequestContext();
  const actor = uid('sf_owner');
  const other = uid('sf_victim');
  setIdentity(actor);
  await expectCode(
    async () =>
      SavedFilter.Create(
        {
          Name: uid('steal'),
          Application: 'web',
          ModelName: 'SavedFilter',
          Condition: {},
          UserId: other,
        } as any,
        ['Id'] as any
      ),
    'PermissionDenied',
    'another user'
  );
});

test('SavedFilter private and shared IsDefault can coexist', async () => {
  resetRequestContext();
  const actor = uid('sf_bucket');
  setIdentity(actor);
  const privateName = uid('priv_def');
  const sharedName = uid('shared_def');
  const priv = await SavedFilter.Create(
    {
      Name: privateName,
      Application: 'web',
      ModelName: 'SavedFilter',
      Condition: {},
      IsDefault: true,
    } as any,
    ['Id', 'IsDefault', 'UserId'] as any
  );
  const shared = await SavedFilter.Create(
    {
      Name: sharedName,
      Application: 'web',
      ModelName: 'SavedFilter',
      Condition: {},
      UserId: null,
      IsDefault: true,
    } as any,
    ['Id', 'IsDefault', 'UserId'] as any
  );
  expect((priv as any).IsDefault).toBe(true);
  expect((shared as any).IsDefault).toBe(true);
  const privAgain = await SavedFilter.Browse(String((priv as any).Id), ['IsDefault'] as any);
  expect((privAgain as any).IsDefault).toBe(true);
  await SavedFilter.DeleteById(String((priv as any).Id));
  await SavedFilter.DeleteById(String((shared as any).Id));
});

test('web bootstrap seeds SavedFilter authz packs (RMA/RFR/RR)', async () => {
  resetRequestContext();
  const expected: Array<{ name: string; model: string }> = [
    { name: 'rma_base_user_web_saved_filter_search', model: 'RoleMethodAccess' },
    { name: 'rma_base_user_web_saved_filter_browse', model: 'RoleMethodAccess' },
    { name: 'rma_base_user_web_saved_filter_create', model: 'RoleMethodAccess' },
    { name: 'rma_base_user_web_saved_filter_update', model: 'RoleMethodAccess' },
    { name: 'rma_base_user_web_saved_filter_update_by_id', model: 'RoleMethodAccess' },
    { name: 'rma_base_user_web_saved_filter_delete', model: 'RoleMethodAccess' },
    { name: 'rma_base_user_web_saved_filter_delete_by_id', model: 'RoleMethodAccess' },
    { name: 'rfr_base_user_web_saved_filter_rw', model: 'RoleFieldRule' },
    { name: 'rrr_base_user_web_saved_filter_rc', model: 'RoleRecordRule' },
    { name: 'rrr_base_user_web_saved_filter_wd_private', model: 'RoleRecordRule' },
    { name: 'rrr_base_user_web_saved_filter_wd_shared', model: 'RoleRecordRule' },
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
    expect(Array.isArray(rows) && rows.length === 1, `missing web.${name}`).toBe(true);
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
  const shared = await SavedFilter.Create(
    {
      Name: uid('sf11'),
      Application: 'web',
      ModelName: 'SavedFilter',
      Condition: {},
      UserId: null,
    } as any,
    ['Id', 'CreateUid'] as any
  );

  disableAllowlist();
  delete (ensureRequestContext() as any)[RR_CACHE_KEY];

  setIdentity(stranger);
  // Write/delete on another user's shared row: targets fail the WD record-rule expr → violation.
  await expectCode(
    async () => SavedFilter.UpdateById(String((shared as any).Id), { Name: uid('hijack') } as any, ['Id'] as any),
    'record_rule_violation',
    'violates record rule'
  );
  await expectCode(
    async () => SavedFilter.DeleteById(String((shared as any).Id)),
    'record_rule_violation',
    'violates record rule'
  );

  setIdentity(creator);
  await SavedFilter.UpdateById(String((shared as any).Id), { Name: uid('ok') } as any, ['Id'] as any);
  const deleted = await SavedFilter.DeleteById(String((shared as any).Id));
  expect(deleted).toBe(1);
});

test('SavedFilter rejects Create without authentication', async () => {
  resetRequestContext();
  setIdentity(undefined);
  await expectCode(
    async () =>
      SavedFilter.Create(
        {
          Name: uid('anon'),
          Application: 'web',
          ModelName: 'SavedFilter',
          Condition: {},
        } as any,
        ['Id'] as any
      ),
    'PermissionDenied',
    'Authentication required'
  );
});

test('SavedFilter rejects duplicate Name in the same ownership bucket', async () => {
  resetRequestContext();
  const actor = uid('sf_dup');
  setIdentity(actor);
  const name = uid('same_name');
  const first = await SavedFilter.Create(
    {
      Name: name,
      Application: 'web',
      ModelName: 'SavedFilter',
      Condition: {},
    } as any,
    ['Id'] as any
  );
  await expectCode(
    async () =>
      SavedFilter.Create(
        {
          Name: name,
          Application: 'web',
          ModelName: 'SavedFilter',
          Condition: {},
        } as any,
        ['Id'] as any
      ),
    'AlreadyExists',
    'already exists'
  );
  await SavedFilter.DeleteById(String((first as any).Id));
});

test('SavedFilter fills Create defaults (UserId/IsDefault/Active/Condition)', async () => {
  resetRequestContext();
  const actor = uid('sf_defaults');
  setIdentity(actor);
  const created = await SavedFilter.Create(
    {
      Name: uid('fills'),
      Application: 'web',
      ModelName: 'SavedFilter',
    } as any,
    ['Id', 'UserId', 'IsDefault', 'Active', 'Condition', 'CreateUid'] as any
  );
  expect(String((created as any).UserId)).toBe(actor);
  expect((created as any).IsDefault).toBe(false);
  expect((created as any).Active).toBe(true);
  expect((created as any).Condition || {}).toEqual({});
  expect(String((created as any).CreateUid)).toBe(actor);
  await SavedFilter.DeleteById(String((created as any).Id));
});

test('SavedFilter Update keeps CreateUid immutable and normalizes UserId', async () => {
  resetRequestContext();
  const actor = uid('sf_upd');
  setIdentity(actor);
  const created = await SavedFilter.Create(
    {
      Name: uid('upd'),
      Application: 'web',
      ModelName: 'SavedFilter',
      Condition: {},
    } as any,
    ['Id', 'CreateUid', 'UserId'] as any
  );
  const createUid = String((created as any).CreateUid);
  const updated = await SavedFilter.UpdateById(
    String((created as any).Id),
    { CreateUid: uid('hijack_uid'), UserId: '' } as any,
    ['Id', 'CreateUid', 'UserId'] as any
  );
  expect(String((updated as any).CreateUid)).toBe(createUid);
  expect((updated as any).UserId == null || (updated as any).UserId === '').toBe(true);

  await expectCode(
    async () =>
      SavedFilter.UpdateById(String((created as any).Id), { UserId: uid('other') } as any, ['Id'] as any),
    'PermissionDenied',
    'another user'
  );
  await SavedFilter.DeleteById(String((created as any).Id));
});

test('SavedFilter private IsDefault update clears other defaults with exceptId', async () => {
  resetRequestContext();
  const actor = uid('sf_except');
  setIdentity(actor);
  const a = await SavedFilter.Create(
    {
      Name: uid('def_a'),
      Application: 'web',
      ModelName: 'SavedFilter',
      Condition: {},
      IsDefault: true,
    } as any,
    ['Id', 'IsDefault'] as any
  );
  const b = await SavedFilter.Create(
    {
      Name: uid('def_b'),
      Application: 'web',
      ModelName: 'SavedFilter',
      Condition: {},
      IsDefault: false,
    } as any,
    ['Id', 'IsDefault'] as any
  );
  await SavedFilter.UpdateById(String((b as any).Id), { IsDefault: true } as any, ['Id', 'IsDefault'] as any);
  const aAgain = await SavedFilter.Browse(String((a as any).Id), ['IsDefault'] as any);
  expect((aAgain as any).IsDefault).toBe(false);
  await SavedFilter.DeleteById(String((a as any).Id));
  await SavedFilter.DeleteById(String((b as any).Id));
});

test('SavedFilter rejects Create missing Name/Application/ModelName', async () => {
  resetRequestContext();
  setIdentity(uid('sf_req'));
  await expectCode(
    async () =>
      SavedFilter.Create(
        {
          Name: '',
          Application: 'web',
          ModelName: 'SavedFilter',
          Condition: {},
        } as any,
        ['Id'] as any
      ),
    'InvalidArgument',
    'required'
  );
});

test('SavedFilter Update reads unchanged fields from current via mergedField', async () => {
  resetRequestContext();
  const actor = uid('sf_merged');
  setIdentity(actor);
  const created = await SavedFilter.Create(
    {
      Name: uid('merged'),
      Application: 'web',
      ModelName: 'SavedFilter',
      Condition: { And: [['A', '=', 1]] },
      Active: true,
    } as any,
    ['Id', 'Name', 'Condition'] as any
  );
  // Touch only Active so Name/Application/ModelName resolve from current.
  const updated = await SavedFilter.UpdateById(
    String((created as any).Id),
    { Active: false } as any,
    ['Id', 'Name', 'Active', 'Application', 'ModelName'] as any
  );
  expect(String((updated as any).Name)).toBe(String((created as any).Name));
  expect((updated as any).Active).toBe(false);
  await SavedFilter.DeleteById(String((created as any).Id));
});

test('SavedFilter shared-default clear PermissionDenied when stranger cannot replace', async () => {
  resetRequestContext();
  const companyId = await resolveAdminCompanyId();
  const creator = await createBaseUser(companyId);
  const stranger = await createBaseUser(companyId);

  setIdentity(creator);
  const shared = await SavedFilter.Create(
    {
      Name: uid('shared_def_owner'),
      Application: 'web',
      ModelName: 'SavedFilter',
      Condition: {},
      UserId: null,
      IsDefault: true,
    } as any,
    ['Id', 'CreateUid', 'IsDefault'] as any
  );
  expect((shared as any).IsDefault).toBe(true);

  disableAllowlist();
  delete (ensureRequestContext() as any)[RR_CACHE_KEY];

  setIdentity(stranger);
  let caught: any;
  try {
    await SavedFilter.Create(
      {
        Name: uid('shared_def_stranger'),
        Application: 'web',
        ModelName: 'SavedFilter',
        Condition: {},
        UserId: null,
        IsDefault: true,
      } as any,
      ['Id'] as any
    );
  } catch (e) {
    caught = e;
  }
  if (!caught) {
    throw new Error('expected shared-default replacement to fail');
  }
  const codes = collectErrorCodes(caught);
  const msg = String((caught as any)?.message || '');
  if (!codes.includes('PermissionDenied') || !msg.includes("another user's shared default")) {
    throw new Error(`expected PermissionDenied with shared-default message, got codes=${codes.join(',')} msg=${msg}`);
  }

  // Creator's shared default must remain the sole default.
  setIdentity(creator);
  const again = await SavedFilter.Browse(String((shared as any).Id), ['IsDefault'] as any);
  expect((again as any).IsDefault).toBe(true);
  await SavedFilter.DeleteById(String((shared as any).Id));
});

test('SavedFilter whitespace UserId normalizes to shared null', async () => {
  resetRequestContext();
  const actor = uid('sf_ws');
  setIdentity(actor);
  const created = await SavedFilter.Create(
    {
      Name: uid('ws_uid'),
      Application: 'web',
      ModelName: 'SavedFilter',
      Condition: {},
      UserId: '   ',
      IsDefault: true,
    } as any,
    ['Id', 'UserId', 'IsDefault'] as any
  );
  expect((created as any).UserId == null || (created as any).UserId === '').toBe(true);
  expect((created as any).IsDefault).toBe(true);
  await SavedFilter.DeleteById(String((created as any).Id));
});

test('SavedFilter creator can replace own shared default', async () => {
  resetRequestContext();
  const actor = uid('sf_shared_ok');
  setIdentity(actor);
  const first = await SavedFilter.Create(
    {
      Name: uid('shared_a'),
      Application: 'web',
      ModelName: 'SavedFilter',
      Condition: {},
      UserId: null,
      IsDefault: true,
    } as any,
    ['Id', 'IsDefault', 'CreateUid'] as any
  );
  expect((first as any).IsDefault).toBe(true);
  const second = await SavedFilter.Create(
    {
      Name: uid('shared_b'),
      Application: 'web',
      ModelName: 'SavedFilter',
      Condition: {},
      UserId: null,
      IsDefault: true,
    } as any,
    ['Id', 'IsDefault'] as any
  );
  expect((second as any).IsDefault).toBe(true);
  const firstAgain = await SavedFilter.Browse(String((first as any).Id), ['IsDefault'] as any);
  expect((firstAgain as any).IsDefault).toBe(false);
  await SavedFilter.DeleteById(String((first as any).Id));
  await SavedFilter.DeleteById(String((second as any).Id));
});

test('SavedFilter Update without IsDefault still clears peers when row is default', async () => {
  resetRequestContext();
  const actor = uid('sf_upd_def');
  setIdentity(actor);
  const a = await SavedFilter.Create(
    {
      Name: uid('upd_def_a'),
      Application: 'web',
      ModelName: 'SavedFilter',
      Condition: {},
      IsDefault: true,
    } as any,
    ['Id', 'IsDefault', 'Name'] as any
  );
  const b = await SavedFilter.Create(
    {
      Name: uid('upd_def_b'),
      Application: 'web',
      ModelName: 'SavedFilter',
      Condition: {},
      IsDefault: false,
    } as any,
    ['Id', 'IsDefault'] as any
  );
  // Promote b via Update; a should clear. Then rename a while it is no longer default
  // and rename the still-default b without sending IsDefault (mergedField path).
  await SavedFilter.UpdateById(String((b as any).Id), { IsDefault: true } as any, ['Id', 'IsDefault'] as any);
  const newName = uid('renamed_def');
  await SavedFilter.UpdateById(String((b as any).Id), { Name: newName } as any, ['Id', 'Name', 'IsDefault'] as any);
  const bAgain = await SavedFilter.Browse(String((b as any).Id), ['Name', 'IsDefault'] as any);
  expect(String((bAgain as any).Name)).toBe(newName);
  expect((bAgain as any).IsDefault).toBe(true);
  const aAgain = await SavedFilter.Browse(String((a as any).Id), ['IsDefault'] as any);
  expect((aAgain as any).IsDefault).toBe(false);
  await SavedFilter.DeleteById(String((a as any).Id));
  await SavedFilter.DeleteById(String((b as any).Id));
});

test('SavedFilter Update rename collision uses exceptId uniqueness', async () => {
  resetRequestContext();
  const actor = uid('sf_rename');
  setIdentity(actor);
  const nameA = uid('rename_a');
  const nameB = uid('rename_b');
  const a = await SavedFilter.Create(
    {
      Name: nameA,
      Application: 'web',
      ModelName: 'SavedFilter',
      Condition: {},
    } as any,
    ['Id', 'Name'] as any
  );
  const b = await SavedFilter.Create(
    {
      Name: nameB,
      Application: 'web',
      ModelName: 'SavedFilter',
      Condition: {},
    } as any,
    ['Id', 'Name'] as any
  );
  await expectCode(
    async () => SavedFilter.UpdateById(String((b as any).Id), { Name: nameA } as any, ['Id'] as any),
    'AlreadyExists',
    'already exists'
  );
  // Same-name update on self must succeed (exceptId excludes current).
  const self = await SavedFilter.UpdateById(String((a as any).Id), { Name: nameA, Active: true } as any, [
    'Id',
    'Name',
  ] as any);
  expect(String((self as any).Name)).toBe(nameA);
  await SavedFilter.DeleteById(String((a as any).Id));
  await SavedFilter.DeleteById(String((b as any).Id));
});
