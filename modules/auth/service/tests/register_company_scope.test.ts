// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { withContext as withModelContext } from '@/core/service/api/context';
import User from '@/auth/service/models/user/user';
import Role from '@/auth/service/models/role';
import UserRole from '@/auth/service/models/user_role';
import { createServiceByModel } from '@/core/service/rpc';
import type Company from '@/base/service/models/company';

const CompanyService = createServiceByModel<typeof Company>('base.Company');

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
    fieldRuleMode: 'skip',
  };
  jsCtx.identity = {};

  delete (jsCtx as any)[Symbol.for('choysum.ctx.override')];
  delete (jsCtx as any)[Symbol.for('choysum.ctx.frozen')];
  delete (jsCtx as any)[RR_CACHE_KEY];
  delete (jsCtx as any)[FR_CACHE_KEY];
}

function setReq(patch: Record<string, any>): void {
  const jsCtx = ensureRequestContext();
  if (!jsCtx.req) jsCtx.req = {};
  Object.assign(jsCtx.req, patch);
}

function setupAllowlistForRegister(): void {
  setReq({
    depth: 0,
    recordRuleMode: 'allowlist',
    recordRuleAllow: [
      'auth.User:read',
      'auth.User:write',
      'auth.User:create',
      'User:read',
      'User:write',
      'User:create',

      'auth.Role:read',
      'Role:read',

      'auth.UserRole:read',
      'auth.UserRole:create',
      'UserRole:read',
      'UserRole:create',

      'auth.RoleMethodAccess:read',
      'auth.RoleMethodAccess:create',
      'RoleMethodAccess:read',
      'RoleMethodAccess:create',

      'meta.MetaModel:read',
      'meta.MetaService:read',
      'MetaModel:read',
      'MetaService:read',

      'base.Company:read',
      'Company:read',
    ],
  });
}

function uid(prefix: string): string {
  const xid = (globalThis as any).$choysum?.xid?.New?.();
  const value = typeof xid === 'string' && xid.trim() ? xid.trim() : String(Date.now());
  return `${prefix}_${value}`;
}

test('Register: anonymous signup assigns base.user inside main company scope', async () => {
  resetRequestContext();
  setupAllowlistForRegister();

  const username = uid('register_user');
  const email = `${uid('register_mail')}@example.com`;

  const userId = await withModelContext(
    {} as any,
    async () => {
      return await User.Register(
        {
          Username: username,
          Email: email,
          FirstName: 'Register',
          LastName: 'User',
        } as any,
        'password-123'
      );
    },
    { merge: false }
  );

  const mainCompanyId = await withModelContext(
    {} as any,
    async () => {
      const rows = await CompanyService.Search(['Code', '=', 'MAIN'] as any, { fields: ['Id'], limit: 1 } as any);
      return String((rows as any)?.[0]?.Id || '').trim();
    },
    { merge: false }
  );
  expect(mainCompanyId === '').toBe(false);

  const baseUserRoleId = await withModelContext(
    {} as any,
    async () => {
      const rows = await Role.Search(['Code', '=', 'base.user'] as any, { fields: ['Id'], limit: 1 } as any);
      return String((rows as any)?.[0]?.Id || '').trim();
    },
    { merge: false }
  );
  expect(baseUserRoleId === '').toBe(false);

  const snapshot = await withModelContext(
    { activeCompanyId: mainCompanyId, enabledCompanyIds: [mainCompanyId] } as any,
    async () => {
      const user = await User.Browse(userId, ['Id', 'CompanyId', 'CompanyIds', 'Preferences'] as any);
      const userRoles = await UserRole.Search(
        {
          And: [
            ['UserId', '=', userId],
            ['RoleId', '=', baseUserRoleId],
            ['CompanyId', '=', mainCompanyId],
          ],
        } as any,
        { fields: ['Id'], limit: 1 } as any
      );
      return { user, userRoles };
    },
    { merge: false }
  );

  expect((snapshot.user as any).CompanyId).toBe(mainCompanyId);
  expect(Array.isArray((snapshot.user as any).CompanyIds)).toBe(true);
  expect(((snapshot.user as any).CompanyIds || []).includes(mainCompanyId)).toBe(true);
  expect((snapshot.user as any).Preferences?.activeCompanyId).toBe(mainCompanyId);
  expect((snapshot.user as any).Preferences?.enabledCompanyIds).toEqual([mainCompanyId]);
  expect(snapshot.userRoles.length).toBe(1);
});
