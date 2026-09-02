// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { withContext as withModelContext } from '@/core/service/api/context';
import User from '@/auth/service/models/user/user';
import { parseJsonStringArray } from '@/auth/service/models/user/_authz_shared';
import Role from '@/auth/service/models/role';
import UserRole from '@/auth/service/models/user_role';
import RoleInheritance from '@/auth/service/models/role_inheritance';
import RoleMethodAccess from '@/auth/service/models/role_method_access';
import RoleUiResource from '@/auth/service/models/role_ui_resource';
import { createServiceByModel } from '@/core/service/rpc';
import type MetaServiceModel from '@/meta/service/models/service';
import type MetaUiResourceModel from '@/meta/service/models/ui_resource';
import type MetaUiResourceMenuRouteModel from '@/meta/service/models/ui_resource_menu_route';
import type MetaUiResourceRouteActionModel from '@/meta/service/models/ui_resource_route_action';
import { metaApplicationId, metaModelId } from './_meta_ids';
const MetaService = createServiceByModel<typeof MetaServiceModel>('meta.MetaService');
const MetaUiResource = createServiceByModel<typeof MetaUiResourceModel>('meta.MetaUiResource');
const MetaUiResourceMenuRoute = createServiceByModel<typeof MetaUiResourceMenuRouteModel>('meta.MetaUiResourceMenuRoute');
const MetaUiResourceRouteAction = createServiceByModel<typeof MetaUiResourceRouteActionModel>('meta.MetaUiResourceRouteAction');

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
    recordRuleMode: 'allowlist',
    recordRuleAllow: [
      // auth
      'auth.User:read',
      'auth.User:write',
      'auth.User:create',
      'auth.User:delete',
      'User:read',
      'User:write',
      'User:create',
      'User:delete',

      'auth.Role:read',
      'auth.Role:write',
      'auth.Role:create',
      'auth.Role:delete',
      'Role:read',
      'Role:write',
      'Role:create',
      'Role:delete',

      'auth.UserRole:read',
      'auth.UserRole:write',
      'auth.UserRole:create',
      'auth.UserRole:delete',
      'UserRole:read',
      'UserRole:write',
      'UserRole:create',
      'UserRole:delete',

      'auth.RoleInheritance:read',
      'auth.RoleInheritance:write',
      'auth.RoleInheritance:create',
      'auth.RoleInheritance:delete',
      'RoleInheritance:read',
      'RoleInheritance:write',
      'RoleInheritance:create',
      'RoleInheritance:delete',

      'auth.RoleMethodAccess:read',
      'auth.RoleMethodAccess:write',
      'auth.RoleMethodAccess:create',
      'auth.RoleMethodAccess:delete',
      'RoleMethodAccess:read',
      'RoleMethodAccess:write',
      'RoleMethodAccess:create',
      'RoleMethodAccess:delete',

      'auth.RoleUiResource:read',
      'auth.RoleUiResource:write',
      'auth.RoleUiResource:create',
      'auth.RoleUiResource:delete',
      'RoleUiResource:read',
      'RoleUiResource:write',
      'RoleUiResource:create',
      'RoleUiResource:delete',

      // meta
      'meta.MetaApplication:read',
      'MetaApplication:read',
      'meta.MetaModel:read',
      'meta.MetaService:read',
      'MetaModel:read',
      'MetaService:read',
    ],
  });
}

function disableAllowlist(): void {
  setReq({ recordRuleMode: '', recordRuleAllow: [] });
}

function busyWait(ms: number): void {
  const start = Date.now();
  while (Date.now() - start < ms) {
    // busy
  }
}

async function resolveModelId(appName: string, modelName: string): Promise<string> {
  const id = await metaModelId(appName, modelName);
  if (!id) throw new Error(`meta model not found: ${appName}.${modelName}`);
  return id;
}

async function resolveServiceId(modelId: string, serviceName: string): Promise<string> {
  const rows = await MetaService.Search(
    {
      And: [['ModelId', '=', modelId]],
    } as any,
    { fields: ['Id', 'Name'], limit: 5000 }
  );

  const target = String(serviceName || '')
    .trim()
    .toLowerCase();
  const hit = (rows || []).find(
    (r: any) =>
      String((r as any).Name || '')
        .trim()
        .toLowerCase() === target
  ) as any;

  const id = String(hit?.Id || '').trim();
  if (!id) throw new Error(`meta service not found: ${modelId}.${serviceName}`);
  return id;
}

async function resolveService(modelId: string, serviceName: string): Promise<{ id: string; name: string }> {
  const rows = await MetaService.Search(
    {
      And: [['ModelId', '=', modelId]],
    } as any,
    { fields: ['Id', 'Name'], limit: 5000 }
  );

  const target = String(serviceName || '')
    .trim()
    .toLowerCase();
  const hit = (rows || []).find(
    (r: any) =>
      String((r as any).Name || '')
        .trim()
        .toLowerCase() === target
  ) as any;
  const id = String(hit?.Id || '').trim();
  const name = String(hit?.Name || '').trim();
  if (!id || !name) throw new Error(`meta service not found: ${modelId}.${serviceName}`);
  return { id, name };
}

async function resolveApplicationId(appName: string): Promise<string> {
  const id = await metaApplicationId(appName);
  if (!id) throw new Error(`meta application not found: ${appName}`);
  return id;
}

async function createUiResource(input: {
  resourceId: string;
  type: 'ROUTE' | 'MENU' | 'ACTION';
  parentId?: string;
  requires?: string[];
  MetaApplicationId?: string | null;
}): Promise<void> {
  const parentDbId = input.parentId ? await resolveUiResourceDbId(input.parentId) : null;
  await MetaUiResource.Create(
    {
      Name: input.resourceId,
      Type: input.type,
      ParentId: parentDbId ? ({ Id: parentDbId } as any) : null,
      Requires: input.requires ?? [],
      MetaApplicationId: input.MetaApplicationId ?? null,
      Module: 'auth',
    } as any,
    ['Id'] as any
  );
}

async function createRoleUiResourceGrant(input: { roleId: string; resourceId: string; Mode?: 'allow' | 'deny' }): Promise<void> {
  const rows = await MetaUiResource.Search(
    {
      And: [['Name', '=', input.resourceId]],
    } as any,
    { fields: ['Id'], limit: 1 } as any
  );
  const uiResourceId = String((rows as any)?.[0]?.Id || '').trim();
  if (!uiResourceId) throw new Error(`MetaUiResource not found: ${input.resourceId}`);

  await RoleUiResource.Create(
    {
      RoleId: { Id: input.roleId } as any,
      MetaApplicationId: null,
      MetaUiResourceId: uiResourceId,
      Mode: input.Mode ?? 'allow',
    } as any,
    ['Id'] as any
  );
}

async function createRoleUiGrant(input: { roleId: string; resourceId?: string; irApplicationId?: string | null; Mode?: 'allow' | 'deny' }): Promise<void> {
  let uiResourceId: string | null = null;
  if (input.resourceId) {
    const rows = await MetaUiResource.Search(
      {
        And: [['Name', '=', input.resourceId]],
      } as any,
      { fields: ['Id'], limit: 1 } as any
    );
    uiResourceId = String((rows as any)?.[0]?.Id || '').trim() || null;
    if (!uiResourceId) throw new Error(`MetaUiResource not found: ${input.resourceId}`);
  }

  await RoleUiResource.Create(
    {
      RoleId: { Id: input.roleId } as any,
      MetaApplicationId: input.irApplicationId ?? null,
      MetaUiResourceId: uiResourceId,
      Mode: input.Mode ?? 'allow',
    } as any,
    ['Id'] as any
  );
}

async function resolveUiResourceDbId(resourceId: string): Promise<string> {
  const rows = await MetaUiResource.Search(
    {
      And: [['Name', '=', resourceId]],
    } as any,
    { fields: ['Id'], limit: 1 } as any
  );
  const id = String((rows as any)?.[0]?.Id || '').trim();
  if (!id) throw new Error(`MetaUiResource not found: ${resourceId}`);
  return id;
}

async function createMenuRouteRelation(input: { menuResourceId: string; routeResourceId: string }): Promise<void> {
  const menuId = await resolveUiResourceDbId(input.menuResourceId);
  const routeId = await resolveUiResourceDbId(input.routeResourceId);

  await MetaUiResourceMenuRoute.Create(
    {
      MenuUiResourceId: { Id: menuId } as any,
      RouteUiResourceId: { Id: routeId } as any,
    } as any,
    ['Id'] as any
  );
}

async function createRouteActionRelation(input: { routeResourceId: string; actionResourceId: string }): Promise<void> {
  const routeId = await resolveUiResourceDbId(input.routeResourceId);
  const actionId = await resolveUiResourceDbId(input.actionResourceId);

  await MetaUiResourceRouteAction.Create(
    {
      RouteUiResourceId: { Id: routeId } as any,
      ActionUiResourceId: { Id: actionId } as any,
    } as any,
    ['Id'] as any
  );
}

async function createUser(companyId: string): Promise<string> {
  const created = await User.Create(
    {
      Username: uid('u'),
      PasswordHash: 'test',
      FirstName: 'T',
      LastName: 'U',
      CompanyId: companyId,
      CompanyIds: [companyId],
      IsActive: true,
    } as any,
    ['Id'] as any
  );
  const id = String((created as any)?.Id || '').trim();
  if (!id) throw new Error('user create failed');
  return id;
}

async function resolveUserByUsername(username: string): Promise<{ id: string; companyId: string }> {
  const rows = await User.Search(
    {
      And: [['Username', '=', username]],
    } as any,
    { fields: ['Id', 'CompanyId'], limit: 1 } as any
  );
  const hit = (rows as any)?.[0] as any;
  const id = String(hit?.Id || '').trim();
  const companyId = String(hit?.CompanyId || '').trim();
  if (!id || !companyId) throw new Error(`bootstrap user not found: ${username}`);
  return { id, companyId };
}

async function resolveRoleByCode(code: string): Promise<string> {
  const rows = await Role.Search(
    {
      And: [['Code', '=', code]],
    } as any,
    { fields: ['Id'], limit: 1 } as any
  );
  const id = String((rows as any)?.[0]?.Id || '').trim();
  if (!id) throw new Error(`bootstrap role not found: ${code}`);
  return id;
}

async function createRole(codePrefix: string): Promise<{ id: string; code: string }> {
  const code = uid(codePrefix);
  const created = await Role.Create(
    {
      Name: uid('role'),
      Code: code,
      Description: 'test',
      IsActive: true,
      IsSystem: false,
    } as any,
    ['Id'] as any
  );
  return { id: created.Id, code };
}

test('PermissionState: roleAllow respects company scopes + inheritance', async () => {
  resetRequestContext();
  const c1 = { Id: uid('C1') };

  setupAllowlistForFixtures();
  const ret = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const userId = await createUser(c1.Id);
      setIdentity(userId);

      const rGlobal = await createRole('ROLE_G');
      const rParent = await createRole('ROLE_P');
      const rChild = await createRole('ROLE_C');

      // grant global role
      await UserRole.Create(
        {
          UserId: { Id: userId } as any,
          RoleId: { Id: rGlobal.id } as any,
          CompanyId: null as any,
        } as any,
        ['Id'] as any
      );

      // grant parent role scoped to c1
      await UserRole.Create(
        {
          UserId: { Id: userId } as any,
          RoleId: { Id: rParent.id } as any,
          CompanyId: c1.Id,
        } as any,
        ['Id'] as any
      );

      // parent -> child inheritance
      await RoleInheritance.Create(
        {
          ParentRoleId: { Id: rParent.id } as any,
          ChildRoleId: { Id: rChild.id } as any,
        } as any,
        ['Id'] as any
      );

      disableAllowlist();
      const ps = await User.GetPermissionState();
      return { ps, rGlobal, rParent, rChild };
    },
    { merge: false }
  );

  expect(ret.ps).toBeTruthy();
  expect(ret.ps.byCompany).toBeTruthy();
  expect(Array.isArray(ret.ps.byCompany['*']?.ui?.routes ?? [])).toBe(true);
  expect(Array.isArray(ret.ps.byCompany[c1.Id]?.ui?.menus ?? [])).toBe(true);
  expect(Array.isArray(ret.ps.byCompany[c1.Id]?.ui?.actions ?? [])).toBe(true);
});

test('PermissionState: rpc allow/deny emits method keys only on mixed service', async () => {
  resetRequestContext();
  const c1 = { Id: uid('C1') };

  setupAllowlistForFixtures();
  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const userId = await createUser(c1.Id);
      setIdentity(userId);

      const r = await createRole('ROLE_RPC');

      await UserRole.Create(
        {
          UserId: { Id: userId } as any,
          RoleId: { Id: r.id } as any,
          CompanyId: null as any,
        } as any,
        ['Id'] as any
      );

      const userModelId = await resolveModelId('auth', 'User');
      const browse = await resolveService(userModelId, 'browse');
      const logout = await resolveService(userModelId, 'logout');

      // allow browse, deny logout (same service => mixed)
      await RoleMethodAccess.Create(
        { RoleId: { Id: r.id } as any, MetaServiceId: browse.id, MetaModelId: null, MetaApplicationId: null, Mode: 'allow' } as any,
        ['Id'] as any
      );
      await RoleMethodAccess.Create(
        { RoleId: { Id: r.id } as any, MetaServiceId: logout.id, MetaModelId: null, MetaApplicationId: null, Mode: 'deny' } as any,
        ['Id'] as any
      );

      disableAllowlist();
      const ps = await User.GetPermissionState();
      return { ps, browseName: browse.name, logoutName: logout.name };
    },
    { merge: false }
  );

  const routes = out.ps.byCompany['*']?.ui?.routes ?? [];
  const menus = out.ps.byCompany['*']?.ui?.menus ?? [];
  const actions = out.ps.byCompany['*']?.ui?.actions ?? [];

  expect(Array.isArray(routes)).toBe(true);
  expect(Array.isArray(menus)).toBe(true);
  expect(Array.isArray(actions)).toBe(true);
});

test('PermissionState: model wildcard allow emits service wildcard', async () => {
  resetRequestContext();
  const c1 = { Id: uid('C1') };

  setupAllowlistForFixtures();
  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const userId = await createUser(c1.Id);
      setIdentity(userId);

      const r = await createRole('ROLE_MW');
      await UserRole.Create(
        {
          UserId: { Id: userId } as any,
          RoleId: { Id: r.id } as any,
          CompanyId: null as any,
        } as any,
        ['Id'] as any
      );

      const modelId = await resolveModelId('meta', 'MetaModel');

      await RoleMethodAccess.Create(
        { RoleId: { Id: r.id } as any, MetaServiceId: null, MetaModelId: modelId, MetaApplicationId: null, Mode: 'allow' } as any,
        ['Id'] as any
      );

      disableAllowlist();
      const ps = await User.GetPermissionState();
      return { ps };
    },
    { merge: false }
  );

  const routes = out.ps.byCompany['*']?.ui?.routes ?? [];
  expect(Array.isArray(routes)).toBe(true);
});

test('PermissionState: application wildcard allow emits service wildcard', async () => {
  resetRequestContext();
  const c1 = { Id: uid('C1') };

  setupAllowlistForFixtures();
  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const userId = await createUser(c1.Id);
      setIdentity(userId);

      const r = await createRole('ROLE_AW');
      await UserRole.Create({ UserId: { Id: userId } as any, RoleId: { Id: r.id } as any, CompanyId: null as any } as any, ['Id'] as any);

      const metaAppId = await resolveApplicationId('meta');
      await RoleMethodAccess.Create(
        { RoleId: { Id: r.id } as any, MetaServiceId: null, MetaModelId: null, MetaApplicationId: metaAppId, Mode: 'allow' } as any,
        ['Id'] as any
      );

      disableAllowlist();
      const ps = await User.GetPermissionState();
      return { ps };
    },
    { merge: false }
  );

  const routes = out.ps.byCompany['*']?.ui?.routes ?? [];
  expect(Array.isArray(routes)).toBe(true);
});

test('PermissionState: global wildcard deny suppresses allow output (deny-wins UX)', async () => {
  resetRequestContext();
  const c1 = { Id: uid('C1') };

  setupAllowlistForFixtures();
  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const userId = await createUser(c1.Id);
      setIdentity(userId);

      const r = await createRole('ROLE_GD');
      await UserRole.Create({ UserId: { Id: userId } as any, RoleId: { Id: r.id } as any, CompanyId: null as any } as any, ['Id'] as any);

      // deny everything
      await RoleMethodAccess.Create(
        { RoleId: { Id: r.id } as any, MetaServiceId: null, MetaModelId: null, MetaApplicationId: null, Mode: 'deny' } as any,
        ['Id'] as any
      );

      // add a narrower allow; should not surface in output for a denied service
      const metaAppId = await resolveApplicationId('meta');
      await RoleMethodAccess.Create(
        { RoleId: { Id: r.id } as any, MetaServiceId: null, MetaModelId: null, MetaApplicationId: metaAppId, Mode: 'allow' } as any,
        ['Id'] as any
      );

      disableAllowlist();
      const ps = await User.GetPermissionState();
      return { ps };
    },
    { merge: false }
  );

  const routes = out.ps.byCompany['*']?.ui?.routes ?? [];
  const menus = out.ps.byCompany['*']?.ui?.menus ?? [];

  expect(Array.isArray(routes)).toBe(true);
  expect(Array.isArray(menus)).toBe(true);
});

test('permStateVersion bumps when related tables updated', async () => {
  resetRequestContext();
  const c1 = { Id: uid('C1') };

  setupAllowlistForFixtures();
  const versions = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const userId = await createUser(c1.Id);
      setIdentity(userId);

      const r = await createRole('ROLE_V');
      await UserRole.Create(
        {
          UserId: { Id: userId } as any,
          RoleId: { Id: r.id } as any,
          CompanyId: null as any,
        } as any,
        ['Id'] as any
      );

      disableAllowlist();
      const v1 = (await User.GetPermissionState()).permStateVersion;

      // Force a version bump by adding one more RoleInheritance edge.
      setupAllowlistForFixtures();
      const r2 = await createRole('ROLE_V2');
      busyWait(3);
      await RoleInheritance.Create(
        {
          ParentRoleId: { Id: r.id } as any,
          ChildRoleId: { Id: r2.id } as any,
        } as any,
        ['Id'] as any
      );

      disableAllowlist();
      const v2 = (await User.GetPermissionState()).permStateVersion;

      return { v1, v2 };
    },
    { merge: false }
  );

  expect(typeof versions.v1).toBe('number');
  expect(typeof versions.v2).toBe('number');
  expect(Boolean(versions.v2 >= versions.v1)).toBe(true);
  expect(Boolean(versions.v2 !== 0)).toBe(true);
});

test("PermissionState: byCompany only contains '*' + enabledCompanyIds", async () => {
  resetRequestContext();
  const c1 = { Id: uid('C1') };
  const c2 = { Id: uid('C2') };

  setupAllowlistForFixtures();
  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id, c2.Id] } as any,
    async () => {
      const userId = await User.Create(
        {
          Username: uid('u'),
          PasswordHash: 'test',
          FirstName: 'T',
          LastName: 'U',
          CompanyId: c1.Id,
          CompanyIds: [c1.Id],
          IsActive: true,
        } as any,
        ['Id'] as any
      ).then(r => r.Id as string);
      setIdentity(userId);

      const rGlobal = await createRole('ROLE_GK');
      const r1 = await createRole('ROLE_C1');
      const r2 = await createRole('ROLE_C2');

      await UserRole.Create({ UserId: { Id: userId } as any, RoleId: { Id: rGlobal.id } as any, CompanyId: null as any } as any, ['Id'] as any);
      await UserRole.Create({ UserId: { Id: userId } as any, RoleId: { Id: r1.id } as any, CompanyId: c1.Id } as any, ['Id'] as any);
      await UserRole.Create({ UserId: { Id: userId } as any, RoleId: { Id: r2.id } as any, CompanyId: c2.Id } as any, ['Id'] as any);

      disableAllowlist();
      const ps = await User.GetPermissionState();
      return { ps };
    },
    { merge: false }
  );

  expect(out.ps).toBeTruthy();
  const keys = Object.keys(out.ps.byCompany || {});
  const allowed = new Set(['*', c1.Id, c2.Id]);
  expect(Boolean(keys.includes('*'))).toBe(true);
  expect(Boolean(keys.includes(c1.Id))).toBe(true);
  expect(Boolean(keys.includes(c2.Id))).toBe(true);
  expect(keys.every(k => allowed.has(k))).toBe(true);
});

test('PermissionState: maps UI resources from requires and backfills menu parents', async () => {
  resetRequestContext();
  const c1 = { Id: uid('C1') };

  setupAllowlistForFixtures();
  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const userId = await createUser(c1.Id);
      setIdentity(userId);

      const r = await createRole('ROLE_UI_CHAIN');
      await UserRole.Create(
        {
          UserId: { Id: userId } as any,
          RoleId: { Id: r.id } as any,
          CompanyId: null as any,
        } as any,
        ['Id'] as any
      );

      await createUiResource({
        resourceId: 'auth.menu.parent_interp',
        type: 'MENU',
        requires: ['rpc:/auth.User/DefinitelyMissingMethod'],
      });

      const userModelId = await resolveModelId('auth', 'User');
      const browse = await resolveService(userModelId, 'browse');

      await createUiResource({
        resourceId: 'auth.menu.user_list_interp',
        type: 'MENU',
        parentId: 'auth.menu.parent_interp',
        requires: [`rpc:/auth.User/${browse.name}`],
      });
      await createUiResource({
        resourceId: 'auth.route.user_list_interp',
        type: 'ROUTE',
        requires: [`rpc:/auth.User/${browse.name}`],
      });
      await createUiResource({
        resourceId: 'auth.action.user_export_interp',
        type: 'ACTION',
        requires: [`rpc:/auth.User/${browse.name}`],
      });

      await RoleMethodAccess.Create(
        {
          RoleId: { Id: r.id } as any,
          MetaServiceId: browse.id,
          MetaModelId: null,
          MetaApplicationId: null,
          Mode: 'allow',
        } as any,
        ['Id'] as any
      );

      disableAllowlist();
      const ps = await User.GetPermissionState();
      return { ps };
    },
    { merge: false }
  );

  const globalUi = out.ps.byCompany['*']?.ui ?? {};
  expect((globalUi.routes ?? []).includes('auth.route.user_list_interp')).toBe(true);
  expect((globalUi.actions ?? []).includes('auth.action.user_export_interp')).toBe(true);
  expect((globalUi.menus ?? []).includes('auth.menu.user_list_interp')).toBe(true);
  // parent has unmet requires but should be interpolated into output for tree integrity.
  expect((globalUi.menus ?? []).includes('auth.menu.parent_interp')).toBe(true);
});

test('PermissionState: emits UI wildcard when role has global wildcard allow', async () => {
  resetRequestContext();
  const c1 = { Id: uid('C1') };

  setupAllowlistForFixtures();
  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const userId = await createUser(c1.Id);
      setIdentity(userId);

      const r = await createRole('ROLE_UI_STAR');
      await UserRole.Create(
        {
          UserId: { Id: userId } as any,
          RoleId: { Id: r.id } as any,
          CompanyId: null as any,
        } as any,
        ['Id'] as any
      );

      await RoleMethodAccess.Create(
        {
          RoleId: { Id: r.id } as any,
          MetaServiceId: null,
          MetaModelId: null,
          MetaApplicationId: null,
          Mode: 'allow',
        } as any,
        ['Id'] as any
      );

      disableAllowlist();
      const ps = await User.GetPermissionState();
      return { ps };
    },
    { merge: false }
  );

  const globalUi = out.ps.byCompany['*']?.ui ?? {};
  expect(globalUi.routes).toEqual(['*']);
  expect(globalUi.menus).toEqual(['*']);
  expect(globalUi.actions).toEqual(['*']);
});

test('PermissionState: explicit RoleUiResource grant materializes UI whitelist without rpc allow', async () => {
  resetRequestContext();
  const c1 = { Id: uid('C1') };

  setupAllowlistForFixtures();
  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const userId = await createUser(c1.Id);
      setIdentity(userId);

      const r = await createRole('ROLE_UI_EXPLICIT');
      await UserRole.Create(
        {
          UserId: { Id: userId } as any,
          RoleId: { Id: r.id } as any,
          CompanyId: null as any,
        } as any,
        ['Id'] as any
      );

      await createUiResource({
        resourceId: 'auth.menu.explicit_parent',
        type: 'MENU',
        requires: ['rpc:/auth.User/DefinitelyMissingMethod'],
      });
      await createUiResource({
        resourceId: 'auth.menu.explicit_child',
        type: 'MENU',
        parentId: 'auth.menu.explicit_parent',
        requires: ['rpc:/auth.User/DefinitelyMissingMethod'],
      });
      await createUiResource({
        resourceId: 'auth.route.explicit_page',
        type: 'ROUTE',
        requires: ['rpc:/auth.User/DefinitelyMissingMethod'],
      });
      await createUiResource({
        resourceId: 'auth.action.explicit_edit',
        type: 'ACTION',
        requires: ['rpc:/auth.User/DefinitelyMissingMethod'],
      });

      await createMenuRouteRelation({
        menuResourceId: 'auth.menu.explicit_child',
        routeResourceId: 'auth.route.explicit_page',
      });
      await createRouteActionRelation({
        routeResourceId: 'auth.route.explicit_page',
        actionResourceId: 'auth.action.explicit_edit',
      });

      await createRoleUiResourceGrant({ roleId: r.id, resourceId: 'auth.route.explicit_page' });

      disableAllowlist();
      const actionRows = await MetaUiResource.Search(
        {
          And: [['Name', '=', 'auth.action.explicit_edit']],
        } as any,
        { fields: ['Id', 'Name', 'Type', 'ParentId', 'MetaApplicationId', 'Requires'], limit: 1 } as any
      );
      expect(parseJsonStringArray(((actionRows as any)?.[0] as any)?.Requires ?? ((actionRows as any)?.[0] as any)?.requires)).toEqual([
        'rpc:/auth.User/DefinitelyMissingMethod',
      ]);
      const ps = await User.GetPermissionState();
      return { ps };
    },
    { merge: false }
  );

  const globalUi = out.ps.byCompany['*']?.ui ?? {};
  expect((globalUi.routes ?? []).includes('auth.route.explicit_page')).toBe(true);
  // route grant should project its owning menu for navigation, then backfill menu ancestors.
  expect((globalUi.menus ?? []).includes('auth.menu.explicit_child')).toBe(true);
  expect((globalUi.menus ?? []).includes('auth.menu.explicit_parent')).toBe(true);
  // route_action only organizes projection and must not implicitly grant action visibility.
  expect((globalUi.actions ?? []).includes('auth.action.explicit_edit')).toBe(false);
});

test('PermissionState: explicit action grant projects owning route and menu chain', async () => {
  // UI-Option-A: granting the ACTION resource derives Method allow for its Requires,
  // so the action remains visible even when Method ACL does not list that RPC.
  resetRequestContext();
  const c1 = { Id: uid('C1') };

  setupAllowlistForFixtures();
  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const userId = await createUser(c1.Id);
      setIdentity(userId);

      const r = await createRole('ROLE_UI_EXPLICIT_ACTION');
      await UserRole.Create(
        {
          UserId: { Id: userId } as any,
          RoleId: { Id: r.id } as any,
          CompanyId: null as any,
        } as any,
        ['Id'] as any
      );

      await createUiResource({
        resourceId: 'auth.menu.explicit_action_parent',
        type: 'MENU',
        requires: ['rpc:/auth.User/DefinitelyMissingMethod'],
      });
      await createUiResource({
        resourceId: 'auth.menu.explicit_action_child',
        type: 'MENU',
        parentId: 'auth.menu.explicit_action_parent',
        requires: ['rpc:/auth.User/DefinitelyMissingMethod'],
      });
      await createUiResource({
        resourceId: 'auth.route.explicit_action_page',
        type: 'ROUTE',
        requires: ['rpc:/auth.User/DefinitelyMissingMethod'],
      });
      await createUiResource({
        resourceId: 'auth.action.explicit_action_edit',
        type: 'ACTION',
        requires: ['rpc:/auth.User/DefinitelyMissingMethod'],
      });

      await createMenuRouteRelation({
        menuResourceId: 'auth.menu.explicit_action_child',
        routeResourceId: 'auth.route.explicit_action_page',
      });
      await createRouteActionRelation({
        routeResourceId: 'auth.route.explicit_action_page',
        actionResourceId: 'auth.action.explicit_action_edit',
      });

      await createRoleUiResourceGrant({ roleId: r.id, resourceId: 'auth.action.explicit_action_edit' });

      disableAllowlist();
      const ps = await User.GetPermissionState();
      return { ps };
    },
    { merge: false }
  );

  const globalUi = out.ps.byCompany['*']?.ui ?? {};
  expect((globalUi.actions ?? []).includes('auth.action.explicit_action_edit')).toBe(true);
  // action grant should project owning route + menus for navigation continuity.
  expect((globalUi.routes ?? []).includes('auth.route.explicit_action_page')).toBe(true);
  expect((globalUi.menus ?? []).includes('auth.menu.explicit_action_child')).toBe(true);
  expect((globalUi.menus ?? []).includes('auth.menu.explicit_action_parent')).toBe(true);
});

test('PermissionState: explicit action allow + route deny must not project route/menu', async () => {
  resetRequestContext();
  const c1 = { Id: uid('C1') };

  setupAllowlistForFixtures();
  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const userId = await createUser(c1.Id);
      setIdentity(userId);

      const r = await createRole('ROLE_UI_ACTION_ALLOW_ROUTE_DENY');
      await UserRole.Create(
        {
          UserId: { Id: userId } as any,
          RoleId: { Id: r.id } as any,
          CompanyId: null as any,
        } as any,
        ['Id'] as any
      );

      await createUiResource({
        resourceId: 'auth.menu.explicit_action_deny_parent',
        type: 'MENU',
        requires: ['rpc:/auth.User/DefinitelyMissingMethod'],
      });
      await createUiResource({
        resourceId: 'auth.menu.explicit_action_deny_child',
        type: 'MENU',
        parentId: 'auth.menu.explicit_action_deny_parent',
        requires: ['rpc:/auth.User/DefinitelyMissingMethod'],
      });
      await createUiResource({
        resourceId: 'auth.route.explicit_action_deny_page',
        type: 'ROUTE',
        requires: ['rpc:/auth.User/DefinitelyMissingMethod'],
      });
      await createUiResource({
        resourceId: 'auth.action.explicit_action_deny_edit',
        type: 'ACTION',
        requires: ['rpc:/auth.User/DefinitelyMissingMethod'],
      });

      await createMenuRouteRelation({
        menuResourceId: 'auth.menu.explicit_action_deny_child',
        routeResourceId: 'auth.route.explicit_action_deny_page',
      });
      await createRouteActionRelation({
        routeResourceId: 'auth.route.explicit_action_deny_page',
        actionResourceId: 'auth.action.explicit_action_deny_edit',
      });

      await createRoleUiResourceGrant({ roleId: r.id, resourceId: 'auth.action.explicit_action_deny_edit', Mode: 'allow' });
      await createRoleUiResourceGrant({ roleId: r.id, resourceId: 'auth.route.explicit_action_deny_page', Mode: 'deny' });

      disableAllowlist();
      const ps = await User.GetPermissionState();
      return { ps };
    },
    { merge: false }
  );

  const globalUi = out.ps.byCompany['*']?.ui ?? {};
  expect((globalUi.actions ?? []).includes('auth.action.explicit_action_deny_edit')).toBe(true);
  expect((globalUi.routes ?? []).includes('auth.route.explicit_action_deny_page')).toBe(false);
  expect((globalUi.menus ?? []).includes('auth.menu.explicit_action_deny_child')).toBe(false);
  expect((globalUi.menus ?? []).includes('auth.menu.explicit_action_deny_parent')).toBe(false);
});

test('PermissionState: Method deny brakes explicit ACTION even when UI grants it (UI∧Method)', async () => {
  resetRequestContext();
  const c1 = { Id: uid('C1') };

  setupAllowlistForFixtures();
  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const userId = await createUser(c1.Id);
      setIdentity(userId);

      const r = await createRole('ROLE_UI_ACTION_METHOD_DENY');
      await UserRole.Create(
        {
          UserId: { Id: userId } as any,
          RoleId: { Id: r.id } as any,
          CompanyId: null as any,
        } as any,
        ['Id'] as any
      );

      const userModelId = await resolveModelId('auth', 'User');
      const browse = await resolveService(userModelId, 'browse');
      const requireKey = `rpc:/auth.User/${browse.name}`;

      await createUiResource({
        resourceId: 'auth.menu.method_deny_parent',
        type: 'MENU',
        requires: [requireKey],
      });
      await createUiResource({
        resourceId: 'auth.route.method_deny_page',
        type: 'ROUTE',
        requires: [requireKey],
      });
      await createUiResource({
        resourceId: 'auth.action.method_deny_edit',
        type: 'ACTION',
        requires: [requireKey],
      });
      await createMenuRouteRelation({
        menuResourceId: 'auth.menu.method_deny_parent',
        routeResourceId: 'auth.route.method_deny_page',
      });
      await createRouteActionRelation({
        routeResourceId: 'auth.route.method_deny_page',
        actionResourceId: 'auth.action.method_deny_edit',
      });

      await createRoleUiResourceGrant({ roleId: r.id, resourceId: 'auth.action.method_deny_edit', Mode: 'allow' });
      await RoleMethodAccess.Create(
        {
          RoleId: { Id: r.id } as any,
          MetaServiceId: browse.id,
          MetaModelId: null,
          MetaApplicationId: null,
          Mode: 'deny',
        } as any,
        ['Id'] as any
      );

      disableAllowlist();
      const ps = await User.GetPermissionState();
      return { ps };
    },
    { merge: false }
  );

  const globalUi = out.ps.byCompany['*']?.ui ?? {};
  expect((globalUi.actions ?? []).includes('auth.action.method_deny_edit')).toBe(false);
});

test('PermissionState: Method deny prevents ACTION wildcard under global UI allow', async () => {
  resetRequestContext();
  const c1 = { Id: uid('C1') };

  setupAllowlistForFixtures();
  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const userId = await createUser(c1.Id);
      setIdentity(userId);

      const r = await createRole('ROLE_UI_GLOBAL_METHOD_DENY');
      await UserRole.Create(
        {
          UserId: { Id: userId } as any,
          RoleId: { Id: r.id } as any,
          CompanyId: null as any,
        } as any,
        ['Id'] as any
      );

      const userModelId = await resolveModelId('auth', 'User');
      const browse = await resolveService(userModelId, 'browse');
      const requireKey = `rpc:/auth.User/${browse.name}`;

      await createUiResource({
        resourceId: 'auth.action.global_method_deny_edit',
        type: 'ACTION',
        requires: [requireKey],
      });
      await createUiResource({
        resourceId: 'auth.action.global_method_deny_empty',
        type: 'ACTION',
        requires: [],
      });

      // Global UI allow (empty app + resource) → routes/menus stay '*', but Method deny must brake actions.
      await RoleUiResource.Create(
        {
          RoleId: { Id: r.id } as any,
          MetaApplicationId: null as any,
          MetaUiResourceId: null as any,
          Mode: 'allow',
        } as any,
        ['Id'] as any
      );
      await RoleMethodAccess.Create(
        {
          RoleId: { Id: r.id } as any,
          MetaServiceId: browse.id,
          MetaModelId: null,
          MetaApplicationId: null,
          Mode: 'deny',
        } as any,
        ['Id'] as any
      );

      disableAllowlist();
      const ps = await User.GetPermissionState();
      return { ps };
    },
    { merge: false }
  );

  const globalUi = out.ps.byCompany['*']?.ui ?? {};
  expect(globalUi.routes).toEqual(['*']);
  expect(globalUi.menus).toEqual(['*']);
  expect(globalUi.actions).not.toEqual(['*']);
  expect((globalUi.actions ?? []).includes('auth.action.global_method_deny_edit')).toBe(false);
  expect((globalUi.actions ?? []).includes('auth.action.global_method_deny_empty')).toBe(true);
});

test('PermissionState smoke: declared resource -> persisted dictionary -> explicit grant -> ui whitelist', async () => {
  resetRequestContext();
  const c1 = { Id: uid('C1') };

  setupAllowlistForFixtures();
  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const userId = await createUser(c1.Id);
      setIdentity(userId);

      const r = await createRole('ROLE_UI_SMOKE');
      await UserRole.Create(
        {
          UserId: { Id: userId } as any,
          RoleId: { Id: r.id } as any,
          CompanyId: null as any,
        } as any,
        ['Id'] as any
      );

      // These IDs come from auth declaration code and should exist after build/persist.
      const routeId = 'auth.route.token_list';
      const menuId = 'auth.menu.token_list';
      const actionId = 'auth.action.token_edit';
      const fallbackActionId = 'auth.action.smoke_fallback';

      await createRoleUiResourceGrant({ roleId: r.id, resourceId: routeId });
      try {
        await createRoleUiResourceGrant({ roleId: r.id, resourceId: actionId });
      } catch {
        await createUiResource({ resourceId: fallbackActionId, type: 'ACTION', requires: [] });
        await createRoleUiResourceGrant({ roleId: r.id, resourceId: fallbackActionId });
      }

      disableAllowlist();
      const ps = await User.GetPermissionState();
      return { ps, routeId, menuId, actionId, fallbackActionId };
    },
    { merge: false }
  );

  const globalUi = out.ps.byCompany['*']?.ui ?? {};
  expect((globalUi.routes ?? []).includes(out.routeId)).toBe(true);
  expect((globalUi.menus ?? []).includes(out.menuId)).toBe(true);
  expect((globalUi.actions ?? []).includes(out.actionId) || (globalUi.actions ?? []).includes(out.fallbackActionId)).toBe(true);
});

test('PermissionState: application scope ui grant takes effect after refresh', async () => {
  resetRequestContext();
  const c1 = { Id: uid('C1') };

  setupAllowlistForFixtures();
  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const userId = await createUser(c1.Id);
      setIdentity(userId);

      const r = await createRole('ROLE_UI_SAVE_APP_PS');
      await UserRole.Create(
        {
          UserId: { Id: userId } as any,
          RoleId: { Id: r.id } as any,
          CompanyId: null as any,
        } as any,
        ['Id'] as any
      );

      const authAppId = await resolveApplicationId('auth');

      await createUiResource({
        resourceId: 'auth.route.save_app_visible',
        type: 'ROUTE',
        requires: ['rpc:/auth.User/DefinitelyMissingMethod'],
        MetaApplicationId: authAppId,
      });

      await createRoleUiGrant({ roleId: r.id, irApplicationId: authAppId });

      disableAllowlist();
      const ps = await User.GetPermissionState();
      return { ps };
    },
    { merge: false }
  );

  const globalUi = out.ps.byCompany['*']?.ui ?? {};
  expect((globalUi.routes ?? []).includes('auth.route.save_app_visible')).toBe(true);
});

test('PermissionState: global ui grant emits UI wildcard after refresh', async () => {
  resetRequestContext();
  const c1 = { Id: uid('C1') };

  setupAllowlistForFixtures();
  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const userId = await createUser(c1.Id);
      setIdentity(userId);

      const r = await createRole('ROLE_UI_SAVE_GLOBAL_PS');
      await UserRole.Create(
        {
          UserId: { Id: userId } as any,
          RoleId: { Id: r.id } as any,
          CompanyId: null as any,
        } as any,
        ['Id'] as any
      );

      await createRoleUiGrant({ roleId: r.id });

      disableAllowlist();
      const ps = await User.GetPermissionState();
      return { ps };
    },
    { merge: false }
  );

  const globalUi = out.ps.byCompany['*']?.ui ?? {};
  expect(globalUi.routes).toEqual(['*']);
  expect(globalUi.menus).toEqual(['*']);
  expect(globalUi.actions).toEqual(['*']);
});

test('PermissionState: explicit ui deny removes resource from whitelist', async () => {
  resetRequestContext();
  const c1 = { Id: uid('C1') };

  setupAllowlistForFixtures();
  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const userId = await createUser(c1.Id);
      setIdentity(userId);

      const r = await createRole('ROLE_UI_DENY_PS');
      await UserRole.Create(
        {
          UserId: { Id: userId } as any,
          RoleId: { Id: r.id } as any,
          CompanyId: null as any,
        } as any,
        ['Id'] as any
      );

      await createUiResource({
        resourceId: 'auth.route.deny_target',
        type: 'ROUTE',
        requires: ['rpc:/auth.User/DefinitelyMissingMethod'],
      });

      await createRoleUiResourceGrant({ roleId: r.id, resourceId: 'auth.route.deny_target', Mode: 'deny' });

      disableAllowlist();
      const ps = await User.GetPermissionState();
      return { ps };
    },
    { merge: false }
  );

  const globalUi = out.ps.byCompany['*']?.ui ?? {};
  expect((globalUi.routes ?? []).includes('auth.route.deny_target')).toBe(false);
});

test('PermissionState: explicit ui deny blocks wildcard materialization', async () => {
  resetRequestContext();
  const c1 = { Id: uid('C1') };

  setupAllowlistForFixtures();
  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const userId = await createUser(c1.Id);
      setIdentity(userId);

      const r = await createRole('ROLE_UI_DENY_WILDCARD');
      await UserRole.Create(
        {
          UserId: { Id: userId } as any,
          RoleId: { Id: r.id } as any,
          CompanyId: null as any,
        } as any,
        ['Id'] as any
      );

      await createUiResource({
        resourceId: 'auth.route.allow_visible',
        type: 'ROUTE',
        requires: ['rpc:/auth.User/DefinitelyMissingMethod'],
      });
      await createUiResource({
        resourceId: 'auth.route.deny_hidden',
        type: 'ROUTE',
        requires: ['rpc:/auth.User/DefinitelyMissingMethod'],
      });

      await createRoleUiGrant({ roleId: r.id, Mode: 'allow' });
      await createRoleUiResourceGrant({ roleId: r.id, resourceId: 'auth.route.deny_hidden', Mode: 'deny' });

      disableAllowlist();
      const ps = await User.GetPermissionState();
      return { ps };
    },
    { merge: false }
  );

  const globalUi = out.ps.byCompany['*']?.ui ?? {};
  expect(JSON.stringify(globalUi.routes ?? []) === JSON.stringify(['*'])).toBe(false);
  expect((globalUi.routes ?? []).includes('auth.route.allow_visible')).toBe(true);
  expect((globalUi.routes ?? []).includes('auth.route.deny_hidden')).toBe(false);
});

test('PermissionState: bootstrap sys admin gets UI wildcard from RoleUiResource global allow', async () => {
  resetRequestContext();
  setupAllowlistForFixtures();

  const out = await withModelContext({} as any, async () => {
    const admin = await resolveUserByUsername('admin');
    setIdentity(admin.id);

    disableAllowlist();
    const ps = await withModelContext(
      { activeCompanyId: admin.companyId, enabledCompanyIds: [admin.companyId] } as any,
      async () => await User.GetPermissionState(),
      { merge: false }
    );
    return { ps };
  });

  const globalUi = out.ps.byCompany['*']?.ui ?? {};
  expect(globalUi.routes).toEqual(['*']);
  expect(globalUi.menus).toEqual(['*']);
  expect(globalUi.actions).toEqual(['*']);
});

test('PermissionState: bootstrap base.user gets home baseline menu', async () => {
  resetRequestContext();
  const c1 = { Id: uid('C1') };

  setupAllowlistForFixtures();

  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const userId = await createUser(c1.Id);
      const baseUserRoleId = await resolveRoleByCode('base.user');
      await resolveUiResourceDbId('web.menu.home');
      await resolveUiResourceDbId('web.route.home');

      await UserRole.Create(
        {
          UserId: { Id: userId } as any,
          RoleId: { Id: baseUserRoleId } as any,
          CompanyId: c1.Id,
        } as any,
        ['Id'] as any
      );

      setIdentity(userId);
      disableAllowlist();

      const ps = await withModelContext({ activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any, async () => await User.GetPermissionState(), {
        merge: false,
      });
      return { ps };
    },
    { merge: false }
  );

  const companyUi = out.ps.byCompany[c1.Id]?.ui ?? {};
  expect((companyUi.menus ?? []).includes('web.menu.home')).toBe(true);
  expect((companyUi.routes ?? []).includes('web.route.home')).toBe(true);
});

test('PermissionState: global role ui grants appear under specific company scope (fallback from *)', async () => {
  resetRequestContext();
  const c1 = { Id: uid('C1') };

  setupAllowlistForFixtures();
  const out = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const userId = await createUser(c1.Id);
      setIdentity(userId);

      const r = await createRole('ROLE_STAR_FALLBACK');
      // global role assignment (CompanyId: null)
      await UserRole.Create(
        {
          UserId: { Id: userId } as any,
          RoleId: { Id: r.id } as any,
          CompanyId: null as any,
        } as any,
        ['Id'] as any
      );

      await createUiResource({
        resourceId: 'auth.route.star_fallback_route',
        type: 'ROUTE',
        requires: ['rpc:/auth.User/DefinitelyMissingMethod'],
      });

      await createRoleUiResourceGrant({ roleId: r.id, resourceId: 'auth.route.star_fallback_route' });

      disableAllowlist();
      const ps = await User.GetPermissionState();
      return { ps };
    },
    { merge: false }
  );

  // Global scope should list the route.
  const globalUi = out.ps.byCompany['*']?.ui ?? {};
  expect((globalUi.routes ?? []).includes('auth.route.star_fallback_route')).toBe(true);

  // Specific company scope must also inherit the global grant via '*' fallback.
  const companyUi = out.ps.byCompany[c1.Id]?.ui ?? {};
  expect((companyUi.routes ?? []).includes('auth.route.star_fallback_route')).toBe(true);
});
