// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { withContext as withModelContext } from '@/core/service/api/context';
import User from '@/auth/service/models/user';
import Role from '@/auth/service/models/role';
import UserRole from '@/auth/service/models/user_role';
import RoleRecordRule from '@/auth/service/models/role_record_rule';
import RoleFieldRule from '@/auth/service/models/role_field_rule';
import { createServiceByModel } from '@/core/service/rpc';
import type IrApplicationModel from '@/meta/service/models/ir_application';

const IrApplication = createServiceByModel<typeof IrApplicationModel>('meta.IrApplication');

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
  jsCtx.req = { depth: 0, fieldRuleMode: 'skip' };
  jsCtx.identity = {};
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

function setupAllowlistForFixtures(): void {
  setReq({
    depth: 0,
    fieldRuleMode: 'skip',
    recordRuleMode: 'allowlist',
    recordRuleAllow: [
      'auth.User:read',
      'auth.User:write',
      'auth.User:create',
      'User:read',
      'User:write',
      'User:create',

      'auth.Role:read',
      'auth.Role:write',
      'auth.Role:create',
      'Role:read',
      'Role:write',
      'Role:create',

      'auth.UserRole:read',
      'auth.UserRole:write',
      'auth.UserRole:create',
      'UserRole:read',
      'UserRole:write',
      'UserRole:create',

      'auth.RoleRecordRule:read',
      'auth.RoleRecordRule:write',
      'auth.RoleRecordRule:create',
      'RoleRecordRule:read',
      'RoleRecordRule:write',
      'RoleRecordRule:create',

      'auth.RoleFieldRule:read',
      'auth.RoleFieldRule:write',
      'auth.RoleFieldRule:create',
      'RoleFieldRule:read',
      'RoleFieldRule:write',
      'RoleFieldRule:create',

      'meta.IrApplication:read',
      'IrApplication:read',
    ],
  });
}

function disableAllowlist(): void {
  setReq({ recordRuleMode: '', recordRuleAllow: [], fieldRuleMode: '' });
}

function clearAuthzCaches(): void {
  const jsCtx = ensureRequestContext();
  delete (jsCtx as any)[RR_CACHE_KEY];
  delete (jsCtx as any)[FR_CACHE_KEY];
}

async function resolveRoleByCode(code: string): Promise<string> {
  const rows = await Role.Search({ And: [['Code', '=', code]] } as any, { fields: ['Id'], limit: 1 } as any);
  const id = String((rows as any)?.[0]?.Id || '').trim();
  if (!id) throw new Error(`bootstrap role not found: ${code}`);
  return id;
}

async function resolveUserByUsername(username: string): Promise<{ id: string; companyId: string }> {
  const rows = await User.Search({ And: [['Username', '=', username]] } as any, { fields: ['Id', 'CompanyId'], limit: 1 } as any);
  const hit = (rows as any)?.[0] as any;
  const id = String(hit?.Id || '').trim();
  const companyId = String(hit?.CompanyId || '').trim();
  if (!id || !companyId) throw new Error(`bootstrap user not found: ${username}`);
  return { id, companyId };
}

async function resolveApplicationId(name: string): Promise<string> {
  const rows = await IrApplication.Search({ And: [['Name', '=', name]] } as any, { fields: ['Id'], limit: 1 } as any);
  const id = String((rows as any)?.[0]?.Id || '').trim();
  if (!id) throw new Error(`IrApplication not found: ${name}`);
  return id;
}

async function createBareUser(companyId: string): Promise<string> {
  const created = await User.Create(
    {
      Username: uid('bare'),
      PasswordHash: 'test',
      FirstName: 'B',
      LastName: 'U',
      CompanyId: companyId,
      CompanyIds: [companyId],
      IsActive: true,
    } as any,
    ['Id'] as any
  );
  return String((created as any)?.Id || '').trim();
}

test('PR-C-2 gift pack: bootstrap seeds sys.admin global RR+FR and base.user app packs', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();

  const companyId = uid('CFIX');
  await withModelContext(
    { activeCompanyId: companyId, enabledCompanyIds: [companyId] } as any,
    async () => {
      const sysAdminRoleId = await resolveRoleByCode('sys.admin');
      const baseUserRoleId = await resolveRoleByCode('base.user');
      const authAppId = await resolveApplicationId('auth');
      const baseAppId = await resolveApplicationId('base');
      const metaAppId = await resolveApplicationId('meta');

      const sysAdminRr = await RoleRecordRule.Search(
        {
          And: [
            ['RoleId', '=', sysAdminRoleId],
            ['Kind', '=', 'grant'],
            ['IrModelId', 'is', null],
            ['IrApplicationId', 'is', null],
            ['PermRead', '=', true],
            ['PermWrite', '=', true],
            ['PermCreate', '=', true],
            ['PermDelete', '=', true],
          ],
        } as any,
        { fields: ['Id'], limit: 1 } as any
      );
      expect((sysAdminRr || []).length > 0).toBe(true);

      const sysAdminFr = await RoleFieldRule.Search(
        {
          And: [
            ['RoleId', '=', sysAdminRoleId],
            ['IrModelId', 'is', null],
            ['IrApplicationId', 'is', null],
            ['IrFieldId', 'is', null],
            ['PermRead', '=', 'allow'],
            ['PermWrite', '=', 'allow'],
          ],
        } as any,
        { fields: ['Id'], limit: 1 } as any
      );
      expect((sysAdminFr || []).length > 0).toBe(true);

      for (const appId of [authAppId, baseAppId, metaAppId]) {
        const rr = await RoleRecordRule.Search(
          {
            And: [
              ['RoleId', '=', baseUserRoleId],
              ['Kind', '=', 'grant'],
              ['IrApplicationId', '=', appId],
              ['IrModelId', 'is', null],
              ['PermRead', '=', true],
            ],
          } as any,
          { fields: ['Id'], limit: 1 } as any
        );
        expect((rr || []).length > 0).toBe(true);

        const fr = await RoleFieldRule.Search(
          {
            And: [
              ['RoleId', '=', baseUserRoleId],
              ['IrApplicationId', '=', appId],
              ['IrModelId', 'is', null],
              ['IrFieldId', 'is', null],
              ['PermRead', '=', 'allow'],
            ],
          } as any,
          { fields: ['Id'], limit: 1 } as any
        );
        expect((fr || []).length > 0).toBe(true);
      }
    },
    { merge: false }
  );
});

test('PR-C-2 gift pack: admin with sys.admin can read Company rows and columns', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();

  const companyId = uid('CADM');
  const out = await withModelContext(
    { activeCompanyId: companyId, enabledCompanyIds: [companyId] } as any,
    async () => {
      const admin = await resolveUserByUsername('admin');
      setIdentity(admin.id);
      clearAuthzCaches();
      disableAllowlist();

      return await withModelContext(
        { activeCompanyId: admin.companyId, enabledCompanyIds: [admin.companyId] } as any,
        async () => {
          const rr = await User.GetRecordRuleCondition('base.Company', 'read');
          const fr = await User.GetFieldRuleSpec('base.Company');
          const ps = await User.GetPermissionState();
          return { rr, fr, ps };
        },
        { merge: false }
      );
    },
    { merge: false }
  );

  expect(String((out.rr as any)?.kind || '')).not.toBe('false');
  expect(Array.isArray((out.fr as any)?.denyReadFields)).toBe(true);
  expect(((out.fr as any)?.denyReadFields || []).includes('Name')).toBe(false);
  expect(((out.fr as any)?.denyWriteFields || []).includes('Name')).toBe(false);
  expect(out.ps?.byCompany?.['*']?.ui?.routes).toEqual(['*']);
});

test('PR-C-2 gift pack: base.user can read Company columns but not write them', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();
  const c1 = { Id: uid('C1') };

  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const userId = await createBareUser(c1.Id);
      const baseUserRoleId = await resolveRoleByCode('base.user');
      await UserRole.Create(
        {
          UserId: { Id: userId } as any,
          RoleId: { Id: baseUserRoleId } as any,
          CompanyId: null as any,
        } as any,
        ['Id'] as any
      );

      setIdentity(userId);
      clearAuthzCaches();
      disableAllowlist();

      return await withModelContext(
        { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
        async () => {
          const rrRead = await User.GetRecordRuleCondition('base.Company', 'read');
          const rrWrite = await User.GetRecordRuleCondition('base.Company', 'write');
          const fr = await User.GetFieldRuleSpec('base.Company');
          return { rrRead, rrWrite, fr };
        },
        { merge: false }
      );
    },
    { merge: false }
  );

  expect(String((out.rrRead as any)?.kind || '')).not.toBe('false');
  expect(String((out.rrWrite as any)?.kind || '')).toBe('false');
  expect(((out.fr as any)?.denyReadFields || []).includes('Name')).toBe(false);
  expect(((out.fr as any)?.denyWriteFields || []).includes('Name')).toBe(true);
});

test('PR-C-2 gift pack: role without packs remains deny-default empty', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();
  const c1 = { Id: uid('C1') };

  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const userId = await createBareUser(c1.Id);
      const bareRole = await Role.Create(
        {
          Name: uid('bare_role'),
          Code: uid('BARE'),
          Description: 'no gift pack',
          IsActive: true,
          IsSystem: false,
        } as any,
        ['Id'] as any
      );
      await UserRole.Create(
        {
          UserId: { Id: userId } as any,
          RoleId: { Id: bareRole.Id } as any,
          CompanyId: null as any,
        } as any,
        ['Id'] as any
      );

      setIdentity(userId);
      clearAuthzCaches();
      disableAllowlist();

      return await withModelContext(
        { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
        async () => {
          const rr = await User.GetRecordRuleCondition('base.Company', 'read');
          const fr = await User.GetFieldRuleSpec('base.Company');
          return { rr, fr };
        },
        { merge: false }
      );
    },
    { merge: false }
  );

  expect(String((out.rr as any)?.kind || '')).toBe('false');
  expect(String((out.rr as any)?.reason || '')).toContain('no_grant');
  expect(((out.fr as any)?.denyReadFields || []).includes('Name')).toBe(true);
  expect(String((out.fr as any)?.reason || '')).toContain('deny_by_default');
});
